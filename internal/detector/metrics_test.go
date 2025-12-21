package detector

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNodeMetricsDetection tests Node.js metrics library detection.
func TestNodeMetricsDetection(t *testing.T) {
	tests := []struct {
		name         string
		packageJSON  string
		wantLibs     []string
		wantPort     int
		wantPath     string
		wantNeedsMetrics bool
	}{
		{
			name: "prom-client detected",
			packageJSON: `{
				"name": "metrics-app",
				"dependencies": {
					"prom-client": "^15.0.0"
				}
			}`,
			wantLibs:     []string{"prom-client"},
			wantPort:     3000,
			wantPath:     "/metrics",
			wantNeedsMetrics: true,
		},
		{
			name: "express-prometheus-middleware detected",
			packageJSON: `{
				"name": "express-app",
				"dependencies": {
					"express": "^4.18.0",
					"express-prometheus-middleware": "^1.0.0"
				}
			}`,
			wantLibs:     []string{"express-prometheus-middleware"},
			wantPort:     3000,
			wantPath:     "/metrics",
			wantNeedsMetrics: true,
		},
		{
			name: "express-prom-bundle detected",
			packageJSON: `{
				"name": "bundled-app",
				"dependencies": {
					"express-prom-bundle": "^6.0.0"
				}
			}`,
			wantLibs:     []string{"express-prom-bundle"},
			wantPort:     3000,
			wantPath:     "/metrics",
			wantNeedsMetrics: true,
		},
		{
			name: "multiple metrics libraries",
			packageJSON: `{
				"name": "multi-metrics",
				"dependencies": {
					"prom-client": "^15.0.0",
					"express-prom-bundle": "^6.0.0"
				}
			}`,
			wantLibs:     []string{"prom-client", "express-prom-bundle"},
			wantPort:     3000,
			wantPath:     "/metrics",
			wantNeedsMetrics: true,
		},
		{
			name: "no metrics libraries",
			packageJSON: `{
				"name": "regular-app",
				"dependencies": {
					"express": "^4.18.0",
					"pg": "^8.0.0"
				}
			}`,
			wantLibs:     nil,
			wantPort:     0,
			wantPath:     "",
			wantNeedsMetrics: false,
		},
		{
			name: "fastify-metrics detected",
			packageJSON: `{
				"name": "fastify-app",
				"dependencies": {
					"fastify": "^4.0.0",
					"fastify-metrics": "^10.0.0"
				}
			}`,
			wantLibs:     []string{"fastify-metrics"},
			wantPort:     3000,
			wantPath:     "/metrics",
			wantNeedsMetrics: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory with package.json
			tmpDir, err := os.MkdirTemp("", "node-metrics-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(tt.packageJSON), 0644); err != nil {
				t.Fatalf("Failed to write package.json: %v", err)
			}

			detector := NewNodeDetector()
			detection, err := detector.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detect() error = %v", err)
			}

			// Check metrics libraries
			if len(detection.MetricsLibraries) != len(tt.wantLibs) {
				t.Errorf("MetricsLibraries count = %d, want %d", len(detection.MetricsLibraries), len(tt.wantLibs))
			}
			for _, wantLib := range tt.wantLibs {
				if !detection.HasMetricsLibrary(wantLib) {
					t.Errorf("Expected metrics library %q not found", wantLib)
				}
			}

			// Check port
			if detection.MetricsPort != tt.wantPort {
				t.Errorf("MetricsPort = %d, want %d", detection.MetricsPort, tt.wantPort)
			}

			// Check path
			if detection.MetricsPath != tt.wantPath {
				t.Errorf("MetricsPath = %q, want %q", detection.MetricsPath, tt.wantPath)
			}

			// Check NeedsMetrics
			if detection.NeedsMetrics() != tt.wantNeedsMetrics {
				t.Errorf("NeedsMetrics() = %v, want %v", detection.NeedsMetrics(), tt.wantNeedsMetrics)
			}
		})
	}
}

// TestGoMetricsDetection tests Go metrics library detection.
func TestGoMetricsDetection(t *testing.T) {
	tests := []struct {
		name         string
		goMod        string
		wantLibs     []string
		wantPort     int
		wantPath     string
		wantNeedsMetrics bool
	}{
		{
			name: "prometheus client_golang detected",
			goMod: `module example.com/metrics-app

go 1.21

require (
	github.com/prometheus/client_golang v1.17.0
)`,
			wantLibs:     []string{"prometheus-client"},
			wantPort:     8080,
			wantPath:     "/metrics",
			wantNeedsMetrics: true,
		},
		{
			name: "promauto detected",
			goMod: `module example.com/promauto-app

go 1.21

require (
	github.com/prometheus/promauto v0.1.0
)`,
			wantLibs:     []string{"promauto"},
			wantPort:     8080,
			wantPath:     "/metrics",
			wantNeedsMetrics: true,
		},
		{
			name: "victoriametrics detected",
			goMod: `module example.com/vm-app

go 1.21

require (
	github.com/VictoriaMetrics/metrics v1.24.0
)`,
			wantLibs:     []string{"victoriametrics"},
			wantPort:     8080,
			wantPath:     "/metrics",
			wantNeedsMetrics: true,
		},
		{
			name: "no metrics libraries",
			goMod: `module example.com/regular-app

go 1.21

require (
	github.com/gin-gonic/gin v1.9.0
	github.com/jackc/pgx/v5 v5.4.0
)`,
			wantLibs:     nil,
			wantPort:     0,
			wantPath:     "",
			wantNeedsMetrics: false,
		},
		{
			name: "opentelemetry prometheus detected",
			goMod: `module example.com/otel-app

go 1.21

require (
	go.opentelemetry.io/otel/exporters/prometheus v0.44.0
)`,
			wantLibs:     []string{"opentelemetry-prometheus"},
			wantPort:     8080,
			wantPath:     "/metrics",
			wantNeedsMetrics: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory with go.mod
			tmpDir, err := os.MkdirTemp("", "go-metrics-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(tt.goMod), 0644); err != nil {
				t.Fatalf("Failed to write go.mod: %v", err)
			}

			detector := NewGoDetector()
			detection, err := detector.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detect() error = %v", err)
			}

			// Check metrics libraries
			if len(detection.MetricsLibraries) != len(tt.wantLibs) {
				t.Errorf("MetricsLibraries count = %d, want %d", len(detection.MetricsLibraries), len(tt.wantLibs))
			}
			for _, wantLib := range tt.wantLibs {
				if !detection.HasMetricsLibrary(wantLib) {
					t.Errorf("Expected metrics library %q not found", wantLib)
				}
			}

			// Check port
			if detection.MetricsPort != tt.wantPort {
				t.Errorf("MetricsPort = %d, want %d", detection.MetricsPort, tt.wantPort)
			}

			// Check path
			if detection.MetricsPath != tt.wantPath {
				t.Errorf("MetricsPath = %q, want %q", detection.MetricsPath, tt.wantPath)
			}

			// Check NeedsMetrics
			if detection.NeedsMetrics() != tt.wantNeedsMetrics {
				t.Errorf("NeedsMetrics() = %v, want %v", detection.NeedsMetrics(), tt.wantNeedsMetrics)
			}
		})
	}
}

// TestPythonMetricsDetection tests Python metrics library detection.
func TestPythonMetricsDetection(t *testing.T) {
	tests := []struct {
		name         string
		requirements string
		wantLibs     []string
		wantPort     int
		wantPath     string
		wantNeedsMetrics bool
	}{
		{
			name:         "prometheus-client detected",
			requirements: "prometheus-client>=0.17.0",
			wantLibs:     []string{"prometheus-client"},
			wantPort:     8000,
			wantPath:     "/metrics",
			wantNeedsMetrics: true,
		},
		{
			name:         "prometheus_client detected (underscore)",
			requirements: "prometheus_client>=0.17.0",
			wantLibs:     []string{"prometheus-client"},
			wantPort:     8000,
			wantPath:     "/metrics",
			wantNeedsMetrics: true,
		},
		{
			name:         "prometheus-fastapi-instrumentator detected",
			requirements: "fastapi>=0.100.0\nprometheus-fastapi-instrumentator>=6.0.0",
			wantLibs:     []string{"prometheus-fastapi-instrumentator"},
			wantPort:     8000,
			wantPath:     "/metrics",
			wantNeedsMetrics: true,
		},
		{
			name:         "prometheus-flask-exporter detected",
			requirements: "flask>=2.0.0\nprometheus-flask-exporter>=0.20.0",
			wantLibs:     []string{"prometheus-flask-exporter"},
			wantPort:     8000,
			wantPath:     "/metrics",
			wantNeedsMetrics: true,
		},
		{
			name:         "django-prometheus detected",
			requirements: "django>=4.0\ndjango-prometheus>=2.2.0",
			wantLibs:     []string{"django-prometheus"},
			wantPort:     8000,
			wantPath:     "/metrics",
			wantNeedsMetrics: true,
		},
		{
			name:         "no metrics libraries",
			requirements: "fastapi>=0.100.0\npsycopg2-binary>=2.9.0",
			wantLibs:     nil,
			wantPort:     0,
			wantPath:     "",
			wantNeedsMetrics: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory with requirements.txt
			tmpDir, err := os.MkdirTemp("", "python-metrics-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if err := os.WriteFile(filepath.Join(tmpDir, "requirements.txt"), []byte(tt.requirements), 0644); err != nil {
				t.Fatalf("Failed to write requirements.txt: %v", err)
			}

			detector := NewPythonDetector()
			detection, err := detector.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detect() error = %v", err)
			}

			// Check metrics libraries
			if len(detection.MetricsLibraries) != len(tt.wantLibs) {
				t.Errorf("MetricsLibraries count = %d, want %d", len(detection.MetricsLibraries), len(tt.wantLibs))
			}
			for _, wantLib := range tt.wantLibs {
				if !detection.HasMetricsLibrary(wantLib) {
					t.Errorf("Expected metrics library %q not found", wantLib)
				}
			}

			// Check port
			if detection.MetricsPort != tt.wantPort {
				t.Errorf("MetricsPort = %d, want %d", detection.MetricsPort, tt.wantPort)
			}

			// Check path
			if detection.MetricsPath != tt.wantPath {
				t.Errorf("MetricsPath = %q, want %q", detection.MetricsPath, tt.wantPath)
			}

			// Check NeedsMetrics
			if detection.NeedsMetrics() != tt.wantNeedsMetrics {
				t.Errorf("NeedsMetrics() = %v, want %v", detection.NeedsMetrics(), tt.wantNeedsMetrics)
			}
		})
	}
}

// TestRustMetricsDetection tests Rust metrics library detection.
func TestRustMetricsDetection(t *testing.T) {
	tests := []struct {
		name         string
		cargoToml    string
		wantLibs     []string
		wantPort     int
		wantPath     string
		wantNeedsMetrics bool
	}{
		{
			name: "prometheus crate detected",
			cargoToml: `[package]
name = "metrics-app"
version = "0.1.0"
edition = "2021"

[dependencies]
prometheus = "0.13"
actix-web = "4"`,
			wantLibs:     []string{"prometheus"},
			wantPort:     8080,
			wantPath:     "/metrics",
			wantNeedsMetrics: true,
		},
		{
			name: "metrics crate detected",
			cargoToml: `[package]
name = "metrics-app"
version = "0.1.0"
edition = "2021"

[dependencies]
metrics = "0.21"
axum = "0.7"`,
			wantLibs:     []string{"metrics"},
			wantPort:     8080,
			wantPath:     "/metrics",
			wantNeedsMetrics: true,
		},
		{
			name: "metrics-exporter-prometheus detected",
			cargoToml: `[package]
name = "exporter-app"
version = "0.1.0"
edition = "2021"

[dependencies]
metrics = "0.21"
metrics-exporter-prometheus = "0.12"`,
			wantLibs:     []string{"metrics", "metrics-exporter-prometheus"},
			wantPort:     8080,
			wantPath:     "/metrics",
			wantNeedsMetrics: true,
		},
		{
			name: "actix-web-prom detected",
			cargoToml: `[package]
name = "actix-prom-app"
version = "0.1.0"
edition = "2021"

[dependencies]
actix-web = "4"
actix-web-prom = "0.6"`,
			wantLibs:     []string{"actix-web-prom"},
			wantPort:     8080,
			wantPath:     "/metrics",
			wantNeedsMetrics: true,
		},
		{
			name: "no metrics libraries",
			cargoToml: `[package]
name = "regular-app"
version = "0.1.0"
edition = "2021"

[dependencies]
actix-web = "4"
sqlx = { version = "0.7", features = ["postgres"] }`,
			wantLibs:     nil,
			wantPort:     0,
			wantPath:     "",
			wantNeedsMetrics: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory with Cargo.toml
			tmpDir, err := os.MkdirTemp("", "rust-metrics-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if err := os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte(tt.cargoToml), 0644); err != nil {
				t.Fatalf("Failed to write Cargo.toml: %v", err)
			}

			detector := NewRustDetector()
			detection, err := detector.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detect() error = %v", err)
			}

			// Check metrics libraries
			if len(detection.MetricsLibraries) != len(tt.wantLibs) {
				t.Errorf("MetricsLibraries count = %d, want %d", len(detection.MetricsLibraries), len(tt.wantLibs))
			}
			for _, wantLib := range tt.wantLibs {
				if !detection.HasMetricsLibrary(wantLib) {
					t.Errorf("Expected metrics library %q not found", wantLib)
				}
			}

			// Check port
			if detection.MetricsPort != tt.wantPort {
				t.Errorf("MetricsPort = %d, want %d", detection.MetricsPort, tt.wantPort)
			}

			// Check path
			if detection.MetricsPath != tt.wantPath {
				t.Errorf("MetricsPath = %q, want %q", detection.MetricsPath, tt.wantPath)
			}

			// Check NeedsMetrics
			if detection.NeedsMetrics() != tt.wantNeedsMetrics {
				t.Errorf("NeedsMetrics() = %v, want %v", detection.NeedsMetrics(), tt.wantNeedsMetrics)
			}
		})
	}
}

// TestMetricsModelHelpers tests the Detection model helper methods for metrics.
func TestMetricsModelHelpers(t *testing.T) {
	t.Run("GetMetricsPath defaults to /metrics", func(t *testing.T) {
		tmpDir, _ := os.MkdirTemp("", "metrics-path-*")
		defer os.RemoveAll(tmpDir)

		os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"name":"test"}`), 0644)
		detector := NewNodeDetector()
		detection, _ := detector.Detect(tmpDir)

		// No metrics library, should return default
		if detection.GetMetricsPath() != "/metrics" {
			t.Errorf("GetMetricsPath() = %q, want /metrics", detection.GetMetricsPath())
		}
	})

	t.Run("GetMetricsPort returns language default", func(t *testing.T) {
		tmpDir, _ := os.MkdirTemp("", "metrics-port-*")
		defer os.RemoveAll(tmpDir)

		os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"name":"test"}`), 0644)
		detector := NewNodeDetector()
		detection, _ := detector.Detect(tmpDir)

		// No metrics library, should return default port for node
		if detection.GetMetricsPort() != 3000 {
			t.Errorf("GetMetricsPort() for node = %d, want 3000", detection.GetMetricsPort())
		}
	})
}
