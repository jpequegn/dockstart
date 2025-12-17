package detector

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGoDetector_Name(t *testing.T) {
	d := NewGoDetector()
	if d.Name() != "go" {
		t.Errorf("expected name 'go', got '%s'", d.Name())
	}
}

func TestGoDetector_Detect_NoGoMod(t *testing.T) {
	d := NewGoDetector()

	// Create a temp directory without go.mod
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
		t.Error("expected nil detection for non-Go project")
	}
}

func TestGoDetector_Detect_BasicGoMod(t *testing.T) {
	d := NewGoDetector()

	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	goMod := `module github.com/test/project

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if detection == nil {
		t.Fatal("expected detection, got nil")
	}

	if detection.Language != "go" {
		t.Errorf("expected language 'go', got '%s'", detection.Language)
	}
	if detection.Version != "1.21" {
		t.Errorf("expected version '1.21', got '%s'", detection.Version)
	}
	if detection.Confidence < 0.6 {
		t.Errorf("expected confidence >= 0.6, got %f", detection.Confidence)
	}
}

func TestGoDetector_Detect_WithVersion123(t *testing.T) {
	d := NewGoDetector()

	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	goMod := `module github.com/test/project

go 1.23
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if detection == nil {
		t.Fatal("expected detection, got nil")
	}

	if detection.Version != "1.23" {
		t.Errorf("expected version '1.23', got '%s'", detection.Version)
	}
}

func TestGoDetector_Detect_PostgresService_Pgx(t *testing.T) {
	d := NewGoDetector()

	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	goMod := `module github.com/test/project

go 1.21

require (
	github.com/jackc/pgx/v5 v5.5.0
	github.com/gin-gonic/gin v1.9.0
)
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
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
		t.Error("expected postgres service to be detected for pgx dependency")
	}
}

func TestGoDetector_Detect_PostgresService_LibPq(t *testing.T) {
	d := NewGoDetector()

	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	goMod := `module github.com/test/project

go 1.21

require github.com/lib/pq v1.10.9
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
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
		t.Error("expected postgres service to be detected for lib/pq dependency")
	}
}

func TestGoDetector_Detect_RedisService(t *testing.T) {
	d := NewGoDetector()

	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	goMod := `module github.com/test/project

go 1.21

require (
	github.com/redis/go-redis/v9 v9.0.0
)
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
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

func TestGoDetector_Detect_MultipleServices(t *testing.T) {
	d := NewGoDetector()

	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	goMod := `module github.com/test/project

go 1.22

require (
	github.com/jackc/pgx/v5 v5.5.0
	github.com/redis/go-redis/v9 v9.0.0
	github.com/gin-gonic/gin v1.9.0
)
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if detection == nil {
		t.Fatal("expected detection, got nil")
	}

	if detection.Version != "1.22" {
		t.Errorf("expected version '1.22', got '%s'", detection.Version)
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

func TestGoDetector_Detect_GormPostgres(t *testing.T) {
	d := NewGoDetector()

	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	goMod := `module github.com/test/project

go 1.21

require (
	gorm.io/gorm v1.25.0
	gorm.io/driver/postgres v1.5.0
)
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
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
		t.Error("expected postgres service to be detected for GORM postgres driver")
	}
}

func TestGoDetector_Detect_HighConfidence(t *testing.T) {
	d := NewGoDetector()

	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// A complete go.mod with module, version, and dependencies
	goMod := `module github.com/test/project

go 1.22

require (
	github.com/gin-gonic/gin v1.9.0
	github.com/jackc/pgx/v5 v5.5.0
)
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if detection == nil {
		t.Fatal("expected detection, got nil")
	}

	// Should have high confidence: base (0.6) + module (0.2) + non-default version (0.1) + deps (0.1) = 1.0
	if detection.Confidence < 0.9 {
		t.Errorf("expected confidence >= 0.9 for complete go.mod, got %f", detection.Confidence)
	}
}

func TestGoDetector_GetVSCodeExtensions(t *testing.T) {
	d := NewGoDetector()
	extensions := d.GetVSCodeExtensions()

	if len(extensions) == 0 {
		t.Error("expected at least one VS Code extension")
	}

	found := false
	for _, ext := range extensions {
		if ext == "golang.go" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected golang.go extension")
	}
}
