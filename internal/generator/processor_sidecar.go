// Package generator provides code generation for devcontainer files.
package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jpequegn/dockstart/internal/models"
)

// ProcessorSidecarConfig holds configuration for generating file processor sidecar container files.
type ProcessorSidecarConfig struct {
	// UseInotify enables inotify-based file watching (Linux only)
	UseInotify bool

	// ProcessImages enables image processing (ImageMagick)
	ProcessImages bool

	// ProcessDocuments enables document processing (Poppler)
	ProcessDocuments bool

	// ProcessVideo enables video processing (FFmpeg)
	ProcessVideo bool

	// PollInterval is the polling interval in seconds (default: 5)
	PollInterval int

	// MaxFileSize is the maximum file size in bytes (default: 50MB)
	MaxFileSize int64

	// ThumbnailSize is the thumbnail dimensions (default: 200x200)
	ThumbnailSize string

	// ProjectName is the name of the project
	ProjectName string
}

// DefaultProcessorConfig returns a ProcessorSidecarConfig with sensible defaults.
func DefaultProcessorConfig() *ProcessorSidecarConfig {
	return &ProcessorSidecarConfig{
		UseInotify:       false, // Use polling for cross-platform compatibility
		ProcessImages:    true,  // Enable by default
		ProcessDocuments: false,
		ProcessVideo:     false,
		PollInterval:     5,
		MaxFileSize:      52428800, // 50MB
		ThumbnailSize:    "200x200",
	}
}

// ProcessorSidecarGenerator generates file processor sidecar container files.
type ProcessorSidecarGenerator struct{}

// NewProcessorSidecarGenerator creates a new processor sidecar generator.
func NewProcessorSidecarGenerator() *ProcessorSidecarGenerator {
	return &ProcessorSidecarGenerator{}
}

// GenerateDockerfile generates the Dockerfile.processor content.
func (g *ProcessorSidecarGenerator) GenerateDockerfile(config *ProcessorSidecarConfig) ([]byte, error) {
	tmpl, err := loadTemplate("Dockerfile.processor.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateProcessScript generates the main process-files.sh script.
func (g *ProcessorSidecarGenerator) GenerateProcessScript(config *ProcessorSidecarConfig) ([]byte, error) {
	tmpl, err := loadTemplate("processor/process-files.sh.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateImageScript generates the image processing script.
func (g *ProcessorSidecarGenerator) GenerateImageScript(config *ProcessorSidecarConfig) ([]byte, error) {
	tmpl, err := loadTemplate("processor/process-image.sh.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateDocumentScript generates the document processing script.
func (g *ProcessorSidecarGenerator) GenerateDocumentScript(config *ProcessorSidecarConfig) ([]byte, error) {
	tmpl, err := loadTemplate("processor/process-document.sh.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateVideoScript generates the video processing script.
func (g *ProcessorSidecarGenerator) GenerateVideoScript(config *ProcessorSidecarConfig) ([]byte, error) {
	tmpl, err := loadTemplate("processor/process-video.sh.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateEntrypoint generates the entrypoint.processor.sh script.
func (g *ProcessorSidecarGenerator) GenerateEntrypoint(config *ProcessorSidecarConfig) ([]byte, error) {
	tmpl, err := loadTemplate("entrypoint.processor.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// Generate writes all processor sidecar files to the target directory.
func (g *ProcessorSidecarGenerator) Generate(detection *models.Detection, projectPath string, projectName string) error {
	devcontainerDir := filepath.Join(projectPath, ".devcontainer")
	scriptsDir := filepath.Join(devcontainerDir, "scripts")

	// Create directories
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		return fmt.Errorf("failed to create scripts directory: %w", err)
	}

	// Create config based on detection
	config := DefaultProcessorConfig()
	config.ProjectName = projectName

	// Determine processing capabilities based on detected libraries
	// For now, enable image processing by default when file upload is detected
	if detection.NeedsFileProcessor() {
		config.ProcessImages = true
	}

	// Generate Dockerfile.processor
	dockerfile, err := g.GenerateDockerfile(config)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(devcontainerDir, "Dockerfile.processor"), dockerfile, 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile.processor: %w", err)
	}

	// Generate main process-files.sh script
	processScript, err := g.GenerateProcessScript(config)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(scriptsDir, "process-files.sh"), processScript, 0755); err != nil {
		return fmt.Errorf("failed to write process-files.sh: %w", err)
	}

	// Generate image processing script if enabled
	if config.ProcessImages {
		imageScript, err := g.GenerateImageScript(config)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(scriptsDir, "process-image.sh"), imageScript, 0755); err != nil {
			return fmt.Errorf("failed to write process-image.sh: %w", err)
		}
	}

	// Generate document processing script if enabled
	if config.ProcessDocuments {
		docScript, err := g.GenerateDocumentScript(config)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(scriptsDir, "process-document.sh"), docScript, 0755); err != nil {
			return fmt.Errorf("failed to write process-document.sh: %w", err)
		}
	}

	// Generate video processing script if enabled
	if config.ProcessVideo {
		videoScript, err := g.GenerateVideoScript(config)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(scriptsDir, "process-video.sh"), videoScript, 0755); err != nil {
			return fmt.Errorf("failed to write process-video.sh: %w", err)
		}
	}

	// Generate entrypoint
	entrypoint, err := g.GenerateEntrypoint(config)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(devcontainerDir, "entrypoint.processor.sh"), entrypoint, 0755); err != nil {
		return fmt.Errorf("failed to write entrypoint.processor.sh: %w", err)
	}

	// Create files directory structure
	filesDir := filepath.Join(devcontainerDir, "files")
	for _, dir := range []string{"pending", "processing", "processed", "failed"} {
		if err := os.MkdirAll(filepath.Join(filesDir, dir), 0755); err != nil {
			return fmt.Errorf("failed to create files/%s directory: %w", dir, err)
		}
	}

	// Create .gitkeep in pending directory
	gitkeep := filepath.Join(filesDir, "pending", ".gitkeep")
	if err := os.WriteFile(gitkeep, []byte{}, 0644); err != nil {
		return fmt.Errorf("failed to write .gitkeep: %w", err)
	}

	return nil
}

// ShouldGenerate checks if processor sidecar should be generated based on detection.
func (g *ProcessorSidecarGenerator) ShouldGenerate(detection *models.Detection) bool {
	return detection.NeedsFileProcessor()
}
