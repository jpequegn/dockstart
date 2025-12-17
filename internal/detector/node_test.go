package detector

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNodeDetector_Name(t *testing.T) {
	d := NewNodeDetector()
	if d.Name() != "node" {
		t.Errorf("expected name 'node', got '%s'", d.Name())
	}
}

func TestNodeDetector_Detect_NoPackageJSON(t *testing.T) {
	d := NewNodeDetector()

	// Create a temp directory without package.json
	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if detection != nil {
		t.Error("expected nil detection for non-Node.js project")
	}
}

func TestNodeDetector_Detect_BasicPackageJSON(t *testing.T) {
	d := NewNodeDetector()

	// Create temp directory with basic package.json
	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	packageJSON := `{
		"name": "test-project",
		"version": "1.0.0"
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatal(err)
	}

	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if detection == nil {
		t.Fatal("expected detection, got nil")
	}

	if detection.Language != "node" {
		t.Errorf("expected language 'node', got '%s'", detection.Language)
	}
	if detection.Version != "20" {
		t.Errorf("expected default version '20', got '%s'", detection.Version)
	}
	if detection.Confidence < 0.5 {
		t.Errorf("expected confidence >= 0.5, got %f", detection.Confidence)
	}
}

func TestNodeDetector_Detect_WithEngines(t *testing.T) {
	d := NewNodeDetector()

	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	packageJSON := `{
		"name": "test-project",
		"engines": {
			"node": ">=18.0.0"
		}
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatal(err)
	}

	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if detection == nil {
		t.Fatal("expected detection, got nil")
	}

	if detection.Version != "18" {
		t.Errorf("expected version '18', got '%s'", detection.Version)
	}
	// Higher confidence when engines is specified
	if detection.Confidence < 0.8 {
		t.Errorf("expected confidence >= 0.8 with engines specified, got %f", detection.Confidence)
	}
}

func TestNodeDetector_Detect_PostgresService(t *testing.T) {
	d := NewNodeDetector()

	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	packageJSON := `{
		"name": "test-project",
		"dependencies": {
			"pg": "^8.11.0",
			"express": "^4.18.0"
		}
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatal(err)
	}

	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if detection == nil {
		t.Fatal("expected detection, got nil")
	}

	if !detection.HasService("postgres") {
		t.Error("expected postgres service to be detected")
	}
}

func TestNodeDetector_Detect_RedisService(t *testing.T) {
	d := NewNodeDetector()

	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	packageJSON := `{
		"name": "test-project",
		"dependencies": {
			"ioredis": "^5.0.0"
		}
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatal(err)
	}

	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if detection == nil {
		t.Fatal("expected detection, got nil")
	}

	if !detection.HasService("redis") {
		t.Error("expected redis service to be detected")
	}
}

func TestNodeDetector_Detect_PrismaService(t *testing.T) {
	d := NewNodeDetector()

	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	packageJSON := `{
		"name": "test-project",
		"dependencies": {
			"@prisma/client": "^5.0.0"
		},
		"devDependencies": {
			"prisma": "^5.0.0"
		}
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatal(err)
	}

	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if detection == nil {
		t.Fatal("expected detection, got nil")
	}

	if !detection.HasService("postgres") {
		t.Error("expected postgres service to be detected for Prisma project")
	}
}

func TestNodeDetector_Detect_MultipleServices(t *testing.T) {
	d := NewNodeDetector()

	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	packageJSON := `{
		"name": "test-project",
		"engines": {
			"node": "^20.0.0"
		},
		"dependencies": {
			"pg": "^8.11.0",
			"redis": "^4.0.0",
			"express": "^4.18.0"
		}
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatal(err)
	}

	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if detection == nil {
		t.Fatal("expected detection, got nil")
	}

	if detection.Version != "20" {
		t.Errorf("expected version '20', got '%s'", detection.Version)
	}
	if !detection.HasService("postgres") {
		t.Error("expected postgres service")
	}
	if !detection.HasService("redis") {
		t.Error("expected redis service")
	}
	if len(detection.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(detection.Services))
	}
}

func TestParseVersionConstraint(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{">=18", "18"},
		{"^20.0.0", "20"},
		{"~18.17.0", "18"},
		{"20.x", "20"},
		{"20", "20"},
		{">=18.0.0 <21.0.0", "18"},
		{"", "20"}, // Default
	}

	for _, test := range tests {
		result := parseVersionConstraint(test.input)
		if result != test.expected {
			t.Errorf("parseVersionConstraint(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}
