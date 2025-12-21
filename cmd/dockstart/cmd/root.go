// Package cmd contains the CLI commands for dockstart.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jpequegn/dockstart/internal/detector"
	"github.com/jpequegn/dockstart/internal/generator"
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

	// Get project name from directory name
	projectName := filepath.Base(absPath)
	fmt.Printf("📂 Analyzing %s...\n", absPath)

	if dryRun {
		fmt.Println("🔍 Dry run mode - no files will be written")
	}

	// Step 1: Detect project language and services
	fmt.Println("\n🔍 Detecting project configuration...")
	registry := detector.NewRegistry()
	detection, err := registry.DetectPrimary(absPath)
	if err != nil {
		return fmt.Errorf("detection failed: %w", err)
	}

	if detection == nil {
		fmt.Println("   ⚠️  No supported language detected")
		fmt.Println("   Supported: Node.js (package.json), Go (go.mod), Python (pyproject.toml/requirements.txt), Rust (Cargo.toml)")
		return nil
	}

	fmt.Printf("   ✅ Detected: %s %s (confidence: %.0f%%)\n",
		detection.Language, detection.Version, detection.Confidence*100)

	if len(detection.Services) > 0 {
		fmt.Printf("   📦 Services: %v\n", detection.Services)
	}

	// Step 2: Generate devcontainer.json
	fmt.Println("\n📝 Generating devcontainer.json...")
	gen := generator.NewDevcontainerGenerator()

	if dryRun {
		// Preview mode - just show what would be generated
		content, err := gen.GenerateContent(detection, projectName)
		if err != nil {
			return fmt.Errorf("generation failed: %w", err)
		}
		fmt.Println("\n--- .devcontainer/devcontainer.json ---")
		fmt.Println(string(content))
		fmt.Println("--- end ---")
	} else {
		// Check if files already exist
		devcontainerPath := filepath.Join(absPath, ".devcontainer", "devcontainer.json")
		if _, err := os.Stat(devcontainerPath); err == nil && !force {
			return fmt.Errorf("devcontainer.json already exists. Use --force to overwrite")
		}

		// Generate and write the file
		if err := gen.Generate(detection, absPath, projectName); err != nil {
			return fmt.Errorf("generation failed: %w", err)
		}
		fmt.Println("   ✅ Created .devcontainer/devcontainer.json")
	}

	// Step 3: Generate docker-compose.yml (when services or sidecars are detected)
	needsCompose := len(detection.Services) > 0 || detection.NeedsMetrics() || detection.NeedsWorker() || detection.NeedsFileProcessor()
	if needsCompose {
		fmt.Println("\n📝 Generating docker-compose.yml...")
		composeGen := generator.NewComposeGenerator()

		if dryRun {
			content, err := composeGen.GenerateContent(detection, projectName)
			if err != nil {
				return fmt.Errorf("compose generation failed: %w", err)
			}
			fmt.Println("\n--- .devcontainer/docker-compose.yml ---")
			fmt.Println(string(content))
			fmt.Println("--- end ---")
		} else {
			composePath := filepath.Join(absPath, ".devcontainer", "docker-compose.yml")
			if _, err := os.Stat(composePath); err == nil && !force {
				return fmt.Errorf("docker-compose.yml already exists. Use --force to overwrite")
			}

			if err := composeGen.Generate(detection, absPath, projectName); err != nil {
				return fmt.Errorf("compose generation failed: %w", err)
			}
			fmt.Println("   ✅ Created .devcontainer/docker-compose.yml")
		}
	}

	// Step 3b: Generate metrics sidecar files (Prometheus + Grafana config)
	metricsGen := generator.NewMetricsSidecarGenerator()
	if metricsGen.ShouldGenerate(detection) {
		fmt.Println("\n📝 Generating metrics stack configuration...")
		if !dryRun {
			if err := metricsGen.Generate(detection, absPath, projectName); err != nil {
				return fmt.Errorf("metrics sidecar generation failed: %w", err)
			}
			fmt.Println("   ✅ Created .devcontainer/prometheus/prometheus.yml")
			fmt.Println("   ✅ Created .devcontainer/grafana/provisioning/datasources/prometheus.yml")
			fmt.Println("   ✅ Created .devcontainer/grafana/provisioning/dashboards/provider.yml")
			fmt.Println("   ✅ Created .devcontainer/grafana/provisioning/dashboards/app-metrics.json")
		} else {
			fmt.Println("   📊 Would create Prometheus and Grafana configuration files")
		}
	}

	// Step 4: Generate Dockerfile
	fmt.Println("\n📝 Generating Dockerfile...")
	dockerfileGen := generator.NewDockerfileGenerator()

	if dryRun {
		content, err := dockerfileGen.GenerateContent(detection, projectName)
		if err != nil {
			return fmt.Errorf("dockerfile generation failed: %w", err)
		}
		fmt.Println("\n--- .devcontainer/Dockerfile ---")
		fmt.Println(string(content))
		fmt.Println("--- end ---")
	} else {
		dockerfilePath := filepath.Join(absPath, ".devcontainer", "Dockerfile")
		if _, err := os.Stat(dockerfilePath); err == nil && !force {
			return fmt.Errorf("Dockerfile already exists. Use --force to overwrite")
		}

		if err := dockerfileGen.Generate(detection, absPath, projectName); err != nil {
			return fmt.Errorf("dockerfile generation failed: %w", err)
		}
		fmt.Println("   ✅ Created .devcontainer/Dockerfile")
	}

	fmt.Println("\n✨ Done!")
	return nil
}
