package detector

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRustDetector_Name(t *testing.T) {
	d := NewRustDetector()
	if d.Name() != "rust" {
		t.Errorf("Name() = %v, want rust", d.Name())
	}
}

func TestRustDetector_Detect_NoCargoToml(t *testing.T) {
	// Create a temporary directory with no Cargo.toml
	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	d := NewRustDetector()
	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if detection != nil {
		t.Error("Expected nil detection for non-Rust project")
	}
}

func TestRustDetector_Detect_BasicCargoToml(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create Cargo.toml
	cargo := `
[package]
name = "my-rust-app"
version = "0.1.0"
edition = "2021"

[dependencies]
tokio = { version = "1.0", features = ["full"] }
serde = { version = "1.0", features = ["derive"] }
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte(cargo), 0644); err != nil {
		t.Fatalf("Failed to write Cargo.toml: %v", err)
	}

	d := NewRustDetector()
	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if detection == nil {
		t.Fatal("Expected detection, got nil")
	}

	if detection.Language != "rust" {
		t.Errorf("Language = %v, want rust", detection.Language)
	}
	if detection.Version != "1.75" {
		t.Errorf("Version = %v, want 1.75 (from edition 2021)", detection.Version)
	}
	if detection.Confidence < 0.7 {
		t.Errorf("Confidence = %v, want >= 0.7", detection.Confidence)
	}
}

func TestRustDetector_Detect_WithRustVersion(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cargo := `
[package]
name = "my-rust-app"
version = "0.1.0"
edition = "2021"
rust-version = "1.70"

[dependencies]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte(cargo), 0644); err != nil {
		t.Fatalf("Failed to write Cargo.toml: %v", err)
	}

	d := NewRustDetector()
	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	// rust-version should take precedence over edition
	if detection.Version != "1.70" {
		t.Errorf("Version = %v, want 1.70 (from rust-version)", detection.Version)
	}
}

func TestRustDetector_Detect_WithSqlx(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cargo := `
[package]
name = "db-app"
version = "0.1.0"
edition = "2021"

[dependencies]
sqlx = { version = "0.7", features = ["runtime-tokio", "postgres"] }
tokio = { version = "1.0", features = ["full"] }
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte(cargo), 0644); err != nil {
		t.Fatalf("Failed to write Cargo.toml: %v", err)
	}

	d := NewRustDetector()
	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if len(detection.Services) != 1 || detection.Services[0] != "postgres" {
		t.Errorf("Services = %v, want [postgres]", detection.Services)
	}
}

func TestRustDetector_Detect_WithRedis(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cargo := `
[package]
name = "cache-app"
version = "0.1.0"
edition = "2021"

[dependencies]
redis = "0.24"
tokio = { version = "1.0", features = ["full"] }
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte(cargo), 0644); err != nil {
		t.Fatalf("Failed to write Cargo.toml: %v", err)
	}

	d := NewRustDetector()
	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if !containsService(detection.Services, "redis") {
		t.Errorf("Services = %v, should contain redis", detection.Services)
	}
}

func TestRustDetector_Detect_MultipleServices(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cargo := `
[package]
name = "fullstack-app"
version = "0.1.0"
edition = "2021"

[dependencies]
diesel = { version = "2.1", features = ["postgres"] }
redis = "0.24"
tokio = { version = "1.0", features = ["full"] }
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte(cargo), 0644); err != nil {
		t.Fatalf("Failed to write Cargo.toml: %v", err)
	}

	d := NewRustDetector()
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

func TestRustDetector_Detect_SeaORM(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cargo := `
[package]
name = "sea-app"
version = "0.1.0"
edition = "2021"

[dependencies]
sea-orm = { version = "0.12", features = ["runtime-tokio-native-tls", "sqlx-postgres"] }
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte(cargo), 0644); err != nil {
		t.Fatalf("Failed to write Cargo.toml: %v", err)
	}

	d := NewRustDetector()
	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if !containsService(detection.Services, "postgres") {
		t.Errorf("Services = %v, should detect postgres from sea-orm", detection.Services)
	}
}

func TestRustDetector_Detect_Edition2018(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cargo := `
[package]
name = "legacy-app"
version = "0.1.0"
edition = "2018"

[dependencies]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte(cargo), 0644); err != nil {
		t.Fatalf("Failed to write Cargo.toml: %v", err)
	}

	d := NewRustDetector()
	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	if detection.Version != "1.31" {
		t.Errorf("Version = %v, want 1.31 (from edition 2018)", detection.Version)
	}
}

func TestRustDetector_HighConfidence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dockstart-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Full Cargo.toml with all fields
	cargo := `
[package]
name = "complete-app"
version = "0.1.0"
edition = "2021"

[dependencies]
tokio = { version = "1.0", features = ["full"] }
serde = "1.0"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte(cargo), 0644); err != nil {
		t.Fatalf("Failed to write Cargo.toml: %v", err)
	}

	d := NewRustDetector()
	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}

	// Should have high confidence with name, edition, and dependencies
	if detection.Confidence < 0.9 {
		t.Errorf("Confidence = %v, want >= 0.9 for complete Cargo.toml", detection.Confidence)
	}
}

func TestRustDetector_GetVSCodeExtensions(t *testing.T) {
	d := NewRustDetector()
	extensions := d.GetVSCodeExtensions()

	if len(extensions) < 1 {
		t.Error("Expected at least one VS Code extension")
	}

	// Should include rust-analyzer extension
	found := false
	for _, ext := range extensions {
		if ext == "rust-lang.rust-analyzer" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected rust-lang.rust-analyzer extension")
	}
}
