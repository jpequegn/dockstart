package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jpequegn/dockstart/internal/models"
	"gopkg.in/yaml.v3"
)

// TestIntegration_DetectionToCompose tests the end-to-end flow from detection to compose generation.
func TestIntegration_DetectionToCompose(t *testing.T) {
	tests := []struct {
		name        string
		detection   *models.Detection
		projectName string
		wantParts   []string
		dontWant    []string
	}{
		{
			name: "node with pino generates sidecar",
			detection: &models.Detection{
				Language:         "node",
				Version:          "20",
				Services:         []string{},
				LoggingLibraries: []string{"pino"},
				LogFormat:        "json",
			},
			projectName: "node-pino-app",
			wantParts: []string{
				"fluent-bit:",
				"fluent/fluent-bit:latest",
				"driver: fluentd",
				"fluentd-address: localhost:24224",
				"LOG_LEVEL=debug",
				"- fluent-bit",
			},
			dontWant: []string{
				"postgres:",
				"redis:",
			},
		},
		{
			name: "go with zap and postgres",
			detection: &models.Detection{
				Language:         "go",
				Version:          "1.21",
				Services:         []string{"postgres"},
				LoggingLibraries: []string{"zap"},
				LogFormat:        "json",
			},
			projectName: "go-zap-app",
			wantParts: []string{
				"fluent-bit:",
				"postgres:",
				"DATABASE_URL",
				"driver: fluentd",
				"- postgres",
				"- fluent-bit",
			},
			dontWant: []string{
				"redis:",
			},
		},
		{
			name: "python with structlog and redis",
			detection: &models.Detection{
				Language:         "python",
				Version:          "3.11",
				Services:         []string{"redis"},
				LoggingLibraries: []string{"structlog"},
				LogFormat:        "json",
			},
			projectName: "python-app",
			wantParts: []string{
				"fluent-bit:",
				"redis:",
				"REDIS_URL",
				"driver: fluentd",
				"- redis",
				"- fluent-bit",
			},
			dontWant: []string{
				"postgres:",
			},
		},
		{
			name: "rust with tracing, postgres, and redis",
			detection: &models.Detection{
				Language:         "rust",
				Version:          "1.75",
				Services:         []string{"postgres", "redis"},
				LoggingLibraries: []string{"tracing"},
				LogFormat:        "json",
			},
			projectName: "rust-full-app",
			wantParts: []string{
				"fluent-bit:",
				"postgres:",
				"redis:",
				"DATABASE_URL",
				"REDIS_URL",
				"driver: fluentd",
				"- postgres",
				"- redis",
				"- fluent-bit",
				"postgres-data:",
				"redis-data:",
				"fluent-bit-logs:",
			},
			dontWant: []string{},
		},
		{
			name: "no logging libraries - no sidecar",
			detection: &models.Detection{
				Language:         "node",
				Version:          "20",
				Services:         []string{"postgres"},
				LoggingLibraries: []string{},
				LogFormat:        "unknown",
			},
			projectName: "node-no-logs",
			wantParts: []string{
				"postgres:",
				"DATABASE_URL",
			},
			dontWant: []string{
				"fluent-bit:",
				"driver: fluentd",
				"24224",
			},
		},
	}

	gen := NewComposeGenerator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := gen.GenerateContent(tt.detection, tt.projectName)
			if err != nil {
				t.Fatalf("GenerateContent() error = %v", err)
			}

			yaml := string(content)

			for _, want := range tt.wantParts {
				if !strings.Contains(yaml, want) {
					t.Errorf("YAML should contain %q, got:\n%s", want, yaml)
				}
			}

			for _, dontWant := range tt.dontWant {
				if strings.Contains(yaml, dontWant) {
					t.Errorf("YAML should NOT contain %q, got:\n%s", dontWant, yaml)
				}
			}
		})
	}
}

// TestIntegration_GeneratedYAMLIsValid tests that generated compose files are valid YAML.
func TestIntegration_GeneratedYAMLIsValid(t *testing.T) {
	tests := []struct {
		name      string
		detection *models.Detection
	}{
		{
			name: "minimal config",
			detection: &models.Detection{
				Language: "node",
				Version:  "20",
			},
		},
		{
			name: "with postgres",
			detection: &models.Detection{
				Language: "go",
				Version:  "1.21",
				Services: []string{"postgres"},
			},
		},
		{
			name: "with redis",
			detection: &models.Detection{
				Language: "python",
				Version:  "3.11",
				Services: []string{"redis"},
			},
		},
		{
			name: "with sidecar",
			detection: &models.Detection{
				Language:         "node",
				Version:          "20",
				LoggingLibraries: []string{"pino"},
				LogFormat:        "json",
			},
		},
		{
			name: "full config",
			detection: &models.Detection{
				Language:         "rust",
				Version:          "1.75",
				Services:         []string{"postgres", "redis"},
				LoggingLibraries: []string{"tracing"},
				LogFormat:        "json",
			},
		},
	}

	gen := NewComposeGenerator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := gen.GenerateContent(tt.detection, "test-app")
			if err != nil {
				t.Fatalf("GenerateContent() error = %v", err)
			}

			// Parse YAML to validate structure
			var parsed map[string]interface{}
			if err := yaml.Unmarshal(content, &parsed); err != nil {
				t.Errorf("Generated YAML is invalid: %v\nContent:\n%s", err, string(content))
			}

			// Check required top-level keys
			if _, ok := parsed["services"]; !ok {
				t.Error("YAML missing 'services' key")
			}
		})
	}
}

// TestIntegration_DevcontainerWithSidecar tests devcontainer generation with sidecar.
func TestIntegration_DevcontainerWithSidecar(t *testing.T) {
	tests := []struct {
		name       string
		detection  *models.Detection
		wantPorts  []int
		wantCompose bool
	}{
		{
			name: "with logging - uses compose and forwards 24224",
			detection: &models.Detection{
				Language:         "node",
				Version:          "20",
				LoggingLibraries: []string{"winston"},
				LogFormat:        "text",
			},
			wantPorts:   []int{3000, 24224},
			wantCompose: true,
		},
		{
			name: "with services - uses compose",
			detection: &models.Detection{
				Language: "go",
				Version:  "1.21",
				Services: []string{"postgres"},
			},
			wantPorts:   []int{8080, 5432},
			wantCompose: true,
		},
		{
			name: "no services or logging - no compose",
			detection: &models.Detection{
				Language: "python",
				Version:  "3.11",
			},
			wantPorts:   []int{8000},
			wantCompose: false,
		},
	}

	gen := NewDevcontainerGenerator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := gen.GenerateContent(tt.detection, "test-app")
			if err != nil {
				t.Fatalf("GenerateContent() error = %v", err)
			}

			// Parse JSON to validate structure
			var parsed map[string]interface{}
			if err := json.Unmarshal(content, &parsed); err != nil {
				t.Errorf("Generated JSON is invalid: %v\nContent:\n%s", err, string(content))
			}

			// Check dockerComposeFile presence
			if tt.wantCompose {
				if _, ok := parsed["dockerComposeFile"]; !ok {
					t.Error("Expected dockerComposeFile for compose-based config")
				}
			}

			// Check forward ports
			if ports, ok := parsed["forwardPorts"].([]interface{}); ok {
				for _, wantPort := range tt.wantPorts {
					found := false
					for _, p := range ports {
						if int(p.(float64)) == wantPort {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected port %d in forwardPorts, got %v", wantPort, ports)
					}
				}
			}
		})
	}
}

// TestIntegration_FluentBitConfigValid tests that generated Fluent Bit config is valid.
func TestIntegration_FluentBitConfigValid(t *testing.T) {
	tests := []struct {
		name      string
		detection *models.Detection
		wantParts []string
	}{
		{
			name: "json format includes parser",
			detection: &models.Detection{
				Language:         "node",
				Version:          "20",
				LoggingLibraries: []string{"pino"},
				LogFormat:        "json",
			},
			wantParts: []string{
				"[SERVICE]",
				"[INPUT]",
				"[FILTER]",
				"[OUTPUT]",
				"Parser          json",
			},
		},
		{
			name: "text format no json parser",
			detection: &models.Detection{
				Language:         "go",
				Version:          "1.21",
				LoggingLibraries: []string{"logrus"},
				LogFormat:        "text",
			},
			wantParts: []string{
				"[SERVICE]",
				"[INPUT]",
				"[OUTPUT]",
			},
		},
	}

	gen := NewLogSidecarGenerator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !gen.ShouldGenerate(tt.detection) {
				t.Fatal("ShouldGenerate returned false unexpectedly")
			}

			content, err := gen.GenerateContent(tt.detection, "test-app")
			if err != nil {
				t.Fatalf("GenerateContent() error = %v", err)
			}

			config := string(content)

			for _, want := range tt.wantParts {
				if !strings.Contains(config, want) {
					t.Errorf("Config should contain %q, got:\n%s", want, config)
				}
			}

			// Validate config has proper structure (basic INI-like validation)
			if !strings.Contains(config, "[SERVICE]") {
				t.Error("Missing [SERVICE] section")
			}
			if !strings.Contains(config, "Flush") {
				t.Error("Missing Flush directive")
			}
		})
	}
}

// TestIntegration_FullGeneration tests full file generation to disk.
func TestIntegration_FullGeneration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dockstart-integration-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	detection := &models.Detection{
		Language:         "node",
		Version:          "20",
		Services:         []string{"postgres"},
		LoggingLibraries: []string{"pino"},
		LogFormat:        "json",
	}

	// Generate all files
	composeGen := NewComposeGenerator()
	devcontainerGen := NewDevcontainerGenerator()
	sidecarGen := NewLogSidecarGenerator()

	if err := composeGen.Generate(detection, tmpDir, "full-test-app"); err != nil {
		t.Fatalf("ComposeGenerator.Generate() error = %v", err)
	}

	if err := devcontainerGen.Generate(detection, tmpDir, "full-test-app"); err != nil {
		t.Fatalf("DevcontainerGenerator.Generate() error = %v", err)
	}

	if sidecarGen.ShouldGenerate(detection) {
		if err := sidecarGen.Generate(detection, tmpDir, "full-test-app"); err != nil {
			t.Fatalf("LogSidecarGenerator.Generate() error = %v", err)
		}
	}

	// Verify all files exist
	expectedFiles := []string{
		".devcontainer/docker-compose.yml",
		".devcontainer/devcontainer.json",
		".devcontainer/fluent-bit.conf",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s to exist", file)
		}
	}

	// Verify docker-compose.yml is valid YAML
	composeContent, err := os.ReadFile(filepath.Join(tmpDir, ".devcontainer/docker-compose.yml"))
	if err != nil {
		t.Fatalf("Failed to read docker-compose.yml: %v", err)
	}

	var composeYAML map[string]interface{}
	if err := yaml.Unmarshal(composeContent, &composeYAML); err != nil {
		t.Errorf("docker-compose.yml is not valid YAML: %v", err)
	}

	// Verify devcontainer.json is valid JSON
	devcontainerContent, err := os.ReadFile(filepath.Join(tmpDir, ".devcontainer/devcontainer.json"))
	if err != nil {
		t.Fatalf("Failed to read devcontainer.json: %v", err)
	}

	var devcontainerJSON map[string]interface{}
	if err := json.Unmarshal(devcontainerContent, &devcontainerJSON); err != nil {
		t.Errorf("devcontainer.json is not valid JSON: %v", err)
	}
}

// TestVolumeConfiguration tests that volumes are correctly configured.
func TestVolumeConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		detection   *models.Detection
		wantVolumes []string
	}{
		{
			name: "postgres only",
			detection: &models.Detection{
				Language: "node",
				Version:  "20",
				Services: []string{"postgres"},
			},
			wantVolumes: []string{"postgres-data:"},
		},
		{
			name: "redis only",
			detection: &models.Detection{
				Language: "go",
				Version:  "1.21",
				Services: []string{"redis"},
			},
			wantVolumes: []string{"redis-data:"},
		},
		{
			name: "all services with sidecar",
			detection: &models.Detection{
				Language:         "python",
				Version:          "3.11",
				Services:         []string{"postgres", "redis"},
				LoggingLibraries: []string{"structlog"},
				LogFormat:        "json",
			},
			wantVolumes: []string{"postgres-data:", "redis-data:", "fluent-bit-logs:"},
		},
	}

	gen := NewComposeGenerator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := gen.GenerateContent(tt.detection, "test-app")
			if err != nil {
				t.Fatalf("GenerateContent() error = %v", err)
			}

			yaml := string(content)

			for _, vol := range tt.wantVolumes {
				if !strings.Contains(yaml, vol) {
					t.Errorf("YAML should contain volume %q, got:\n%s", vol, yaml)
				}
			}
		})
	}
}
