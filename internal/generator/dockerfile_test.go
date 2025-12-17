package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jpequegn/dockstart/internal/models"
)

// TestDockerfileGenerator_GenerateContent tests the GenerateContent method.
func TestDockerfileGenerator_GenerateContent(t *testing.T) {
	tests := []struct {
		name        string
		detection   *models.Detection
		projectName string
		wantImage   string
		wantPkgMgr  string
		wantInFile  []string
		dontWant    []string
	}{
		{
			name: "node project",
			detection: &models.Detection{
				Language:   "node",
				Version:    "20",
				Confidence: 1.0,
			},
			projectName: "my-node-app",
			wantImage:   "node:20",
			wantPkgMgr:  "apt-get",
			wantInFile: []string{
				"FROM node:20",
				"apt-get update",
				"apt-get install -y",
				"git",
				"curl",
				"WORKDIR /workspace",
				"my-node-app",
			},
			dontWant: []string{
				"pip install",
				"rustup",
			},
		},
		{
			name: "go project",
			detection: &models.Detection{
				Language:   "go",
				Version:    "1.23",
				Confidence: 1.0,
			},
			projectName: "my-go-app",
			wantImage:   "golang:1.23",
			wantPkgMgr:  "apt-get",
			wantInFile: []string{
				"FROM golang:1.23",
				"apt-get update",
				"WORKDIR /workspace",
			},
			dontWant: []string{
				"pip install",
				"rustup",
			},
		},
		{
			name: "python project",
			detection: &models.Detection{
				Language:   "python",
				Version:    "3.11",
				Confidence: 1.0,
			},
			projectName: "my-python-app",
			wantImage:   "python:3.11",
			wantPkgMgr:  "apt-get",
			wantInFile: []string{
				"FROM python:3.11",
				"apt-get update",
				"WORKDIR /workspace",
				"pip install --upgrade pip",
			},
			dontWant: []string{
				"rustup",
			},
		},
		{
			name: "rust project",
			detection: &models.Detection{
				Language:   "rust",
				Version:    "1.75",
				Confidence: 1.0,
			},
			projectName: "my-rust-app",
			wantImage:   "rust:1.75",
			wantPkgMgr:  "apt-get",
			wantInFile: []string{
				"FROM rust:1.75",
				"apt-get update",
				"WORKDIR /workspace",
				"rustup component add rustfmt clippy",
			},
			dontWant: []string{
				"pip install",
			},
		},
		{
			name: "unknown language defaults to ubuntu",
			detection: &models.Detection{
				Language:   "unknown",
				Version:    "",
				Confidence: 0.5,
			},
			projectName: "my-app",
			wantImage:   "ubuntu:22.04",
			wantPkgMgr:  "apt-get",
			wantInFile: []string{
				"FROM ubuntu:22.04",
				"apt-get update",
				"WORKDIR /workspace",
			},
			dontWant: []string{
				"pip install",
				"rustup",
			},
		},
	}

	gen := NewDockerfileGenerator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := gen.GenerateContent(tt.detection, tt.projectName)
			if err != nil {
				t.Fatalf("GenerateContent() error = %v", err)
			}

			dockerfile := string(content)

			// Check required strings are present
			for _, want := range tt.wantInFile {
				if !strings.Contains(dockerfile, want) {
					t.Errorf("Dockerfile should contain %q, got:\n%s", want, dockerfile)
				}
			}

			// Check unwanted strings are absent
			for _, dontWant := range tt.dontWant {
				if strings.Contains(dockerfile, dontWant) {
					t.Errorf("Dockerfile should NOT contain %q", dontWant)
				}
			}
		})
	}
}

// TestDockerfileGenerator_Generate tests the Generate method which writes to disk.
func TestDockerfileGenerator_Generate(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "dockstart-dockerfile-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gen := NewDockerfileGenerator()
	detection := &models.Detection{
		Language:   "go",
		Version:    "1.23",
		Confidence: 1.0,
	}

	// Generate the Dockerfile
	err = gen.Generate(detection, tmpDir, "test-project")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify the .devcontainer directory was created
	devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
	if _, err := os.Stat(devcontainerDir); os.IsNotExist(err) {
		t.Error(".devcontainer directory was not created")
	}

	// Verify Dockerfile was created
	dockerfilePath := filepath.Join(devcontainerDir, "Dockerfile")
	content, err := os.ReadFile(dockerfilePath)
	if err != nil {
		t.Fatalf("Failed to read Dockerfile: %v", err)
	}

	// Verify some key content
	dockerfile := string(content)
	if !strings.Contains(dockerfile, "golang:1.23") {
		t.Error("Dockerfile should contain golang:1.23")
	}
	if !strings.Contains(dockerfile, "test-project") {
		t.Error("Dockerfile should contain project name in comment")
	}
}

// TestBuildDockerfileConfig tests the internal buildConfig function.
func TestBuildDockerfileConfig(t *testing.T) {
	gen := NewDockerfileGenerator()

	tests := []struct {
		name        string
		detection   *models.Detection
		projectName string
		wantImage   string
		wantPkgMgr  string
	}{
		{
			name: "node config",
			detection: &models.Detection{
				Language: "node",
				Version:  "18",
			},
			projectName: "node-app",
			wantImage:   "node:18",
			wantPkgMgr:  "apt-get",
		},
		{
			name: "go config",
			detection: &models.Detection{
				Language: "go",
				Version:  "1.22",
			},
			projectName: "go-app",
			wantImage:   "golang:1.22",
			wantPkgMgr:  "apt-get",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := gen.buildConfig(tt.detection, tt.projectName)

			if config.Name != tt.projectName {
				t.Errorf("Name = %v, want %v", config.Name, tt.projectName)
			}
			if config.BaseImage != tt.wantImage {
				t.Errorf("BaseImage = %v, want %v", config.BaseImage, tt.wantImage)
			}
			if config.PackageManager != tt.wantPkgMgr {
				t.Errorf("PackageManager = %v, want %v", config.PackageManager, tt.wantPkgMgr)
			}
		})
	}
}

// TestDockerfileGenerator_HeaderComment tests that header comments are generated.
func TestDockerfileGenerator_HeaderComment(t *testing.T) {
	gen := NewDockerfileGenerator()
	detection := &models.Detection{
		Language: "node",
		Version:  "20",
	}

	content, err := gen.GenerateContent(detection, "test-app")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	dockerfile := string(content)
	if !strings.Contains(dockerfile, "# Dockerfile for test-app") {
		t.Error("Dockerfile should contain header comment with project name")
	}
	if !strings.Contains(dockerfile, "Generated by dockstart") {
		t.Error("Dockerfile should contain 'Generated by dockstart' attribution")
	}
}

// TestDockerfileGenerator_DevTools tests that common dev tools are installed.
func TestDockerfileGenerator_DevTools(t *testing.T) {
	gen := NewDockerfileGenerator()
	detection := &models.Detection{
		Language: "node",
		Version:  "20",
	}

	content, err := gen.GenerateContent(detection, "test-app")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	dockerfile := string(content)

	// Check common dev tools are installed
	devTools := []string{"git", "curl", "wget", "vim"}
	for _, tool := range devTools {
		if !strings.Contains(dockerfile, tool) {
			t.Errorf("Dockerfile should install %s", tool)
		}
	}
}

// TestDockerfileGenerator_CacheCleanup tests that package cache is cleaned.
func TestDockerfileGenerator_CacheCleanup(t *testing.T) {
	gen := NewDockerfileGenerator()
	detection := &models.Detection{
		Language: "go",
		Version:  "1.23",
	}

	content, err := gen.GenerateContent(detection, "test-app")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	dockerfile := string(content)

	// Check that apt cache is cleaned (reduces image size)
	if !strings.Contains(dockerfile, "rm -rf /var/lib/apt/lists/*") {
		t.Error("Dockerfile should clean apt cache to reduce image size")
	}
}

// TestDockerfileGenerator_SleepCommand tests that sleep infinity is used.
func TestDockerfileGenerator_SleepCommand(t *testing.T) {
	gen := NewDockerfileGenerator()
	detection := &models.Detection{
		Language: "python",
		Version:  "3.11",
	}

	content, err := gen.GenerateContent(detection, "test-app")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	dockerfile := string(content)

	// Check that sleep infinity is the default command
	// This keeps the container running for VS Code to attach
	if !strings.Contains(dockerfile, `CMD ["sleep", "infinity"]`) {
		t.Error("Dockerfile should use 'sleep infinity' as default command")
	}
}
