package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jpequegn/dockstart/internal/models"
)

// TestDevcontainerGenerator_GenerateContent tests the GenerateContent method
// which returns the generated JSON without writing to disk.
func TestDevcontainerGenerator_GenerateContent(t *testing.T) {
	tests := []struct {
		name        string
		detection   *models.Detection
		projectName string
		wantImage   string
		wantUser    string
		wantPorts   []int
		wantExts    []string
		wantCmd     string
	}{
		{
			name: "node project",
			detection: &models.Detection{
				Language:   "node",
				Version:    "20",
				Confidence: 1.0,
			},
			projectName: "my-node-app",
			wantImage:   "mcr.microsoft.com/devcontainers/javascript-node:20",
			wantUser:    "node",
			wantPorts:   []int{3000},
			wantExts:    []string{"dbaeumer.vscode-eslint"},
			wantCmd:     "npm install",
		},
		{
			name: "go project",
			detection: &models.Detection{
				Language:   "go",
				Version:    "1.23",
				Confidence: 1.0,
			},
			projectName: "my-go-app",
			wantImage:   "mcr.microsoft.com/devcontainers/go:1.23",
			wantUser:    "vscode",
			wantPorts:   []int{8080},
			wantExts:    []string{"golang.go"},
			wantCmd:     "go mod download",
		},
		{
			name: "python project",
			detection: &models.Detection{
				Language:   "python",
				Version:    "3.11",
				Confidence: 1.0,
			},
			projectName: "my-python-app",
			wantImage:   "mcr.microsoft.com/devcontainers/python:3.11",
			wantUser:    "vscode",
			wantPorts:   []int{8000},
			wantExts:    []string{"ms-python.python", "ms-python.vscode-pylance"},
			wantCmd:     "pip install -r requirements.txt",
		},
		{
			name: "rust project",
			detection: &models.Detection{
				Language:   "rust",
				Version:    "1.75",
				Confidence: 1.0,
			},
			projectName: "my-rust-app",
			wantImage:   "mcr.microsoft.com/devcontainers/rust:1.75",
			wantUser:    "vscode",
			wantPorts:   []int{8080},
			wantExts:    []string{"rust-lang.rust-analyzer"},
			wantCmd:     "cargo build",
		},
		{
			name: "unknown language falls back to base",
			detection: &models.Detection{
				Language:   "unknown",
				Version:    "",
				Confidence: 0.5,
			},
			projectName: "my-app",
			wantImage:   "mcr.microsoft.com/devcontainers/base:ubuntu",
			wantUser:    "vscode",
			wantPorts:   nil, // No default ports for unknown
			wantExts:    nil, // No extensions for unknown
			wantCmd:     "",  // No postCreateCommand for unknown
		},
		{
			name: "node with postgres service",
			detection: &models.Detection{
				Language:   "node",
				Version:    "20",
				Services:   []string{"postgres"},
				Confidence: 1.0,
			},
			projectName: "node-pg-app",
			wantImage:   "mcr.microsoft.com/devcontainers/javascript-node:20",
			wantUser:    "node",
			wantPorts:   []int{3000, 5432},
			wantExts:    []string{"dbaeumer.vscode-eslint"},
			wantCmd:     "npm install",
		},
		{
			name: "go with redis service",
			detection: &models.Detection{
				Language:   "go",
				Version:    "1.23",
				Services:   []string{"redis"},
				Confidence: 1.0,
			},
			projectName: "go-redis-app",
			wantImage:   "mcr.microsoft.com/devcontainers/go:1.23",
			wantUser:    "vscode",
			wantPorts:   []int{8080, 6379},
			wantExts:    []string{"golang.go"},
			wantCmd:     "go mod download",
		},
		{
			name: "python with multiple services",
			detection: &models.Detection{
				Language:   "python",
				Version:    "3.11",
				Services:   []string{"postgres", "redis"},
				Confidence: 1.0,
			},
			projectName: "python-full-app",
			wantImage:   "mcr.microsoft.com/devcontainers/python:3.11",
			wantUser:    "vscode",
			wantPorts:   []int{8000, 5432, 6379},
			wantExts:    []string{"ms-python.python", "ms-python.vscode-pylance"},
			wantCmd:     "pip install -r requirements.txt",
		},
	}

	gen := NewDevcontainerGenerator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := gen.GenerateContent(tt.detection, tt.projectName)
			if err != nil {
				t.Fatalf("GenerateContent() error = %v", err)
			}

			// Parse the JSON to verify structure
			var result map[string]interface{}
			if err := json.Unmarshal(content, &result); err != nil {
				t.Fatalf("Generated invalid JSON: %v", err)
			}

			// Check name
			if name, ok := result["name"].(string); !ok || name != tt.projectName {
				t.Errorf("name = %v, want %v", result["name"], tt.projectName)
			}

			// Check image (only if not using compose)
			if tt.detection.Services == nil || len(tt.detection.Services) == 0 {
				if image, ok := result["image"].(string); !ok || image != tt.wantImage {
					t.Errorf("image = %v, want %v", result["image"], tt.wantImage)
				}
			}

			// Check remoteUser
			if user, ok := result["remoteUser"].(string); !ok || user != tt.wantUser {
				t.Errorf("remoteUser = %v, want %v", result["remoteUser"], tt.wantUser)
			}

			// Check forwardPorts
			if tt.wantPorts != nil {
				ports, ok := result["forwardPorts"].([]interface{})
				if !ok {
					t.Errorf("forwardPorts not found or wrong type")
				} else {
					if len(ports) != len(tt.wantPorts) {
						t.Errorf("forwardPorts count = %d, want %d", len(ports), len(tt.wantPorts))
					}
				}
			}

			// Check extensions in customizations
			if tt.wantExts != nil {
				customizations, ok := result["customizations"].(map[string]interface{})
				if !ok {
					t.Errorf("customizations not found")
				} else {
					vscode, ok := customizations["vscode"].(map[string]interface{})
					if !ok {
						t.Errorf("vscode customizations not found")
					} else {
						exts, ok := vscode["extensions"].([]interface{})
						if !ok {
							t.Errorf("extensions not found")
						} else if len(exts) != len(tt.wantExts) {
							t.Errorf("extensions count = %d, want %d", len(exts), len(tt.wantExts))
						}
					}
				}
			}

			// Check postCreateCommand
			if tt.wantCmd != "" {
				if cmd, ok := result["postCreateCommand"].(string); !ok || cmd != tt.wantCmd {
					t.Errorf("postCreateCommand = %v, want %v", result["postCreateCommand"], tt.wantCmd)
				}
			}
		})
	}
}

// TestDevcontainerGenerator_Generate tests the Generate method which writes to disk.
func TestDevcontainerGenerator_Generate(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gen := NewDevcontainerGenerator()
	detection := &models.Detection{
		Language:   "go",
		Version:    "1.23",
		Confidence: 1.0,
	}

	// Generate the devcontainer.json
	err = gen.Generate(detection, tmpDir, "test-project")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify the .devcontainer directory was created
	devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
	if _, err := os.Stat(devcontainerDir); os.IsNotExist(err) {
		t.Error(".devcontainer directory was not created")
	}

	// Verify devcontainer.json was created
	devcontainerFile := filepath.Join(devcontainerDir, "devcontainer.json")
	content, err := os.ReadFile(devcontainerFile)
	if err != nil {
		t.Fatalf("Failed to read devcontainer.json: %v", err)
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(content, &result); err != nil {
		t.Fatalf("Generated invalid JSON: %v", err)
	}

	// Verify some key fields
	if name, ok := result["name"].(string); !ok || name != "test-project" {
		t.Errorf("name = %v, want test-project", result["name"])
	}
}

// TestDevcontainerGenerator_UseCompose tests that UseCompose is set when services are detected.
func TestDevcontainerGenerator_UseCompose(t *testing.T) {
	gen := NewDevcontainerGenerator()

	// Detection with services should use compose
	detectionWithServices := &models.Detection{
		Language:   "node",
		Version:    "20",
		Services:   []string{"postgres"},
		Confidence: 1.0,
	}

	content, err := gen.GenerateContent(detectionWithServices, "test-app")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(content, &result); err != nil {
		t.Fatalf("Generated invalid JSON: %v", err)
	}

	// When using compose, should have dockerComposeFile instead of image
	if _, hasImage := result["image"]; hasImage {
		t.Error("Should not have 'image' when using compose")
	}
	if _, hasCompose := result["dockerComposeFile"]; !hasCompose {
		t.Error("Should have 'dockerComposeFile' when services detected")
	}
	if service, ok := result["service"].(string); !ok || service != "app" {
		t.Errorf("service = %v, want app", result["service"])
	}
}

// TestBuildConfig tests the internal buildConfig function.
func TestBuildConfig(t *testing.T) {
	gen := NewDevcontainerGenerator()

	detection := &models.Detection{
		Language:   "node",
		Version:    "18",
		Services:   []string{"postgres", "redis"},
		Confidence: 1.0,
	}

	config := gen.buildConfig(detection, "my-app")

	// Verify config fields
	if config.Name != "my-app" {
		t.Errorf("Name = %v, want my-app", config.Name)
	}
	if !config.UseCompose {
		t.Error("UseCompose should be true when services detected")
	}
	if config.RemoteUser != "node" {
		t.Errorf("RemoteUser = %v, want node", config.RemoteUser)
	}
	// Should have 3000 (node default) + 5432 (postgres) + 6379 (redis)
	if len(config.ForwardPorts) != 3 {
		t.Errorf("ForwardPorts count = %d, want 3", len(config.ForwardPorts))
	}
}
