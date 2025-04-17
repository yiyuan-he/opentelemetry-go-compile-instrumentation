// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"github.com/dave/dst"
	"go/token"
)

const (
	AWS_OTELAWS_IMPORT_PATH = "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
)

// createAwsDirectInstrumentationCall creates a direct call to otelaws.AppendMiddlewares
// This generates the equivalent of: otelaws.AppendMiddlewares(&cfg.APIOptions)
func createAwsDirectInstrumentationCall(configVarName string) *dst.ExprStmt {
	return &dst.ExprStmt{
		X: &dst.CallExpr{
			Fun: &dst.SelectorExpr{
				X: &dst.Ident{
					Name: "otelaws",
				},
				Sel: &dst.Ident{
					Name: "AppendMiddlewares",
				},
			},
			Args: []dst.Expr{
				&dst.UnaryExpr{
					Op: token.AND,
					X: &dst.SelectorExpr{
						X: &dst.Ident{
							Name: configVarName,
						},
						Sel: &dst.Ident{
							Name: "APIOptions",
						},
					},
				},
			},
		},
	}
}

// ensureAwsImports adds the necessary imports for AWS SDK instrumentation
func ensureAwsImports(file *dst.File) {
	// Check if otelaws is already imported
	otelImportExists := false
	for _, imp := range file.Imports {
		if imp.Path != nil && imp.Path.Value == `"`+AWS_OTELAWS_IMPORT_PATH+`"` {
			otelImportExists = true
			break
		}
	}

	// Add the import if needed
	if !otelImportExists {
		// Add import at the beginning of the import block
		file.Imports = append([]*dst.ImportSpec{
			{
				Path: &dst.BasicLit{
					Kind:  token.STRING,
					Value: `"` + AWS_OTELAWS_IMPORT_PATH + `"`,
				},
			},
		}, file.Imports...)

		// Make sure imports are actually declared in the file
		hasImportDecl := false
		for _, decl := range file.Decls {
			if genDecl, ok := decl.(*dst.GenDecl); ok && genDecl.Tok == token.IMPORT {
				hasImportDecl = true
				genDecl.Specs = append(genDecl.Specs, &dst.ImportSpec{
					Path: &dst.BasicLit{
						Kind:  token.STRING,
						Value: `"` + AWS_OTELAWS_IMPORT_PATH + `"`,
					},
				})
				break
			}
		}

		// If no import declaration exists, create one
		if !hasImportDecl {
			importDecl := &dst.GenDecl{
				Tok: token.IMPORT,
				Specs: []dst.Spec{
					&dst.ImportSpec{
						Path: &dst.BasicLit{
							Kind:  token.STRING,
							Value: `"` + AWS_OTELAWS_IMPORT_PATH + `"`,
						},
					},
				},
			}
			file.Decls = append([]dst.Decl{importDecl}, file.Decls...)
		}
	}
}

// instrumentAwsConfig adds direct instrumentation call after AWS config loading
func instrumentAwsConfig(astFile *dst.File, fn *dst.FuncDecl) bool {
	foundAwsConfig := false

	// Process the function body to find AWS SDK config calls
	dst.Inspect(fn.Body, func(node dst.Node) bool {
		// Check for assignment statements with AWS SDK calls
		assignStmt, ok := node.(*dst.AssignStmt)
		if !ok {
			return true
		}

		// Check right side of assignment for AWS calls
		for _, rhs := range assignStmt.Rhs {
			if isAwsConfigCall(rhs) {
				foundAwsConfig = true

				// Inject both the hook call and direct instrumentation
				if len(assignStmt.Lhs) > 0 {
					lhsIdent, ok := assignStmt.Lhs[0].(*dst.Ident)
					if ok {
						// Hook call for debugging info
						hookCall := &dst.ExprStmt{
							X: &dst.CallExpr{
								Fun: &dst.Ident{
									Name: AwsConfigHookName,
								},
								Args: []dst.Expr{
									&dst.Ident{
										Name: lhsIdent.Name,
									},
								},
							},
						}

						// Direct instrumentation call
						directCall := createAwsDirectInstrumentationCall(lhsIdent.Name)

						// Find the index of this statement in the parent block to insert after it
						for i, stmt := range fn.Body.List {
							if stmt == assignStmt {
								// Add the hook call and direct instrumentation call after the assignment
								if i+1 < len(fn.Body.List) {
									fn.Body.List = append(fn.Body.List[:i+1], append([]dst.Stmt{hookCall, directCall}, fn.Body.List[i+1:]...)...)
								} else {
									fn.Body.List = append(fn.Body.List, hookCall, directCall)
								}
								break
							}
						}

						// Add the necessary imports
						ensureAwsImports(astFile)
					}
				}
			}
		}

		return true
	})

	return foundAwsConfig
}
