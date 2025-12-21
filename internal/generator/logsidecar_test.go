package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jpequegn/dockstart/internal/models"
)

func TestLogSidecarGenerator_ShouldGenerate(t *testing.T) {
	g := NewLogSidecarGenerator()

	tests := []struct {
		name      string
		detection *models.Detection
		want      bool
	}{
		{
			name: "with logging libraries",
			detection: &models.Detection{
				Language:         "node",
				LoggingLibraries: []string{"winston", "pino"},
				LogFormat:        "json",
			},
			want: true,
		},
		{
			name: "without logging libraries",
			detection: &models.Detection{
				Language:         "node",
				LoggingLibraries: []string{},
				LogFormat:        "unknown",
			},
			want: false,
		},
		{
			name: "nil logging libraries",
			detection: &models.Detection{
				Language:  "go",
				LogFormat: "unknown",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := g.ShouldGenerate(tt.detection)
			if got != tt.want {
				t.Errorf("ShouldGenerate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLogSidecarGenerator_GenerateContent_Basic(t *testing.T) {
	g := NewLogSidecarGenerator()

	detection := &models.Detection{
		Language:         "node",
		LoggingLibraries: []string{"pino"},
		LogFormat:        "json",
	}

	content, err := g.GenerateContent(detection, "my-app")
	if err != nil {
		t.Fatalf("GenerateContent failed: %v", err)
	}

	contentStr := string(content)

	// Check basic structure
	if !strings.Contains(contentStr, "[SERVICE]") {
		t.Error("Expected [SERVICE] section")
	}
	if !strings.Contains(contentStr, "[INPUT]") {
		t.Error("Expected [INPUT] section")
	}
	if !strings.Contains(contentStr, "[OUTPUT]") {
		t.Error("Expected [OUTPUT] section")
	}

	// Check project name is included
	if !strings.Contains(contentStr, "my-app") {
		t.Error("Expected project name 'my-app' in config")
	}

	// Check forward input configuration
	if !strings.Contains(contentStr, "Name            forward") {
		t.Error("Expected forward input plugin")
	}
	if !strings.Contains(contentStr, "Port            24224") {
		t.Error("Expected port 24224")
	}
}

func TestLogSidecarGenerator_GenerateContent_JSONFormat(t *testing.T) {
	g := NewLogSidecarGenerator()

	detection := &models.Detection{
		Language:         "node",
		LoggingLibraries: []string{"pino"},
		LogFormat:        "json",
	}

	content, err := g.GenerateContent(detection, "test-app")
	if err != nil {
		t.Fatalf("GenerateContent failed: %v", err)
	}

	contentStr := string(content)

	// JSON format should include parser filter
	if !strings.Contains(contentStr, "[FILTER]") {
		t.Error("Expected [FILTER] section for JSON parsing")
	}
	if !strings.Contains(contentStr, "Name            parser") {
		t.Error("Expected parser filter for JSON logs")
	}
	if !strings.Contains(contentStr, "Parser          json") {
		t.Error("Expected json parser")
	}
}

func TestLogSidecarGenerator_GenerateContent_TextFormat(t *testing.T) {
	g := NewLogSidecarGenerator()

	detection := &models.Detection{
		Language:         "go",
		LoggingLibraries: []string{"logrus"},
		LogFormat:        "text",
	}

	content, err := g.GenerateContent(detection, "test-app")
	if err != nil {
		t.Fatalf("GenerateContent failed: %v", err)
	}

	contentStr := string(content)

	// Text format should NOT include JSON parser filter
	if strings.Contains(contentStr, "Parser          json") {
		t.Error("Text format should not include JSON parser")
	}

	// Should still have the modify filter for metadata
	if !strings.Contains(contentStr, "Name            modify") {
		t.Error("Expected modify filter for metadata")
	}
}

func TestLogSidecarGenerator_GenerateContent_MetadataFilter(t *testing.T) {
	g := NewLogSidecarGenerator()

	detection := &models.Detection{
		Language:         "python",
		LoggingLibraries: []string{"structlog"},
		LogFormat:        "json",
	}

	content, err := g.GenerateContent(detection, "my-python-app")
	if err != nil {
		t.Fatalf("GenerateContent failed: %v", err)
	}

	contentStr := string(content)

	// Check metadata enrichment
	if !strings.Contains(contentStr, "Add             environment development") {
		t.Error("Expected environment metadata")
	}
	if !strings.Contains(contentStr, "Add             project my-python-app") {
		t.Error("Expected project metadata")
	}
}

func TestLogSidecarGenerator_Generate_WritesFile(t *testing.T) {
	g := NewLogSidecarGenerator()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "dockstart-logsidecar-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	detection := &models.Detection{
		Language:         "rust",
		LoggingLibraries: []string{"tracing"},
		LogFormat:        "json",
	}

	err = g.Generate(detection, tmpDir, "rust-app")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check file was created
	outputPath := filepath.Join(tmpDir, ".devcontainer", "fluent-bit.conf")
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Expected fluent-bit.conf to be created")
	}

	// Read and verify content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	if !strings.Contains(string(content), "rust-app") {
		t.Error("Expected project name in generated file")
	}
}

func TestLogSidecarGenerator_GetComposeService(t *testing.T) {
	g := NewLogSidecarGenerator()

	service := g.GetComposeService("my-app")

	// Check service definition
	if !strings.Contains(service, "fluent-bit:") {
		t.Error("Expected fluent-bit service name")
	}
	if !strings.Contains(service, "fluent/fluent-bit:latest") {
		t.Error("Expected fluent-bit image")
	}
	if !strings.Contains(service, "./fluent-bit.conf:/fluent-bit/etc/fluent-bit.conf") {
		t.Error("Expected config volume mount")
	}
	if !strings.Contains(service, "24224:24224") {
		t.Error("Expected port mapping")
	}
}

func TestLogSidecarGenerator_AllLanguages(t *testing.T) {
	g := NewLogSidecarGenerator()

	tests := []struct {
		language string
		libs     []string
		format   string
	}{
		{"node", []string{"pino", "morgan"}, "json"},
		{"go", []string{"zap"}, "json"},
		{"python", []string{"structlog"}, "json"},
		{"rust", []string{"tracing"}, "json"},
		{"node", []string{"winston"}, "text"},
		{"go", []string{"logrus"}, "text"},
		{"python", []string{"loguru"}, "text"},
		{"rust", []string{"log", "env_logger"}, "text"},
	}

	for _, tt := range tests {
		t.Run(tt.language+"_"+tt.format, func(t *testing.T) {
			detection := &models.Detection{
				Language:         tt.language,
				LoggingLibraries: tt.libs,
				LogFormat:        tt.format,
			}

			content, err := g.GenerateContent(detection, tt.language+"-app")
			if err != nil {
				t.Fatalf("GenerateContent failed for %s: %v", tt.language, err)
			}

			// Basic validation
			contentStr := string(content)
			if !strings.Contains(contentStr, "[SERVICE]") {
				t.Errorf("Missing [SERVICE] for %s", tt.language)
			}
			if !strings.Contains(contentStr, tt.language+"-app") {
				t.Errorf("Missing project name for %s", tt.language)
			}

			// Format-specific validation
			if tt.format == "json" {
				if !strings.Contains(contentStr, "Parser          json") {
					t.Errorf("Missing JSON parser for %s with json format", tt.language)
				}
			}
		})
	}
}
