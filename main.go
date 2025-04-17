// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/open-telemetry/opentelemetry-go-compile-instrumentation/internal"
)

func runCmd(args ...string) error {
	path := args[0]
	args = args[1:]
	cmd := exec.Command(path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func isCompilePackage(args []string, pkg string) bool {
	if len(args) < 2 {
		return false
	}
	if !strings.HasSuffix(args[0], "compile") &&
		!strings.HasSuffix(args[0], "compile.exe") {
		return false
	}
	for i, arg := range args[1:] {
		if arg == "-p" {
			if i+1 < len(args) && args[i+2] == pkg {
				return true
			}
		}
	}
	return false
}

func main() {
	args := os.Args[1:]
	
	// Check if we're running in initialization mode
	if len(args) > 0 && args[0] == "init" {
		if len(args) < 2 {
			fmt.Println("Usage: otel init <target-directory>")
			os.Exit(1)
		}
		targetDir := args[1]
		
		// Ensure project has necessary imports
		if err := internal.EnsureInstrumentationImports(targetDir); err != nil {
			panic("failed to initialize instrumentation: " + err.Error())
		}
		
		fmt.Println("Instrumentation initialized successfully")
		return
	}
	
	// Normal instrumentation mode
	if isCompilePackage(args, internal.TargetPkg) {
		// It's the compile command, intercept it and inject hook code
		args = internal.Instrument(args)
	}
	
	err := runCmd(args...)
	if err != nil {
		panic("failed to run command: " + err.Error())
	}
}
