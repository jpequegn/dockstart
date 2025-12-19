package detector

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoggingDetection_Node tests logging library detection for Node.js projects.
func TestLoggingDetection_Node(t *testing.T) {
	tests := []struct {
		name            string
		packageJSON     string
		wantLibraries   []string
		wantLogFormat   string
	}{
		{
			name: "pino (JSON logger)",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"pino": "^8.0.0", "express": "^4.18.0"}
			}`,
			wantLibraries: []string{"pino"},
			wantLogFormat: "json",
		},
		{
			name: "winston (configurable)",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"winston": "^3.0.0"}
			}`,
			wantLibraries: []string{"winston"},
			wantLogFormat: "text",
		},
		{
			name: "bunyan (JSON logger)",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"bunyan": "^1.8.0"}
			}`,
			wantLibraries: []string{"bunyan"},
			wantLogFormat: "json",
		},
		{
			name: "morgan (request logger)",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"morgan": "^1.10.0", "express": "^4.18.0"}
			}`,
			wantLibraries: []string{"morgan"},
			wantLogFormat: "unknown",
		},
		{
			name: "multiple loggers",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {
					"pino": "^8.0.0",
					"morgan": "^1.10.0"
				}
			}`,
			wantLibraries: []string{"pino", "morgan"},
			wantLogFormat: "json",
		},
		{
			name: "no logging library",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"express": "^4.18.0"}
			}`,
			wantLibraries: []string{},
			wantLogFormat: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "dockstart-logging-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(tt.packageJSON), 0644); err != nil {
				t.Fatalf("Failed to write package.json: %v", err)
			}

			d := NewNodeDetector()
			detection, err := d.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detection failed: %v", err)
			}
			if detection == nil {
				t.Fatal("Expected detection, got nil")
			}

			// Check log format
			if detection.LogFormat != tt.wantLogFormat {
				t.Errorf("LogFormat = %q, want %q", detection.LogFormat, tt.wantLogFormat)
			}

			// Check libraries count
			if len(detection.LoggingLibraries) != len(tt.wantLibraries) {
				t.Errorf("LoggingLibraries count = %d, want %d (got: %v)",
					len(detection.LoggingLibraries), len(tt.wantLibraries), detection.LoggingLibraries)
			}

			// Check each expected library is present
			for _, wantLib := range tt.wantLibraries {
				if !detection.HasLoggingLibrary(wantLib) {
					t.Errorf("Expected library %q not found in %v", wantLib, detection.LoggingLibraries)
				}
			}
		})
	}
}

// TestLoggingDetection_Go tests logging library detection for Go projects.
func TestLoggingDetection_Go(t *testing.T) {
	tests := []struct {
		name          string
		goMod         string
		wantLibraries []string
		wantLogFormat string
	}{
		{
			name: "zap (JSON logger)",
			goMod: `
module test-app
go 1.21
require go.uber.org/zap v1.26.0
`,
			wantLibraries: []string{"zap"},
			wantLogFormat: "json",
		},
		{
			name: "zerolog (JSON logger)",
			goMod: `
module test-app
go 1.21
require github.com/rs/zerolog v1.31.0
`,
			wantLibraries: []string{"zerolog"},
			wantLogFormat: "json",
		},
		{
			name: "logrus (text logger)",
			goMod: `
module test-app
go 1.21
require github.com/sirupsen/logrus v1.9.3
`,
			wantLibraries: []string{"logrus"},
			wantLogFormat: "text",
		},
		{
			name: "multiple loggers",
			goMod: `
module test-app
go 1.21
require (
	go.uber.org/zap v1.26.0
	github.com/sirupsen/logrus v1.9.3
)
`,
			wantLibraries: []string{"zap", "logrus"},
			wantLogFormat: "json",
		},
		{
			name: "no logging library",
			goMod: `
module test-app
go 1.21
require github.com/gin-gonic/gin v1.9.0
`,
			wantLibraries: []string{},
			wantLogFormat: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "dockstart-logging-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(tt.goMod), 0644); err != nil {
				t.Fatalf("Failed to write go.mod: %v", err)
			}

			d := NewGoDetector()
			detection, err := d.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detection failed: %v", err)
			}
			if detection == nil {
				t.Fatal("Expected detection, got nil")
			}

			// Check log format
			if detection.LogFormat != tt.wantLogFormat {
				t.Errorf("LogFormat = %q, want %q", detection.LogFormat, tt.wantLogFormat)
			}

			// Check libraries count
			if len(detection.LoggingLibraries) != len(tt.wantLibraries) {
				t.Errorf("LoggingLibraries count = %d, want %d (got: %v)",
					len(detection.LoggingLibraries), len(tt.wantLibraries), detection.LoggingLibraries)
			}

			// Check each expected library is present
			for _, wantLib := range tt.wantLibraries {
				if !detection.HasLoggingLibrary(wantLib) {
					t.Errorf("Expected library %q not found in %v", wantLib, detection.LoggingLibraries)
				}
			}
		})
	}
}

// TestLoggingDetection_Python tests logging library detection for Python projects.
func TestLoggingDetection_Python(t *testing.T) {
	tests := []struct {
		name          string
		pyproject     string
		wantLibraries []string
		wantLogFormat string
	}{
		{
			name: "structlog (JSON logger)",
			pyproject: `
[project]
name = "test-app"
dependencies = ["structlog>=23.0.0"]
`,
			wantLibraries: []string{"structlog"},
			wantLogFormat: "json",
		},
		{
			name: "loguru (text logger)",
			pyproject: `
[project]
name = "test-app"
dependencies = ["loguru>=0.7.0"]
`,
			wantLibraries: []string{"loguru"},
			wantLogFormat: "text",
		},
		{
			name: "python-json-logger",
			pyproject: `
[project]
name = "test-app"
dependencies = ["python-json-logger>=2.0.0"]
`,
			wantLibraries: []string{"python-json-logger"},
			wantLogFormat: "json",
		},
		{
			name: "multiple loggers",
			pyproject: `
[project]
name = "test-app"
dependencies = ["structlog>=23.0.0", "rich>=13.0.0"]
`,
			wantLibraries: []string{"structlog", "rich"},
			wantLogFormat: "json",
		},
		{
			name: "no logging library",
			pyproject: `
[project]
name = "test-app"
dependencies = ["fastapi>=0.100.0"]
`,
			wantLibraries: []string{},
			wantLogFormat: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "dockstart-logging-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if err := os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte(tt.pyproject), 0644); err != nil {
				t.Fatalf("Failed to write pyproject.toml: %v", err)
			}

			d := NewPythonDetector()
			detection, err := d.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detection failed: %v", err)
			}
			if detection == nil {
				t.Fatal("Expected detection, got nil")
			}

			// Check log format
			if detection.LogFormat != tt.wantLogFormat {
				t.Errorf("LogFormat = %q, want %q", detection.LogFormat, tt.wantLogFormat)
			}

			// Check libraries count
			if len(detection.LoggingLibraries) != len(tt.wantLibraries) {
				t.Errorf("LoggingLibraries count = %d, want %d (got: %v)",
					len(detection.LoggingLibraries), len(tt.wantLibraries), detection.LoggingLibraries)
			}

			// Check each expected library is present
			for _, wantLib := range tt.wantLibraries {
				if !detection.HasLoggingLibrary(wantLib) {
					t.Errorf("Expected library %q not found in %v", wantLib, detection.LoggingLibraries)
				}
			}
		})
	}
}

// TestLoggingDetection_Rust tests logging library detection for Rust projects.
func TestLoggingDetection_Rust(t *testing.T) {
	tests := []struct {
		name          string
		cargoToml     string
		wantLibraries []string
		wantLogFormat string
	}{
		{
			name: "tracing (JSON capable)",
			cargoToml: `
[package]
name = "test-app"
version = "0.1.0"
edition = "2021"

[dependencies]
tracing = "0.1"
tracing-subscriber = "0.3"
`,
			wantLibraries: []string{"tracing"},
			wantLogFormat: "json",
		},
		{
			name: "log with env_logger (text)",
			cargoToml: `
[package]
name = "test-app"
version = "0.1.0"
edition = "2021"

[dependencies]
log = "0.4"
env_logger = "0.10"
`,
			wantLibraries: []string{"log", "env_logger"},
			wantLogFormat: "text",
		},
		{
			name: "log4rs (text)",
			cargoToml: `
[package]
name = "test-app"
version = "0.1.0"
edition = "2021"

[dependencies]
log4rs = "1.2"
`,
			wantLibraries: []string{"log4rs"},
			wantLogFormat: "text",
		},
		{
			name: "no logging library",
			cargoToml: `
[package]
name = "test-app"
version = "0.1.0"
edition = "2021"

[dependencies]
actix-web = "4.0"
`,
			wantLibraries: []string{},
			wantLogFormat: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "dockstart-logging-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if err := os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte(tt.cargoToml), 0644); err != nil {
				t.Fatalf("Failed to write Cargo.toml: %v", err)
			}

			d := NewRustDetector()
			detection, err := d.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detection failed: %v", err)
			}
			if detection == nil {
				t.Fatal("Expected detection, got nil")
			}

			// Check log format
			if detection.LogFormat != tt.wantLogFormat {
				t.Errorf("LogFormat = %q, want %q", detection.LogFormat, tt.wantLogFormat)
			}

			// Check libraries count
			if len(detection.LoggingLibraries) != len(tt.wantLibraries) {
				t.Errorf("LoggingLibraries count = %d, want %d (got: %v)",
					len(detection.LoggingLibraries), len(tt.wantLibraries), detection.LoggingLibraries)
			}

			// Check each expected library is present
			for _, wantLib := range tt.wantLibraries {
				if !detection.HasLoggingLibrary(wantLib) {
					t.Errorf("Expected library %q not found in %v", wantLib, detection.LoggingLibraries)
				}
			}
		})
	}
}

// TestLoggingDetection_PythonRequirements tests logging detection from requirements.txt
func TestLoggingDetection_PythonRequirements(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dockstart-logging-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	requirements := `
# Logging
structlog>=23.0.0
coloredlogs>=15.0.0

# Web framework
fastapi>=0.100.0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "requirements.txt"), []byte(requirements), 0644); err != nil {
		t.Fatalf("Failed to write requirements.txt: %v", err)
	}

	d := NewPythonDetector()
	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detection failed: %v", err)
	}
	if detection == nil {
		t.Fatal("Expected detection, got nil")
	}

	if detection.LogFormat != "json" {
		t.Errorf("LogFormat = %q, want %q", detection.LogFormat, "json")
	}

	if !detection.HasLoggingLibrary("structlog") {
		t.Error("Expected structlog to be detected")
	}
	if !detection.HasLoggingLibrary("coloredlogs") {
		t.Error("Expected coloredlogs to be detected")
	}
}

// TestHasStructuredLogging tests the HasStructuredLogging helper method
func TestHasStructuredLogging(t *testing.T) {
	tests := []struct {
		name        string
		packageJSON string
		want        bool
	}{
		{
			name: "with logging library",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"winston": "^3.0.0"}
			}`,
			want: true,
		},
		{
			name: "without logging library",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"express": "^4.18.0"}
			}`,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "dockstart-logging-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(tt.packageJSON), 0644); err != nil {
				t.Fatalf("Failed to write package.json: %v", err)
			}

			d := NewNodeDetector()
			detection, err := d.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detection failed: %v", err)
			}

			if detection.HasStructuredLogging() != tt.want {
				t.Errorf("HasStructuredLogging() = %v, want %v", detection.HasStructuredLogging(), tt.want)
			}
		})
	}
}
