// Package main is the entry point for the dockstart CLI tool.
// dockstart analyzes a project and generates Docker development environment files.
package main

import (
	"os"

	"github.com/jpequegn/dockstart/cmd/dockstart/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
