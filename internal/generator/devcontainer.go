package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jpequegn/dockstart/internal/models"
)

// DevcontainerConfig holds the configuration for generating devcontainer.json.
type DevcontainerConfig struct {
	// Name is the project/container name
	Name string

	// Image is the Docker image to use (when not using Compose)
	Image string

	// UseCompose indicates whether to use docker-compose.yml
	UseCompose bool

	// Extensions is a list of VS Code extension IDs
	Extensions []string

	// ForwardPorts is a list of ports to forward from the container
	ForwardPorts []int

	// PostCreateCommand is the command to run after container creation
	PostCreateCommand string

	// RemoteUser is the user to run as in the container
	RemoteUser string
}

// DevcontainerGenerator generates devcontainer.json files.
type DevcontainerGenerator struct{}

// NewDevcontainerGenerator creates a new devcontainer generator.
func NewDevcontainerGenerator() *DevcontainerGenerator {
	return &DevcontainerGenerator{}
}

// Generate creates a devcontainer.json file from a Detection.
func (g *DevcontainerGenerator) Generate(detection *models.Detection, projectPath string, projectName string) error {
	config := g.buildConfig(detection, projectName)

	// Create .devcontainer directory
	devcontainerDir := filepath.Join(projectPath, ".devcontainer")
	if err := os.MkdirAll(devcontainerDir, 0755); err != nil {
		return fmt.Errorf("failed to create .devcontainer directory: %w", err)
	}

	// Generate devcontainer.json content
	content, err := g.render(config)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Write to file
	outputPath := filepath.Join(devcontainerDir, "devcontainer.json")
	if err := os.WriteFile(outputPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write devcontainer.json: %w", err)
	}

	return nil
}

// GenerateContent returns the generated devcontainer.json content without writing to disk.
// Useful for dry-run mode.
func (g *DevcontainerGenerator) GenerateContent(detection *models.Detection, projectName string) ([]byte, error) {
	config := g.buildConfig(detection, projectName)
	return g.render(config)
}

// buildConfig creates a DevcontainerConfig from a Detection.
func (g *DevcontainerGenerator) buildConfig(detection *models.Detection, projectName string) *DevcontainerConfig {
	config := &DevcontainerConfig{
		Name:       projectName,
		RemoteUser: "root", // Default, will be overridden per language
	}

	// Determine if we need docker-compose (when services are detected)
	config.UseCompose = len(detection.Services) > 0

	// Language-specific configuration
	switch detection.Language {
	case "node":
		config.Image = fmt.Sprintf("mcr.microsoft.com/devcontainers/javascript-node:%s", detection.Version)
		config.Extensions = []string{
			"dbaeumer.vscode-eslint",
		}
		config.PostCreateCommand = "npm install"
		config.RemoteUser = "node"
		config.ForwardPorts = []int{3000}

	case "go":
		config.Image = fmt.Sprintf("mcr.microsoft.com/devcontainers/go:%s", detection.Version)
		config.Extensions = []string{
			"golang.go",
		}
		config.PostCreateCommand = "go mod download"
		config.RemoteUser = "vscode"
		config.ForwardPorts = []int{8080}

	case "python":
		config.Image = fmt.Sprintf("mcr.microsoft.com/devcontainers/python:%s", detection.Version)
		config.Extensions = []string{
			"ms-python.python",
			"ms-python.vscode-pylance",
		}
		config.PostCreateCommand = "pip install -r requirements.txt"
		config.RemoteUser = "vscode"
		config.ForwardPorts = []int{8000}

	case "rust":
		config.Image = fmt.Sprintf("mcr.microsoft.com/devcontainers/rust:%s", detection.Version)
		config.Extensions = []string{
			"rust-lang.rust-analyzer",
		}
		config.PostCreateCommand = "cargo build"
		config.RemoteUser = "vscode"
		config.ForwardPorts = []int{8080}

	default:
		config.Image = "mcr.microsoft.com/devcontainers/base:ubuntu"
		config.RemoteUser = "vscode"
	}

	// Add service-specific ports
	for _, service := range detection.Services {
		switch service {
		case "postgres":
			config.ForwardPorts = append(config.ForwardPorts, 5432)
		case "redis":
			config.ForwardPorts = append(config.ForwardPorts, 6379)
		}
	}

	return config
}

// render executes the template with the given config.
func (g *DevcontainerGenerator) render(config *DevcontainerConfig) ([]byte, error) {
	tmpl, err := loadTemplate("devcontainer.json.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	// Validate JSON output
	var js json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &js); err != nil {
		return nil, fmt.Errorf("generated invalid JSON: %w", err)
	}

	// Pretty-print the JSON
	var prettyBuf bytes.Buffer
	if err := json.Indent(&prettyBuf, buf.Bytes(), "", "\t"); err != nil {
		return buf.Bytes(), nil // Return original if pretty-print fails
	}

	return prettyBuf.Bytes(), nil
}
