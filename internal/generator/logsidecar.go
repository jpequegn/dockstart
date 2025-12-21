package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jpequegn/dockstart/internal/models"
)

// LogSidecarConfig holds the configuration for generating log sidecar configs.
type LogSidecarConfig struct {
	// Name is the project name
	Name string

	// LogFormat is the detected log format ("json", "text", "unknown")
	LogFormat string

	// EnableFileOutput enables writing logs to files in addition to stdout
	EnableFileOutput bool

	// LoggingLibraries is the list of detected logging libraries
	LoggingLibraries []string
}

// LogSidecarGenerator generates Fluent Bit configuration files.
type LogSidecarGenerator struct{}

// NewLogSidecarGenerator creates a new log sidecar generator.
func NewLogSidecarGenerator() *LogSidecarGenerator {
	return &LogSidecarGenerator{}
}

// ShouldGenerate returns true if log sidecar configuration should be generated.
// This is based on whether structured logging libraries were detected.
func (g *LogSidecarGenerator) ShouldGenerate(detection *models.Detection) bool {
	return detection.HasStructuredLogging()
}

// Generate creates a Fluent Bit configuration file from a Detection.
// The file is written to .devcontainer/fluent-bit.conf.
func (g *LogSidecarGenerator) Generate(detection *models.Detection, projectPath string, projectName string) error {
	config := g.buildConfig(detection, projectName)

	// Create .devcontainer directory (may already exist)
	devcontainerDir := filepath.Join(projectPath, ".devcontainer")
	if err := os.MkdirAll(devcontainerDir, 0755); err != nil {
		return fmt.Errorf("failed to create .devcontainer directory: %w", err)
	}

	// Generate fluent-bit.conf content
	content, err := g.render(config)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Write to file
	outputPath := filepath.Join(devcontainerDir, "fluent-bit.conf")
	if err := os.WriteFile(outputPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write fluent-bit.conf: %w", err)
	}

	return nil
}

// GenerateContent returns the generated fluent-bit.conf content without writing to disk.
// Useful for dry-run mode.
func (g *LogSidecarGenerator) GenerateContent(detection *models.Detection, projectName string) ([]byte, error) {
	config := g.buildConfig(detection, projectName)
	return g.render(config)
}

// buildConfig creates a LogSidecarConfig from a Detection.
func (g *LogSidecarGenerator) buildConfig(detection *models.Detection, projectName string) *LogSidecarConfig {
	return &LogSidecarConfig{
		Name:             projectName,
		LogFormat:        detection.LogFormat,
		EnableFileOutput: false, // Default to stdout only for dev
		LoggingLibraries: detection.LoggingLibraries,
	}
}

// render executes the template with the given config.
func (g *LogSidecarGenerator) render(config *LogSidecarConfig) ([]byte, error) {
	tmpl, err := loadTemplate("fluent-bit.conf.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// GetComposeService returns the docker-compose service definition for Fluent Bit.
// This can be added to the main docker-compose.yml.
func (g *LogSidecarGenerator) GetComposeService(projectName string) string {
	return fmt.Sprintf(`  # Log aggregator sidecar
  fluent-bit:
    image: fluent/fluent-bit:latest
    volumes:
      - ./fluent-bit.conf:/fluent-bit/etc/fluent-bit.conf:ro
    ports:
      - "24224:24224"
    restart: unless-stopped
`)
}

// GetLoggingDriverConfig returns the Docker logging driver configuration
// to add to the app service in docker-compose.yml.
func (g *LogSidecarGenerator) GetLoggingDriverConfig() string {
	return `    logging:
      driver: fluentd
      options:
        fluentd-address: localhost:24224
        tag: app.{{.Name}}
`
}
