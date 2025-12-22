package detector

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNodeTracingDetection tests Node.js tracing library detection.
func TestNodeTracingDetection(t *testing.T) {
	tests := []struct {
		name             string
		packageJSON      string
		expectedLibs     []string
		expectedProtocol string
	}{
		{
			name: "OpenTelemetry SDK Node",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {
					"@opentelemetry/sdk-node": "^0.45.0",
					"@opentelemetry/auto-instrumentations-node": "^0.40.0"
				}
			}`,
			expectedLibs:     []string{"@opentelemetry/sdk-node", "@opentelemetry/auto-instrumentations-node"},
			expectedProtocol: "otlp",
		},
		{
			name: "OpenTelemetry OTLP Exporter",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {
					"@opentelemetry/api": "^1.7.0",
					"@opentelemetry/exporter-trace-otlp-http": "^0.45.0"
				}
			}`,
			expectedLibs:     []string{"@opentelemetry/api", "@opentelemetry/exporter-trace-otlp-http"},
			expectedProtocol: "otlp",
		},
		{
			name: "Jaeger Client",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {
					"jaeger-client": "^3.19.0"
				}
			}`,
			expectedLibs:     []string{"jaeger-client"},
			expectedProtocol: "jaeger",
		},
		{
			name: "OpenTelemetry with Jaeger Exporter",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {
					"@opentelemetry/sdk-node": "^0.45.0",
					"@opentelemetry/exporter-jaeger": "^1.20.0"
				}
			}`,
			expectedLibs:     []string{"@opentelemetry/sdk-node", "@opentelemetry/exporter-jaeger"},
			expectedProtocol: "otlp",
		},
		{
			name: "Zipkin",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {
					"zipkin": "^0.22.0"
				}
			}`,
			expectedLibs:     []string{"zipkin"},
			expectedProtocol: "zipkin",
		},
		{
			name: "No Tracing Libraries",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {
					"express": "^4.18.0"
				}
			}`,
			expectedLibs:     nil,
			expectedProtocol: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir, err := os.MkdirTemp("", "node-tracing-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Write package.json
			if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(tt.packageJSON), 0644); err != nil {
				t.Fatalf("Failed to write package.json: %v", err)
			}

			// Run detection
			detector := NewNodeDetector()
			detection, err := detector.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detection failed: %v", err)
			}

			// Verify tracing libraries
			if len(tt.expectedLibs) == 0 {
				if len(detection.TracingLibraries) != 0 {
					t.Errorf("Expected no tracing libraries, got %v", detection.TracingLibraries)
				}
			} else {
				for _, expectedLib := range tt.expectedLibs {
					found := false
					for _, lib := range detection.TracingLibraries {
						if lib == expectedLib {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected tracing library %q not found in %v", expectedLib, detection.TracingLibraries)
					}
				}
			}

			// Verify protocol
			if tt.expectedProtocol == "" {
				if detection.TracingProtocol != "" && detection.TracingProtocol != "unknown" {
					t.Errorf("Expected no protocol, got %q", detection.TracingProtocol)
				}
			} else {
				if detection.TracingProtocol != tt.expectedProtocol {
					t.Errorf("Expected protocol %q, got %q", tt.expectedProtocol, detection.TracingProtocol)
				}
			}
		})
	}
}

// TestGoTracingDetection tests Go tracing library detection.
func TestGoTracingDetection(t *testing.T) {
	tests := []struct {
		name             string
		goMod            string
		expectedLibs     []string
		expectedProtocol string
	}{
		{
			name: "OpenTelemetry SDK",
			goMod: `module github.com/test/app

go 1.21

require (
	go.opentelemetry.io/otel v1.21.0
	go.opentelemetry.io/otel/sdk v1.21.0
)`,
			expectedLibs:     []string{"go.opentelemetry.io/otel"},
			expectedProtocol: "otlp",
		},
		{
			name: "OpenTelemetry with OTLP Exporter",
			goMod: `module github.com/test/app

go 1.21

require (
	go.opentelemetry.io/otel v1.21.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.21.0
)`,
			expectedLibs:     []string{"go.opentelemetry.io/otel"},
			expectedProtocol: "otlp",
		},
		{
			name: "Jaeger Client",
			goMod: `module github.com/test/app

go 1.21

require github.com/uber/jaeger-client-go v2.30.0`,
			expectedLibs:     []string{"github.com/uber/jaeger-client-go"},
			expectedProtocol: "jaeger",
		},
		{
			name: "Zipkin",
			goMod: `module github.com/test/app

go 1.21

require github.com/openzipkin/zipkin-go v0.4.2`,
			expectedLibs:     []string{"github.com/openzipkin/zipkin-go"},
			expectedProtocol: "zipkin",
		},
		{
			name: "No Tracing Libraries",
			goMod: `module github.com/test/app

go 1.21

require github.com/gin-gonic/gin v1.9.1`,
			expectedLibs:     nil,
			expectedProtocol: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir, err := os.MkdirTemp("", "go-tracing-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Write go.mod
			if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(tt.goMod), 0644); err != nil {
				t.Fatalf("Failed to write go.mod: %v", err)
			}

			// Run detection
			detector := NewGoDetector()
			detection, err := detector.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detection failed: %v", err)
			}

			// Verify tracing libraries
			if len(tt.expectedLibs) == 0 {
				if len(detection.TracingLibraries) != 0 {
					t.Errorf("Expected no tracing libraries, got %v", detection.TracingLibraries)
				}
			} else {
				for _, expectedLib := range tt.expectedLibs {
					found := false
					for _, lib := range detection.TracingLibraries {
						if lib == expectedLib {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected tracing library %q not found in %v", expectedLib, detection.TracingLibraries)
					}
				}
			}

			// Verify protocol
			if tt.expectedProtocol == "" {
				if detection.TracingProtocol != "" && detection.TracingProtocol != "unknown" {
					t.Errorf("Expected no protocol, got %q", detection.TracingProtocol)
				}
			} else {
				if detection.TracingProtocol != tt.expectedProtocol {
					t.Errorf("Expected protocol %q, got %q", tt.expectedProtocol, detection.TracingProtocol)
				}
			}
		})
	}
}

// TestPythonTracingDetection tests Python tracing library detection.
func TestPythonTracingDetection(t *testing.T) {
	tests := []struct {
		name             string
		requirements     string
		expectedLibs     []string
		expectedProtocol string
	}{
		{
			name: "OpenTelemetry SDK",
			requirements: `opentelemetry-sdk>=1.21.0
opentelemetry-api>=1.21.0`,
			expectedLibs:     []string{"opentelemetry-sdk", "opentelemetry-api"},
			expectedProtocol: "otlp",
		},
		{
			name: "OpenTelemetry with OTLP Exporter",
			requirements: `opentelemetry-sdk>=1.21.0
opentelemetry-exporter-otlp>=1.21.0`,
			expectedLibs:     []string{"opentelemetry-sdk", "opentelemetry-exporter-otlp"},
			expectedProtocol: "otlp",
		},
		{
			name: "OpenTelemetry FastAPI Instrumentation",
			requirements: `opentelemetry-instrumentation-fastapi>=0.42b0
fastapi>=0.104.0`,
			expectedLibs:     []string{"opentelemetry-instrumentation-fastapi"},
			expectedProtocol: "otlp",
		},
		{
			name: "Jaeger Client",
			requirements: `jaeger-client>=4.8.0`,
			expectedLibs:     []string{"jaeger-client"},
			expectedProtocol: "jaeger",
		},
		{
			name: "Zipkin",
			requirements: `py-zipkin>=1.0.0`,
			expectedLibs:     []string{"py-zipkin"},
			expectedProtocol: "zipkin",
		},
		{
			name: "No Tracing Libraries",
			requirements: `flask>=2.0.0
redis>=4.0.0`,
			expectedLibs:     nil,
			expectedProtocol: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir, err := os.MkdirTemp("", "python-tracing-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Write requirements.txt
			if err := os.WriteFile(filepath.Join(tmpDir, "requirements.txt"), []byte(tt.requirements), 0644); err != nil {
				t.Fatalf("Failed to write requirements.txt: %v", err)
			}

			// Run detection
			detector := NewPythonDetector()
			detection, err := detector.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detection failed: %v", err)
			}

			// Verify tracing libraries
			if len(tt.expectedLibs) == 0 {
				if len(detection.TracingLibraries) != 0 {
					t.Errorf("Expected no tracing libraries, got %v", detection.TracingLibraries)
				}
			} else {
				for _, expectedLib := range tt.expectedLibs {
					found := false
					for _, lib := range detection.TracingLibraries {
						if lib == expectedLib {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected tracing library %q not found in %v", expectedLib, detection.TracingLibraries)
					}
				}
			}

			// Verify protocol
			if tt.expectedProtocol == "" {
				if detection.TracingProtocol != "" && detection.TracingProtocol != "unknown" {
					t.Errorf("Expected no protocol, got %q", detection.TracingProtocol)
				}
			} else {
				if detection.TracingProtocol != tt.expectedProtocol {
					t.Errorf("Expected protocol %q, got %q", tt.expectedProtocol, detection.TracingProtocol)
				}
			}
		})
	}
}

// TestRustTracingDetection tests Rust tracing library detection.
func TestRustTracingDetection(t *testing.T) {
	tests := []struct {
		name             string
		cargoTOML        string
		expectedLibs     []string
		expectedProtocol string
	}{
		{
			name: "OpenTelemetry Crate",
			cargoTOML: `[package]
name = "test-app"
version = "0.1.0"
edition = "2021"

[dependencies]
opentelemetry = "0.21"
opentelemetry-otlp = "0.14"`,
			expectedLibs:     []string{"opentelemetry", "opentelemetry-otlp"},
			expectedProtocol: "otlp",
		},
		{
			name: "Tracing OpenTelemetry",
			cargoTOML: `[package]
name = "test-app"
version = "0.1.0"
edition = "2021"

[dependencies]
tracing = "0.1"
tracing-opentelemetry = "0.22"`,
			expectedLibs:     []string{"tracing-opentelemetry"},
			expectedProtocol: "otlp",
		},
		{
			name: "OpenTelemetry Jaeger Exporter",
			cargoTOML: `[package]
name = "test-app"
version = "0.1.0"
edition = "2021"

[dependencies]
opentelemetry = "0.21"
opentelemetry-jaeger = "0.20"`,
			expectedLibs:     []string{"opentelemetry", "opentelemetry-jaeger"},
			expectedProtocol: "otlp", // OTLP detected first via opentelemetry base crate
		},
		{
			name: "OpenTelemetry Zipkin Exporter",
			cargoTOML: `[package]
name = "test-app"
version = "0.1.0"
edition = "2021"

[dependencies]
opentelemetry-zipkin = "0.18"`,
			expectedLibs:     []string{"opentelemetry-zipkin"},
			expectedProtocol: "zipkin",
		},
		{
			name: "No Tracing Libraries",
			cargoTOML: `[package]
name = "test-app"
version = "0.1.0"
edition = "2021"

[dependencies]
actix-web = "4.0"`,
			expectedLibs:     nil,
			expectedProtocol: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir, err := os.MkdirTemp("", "rust-tracing-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Write Cargo.toml
			if err := os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte(tt.cargoTOML), 0644); err != nil {
				t.Fatalf("Failed to write Cargo.toml: %v", err)
			}

			// Run detection
			detector := NewRustDetector()
			detection, err := detector.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detection failed: %v", err)
			}

			// Verify tracing libraries
			if len(tt.expectedLibs) == 0 {
				if len(detection.TracingLibraries) != 0 {
					t.Errorf("Expected no tracing libraries, got %v", detection.TracingLibraries)
				}
			} else {
				for _, expectedLib := range tt.expectedLibs {
					found := false
					for _, lib := range detection.TracingLibraries {
						if lib == expectedLib {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected tracing library %q not found in %v", expectedLib, detection.TracingLibraries)
					}
				}
			}

			// Verify protocol
			if tt.expectedProtocol == "" {
				if detection.TracingProtocol != "" && detection.TracingProtocol != "unknown" {
					t.Errorf("Expected no protocol, got %q", detection.TracingProtocol)
				}
			} else {
				if detection.TracingProtocol != tt.expectedProtocol {
					t.Errorf("Expected protocol %q, got %q", tt.expectedProtocol, detection.TracingProtocol)
				}
			}
		})
	}
}

// TestNeedsTracing tests the NeedsTracing helper method.
func TestNeedsTracing(t *testing.T) {
	tests := []struct {
		name         string
		packageJSON  string
		needsTracing bool
	}{
		{
			name: "With OpenTelemetry",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {
					"@opentelemetry/sdk-node": "^0.45.0"
				}
			}`,
			needsTracing: true,
		},
		{
			name: "Without Tracing",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {
					"express": "^4.18.0"
				}
			}`,
			needsTracing: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir, err := os.MkdirTemp("", "needs-tracing-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Write package.json
			if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(tt.packageJSON), 0644); err != nil {
				t.Fatalf("Failed to write package.json: %v", err)
			}

			// Run detection
			detector := NewNodeDetector()
			detection, err := detector.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detection failed: %v", err)
			}

			// Verify NeedsTracing
			if detection.NeedsTracing() != tt.needsTracing {
				t.Errorf("Expected NeedsTracing() = %v, got %v", tt.needsTracing, detection.NeedsTracing())
			}
		})
	}
}

// TestGetTracingProtocol tests the GetTracingProtocol helper method.
func TestGetTracingProtocol(t *testing.T) {
	tests := []struct {
		name             string
		packageJSON      string
		expectedProtocol string
	}{
		{
			name: "With OTLP",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {
					"@opentelemetry/sdk-node": "^0.45.0"
				}
			}`,
			expectedProtocol: "otlp",
		},
		{
			name: "With Jaeger",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {
					"jaeger-client": "^3.19.0"
				}
			}`,
			expectedProtocol: "jaeger",
		},
		{
			name: "Without Tracing",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {
					"express": "^4.18.0"
				}
			}`,
			expectedProtocol: "otlp", // GetTracingProtocol defaults to otlp
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir, err := os.MkdirTemp("", "get-protocol-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Write package.json
			if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(tt.packageJSON), 0644); err != nil {
				t.Fatalf("Failed to write package.json: %v", err)
			}

			// Run detection
			detector := NewNodeDetector()
			detection, err := detector.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detection failed: %v", err)
			}

			// Verify GetTracingProtocol
			if detection.GetTracingProtocol() != tt.expectedProtocol {
				t.Errorf("Expected GetTracingProtocol() = %q, got %q", tt.expectedProtocol, detection.GetTracingProtocol())
			}
		})
	}
}
