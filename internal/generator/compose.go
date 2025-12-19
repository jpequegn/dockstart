package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jpequegn/dockstart/internal/models"
)

// ServiceConfig holds configuration for a single Docker Compose service.
type ServiceConfig struct {
	// Name is the service name (e.g., "postgres", "redis")
	Name string
}

// LogSidecarComposeConfig holds configuration for the log aggregator sidecar.
type LogSidecarComposeConfig struct {
	// Enabled indicates whether to include the log sidecar
	Enabled bool

	// LogFormat is the detected log format ("json", "text", "unknown")
	LogFormat string

	// LoggingLibraries is the list of detected logging libraries
	LoggingLibraries []string
}

// ComposeConfig holds the configuration for generating docker-compose.yml.
type ComposeConfig struct {
	// Name is the project name (used for database names, etc.)
	Name string

	// Services is a list of additional services to include
	Services []ServiceConfig

	// LogSidecar holds configuration for the log aggregator sidecar
	LogSidecar LogSidecarComposeConfig
}

// ComposeGenerator generates docker-compose.yml files.
type ComposeGenerator struct{}

// NewComposeGenerator creates a new compose generator.
func NewComposeGenerator() *ComposeGenerator {
	return &ComposeGenerator{}
}

// Generate creates a docker-compose.yml file from a Detection.
// The file is written to .devcontainer/docker-compose.yml.
func (g *ComposeGenerator) Generate(detection *models.Detection, projectPath string, projectName string) error {
	config := g.buildConfig(detection, projectName)

	// Create .devcontainer directory (may already exist)
	devcontainerDir := filepath.Join(projectPath, ".devcontainer")
	if err := os.MkdirAll(devcontainerDir, 0755); err != nil {
		return fmt.Errorf("failed to create .devcontainer directory: %w", err)
	}

	// Generate docker-compose.yml content
	content, err := g.render(config)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Write to file
	outputPath := filepath.Join(devcontainerDir, "docker-compose.yml")
	if err := os.WriteFile(outputPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write docker-compose.yml: %w", err)
	}

	return nil
}

// GenerateContent returns the generated docker-compose.yml content without writing to disk.
// Useful for dry-run mode.
func (g *ComposeGenerator) GenerateContent(detection *models.Detection, projectName string) ([]byte, error) {
	config := g.buildConfig(detection, projectName)
	return g.render(config)
}

// buildConfig creates a ComposeConfig from a Detection.
func (g *ComposeGenerator) buildConfig(detection *models.Detection, projectName string) *ComposeConfig {
	config := &ComposeConfig{
		Name:     projectName,
		Services: make([]ServiceConfig, 0, len(detection.Services)),
	}

	// Convert detected services to ServiceConfig
	for _, service := range detection.Services {
		config.Services = append(config.Services, ServiceConfig{
			Name: service,
		})
	}

	// Configure log sidecar if structured logging is detected
	if detection.HasStructuredLogging() {
		config.LogSidecar = LogSidecarComposeConfig{
			Enabled:          true,
			LogFormat:        detection.LogFormat,
			LoggingLibraries: detection.LoggingLibraries,
		}
	}

	return config
}

// render executes the template with the given config.
func (g *ComposeGenerator) render(config *ComposeConfig) ([]byte, error) {
	tmpl, err := loadTemplate("docker-compose.yml.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}
