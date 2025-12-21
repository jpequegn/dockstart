package generator

import (
	"strings"
	"testing"

	"github.com/jpequegn/dockstart/internal/models"
)

// TestComposeGenerator_FileProcessorSidecar tests file processor sidecar generation.
func TestComposeGenerator_FileProcessorSidecar(t *testing.T) {
	gen := NewComposeGenerator()
	detection := &models.Detection{
		Language:            "node",
		Version:             "20",
		Services:            []string{},
		FileUploadLibraries: []string{"multer"},
		UploadPath:          "/uploads",
	}

	content, err := gen.GenerateContent(detection, "upload-app")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	yaml := string(content)

	// Check file processor service is included
	expectedParts := []string{
		"file-processor:",
		"dockerfile: Dockerfile.processor",
		"uploads:/uploads",
		"PENDING_PATH=/uploads/pending",
		"PROCESSING_PATH=/uploads/processing",
		"PROCESSED_PATH=/uploads/processed",
		"FAILED_PATH=/uploads/failed",
		"POLL_INTERVAL=5",
		"MAX_FILE_SIZE=52428800",
		"RETRY_COUNT=3",
		"NOTIFY_METHOD=file",
		"deploy:",
		"resources:",
		"limits:",
		"memory: 512M",
		"cpus: '0.5'",
		"restart: unless-stopped",
		"volumes:",
		"uploads:",
	}

	for _, part := range expectedParts {
		if !strings.Contains(yaml, part) {
			t.Errorf("YAML should contain %q for file processor sidecar, got:\n%s", part, yaml)
		}
	}
}

// TestComposeGenerator_FileProcessorSidecar_AppVolume tests that app service gets uploads volume.
func TestComposeGenerator_FileProcessorSidecar_AppVolume(t *testing.T) {
	gen := NewComposeGenerator()
	detection := &models.Detection{
		Language:            "python",
		Version:             "3.11",
		Services:            []string{},
		FileUploadLibraries: []string{"python-multipart"},
		UploadPath:          "/data/uploads",
	}

	content, err := gen.GenerateContent(detection, "py-upload")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	yaml := string(content)

	// Check app service has uploads volume
	if !strings.Contains(yaml, "- uploads:/uploads") {
		t.Errorf("App service should have uploads volume, got:\n%s", yaml)
	}
}

// TestComposeGenerator_FileProcessorSidecar_EnvironmentVars tests app environment variables.
func TestComposeGenerator_FileProcessorSidecar_EnvironmentVars(t *testing.T) {
	gen := NewComposeGenerator()
	detection := &models.Detection{
		Language:            "node",
		Version:             "20",
		Services:            []string{},
		FileUploadLibraries: []string{"formidable"},
	}

	content, err := gen.GenerateContent(detection, "env-test")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	yaml := string(content)

	// Check environment variables for app service
	expectedEnvVars := []string{
		"UPLOAD_PATH=/uploads/pending",
		"PROCESSED_PATH=/uploads/processed",
		"FAILED_PATH=/uploads/failed",
	}

	for _, envVar := range expectedEnvVars {
		if !strings.Contains(yaml, envVar) {
			t.Errorf("App should have environment variable %q, got:\n%s", envVar, yaml)
		}
	}
}

// TestComposeGenerator_FileProcessorSidecar_NotGenerated tests sidecar not generated without upload libraries.
func TestComposeGenerator_FileProcessorSidecar_NotGenerated(t *testing.T) {
	gen := NewComposeGenerator()
	detection := &models.Detection{
		Language:            "go",
		Version:             "1.23",
		Services:            []string{"postgres"},
		FileUploadLibraries: []string{}, // No file upload libraries
	}

	content, err := gen.GenerateContent(detection, "no-uploads")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	yaml := string(content)

	// Check file processor service is NOT included
	unwantedParts := []string{
		"file-processor:",
		"Dockerfile.processor",
		"PENDING_PATH",
		"PROCESSING_PATH",
		"PROCESSED_PATH",
		"FAILED_PATH",
	}

	for _, part := range unwantedParts {
		if strings.Contains(yaml, part) {
			t.Errorf("YAML should NOT contain %q when no upload libraries detected, got:\n%s", part, yaml)
		}
	}
}

// TestComposeGenerator_FileProcessorSidecar_WithOtherServices tests file processor with postgres/redis.
func TestComposeGenerator_FileProcessorSidecar_WithOtherServices(t *testing.T) {
	gen := NewComposeGenerator()
	detection := &models.Detection{
		Language:            "node",
		Version:             "20",
		Services:            []string{"postgres", "redis"},
		FileUploadLibraries: []string{"multer"},
		UploadPath:          "/uploads",
	}

	content, err := gen.GenerateContent(detection, "full-stack")
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
		// File processor
		"file-processor:",
		"Dockerfile.processor",
		"PENDING_PATH",
		// Volumes
		"volumes:",
		"postgres-data:",
		"redis-data:",
		"uploads:",
	}

	for _, part := range expectedParts {
		if !strings.Contains(yaml, part) {
			t.Errorf("YAML should contain %q, got:\n%s", part, yaml)
		}
	}
}

// TestComposeGenerator_FileProcessorSidecar_WithWorker tests file processor with worker sidecar.
func TestComposeGenerator_FileProcessorSidecar_WithWorker(t *testing.T) {
	gen := NewComposeGenerator()
	detection := &models.Detection{
		Language:            "node",
		Version:             "20",
		Services:            []string{"redis"},
		FileUploadLibraries: []string{"multer"},
		QueueLibraries:      []string{"bullmq"},
		WorkerCommand:       "npm run worker",
	}

	content, err := gen.GenerateContent(detection, "worker-upload")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	yaml := string(content)

	// Check worker also gets upload environment variables
	expectedParts := []string{
		"worker:",
		"npm run worker",
		"UPLOAD_PATH=/uploads/pending",
		"PROCESSED_PATH=/uploads/processed",
		"file-processor:",
	}

	for _, part := range expectedParts {
		if !strings.Contains(yaml, part) {
			t.Errorf("YAML should contain %q, got:\n%s", part, yaml)
		}
	}

	// Worker should also get uploads volume
	// Count occurrences of uploads:/uploads to verify both app and worker have it
	volumeCount := strings.Count(yaml, "uploads:/uploads")
	if volumeCount < 2 {
		t.Errorf("Both app and worker should have uploads volume, found %d occurrences", volumeCount)
	}
}

// TestBuildComposeConfig_FileProcessorSidecar tests buildConfig includes file processor config.
func TestBuildComposeConfig_FileProcessorSidecar(t *testing.T) {
	gen := NewComposeGenerator()

	detection := &models.Detection{
		Language:            "node",
		Version:             "20",
		Services:            []string{},
		FileUploadLibraries: []string{"multer", "sharp"},
		UploadPath:          "/custom/uploads",
	}

	config := gen.buildConfig(detection, "upload-app")

	// Verify file processor sidecar config
	if !config.FileProcessorSidecar.Enabled {
		t.Error("FileProcessorSidecar.Enabled should be true")
	}
	if config.FileProcessorSidecar.UploadPath != "/custom/uploads" {
		t.Errorf("FileProcessorSidecar.UploadPath = %v, want /custom/uploads", config.FileProcessorSidecar.UploadPath)
	}
	if len(config.FileProcessorSidecar.FileUploadLibraries) != 2 {
		t.Errorf("FileProcessorSidecar.FileUploadLibraries count = %d, want 2", len(config.FileProcessorSidecar.FileUploadLibraries))
	}
	if !config.FileProcessorSidecar.ProcessImages {
		t.Error("FileProcessorSidecar.ProcessImages should be true by default")
	}
	if config.FileProcessorSidecar.MemoryLimit != "512M" {
		t.Errorf("FileProcessorSidecar.MemoryLimit = %v, want 512M", config.FileProcessorSidecar.MemoryLimit)
	}
	if config.FileProcessorSidecar.CPULimit != "0.5" {
		t.Errorf("FileProcessorSidecar.CPULimit = %v, want 0.5", config.FileProcessorSidecar.CPULimit)
	}
}

// TestBuildComposeConfig_FileProcessorSidecar_DefaultUploadPath tests default upload path.
func TestBuildComposeConfig_FileProcessorSidecar_DefaultUploadPath(t *testing.T) {
	gen := NewComposeGenerator()

	detection := &models.Detection{
		Language:            "python",
		Version:             "3.11",
		FileUploadLibraries: []string{"python-multipart"},
		UploadPath:          "", // Empty, should default to /uploads
	}

	config := gen.buildConfig(detection, "default-path-app")

	if config.FileProcessorSidecar.UploadPath != "/uploads" {
		t.Errorf("FileProcessorSidecar.UploadPath = %v, want /uploads (default)", config.FileProcessorSidecar.UploadPath)
	}
}

// TestBuildComposeConfig_NoFileProcessorSidecar tests buildConfig without upload libraries.
func TestBuildComposeConfig_NoFileProcessorSidecar(t *testing.T) {
	gen := NewComposeGenerator()

	detection := &models.Detection{
		Language:            "rust",
		Version:             "1.75",
		Services:            []string{"redis"},
		FileUploadLibraries: []string{},
	}

	config := gen.buildConfig(detection, "no-upload-app")

	// Verify file processor sidecar is NOT enabled
	if config.FileProcessorSidecar.Enabled {
		t.Error("FileProcessorSidecar.Enabled should be false when no upload libraries")
	}
}

// TestComposeGenerator_FileProcessorSidecar_DependsOnApp tests file-processor depends_on app.
func TestComposeGenerator_FileProcessorSidecar_DependsOnApp(t *testing.T) {
	gen := NewComposeGenerator()
	detection := &models.Detection{
		Language:            "node",
		Version:             "20",
		Services:            []string{},
		FileUploadLibraries: []string{"multer"},
	}

	content, err := gen.GenerateContent(detection, "deps-test")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	yaml := string(content)

	// Check file-processor has depends_on app
	if !strings.Contains(yaml, "depends_on:") {
		t.Error("file-processor should have depends_on")
	}
}
