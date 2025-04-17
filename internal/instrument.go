// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

const (
	TargetPkg         = "main"
	TargetFunc        = "main"
	TrampolineName    = "Trampoline"
	HookName          = "Hook"
	AwsConfigHookName = "AwsConfigHook"
	AwsClientHookName = "AwsClientHook"
)

func loadAst(filePath string) *dst.File {
	name := filepath.Base(filePath)
	fset := token.NewFileSet()
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	astFile, err := parser.ParseFile(fset, name, file, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	dec := decorator.NewDecorator(fset)
	dstFile, err := dec.DecorateFile(astFile)
	if err != nil {
		panic(err)
	}
	return dstFile
}

func storeAst(filePath string, ast *dst.File) {
	f, err := os.Create(filePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	r := decorator.NewRestorer()
	err = r.Fprint(f, ast)
	if err != nil {
		panic(err)
	}
}

func newTrampolineFunc() *dst.FuncDecl {
	trampoline := &dst.FuncDecl{
		Name: &dst.Ident{
			Name: TrampolineName,
		},
		Type: &dst.FuncType{},
		Body: &dst.BlockStmt{
			List: []dst.Stmt{newFuncCall(HookName)},
		},
	}
	return trampoline
}

func newFuncCall(target string) *dst.ExprStmt {
	return &dst.ExprStmt{
		X: &dst.CallExpr{
			Fun: &dst.Ident{
				Name: target,
			},
		},
	}
}

func newHookFunc(name string) *dst.FuncDecl {
	return &dst.FuncDecl{
		Name: &dst.Ident{
			Name: name,
		},
		Type: &dst.FuncType{
			Params: &dst.FieldList{},
		},
	}
}

func newAwsConfigHookFunc() *dst.FuncDecl {
	return &dst.FuncDecl{
		Name: &dst.Ident{
			Name: AwsConfigHookName,
		},
		Type: &dst.FuncType{
			Params: &dst.FieldList{
				List: []*dst.Field{
					{
						Names: []*dst.Ident{
							{
								Name: "cfg",
							},
						},
						Type: &dst.InterfaceType{
							Methods: &dst.FieldList{},
						},
					},
				},
			},
		},
	}
}

func newAwsClientHookFunc() *dst.FuncDecl {
	return &dst.FuncDecl{
		Name: &dst.Ident{
			Name: AwsClientHookName,
		},
		Type: &dst.FuncType{
			Params: &dst.FieldList{
				List: []*dst.Field{
					{
						Names: []*dst.Ident{
							{
								Name: "client",
							},
						},
						Type: &dst.InterfaceType{
							Methods: &dst.FieldList{},
						},
					},
				},
			},
		},
	}
}

func findOutputDir(args []string) string {
	for i, arg := range args {
		if arg == "-o" {
			return filepath.Dir(args[i+1])
		}
	}
	return ""
}

// isAwsConfigCall checks if an expression is an AWS config loading call
func isAwsConfigCall(expr dst.Expr) bool {
	callExpr, ok := expr.(*dst.CallExpr)
	if !ok {
		return false
	}

	// Check for LoadDefaultConfig selector expression
	selectorExpr, ok := callExpr.Fun.(*dst.SelectorExpr)
	if !ok {
		return false
	}

	// Common AWS config loading methods
	awsConfigMethods := []string{"LoadDefaultConfig", "FromEnv", "New"}

	for _, method := range awsConfigMethods {
		if selectorExpr.Sel.Name == method {
			// Check if it's from the aws-sdk-go-v2/config package
			xident, ok := selectorExpr.X.(*dst.Ident)
			if ok && (xident.Name == "awsConfig" || xident.Name == "config" || xident.Name == "awscfg") {
				return true
			}
		}
	}

	return false
}

// isAwsClientCreation checks if an expression is an AWS client creation
func isAwsClientCreation(expr dst.Expr) bool {
	callExpr, ok := expr.(*dst.CallExpr)
	if !ok {
		return false
	}

	// Check for NewFromConfig selector expression
	selectorExpr, ok := callExpr.Fun.(*dst.SelectorExpr)
	if !ok {
		return false
	}

	// Common AWS client creation patterns
	awsClientMethods := []string{
		"NewFromConfig",
		"New",
		"NewClient",
		"NewClientWithOptions",
	}

	for _, method := range awsClientMethods {
		if selectorExpr.Sel.Name == method {
			// For NewFromConfig, we can be pretty sure it's AWS
			if method == "NewFromConfig" {
				return true
			}

			// For other methods, check if it has at least one argument
			// (typically the config object)
			if len(callExpr.Args) > 0 {
				// If the argument is a reference to a config variable, assume it's an AWS client
				// This is a best-effort check and might need refinement
				if argIdent, ok := callExpr.Args[0].(*dst.Ident); ok {
					if argIdent.Name == "cfg" || argIdent.Name == "config" || argIdent.Name == "awsConfig" {
						return true
					}
				}
			}
		}
	}

	return false
}

func rewriteAst(astFile *dst.File, fn *dst.FuncDecl) {
	fmt.Printf("Instrumenting function: %s\n", fn.Name.Name)

	// Check for AWS SDK usage within the function body
	// Use our enhanced AWS instrumentation which adds direct instrumentation calls
	foundAwsConfig := instrumentAwsConfig(astFile, fn)

	// Process AWS client creation with hook only for now
	foundAwsClient := false

	// Process the function body to find AWS SDK client calls
	dst.Inspect(fn.Body, func(node dst.Node) bool {
		// Check for assignment statements with AWS SDK calls
		assignStmt, ok := node.(*dst.AssignStmt)
		if !ok {
			return true
		}

		// Check right side of assignment for AWS client calls
		for _, rhs := range assignStmt.Rhs {
			if isAwsClientCreation(rhs) {
				foundAwsClient = true

				// Inject AWS client hook call after assignment
				if len(assignStmt.Lhs) > 0 {
					lhsIdent, ok := assignStmt.Lhs[0].(*dst.Ident)
					if ok {
						hookCall := &dst.ExprStmt{
							X: &dst.CallExpr{
								Fun: &dst.Ident{
									Name: AwsClientHookName,
								},
								Args: []dst.Expr{
									&dst.Ident{
										Name: lhsIdent.Name,
									},
								},
							},
						}

						// Find the index of this statement in the parent block to insert after it
						for i, stmt := range fn.Body.List {
							if stmt == assignStmt {
								if i+1 < len(fn.Body.List) {
									fn.Body.List = append(fn.Body.List[:i+1], append([]dst.Stmt{hookCall}, fn.Body.List[i+1:]...)...)
								} else {
									fn.Body.List = append(fn.Body.List, hookCall)
								}
								break
							}
						}
					}
				}
			}
		}

		return true
	})

	// Ensure we add AWS imports if we found AWS usage
	if foundAwsConfig || foundAwsClient {
		ensureAwsImports(astFile)
	}

	// Insert a call to the trampoline function at the beginning of the function
	callToTrampoline := newFuncCall(TrampolineName)
	fn.Body.List = append([]dst.Stmt{callToTrampoline}, fn.Body.List...)

	// Add the trampoline function to the AST
	astFile.Decls = append(astFile.Decls, newTrampolineFunc())

	// Add the hook function to the AST
	hook := newHookFunc(HookName)
	astFile.Decls = append(astFile.Decls, hook)

	// Add AWS hook functions if needed
	if foundAwsConfig {
		awsConfigHook := newAwsConfigHookFunc()
		astFile.Decls = append(astFile.Decls, awsConfigHook)
	}

	if foundAwsClient {
		awsClientHook := newAwsClientHookFunc()
		astFile.Decls = append(astFile.Decls, awsClientHook)
	}
}

func Instrument(args []string) []string {
	for i, arg := range args {
		if strings.HasSuffix(arg, ".go") {
			astFile := loadAst(arg)
			for _, decl := range astFile.Decls {
				// Find target function
				fn, ok := decl.(*dst.FuncDecl)
				if !ok {
					continue
				}
				if fn.Name.Name != TargetFunc {
					continue
				}
				// Instrument the function
				rewriteAst(astFile, fn)
				// Update the compilation command
				modified := filepath.Join(findOutputDir(args), "modified.go")
				storeAst(modified, astFile)
				args[i] = modified
				break
			}
		}
	}

	// Remove the -complete flag because we added a body-less hook function
	// and the compiler will complain about it
	for i, arg := range args {
		if arg == "-complete" {
			args = append(args[:i], args[i+1:]...)
			break
		}
	}
	return args
}
