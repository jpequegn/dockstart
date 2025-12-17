package detector

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPythonDetector_Name(t *testing.T) {
	d := NewPythonDetector()
	if d.Name() != "python" {
		t.Errorf("Name() = %v, want python", d.Name())
	}
}

func TestPythonDetector_Detect_NoPythonFiles(t *testing.T) {
	// Create a temporary directory with no Python files
	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	d := NewPythonDetector()
	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if detection != nil {
		t.Error("Expected nil detection for non-Python project")
	}
}

func TestPythonDetector_Detect_PyprojectBasic(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create pyproject.toml
	pyproject := `
[project]
name = "my-python-app"
requires-python = ">=3.10"
dependencies = [
    "fastapi>=0.100.0",
    "uvicorn>=0.23.0",
]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte(pyproject), 0644); err != nil {
		t.Fatalf("Failed to write pyproject.toml: %v", err)
	}

	d := NewPythonDetector()
	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if detection == nil {
		t.Fatal("Expected detection, got nil")
	}

	if detection.Language != "python" {
		t.Errorf("Language = %v, want python", detection.Language)
	}
	if detection.Version != "3.10" {
		t.Errorf("Version = %v, want 3.10", detection.Version)
	}
	if detection.Confidence < 0.7 {
		t.Errorf("Confidence = %v, want >= 0.7", detection.Confidence)
	}
}

func TestPythonDetector_Detect_PyprojectWithPostgres(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pyproject := `
[project]
name = "db-app"
requires-python = ">=3.11"
dependencies = [
    "psycopg2-binary>=2.9.0",
    "sqlalchemy>=2.0.0",
]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte(pyproject), 0644); err != nil {
		t.Fatalf("Failed to write pyproject.toml: %v", err)
	}

	d := NewPythonDetector()
	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if len(detection.Services) != 1 || detection.Services[0] != "postgres" {
		t.Errorf("Services = %v, want [postgres]", detection.Services)
	}
}

func TestPythonDetector_Detect_PyprojectWithRedis(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pyproject := `
[project]
name = "cache-app"
requires-python = ">=3.11"
dependencies = [
    "redis>=4.0.0",
    "celery>=5.3.0",
]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte(pyproject), 0644); err != nil {
		t.Fatalf("Failed to write pyproject.toml: %v", err)
	}

	d := NewPythonDetector()
	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	// Should detect redis (both redis and celery are redis indicators)
	if !containsService(detection.Services, "redis") {
		t.Errorf("Services = %v, should contain redis", detection.Services)
	}
}

func TestPythonDetector_Detect_PyprojectMultipleServices(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pyproject := `
[project]
name = "full-stack-app"
requires-python = ">=3.11"
dependencies = [
    "django>=4.2.0",
    "psycopg2-binary>=2.9.0",
    "redis>=4.0.0",
]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte(pyproject), 0644); err != nil {
		t.Fatalf("Failed to write pyproject.toml: %v", err)
	}

	d := NewPythonDetector()
	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if len(detection.Services) != 2 {
		t.Errorf("Services count = %d, want 2", len(detection.Services))
	}
	if !containsService(detection.Services, "postgres") {
		t.Error("Should detect postgres")
	}
	if !containsService(detection.Services, "redis") {
		t.Error("Should detect redis")
	}
}

func TestPythonDetector_Detect_PoetryFormat(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pyproject := `
[tool.poetry]
name = "poetry-app"

[tool.poetry.dependencies]
python = "^3.10"
fastapi = "^0.100.0"
psycopg2-binary = "^2.9.0"

[tool.poetry.dev-dependencies]
pytest = "^7.0.0"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte(pyproject), 0644); err != nil {
		t.Fatalf("Failed to write pyproject.toml: %v", err)
	}

	d := NewPythonDetector()
	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if detection.Language != "python" {
		t.Errorf("Language = %v, want python", detection.Language)
	}
	if detection.Version != "3.10" {
		t.Errorf("Version = %v, want 3.10", detection.Version)
	}
	if !containsService(detection.Services, "postgres") {
		t.Error("Should detect postgres from psycopg2-binary")
	}
}

func TestPythonDetector_Detect_RequirementsTxt(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	requirements := `
# Web framework
flask>=2.0.0
gunicorn>=21.0.0

# Database
psycopg2>=2.9.0
sqlalchemy>=2.0.0

# Cache
redis>=4.0.0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "requirements.txt"), []byte(requirements), 0644); err != nil {
		t.Fatalf("Failed to write requirements.txt: %v", err)
	}

	d := NewPythonDetector()
	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if detection == nil {
		t.Fatal("Expected detection, got nil")
	}
	if detection.Language != "python" {
		t.Errorf("Language = %v, want python", detection.Language)
	}
	// Default version when not specified
	if detection.Version != "3.11" {
		t.Errorf("Version = %v, want 3.11 (default)", detection.Version)
	}
	if !containsService(detection.Services, "postgres") {
		t.Error("Should detect postgres")
	}
	if !containsService(detection.Services, "redis") {
		t.Error("Should detect redis")
	}
}

func TestPythonDetector_Detect_PyprojectTakesPrecedence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create both files
	pyproject := `
[project]
name = "modern-app"
requires-python = ">=3.12"
dependencies = ["fastapi>=0.100.0"]
`
	requirements := `
flask>=2.0.0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte(pyproject), 0644); err != nil {
		t.Fatalf("Failed to write pyproject.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "requirements.txt"), []byte(requirements), 0644); err != nil {
		t.Fatalf("Failed to write requirements.txt: %v", err)
	}

	d := NewPythonDetector()
	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	// pyproject.toml should take precedence
	if detection.Version != "3.12" {
		t.Errorf("Version = %v, want 3.12 (from pyproject.toml)", detection.Version)
	}
}

func TestPythonDetector_ParseVersionConstraint(t *testing.T) {
	d := NewPythonDetector()

	tests := []struct {
		constraint string
		want       string
	}{
		{">=3.10", "3.10"},
		{"^3.11", "3.11"},
		{">=3.9,<4.0", "3.9"},
		{"~=3.10.0", "3.10"},
		{"3.11", "3.11"},
		{"invalid", "3.11"}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.constraint, func(t *testing.T) {
			got := d.parseVersionConstraint(tt.constraint)
			if got != tt.want {
				t.Errorf("parseVersionConstraint(%q) = %v, want %v", tt.constraint, got, tt.want)
			}
		})
	}
}

func TestPythonDetector_HighConfidence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Full pyproject.toml with all fields
	pyproject := `
[project]
name = "complete-app"
requires-python = ">=3.11"
dependencies = [
    "fastapi>=0.100.0",
    "uvicorn>=0.23.0",
]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte(pyproject), 0644); err != nil {
		t.Fatalf("Failed to write pyproject.toml: %v", err)
	}

	d := NewPythonDetector()
	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	// Should have high confidence with name, version, and dependencies
	if detection.Confidence < 0.9 {
		t.Errorf("Confidence = %v, want >= 0.9 for complete pyproject.toml", detection.Confidence)
	}
}

func TestPythonDetector_GetVSCodeExtensions(t *testing.T) {
	d := NewPythonDetector()
	extensions := d.GetVSCodeExtensions()

	if len(extensions) < 1 {
		t.Error("Expected at least one VS Code extension")
	}

	// Should include Python extension
	found := false
	for _, ext := range extensions {
		if ext == "ms-python.python" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected ms-python.python extension")
	}
}
