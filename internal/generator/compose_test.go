package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jpequegn/dockstart/internal/models"
)

// TestComposeGenerator_GenerateContent tests the GenerateContent method.
func TestComposeGenerator_GenerateContent(t *testing.T) {
	tests := []struct {
		name        string
		detection   *models.Detection
		projectName string
		wantInYAML  []string
		dontWant    []string
	}{
		{
			name: "no services - minimal compose",
			detection: &models.Detection{
				Language:   "node",
				Version:    "20",
				Services:   []string{},
				Confidence: 1.0,
			},
			projectName: "my-app",
			wantInYAML: []string{
				"services:",
				"app:",
				"build:",
				"context: ..",
				"dockerfile: .devcontainer/Dockerfile",
				"volumes:",
				"..:/workspace:cached",
				"sleep infinity",
			},
			dontWant: []string{
				"depends_on:",
				"postgres:",
				"redis:",
				"volumes:\n  postgres-data:",
			},
		},
		{
			name: "postgres only",
			detection: &models.Detection{
				Language:   "go",
				Version:    "1.23",
				Services:   []string{"postgres"},
				Confidence: 1.0,
			},
			projectName: "go-app",
			wantInYAML: []string{
				"depends_on:",
				"- postgres",
				"DATABASE_URL=postgres://postgres:postgres@postgres:5432/go-app_dev",
				"postgres:",
				"image: postgres:16-alpine",
				"POSTGRES_DB: go-app_dev",
				"postgres-data:/var/lib/postgresql/data",
				"volumes:",
				"postgres-data:",
			},
			dontWant: []string{
				"redis:",
				"REDIS_URL",
				"redis-data:",
			},
		},
		{
			name: "redis only",
			detection: &models.Detection{
				Language:   "python",
				Version:    "3.11",
				Services:   []string{"redis"},
				Confidence: 1.0,
			},
			projectName: "python-app",
			wantInYAML: []string{
				"depends_on:",
				"- redis",
				"REDIS_URL=redis://redis:6379",
				"redis:",
				"image: redis:7-alpine",
				"redis-data:/data",
				"volumes:",
				"redis-data:",
			},
			dontWant: []string{
				"postgres:",
				"DATABASE_URL",
				"postgres-data:",
			},
		},
		{
			name: "both postgres and redis",
			detection: &models.Detection{
				Language:   "node",
				Version:    "20",
				Services:   []string{"postgres", "redis"},
				Confidence: 1.0,
			},
			projectName: "fullstack-app",
			wantInYAML: []string{
				"depends_on:",
				"- postgres",
				"- redis",
				"DATABASE_URL=postgres://postgres:postgres@postgres:5432/fullstack-app_dev",
				"REDIS_URL=redis://redis:6379",
				"postgres:",
				"image: postgres:16-alpine",
				"redis:",
				"image: redis:7-alpine",
				"volumes:",
				"postgres-data:",
				"redis-data:",
			},
			dontWant: []string{},
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

			// Check required strings are present
			for _, want := range tt.wantInYAML {
				if !strings.Contains(yaml, want) {
					t.Errorf("YAML should contain %q, got:\n%s", want, yaml)
				}
			}

			// Check unwanted strings are absent
			for _, dontWant := range tt.dontWant {
				if strings.Contains(yaml, dontWant) {
					t.Errorf("YAML should NOT contain %q, got:\n%s", dontWant, yaml)
				}
			}
		})
	}
}

// TestComposeGenerator_Generate tests the Generate method which writes to disk.
func TestComposeGenerator_Generate(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "dockstart-compose-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gen := NewComposeGenerator()
	detection := &models.Detection{
		Language:   "node",
		Version:    "20",
		Services:   []string{"postgres"},
		Confidence: 1.0,
	}

	// Generate the docker-compose.yml
	err = gen.Generate(detection, tmpDir, "test-project")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify the .devcontainer directory was created
	devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
	if _, err := os.Stat(devcontainerDir); os.IsNotExist(err) {
		t.Error(".devcontainer directory was not created")
	}

	// Verify docker-compose.yml was created
	composeFile := filepath.Join(devcontainerDir, "docker-compose.yml")
	content, err := os.ReadFile(composeFile)
	if err != nil {
		t.Fatalf("Failed to read docker-compose.yml: %v", err)
	}

	// Verify some key content
	yaml := string(content)
	if !strings.Contains(yaml, "test-project") {
		t.Error("YAML should contain project name")
	}
	if !strings.Contains(yaml, "postgres:") {
		t.Error("YAML should contain postgres service")
	}
}

// TestBuildComposeConfig tests the internal buildConfig function.
func TestBuildComposeConfig(t *testing.T) {
	gen := NewComposeGenerator()

	detection := &models.Detection{
		Language:   "node",
		Version:    "20",
		Services:   []string{"postgres", "redis"},
		Confidence: 1.0,
	}

	config := gen.buildConfig(detection, "my-app")

	// Verify config fields
	if config.Name != "my-app" {
		t.Errorf("Name = %v, want my-app", config.Name)
	}
	if len(config.Services) != 2 {
		t.Errorf("Services count = %d, want 2", len(config.Services))
	}
	if config.Services[0].Name != "postgres" {
		t.Errorf("Services[0].Name = %v, want postgres", config.Services[0].Name)
	}
	if config.Services[1].Name != "redis" {
		t.Errorf("Services[1].Name = %v, want redis", config.Services[1].Name)
	}
}

// TestComposeGenerator_HeaderComment tests that the header comment is generated.
func TestComposeGenerator_HeaderComment(t *testing.T) {
	gen := NewComposeGenerator()
	detection := &models.Detection{
		Language: "go",
		Version:  "1.23",
		Services: []string{},
	}

	content, err := gen.GenerateContent(detection, "test-app")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	yaml := string(content)
	if !strings.Contains(yaml, "# Docker Compose configuration for test-app") {
		t.Error("YAML should contain header comment with project name")
	}
	if !strings.Contains(yaml, "Generated by dockstart") {
		t.Error("YAML should contain 'Generated by dockstart' attribution")
	}
}

// TestComposeGenerator_PostgresDefaults tests postgres service defaults.
func TestComposeGenerator_PostgresDefaults(t *testing.T) {
	gen := NewComposeGenerator()
	detection := &models.Detection{
		Language: "node",
		Version:  "20",
		Services: []string{"postgres"},
	}

	content, err := gen.GenerateContent(detection, "my-db-app")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	yaml := string(content)

	// Check postgres-specific settings
	expectedParts := []string{
		"postgres:16-alpine",           // Latest stable alpine image
		"POSTGRES_USER: postgres",      // Default user
		"POSTGRES_PASSWORD: postgres",  // Default password
		"POSTGRES_DB: my-db-app_dev",   // Database named after project
		"5432:5432",                    // Default port mapping
		"unless-stopped",               // Restart policy
	}

	for _, part := range expectedParts {
		if !strings.Contains(yaml, part) {
			t.Errorf("YAML should contain %q for postgres defaults", part)
		}
	}
}

// TestComposeGenerator_RedisDefaults tests redis service defaults.
func TestComposeGenerator_RedisDefaults(t *testing.T) {
	gen := NewComposeGenerator()
	detection := &models.Detection{
		Language: "python",
		Version:  "3.11",
		Services: []string{"redis"},
	}

	content, err := gen.GenerateContent(detection, "cache-app")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	yaml := string(content)

	// Check redis-specific settings
	expectedParts := []string{
		"redis:7-alpine",      // Latest stable alpine image
		"6379:6379",           // Default port mapping
		"redis-data:/data",    // Data persistence volume
		"unless-stopped",      // Restart policy
	}

	for _, part := range expectedParts {
		if !strings.Contains(yaml, part) {
			t.Errorf("YAML should contain %q for redis defaults", part)
		}
	}
}

// TestComposeGenerator_LogSidecar tests Fluent Bit sidecar generation.
func TestComposeGenerator_LogSidecar(t *testing.T) {
	gen := NewComposeGenerator()
	detection := &models.Detection{
		Language:         "node",
		Version:          "20",
		Services:         []string{},
		LoggingLibraries: []string{"pino"},
		LogFormat:        "json",
	}

	content, err := gen.GenerateContent(detection, "logged-app")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	yaml := string(content)

	// Check Fluent Bit service is included
	expectedParts := []string{
		"fluent-bit:",
		"fluent/fluent-bit:latest",
		"./fluent-bit.conf:/fluent-bit/etc/fluent-bit.conf",
		"24224:24224",
		"depends_on:",
		"- fluent-bit",
		"driver: fluentd",
		"fluentd-address: localhost:24224",
		"LOG_LEVEL=debug",
	}

	for _, part := range expectedParts {
		if !strings.Contains(yaml, part) {
			t.Errorf("YAML should contain %q for log sidecar, got:\n%s", part, yaml)
		}
	}
}

// TestComposeGenerator_LogSidecar_NotGenerated tests that sidecar is not generated without logging libraries.
func TestComposeGenerator_LogSidecar_NotGenerated(t *testing.T) {
	gen := NewComposeGenerator()
	detection := &models.Detection{
		Language:         "node",
		Version:          "20",
		Services:         []string{},
		LoggingLibraries: []string{}, // No logging libraries
		LogFormat:        "unknown",
	}

	content, err := gen.GenerateContent(detection, "no-logs-app")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	yaml := string(content)

	// Check Fluent Bit service is NOT included
	unwantedParts := []string{
		"fluent-bit:",
		"fluent/fluent-bit",
		"driver: fluentd",
		"24224",
	}

	for _, part := range unwantedParts {
		if strings.Contains(yaml, part) {
			t.Errorf("YAML should NOT contain %q when no logging libraries detected, got:\n%s", part, yaml)
		}
	}
}

// TestComposeGenerator_LogSidecar_WithServices tests log sidecar combined with postgres/redis.
func TestComposeGenerator_LogSidecar_WithServices(t *testing.T) {
	gen := NewComposeGenerator()
	detection := &models.Detection{
		Language:         "go",
		Version:          "1.23",
		Services:         []string{"postgres", "redis"},
		LoggingLibraries: []string{"zap"},
		LogFormat:        "json",
	}

	content, err := gen.GenerateContent(detection, "full-app")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	yaml := string(content)

	// Check all services are present
	expectedParts := []string{
		// Postgres
		"postgres:",
		"postgres:16-alpine",
		"DATABASE_URL",
		// Redis
		"redis:",
		"redis:7-alpine",
		"REDIS_URL",
		// Fluent Bit
		"fluent-bit:",
		"fluent/fluent-bit:latest",
		"driver: fluentd",
		// Dependencies
		"depends_on:",
		"- postgres",
		"- redis",
		"- fluent-bit",
		// Volumes
		"volumes:",
		"postgres-data:",
		"redis-data:",
		"fluent-bit-logs:",
	}

	for _, part := range expectedParts {
		if !strings.Contains(yaml, part) {
			t.Errorf("YAML should contain %q, got:\n%s", part, yaml)
		}
	}
}

// TestBuildComposeConfig_WithLogSidecar tests buildConfig includes log sidecar config.
func TestBuildComposeConfig_WithLogSidecar(t *testing.T) {
	gen := NewComposeGenerator()

	detection := &models.Detection{
		Language:         "python",
		Version:          "3.11",
		Services:         []string{"postgres"},
		LoggingLibraries: []string{"structlog", "rich"},
		LogFormat:        "json",
	}

	config := gen.buildConfig(detection, "python-app")

	// Verify log sidecar config
	if !config.LogSidecar.Enabled {
		t.Error("LogSidecar.Enabled should be true")
	}
	if config.LogSidecar.LogFormat != "json" {
		t.Errorf("LogSidecar.LogFormat = %v, want json", config.LogSidecar.LogFormat)
	}
	if len(config.LogSidecar.LoggingLibraries) != 2 {
		t.Errorf("LogSidecar.LoggingLibraries count = %d, want 2", len(config.LogSidecar.LoggingLibraries))
	}
}

// TestBuildComposeConfig_NoLogSidecar tests buildConfig without logging libraries.
func TestBuildComposeConfig_NoLogSidecar(t *testing.T) {
	gen := NewComposeGenerator()

	detection := &models.Detection{
		Language:         "rust",
		Version:          "1.75",
		Services:         []string{"redis"},
		LoggingLibraries: []string{},
		LogFormat:        "unknown",
	}

	config := gen.buildConfig(detection, "rust-app")

	// Verify log sidecar is NOT enabled
	if config.LogSidecar.Enabled {
		t.Error("LogSidecar.Enabled should be false when no logging libraries")
	}
}
