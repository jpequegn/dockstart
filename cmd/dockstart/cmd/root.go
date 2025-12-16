// Package cmd contains the CLI commands for dockstart.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	// Version is set at build time
	Version = "dev"

	// Flags
	dryRun bool
	force  bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "dockstart <path>",
	Short: "Generate Docker development environment files",
	Long: `dockstart analyzes a project directory and generates Docker
development environment files including:

  - .devcontainer/devcontainer.json
  - .devcontainer/docker-compose.yml
  - .devcontainer/Dockerfile

It detects the project's language (Node.js, Go, Python, Rust) and
any services (PostgreSQL, Redis) to create an optimized dev environment.`,
	Args: cobra.MaximumNArgs(1),
	RunE: run,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview output without writing files")
	rootCmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files")
}

func run(cmd *cobra.Command, args []string) error {
	// Default to current directory if no path provided
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Verify path exists and is a directory
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", absPath)
		}
		return fmt.Errorf("cannot access path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", absPath)
	}

	fmt.Printf("📂 Analyzing %s...\n", absPath)

	if dryRun {
		fmt.Println("🔍 Dry run mode - no files will be written")
	}

	// TODO: Implement detection logic (Issue #3, #4)
	fmt.Println("   ⏳ Detection not yet implemented")

	// TODO: Implement generation logic (Issue #5, #6, #7)
	fmt.Println("   ⏳ Generation not yet implemented")

	return nil
}
