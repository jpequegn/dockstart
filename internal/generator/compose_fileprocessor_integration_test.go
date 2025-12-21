package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jpequegn/dockstart/internal/models"
)

// TestComposeGenerator_FileProcessorSidecar_Integration tests the full integration
// of file processor sidecar generation with docker-compose.yml.
func TestComposeGenerator_FileProcessorSidecar_Integration(t *testing.T) {
	tests := []struct {
		name          string
		detection     *models.Detection
		projectName   string
		wantProcessor bool
		wantInYAML    []string
		dontWant      []string
		wantInEnv     []string
		wantVolumes   []string
		wantDependsOn []string
	}{
		{
			name: "node multer project - full file processor",
			detection: &models.Detection{
				Language:            "node",
				Version:             "20",
				Services:            []string{},
				FileUploadLibraries: []string{"multer"},
				UploadPath:          "/uploads",
				Confidence:          1.0,
			},
			projectName:   "upload-app",
			wantProcessor: true,
			wantInYAML: []string{
				"file-processor:",
				"Dockerfile.processor",
				"uploads:/uploads",
			},
			wantInEnv: []string{
				"PENDING_PATH=/uploads/pending",
				"PROCESSING_PATH=/uploads/processing",
				"PROCESSED_PATH=/uploads/processed",
				"FAILED_PATH=/uploads/failed",
				"POLL_INTERVAL=5",
				"MAX_FILE_SIZE=52428800",
				"RETRY_COUNT=3",
			},
			wantVolumes:   []string{"uploads:"},
			wantDependsOn: []string{"- app"},
		},
		{
			name: "python fastapi with python-multipart",
			detection: &models.Detection{
				Language:            "python",
				Version:             "3.11",
				Services:            []string{},
				FileUploadLibraries: []string{"python-multipart"},
				Confidence:          1.0,
			},
			projectName:   "py-api",
			wantProcessor: true,
			wantInYAML: []string{
				"file-processor:",
				"uploads:/uploads",
			},
			wantInEnv: []string{
				"PENDING_PATH=/uploads/pending",
				"PROCESSED_PATH=/uploads/processed",
			},
			wantVolumes: []string{"uploads:"},
		},
		{
			name: "go with upload directory detected",
			detection: &models.Detection{
				Language:            "go",
				Version:             "1.23",
				Services:            []string{"postgres"},
				FileUploadLibraries: []string{"multipart"},
				UploadPath:          "uploads",
				Confidence:          1.0,
			},
			projectName:   "go-upload",
			wantProcessor: true,
			wantInYAML: []string{
				"file-processor:",
				"uploads:/uploads",
				"postgres:",
			},
			wantVolumes: []string{"uploads:", "postgres-data:"},
		},
		{
			name: "rust with actix-multipart",
			detection: &models.Detection{
				Language:            "rust",
				Version:             "1.75",
				Services:            []string{"redis"},
				FileUploadLibraries: []string{"actix-multipart"},
				Confidence:          1.0,
			},
			projectName:   "rust-upload",
			wantProcessor: true,
			wantInYAML: []string{
				"file-processor:",
				"redis:",
			},
			wantVolumes: []string{"uploads:", "redis-data:"},
		},
		{
			name: "no upload library - no file processor",
			detection: &models.Detection{
				Language:            "node",
				Version:             "20",
				Services:            []string{"postgres"},
				FileUploadLibraries: []string{},
				Confidence:          1.0,
			},
			projectName:   "regular-app",
			wantProcessor: false,
			dontWant: []string{
				"file-processor:",
				"Dockerfile.processor",
				"PENDING_PATH",
				"PROCESSING_PATH",
				"PROCESSED_PATH",
				"FAILED_PATH",
			},
		},
		{
			name: "multiple upload libraries",
			detection: &models.Detection{
				Language:            "node",
				Version:             "20",
				Services:            []string{},
				FileUploadLibraries: []string{"multer", "formidable", "sharp"},
				Confidence:          1.0,
			},
			projectName:   "multi-upload",
			wantProcessor: true,
			wantInYAML: []string{
				"file-processor:",
				"uploads:/uploads",
			},
		},
		{
			name: "file processor with worker sidecar",
			detection: &models.Detection{
				Language:            "node",
				Version:             "20",
				Services:            []string{"redis"},
				FileUploadLibraries: []string{"multer"},
				QueueLibraries:      []string{"bullmq"},
				WorkerCommand:       "npm run worker",
				Confidence:          1.0,
			},
			projectName:   "full-app",
			wantProcessor: true,
			wantInYAML: []string{
				"file-processor:",
				"worker:",
				"npm run worker",
				"redis:",
			},
			wantVolumes: []string{"uploads:", "redis-data:"},
		},
		{
			name: "file processor with backup sidecar",
			detection: &models.Detection{
				Language:            "node",
				Version:             "20",
				Services:            []string{"postgres"},
				FileUploadLibraries: []string{"multer"},
				Confidence:          1.0,
			},
			projectName:   "backup-upload",
			wantProcessor: true,
			wantInYAML: []string{
				"file-processor:",
				"db-backup:",
				"postgres:",
			},
			wantVolumes: []string{"uploads:", "postgres-data:", "backups:"},
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

			// Check if file processor presence matches expectation
			hasProcessor := strings.Contains(yaml, "file-processor:")
			if hasProcessor != tt.wantProcessor {
				t.Errorf("file-processor presence = %v, want %v\nYAML:\n%s", hasProcessor, tt.wantProcessor, yaml)
			}

			// Check required strings
			for _, want := range tt.wantInYAML {
				if !strings.Contains(yaml, want) {
					t.Errorf("YAML should contain %q\nGot:\n%s", want, yaml)
				}
			}

			// Check unwanted strings
			for _, dontWant := range tt.dontWant {
				if strings.Contains(yaml, dontWant) {
					t.Errorf("YAML should NOT contain %q\nGot:\n%s", dontWant, yaml)
				}
			}

			// Check environment variables
			for _, env := range tt.wantInEnv {
				if !strings.Contains(yaml, env) {
					t.Errorf("YAML should contain environment variable %q\nGot:\n%s", env, yaml)
				}
			}

			// Check volumes
			for _, vol := range tt.wantVolumes {
				if !strings.Contains(yaml, vol) {
					t.Errorf("YAML should contain volume %q\nGot:\n%s", vol, yaml)
				}
			}

			// Check depends_on
			for _, dep := range tt.wantDependsOn {
				if !strings.Contains(yaml, dep) {
					t.Errorf("YAML should contain depends_on %q\nGot:\n%s", dep, yaml)
				}
			}
		})
	}
}

// TestComposeGenerator_FileProcessorSidecar_ResourceLimits tests resource limits.
func TestComposeGenerator_FileProcessorSidecar_ResourceLimits(t *testing.T) {
	gen := NewComposeGenerator()
	detection := &models.Detection{
		Language:            "node",
		Version:             "20",
		FileUploadLibraries: []string{"multer"},
	}

	content, err := gen.GenerateContent(detection, "limits-test")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	yaml := string(content)

	// Check resource limits
	expectedLimits := []string{
		"deploy:",
		"resources:",
		"limits:",
		"memory: 512M",
		"cpus: '0.5'",
	}

	for _, limit := range expectedLimits {
		if !strings.Contains(yaml, limit) {
			t.Errorf("YAML should contain resource limit %q\nGot:\n%s", limit, yaml)
		}
	}
}

// TestEndToEndFileProcessorGeneration tests complete file generation to disk.
func TestEndToEndFileProcessorGeneration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dockstart-e2e-processor-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	detection := &models.Detection{
		Language:            "node",
		Version:             "20",
		Services:            []string{"postgres"},
		FileUploadLibraries: []string{"multer"},
		UploadPath:          "/uploads",
		Confidence:          1.0,
	}

	// Generate compose file
	composeGen := NewComposeGenerator()
	err = composeGen.Generate(detection, tmpDir, "e2e-test")
	if err != nil {
		t.Fatalf("ComposeGenerator.Generate() error = %v", err)
	}

	// Generate processor sidecar files
	processorGen := NewProcessorSidecarGenerator()
	err = processorGen.Generate(detection, tmpDir, "e2e-test")
	if err != nil {
		t.Fatalf("ProcessorSidecarGenerator.Generate() error = %v", err)
	}

	devcontainerDir := filepath.Join(tmpDir, ".devcontainer")

	// Verify all expected files exist
	expectedFiles := []string{
		"docker-compose.yml",
		"Dockerfile.processor",
		"entrypoint.processor.sh",
		"scripts/process-files.sh",
		"scripts/process-image.sh",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(devcontainerDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s does not exist", file)
		}
	}

	// Verify directory structure
	expectedDirs := []string{
		"scripts",
		"files/pending",
		"files/processing",
		"files/processed",
		"files/failed",
	}

	for _, dir := range expectedDirs {
		path := filepath.Join(devcontainerDir, dir)
		stat, err := os.Stat(path)
		if os.IsNotExist(err) {
			t.Errorf("Expected directory %s does not exist", dir)
		} else if !stat.IsDir() {
			t.Errorf("Expected %s to be a directory", dir)
		}
	}

	// Verify compose file contains processor service
	composeContent, err := os.ReadFile(filepath.Join(devcontainerDir, "docker-compose.yml"))
	if err != nil {
		t.Fatalf("Failed to read docker-compose.yml: %v", err)
	}

	if !strings.Contains(string(composeContent), "file-processor:") {
		t.Error("docker-compose.yml should contain file-processor service")
	}
}

// TestGeneratedBashScriptsValid validates all generated bash scripts.
func TestGeneratedBashScriptsValid(t *testing.T) {
	// Check if bash is available
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not available, skipping script validation")
	}

	gen := NewProcessorSidecarGenerator()
	config := &ProcessorSidecarConfig{
		ProcessImages:    true,
		ProcessDocuments: true,
		ProcessVideo:     true,
		UseInotify:       true,
		PollInterval:     5,
		MaxFileSize:      52428800,
		ThumbnailSize:    "200x200",
		ProjectName:      "test",
	}

	scripts := []struct {
		name     string
		generate func() ([]byte, error)
	}{
		{"process-files.sh", func() ([]byte, error) { return gen.GenerateProcessScript(config) }},
		{"process-image.sh", func() ([]byte, error) { return gen.GenerateImageScript(config) }},
		{"process-document.sh", func() ([]byte, error) { return gen.GenerateDocumentScript(config) }},
		{"process-video.sh", func() ([]byte, error) { return gen.GenerateVideoScript(config) }},
		{"entrypoint.processor.sh", func() ([]byte, error) { return gen.GenerateEntrypoint(config) }},
	}

	for _, script := range scripts {
		t.Run(script.name, func(t *testing.T) {
			content, err := script.generate()
			if err != nil {
				t.Fatalf("Failed to generate %s: %v", script.name, err)
			}

			// Write to temp file
			tmpFile, err := os.CreateTemp("", "script-*.sh")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.Write(content); err != nil {
				t.Fatalf("Failed to write script: %v", err)
			}
			tmpFile.Close()

			// Validate with bash -n
			cmd := exec.Command(bashPath, "-n", tmpFile.Name())
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("%s has syntax errors:\n%s", script.name, string(output))
			}
		})
	}
}

// TestDockerfileProcessorContent validates Dockerfile.processor content.
func TestDockerfileProcessorContent(t *testing.T) {
	gen := NewProcessorSidecarGenerator()

	tests := []struct {
		name     string
		config   *ProcessorSidecarConfig
		contains []string
	}{
		{
			name: "basic image processing",
			config: &ProcessorSidecarConfig{
				ProcessImages: true,
			},
			contains: []string{
				"FROM alpine",
				"imagemagick",
				"HEALTHCHECK",
				"ENTRYPOINT",
			},
		},
		{
			name: "all processing types",
			config: &ProcessorSidecarConfig{
				ProcessImages:    true,
				ProcessDocuments: true,
				ProcessVideo:     true,
				UseInotify:       true,
			},
			contains: []string{
				"imagemagick",
				"poppler-utils",
				"ffmpeg",
				"inotify-tools",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := gen.GenerateDockerfile(tt.config)
			if err != nil {
				t.Fatalf("GenerateDockerfile() error = %v", err)
			}

			dockerfile := string(content)
			for _, want := range tt.contains {
				if !strings.Contains(dockerfile, want) {
					t.Errorf("Dockerfile should contain %q\nGot:\n%s", want, dockerfile)
				}
			}
		})
	}
}

// TestFileProcessorWithAllSidecars tests file processor combined with all other sidecars.
func TestFileProcessorWithAllSidecars(t *testing.T) {
	gen := NewComposeGenerator()
	detection := &models.Detection{
		Language:            "node",
		Version:             "20",
		Services:            []string{"postgres", "redis"},
		FileUploadLibraries: []string{"multer"},
		QueueLibraries:      []string{"bullmq"},
		WorkerCommand:       "npm run worker",
		LoggingLibraries:    []string{"pino"},
		LogFormat:           "json",
		Confidence:          1.0,
	}

	content, err := gen.GenerateContent(detection, "full-stack")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	yaml := string(content)

	// All sidecars should be present
	expectedServices := []string{
		"app:",
		"worker:",
		"postgres:",
		"redis:",
		"fluent-bit:",
		"file-processor:",
		"db-backup:",
	}

	for _, service := range expectedServices {
		if !strings.Contains(yaml, service) {
			t.Errorf("YAML should contain service %q\nGot:\n%s", service, yaml)
		}
	}

	// All volumes should be present
	expectedVolumes := []string{
		"postgres-data:",
		"redis-data:",
		"uploads:",
		"backups:",
		"fluent-bit-logs:",
	}

	for _, vol := range expectedVolumes {
		if !strings.Contains(yaml, vol) {
			t.Errorf("YAML should contain volume %q\nGot:\n%s", vol, yaml)
		}
	}
}

// TestFileProcessorAppEnvironmentVariables verifies app service gets upload env vars.
func TestFileProcessorAppEnvironmentVariables(t *testing.T) {
	gen := NewComposeGenerator()
	detection := &models.Detection{
		Language:            "node",
		Version:             "20",
		FileUploadLibraries: []string{"multer"},
	}

	content, err := gen.GenerateContent(detection, "env-app")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	yaml := string(content)

	// App service should have upload environment variables
	appEnvVars := []string{
		"UPLOAD_PATH=/uploads/pending",
		"PROCESSED_PATH=/uploads/processed",
		"FAILED_PATH=/uploads/failed",
	}

	for _, env := range appEnvVars {
		if !strings.Contains(yaml, env) {
			t.Errorf("App service should have environment variable %q\nGot:\n%s", env, yaml)
		}
	}
}
