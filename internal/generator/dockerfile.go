package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jpequegn/dockstart/internal/models"
)

// DockerfileConfig holds the configuration for generating a Dockerfile.
type DockerfileConfig struct {
	// Name is the project name (used in comments)
	Name string

	// BaseImage is the Docker base image (e.g., "node:20", "golang:1.23")
	BaseImage string

	// PackageManager is the OS package manager command (apt-get, apk, etc.)
	PackageManager string

	// CacheCleanup is the command to clean package cache (varies by OS)
	CacheCleanup string

	// PostInstall is optional language-specific setup commands
	PostInstall string
}

// DockerfileGenerator generates Dockerfile files.
type DockerfileGenerator struct{}

// NewDockerfileGenerator creates a new dockerfile generator.
func NewDockerfileGenerator() *DockerfileGenerator {
	return &DockerfileGenerator{}
}

// Generate creates a Dockerfile from a Detection.
// The file is written to .devcontainer/Dockerfile.
func (g *DockerfileGenerator) Generate(detection *models.Detection, projectPath string, projectName string) error {
	config := g.buildConfig(detection, projectName)

	// Create .devcontainer directory (may already exist)
	devcontainerDir := filepath.Join(projectPath, ".devcontainer")
	if err := os.MkdirAll(devcontainerDir, 0755); err != nil {
		return fmt.Errorf("failed to create .devcontainer directory: %w", err)
	}

	// Generate Dockerfile content
	content, err := g.render(config)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Write to file
	outputPath := filepath.Join(devcontainerDir, "Dockerfile")
	if err := os.WriteFile(outputPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	return nil
}

// GenerateContent returns the generated Dockerfile content without writing to disk.
// Useful for dry-run mode.
func (g *DockerfileGenerator) GenerateContent(detection *models.Detection, projectName string) ([]byte, error) {
	config := g.buildConfig(detection, projectName)
	return g.render(config)
}

// buildConfig creates a DockerfileConfig from a Detection.
func (g *DockerfileGenerator) buildConfig(detection *models.Detection, projectName string) *DockerfileConfig {
	config := &DockerfileConfig{
		Name: projectName,
	}

	// Language-specific configuration
	// Using official Docker Hub images for each language
	switch detection.Language {
	case "node":
		// Node.js - using official node image (Debian-based)
		config.BaseImage = fmt.Sprintf("node:%s", detection.Version)
		config.PackageManager = "apt-get"
		config.CacheCleanup = "/var/lib/apt/lists/*"
		// npm is already available in the node image

	case "go":
		// Go - using official golang image (Debian-based)
		config.BaseImage = fmt.Sprintf("golang:%s", detection.Version)
		config.PackageManager = "apt-get"
		config.CacheCleanup = "/var/lib/apt/lists/*"
		// Go tools like gopls will be installed by VS Code extension

	case "python":
		// Python - using official python image (Debian-based)
		config.BaseImage = fmt.Sprintf("python:%s", detection.Version)
		config.PackageManager = "apt-get"
		config.CacheCleanup = "/var/lib/apt/lists/*"
		// pip is already available in the python image
		config.PostInstall = "RUN pip install --upgrade pip"

	case "rust":
		// Rust - using official rust image (Debian-based)
		config.BaseImage = fmt.Sprintf("rust:%s", detection.Version)
		config.PackageManager = "apt-get"
		config.CacheCleanup = "/var/lib/apt/lists/*"
		// rustup, cargo, and rustc are already available
		config.PostInstall = "RUN rustup component add rustfmt clippy"

	default:
		// Default to Ubuntu for unknown languages
		config.BaseImage = "ubuntu:22.04"
		config.PackageManager = "apt-get"
		config.CacheCleanup = "/var/lib/apt/lists/*"
	}

	return config
}

// render executes the template with the given config.
func (g *DockerfileGenerator) render(config *DockerfileConfig) ([]byte, error) {
	tmpl, err := loadTemplate("Dockerfile.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}
