// Package generator provides code generation for devcontainer files.
package generator

import (
	"strings"
	"testing"

	"github.com/jpequegn/dockstart/internal/models"
)

func TestComposeConfig_TracingSidecar(t *testing.T) {
	gen := NewComposeGenerator()

	tests := []struct {
		name        string
		detection   *models.Detection
		checkConfig func(*testing.T, *ComposeConfig)
	}{
		{
			name: "tracing library enables tracing sidecar",
			detection: &models.Detection{
				Language:         "nodejs",
				TracingLibraries: []string{"@opentelemetry/sdk-node"},
				TracingProtocol:  "otlp",
			},
			checkConfig: func(t *testing.T, config *ComposeConfig) {
				if !config.TracingSidecar.Enabled {
					t.Error("TracingSidecar.Enabled should be true when tracing libraries detected")
				}
				if config.TracingSidecar.TracingProtocol != "otlp" {
					t.Errorf("TracingSidecar.TracingProtocol = %q, want otlp", config.TracingSidecar.TracingProtocol)
				}
				if config.TracingSidecar.JaegerUIPort != 16686 {
					t.Errorf("TracingSidecar.JaegerUIPort = %d, want 16686", config.TracingSidecar.JaegerUIPort)
				}
				if config.TracingSidecar.OTLPGRPCPort != 4317 {
					t.Errorf("TracingSidecar.OTLPGRPCPort = %d, want 4317", config.TracingSidecar.OTLPGRPCPort)
				}
				if config.TracingSidecar.OTLPHTTPPort != 4318 {
					t.Errorf("TracingSidecar.OTLPHTTPPort = %d, want 4318", config.TracingSidecar.OTLPHTTPPort)
				}
			},
		},
		{
			name: "no tracing library disables tracing sidecar",
			detection: &models.Detection{
				Language:         "nodejs",
				TracingLibraries: nil,
			},
			checkConfig: func(t *testing.T, config *ComposeConfig) {
				if config.TracingSidecar.Enabled {
					t.Error("TracingSidecar.Enabled should be false when no tracing libraries detected")
				}
			},
		},
		{
			name: "uses default protocol when unknown",
			detection: &models.Detection{
				Language:         "nodejs",
				TracingLibraries: []string{"@opentelemetry/sdk-node"},
				TracingProtocol:  "unknown",
			},
			checkConfig: func(t *testing.T, config *ComposeConfig) {
				if !config.TracingSidecar.Enabled {
					t.Error("TracingSidecar.Enabled should be true")
				}
				if config.TracingSidecar.TracingProtocol != "otlp" {
					t.Errorf("TracingSidecar.TracingProtocol = %q, want otlp (should default)", config.TracingSidecar.TracingProtocol)
				}
			},
		},
		{
			name: "preserves jaeger protocol",
			detection: &models.Detection{
				Language:         "go",
				TracingLibraries: []string{"github.com/uber/jaeger-client-go"},
				TracingProtocol:  "jaeger",
			},
			checkConfig: func(t *testing.T, config *ComposeConfig) {
				if config.TracingSidecar.TracingProtocol != "jaeger" {
					t.Errorf("TracingSidecar.TracingProtocol = %q, want jaeger", config.TracingSidecar.TracingProtocol)
				}
			},
		},
		{
			name: "sets service name from project name",
			detection: &models.Detection{
				Language:         "python",
				TracingLibraries: []string{"opentelemetry-sdk"},
				TracingProtocol:  "otlp",
			},
			checkConfig: func(t *testing.T, config *ComposeConfig) {
				if config.TracingSidecar.ServiceName != "myproject" {
					t.Errorf("TracingSidecar.ServiceName = %q, want myproject", config.TracingSidecar.ServiceName)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := gen.buildConfig(tt.detection, "myproject")
			tt.checkConfig(t, config)
		})
	}
}

func TestComposeGenerator_TracingServices(t *testing.T) {
	gen := NewComposeGenerator()

	tests := []struct {
		name            string
		detection       *models.Detection
		expectedParts   []string
		unexpectedParts []string
	}{
		{
			name: "tracing enabled generates jaeger",
			detection: &models.Detection{
				Language:         "nodejs",
				TracingLibraries: []string{"@opentelemetry/sdk-node"},
				TracingProtocol:  "otlp",
			},
			expectedParts: []string{
				"jaeger:",
				"image: jaegertracing/all-in-one:latest",
				"COLLECTOR_OTLP_ENABLED=true",
				"16686:16686",
				"4317:4317",
				"4318:4318",
				"OTEL_SERVICE_NAME=testproject",
				"OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318",
				"OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf",
				"OTEL_TRACES_SAMPLER=always_on",
			},
		},
		{
			name: "no tracing libraries no jaeger",
			detection: &models.Detection{
				Language: "nodejs",
			},
			unexpectedParts: []string{
				"jaeger:",
				"OTEL_SERVICE_NAME",
				"OTEL_EXPORTER_OTLP_ENDPOINT",
			},
		},
		{
			name: "tracing with worker adds OTEL env to worker",
			detection: &models.Detection{
				Language:         "nodejs",
				TracingLibraries: []string{"@opentelemetry/sdk-node"},
				TracingProtocol:  "otlp",
				QueueLibraries:   []string{"bull"},
				WorkerCommand:    "npm run worker",
			},
			expectedParts: []string{
				"jaeger:",
				"OTEL_SERVICE_NAME=testproject",
				"OTEL_SERVICE_NAME=testproject-worker",
			},
		},
		{
			name: "jaeger healthcheck configured",
			detection: &models.Detection{
				Language:         "nodejs",
				TracingLibraries: []string{"@opentelemetry/sdk-node"},
			},
			expectedParts: []string{
				"healthcheck:",
				"wget",
				"http://localhost:16686",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := gen.GenerateContent(tt.detection, "testproject")
			if err != nil {
				t.Fatalf("GenerateContent() error = %v", err)
			}

			contentStr := string(content)

			for _, part := range tt.expectedParts {
				if !strings.Contains(contentStr, part) {
					t.Errorf("GenerateContent() missing expected content: %q", part)
				}
			}

			for _, part := range tt.unexpectedParts {
				if strings.Contains(contentStr, part) {
					t.Errorf("GenerateContent() contains unexpected content: %q", part)
				}
			}
		})
	}
}

func TestDevcontainerGenerator_TracingPorts(t *testing.T) {
	gen := NewDevcontainerGenerator()

	tests := []struct {
		name        string
		detection   *models.Detection
		expectPorts []int
	}{
		{
			name: "tracing adds jaeger UI port",
			detection: &models.Detection{
				Language:         "node",
				Version:          "20",
				TracingLibraries: []string{"@opentelemetry/sdk-node"},
			},
			expectPorts: []int{3000, 16686}, // app + jaeger UI
		},
		{
			name: "no tracing no jaeger port",
			detection: &models.Detection{
				Language: "node",
				Version:  "20",
			},
			expectPorts: []int{3000}, // just app
		},
		{
			name: "tracing with metrics adds all ports",
			detection: &models.Detection{
				Language:         "node",
				Version:          "20",
				TracingLibraries: []string{"@opentelemetry/sdk-node"},
				MetricsLibraries: []string{"prom-client"},
			},
			expectPorts: []int{3000, 9090, 3001, 16686}, // app + prometheus + grafana + jaeger UI
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := gen.buildConfig(tt.detection, "testproject")

			for _, port := range tt.expectPorts {
				found := false
				for _, p := range config.ForwardPorts {
					if p == port {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected port %d not found in ForwardPorts: %v", port, config.ForwardPorts)
				}
			}
		})
	}
}

func TestDevcontainerGenerator_UseComposeWithTracing(t *testing.T) {
	gen := NewDevcontainerGenerator()

	tests := []struct {
		name             string
		detection        *models.Detection
		expectUseCompose bool
	}{
		{
			name: "tracing enables compose",
			detection: &models.Detection{
				Language:         "node",
				Version:          "20",
				TracingLibraries: []string{"@opentelemetry/sdk-node"},
			},
			expectUseCompose: true,
		},
		{
			name: "no sidecars no compose",
			detection: &models.Detection{
				Language: "node",
				Version:  "20",
			},
			expectUseCompose: false,
		},
		{
			name: "tracing and metrics enable compose",
			detection: &models.Detection{
				Language:         "node",
				Version:          "20",
				TracingLibraries: []string{"@opentelemetry/sdk-node"},
				MetricsLibraries: []string{"prom-client"},
			},
			expectUseCompose: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := gen.buildConfig(tt.detection, "testproject")

			if config.UseCompose != tt.expectUseCompose {
				t.Errorf("UseCompose = %v, want %v", config.UseCompose, tt.expectUseCompose)
			}
		})
	}
}

func TestTracingWithAllLanguages(t *testing.T) {
	gen := NewComposeGenerator()

	tests := []struct {
		name      string
		detection *models.Detection
	}{
		{
			name: "nodejs with opentelemetry",
			detection: &models.Detection{
				Language:         "nodejs",
				TracingLibraries: []string{"@opentelemetry/sdk-node"},
				TracingProtocol:  "otlp",
			},
		},
		{
			name: "go with opentelemetry",
			detection: &models.Detection{
				Language:         "go",
				TracingLibraries: []string{"go.opentelemetry.io/otel"},
				TracingProtocol:  "otlp",
			},
		},
		{
			name: "python with opentelemetry",
			detection: &models.Detection{
				Language:         "python",
				TracingLibraries: []string{"opentelemetry-sdk"},
				TracingProtocol:  "otlp",
			},
		},
		{
			name: "rust with opentelemetry",
			detection: &models.Detection{
				Language:         "rust",
				TracingLibraries: []string{"opentelemetry"},
				TracingProtocol:  "otlp",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := gen.GenerateContent(tt.detection, "testproject")
			if err != nil {
				t.Fatalf("GenerateContent() error = %v", err)
			}

			contentStr := string(content)
			requiredParts := []string{
				"jaeger:",
				"jaegertracing/all-in-one:latest",
				"OTEL_SERVICE_NAME=testproject",
				"OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318",
			}

			for _, part := range requiredParts {
				if !strings.Contains(contentStr, part) {
					t.Errorf("GenerateContent() for %s missing expected content: %q", tt.name, part)
				}
			}
		})
	}
}

func TestTracingCombinedWithOtherSidecars(t *testing.T) {
	gen := NewComposeGenerator()

	// Full stack detection with tracing, metrics, logging, worker, and services
	detection := &models.Detection{
		Language:         "nodejs",
		Version:          "20",
		TracingLibraries: []string{"@opentelemetry/sdk-node"},
		TracingProtocol:  "otlp",
		MetricsLibraries: []string{"prom-client"},
		MetricsPort:      3000,
		MetricsPath:      "/metrics",
		LoggingLibraries: []string{"pino"},
		LogFormat:        "json",
		QueueLibraries:   []string{"bull"},
		WorkerCommand:    "npm run worker",
		Services:         []string{"postgres", "redis"},
	}

	content, err := gen.GenerateContent(detection, "fullstack")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	contentStr := string(content)
	requiredParts := []string{
		// Tracing
		"jaeger:",
		"OTEL_SERVICE_NAME=fullstack",
		"OTEL_SERVICE_NAME=fullstack-worker",
		// Metrics
		"prometheus:",
		"grafana:",
		"prometheus.scrape=true",
		// Logging
		"fluent-bit:",
		// Worker
		"worker:",
		"npm run worker",
		// Services
		"postgres:",
		"redis:",
		// Volumes
		"prometheus-data:",
		"grafana-data:",
	}

	for _, part := range requiredParts {
		if !strings.Contains(contentStr, part) {
			t.Errorf("Full stack GenerateContent() missing expected content: %q", part)
		}
	}
}
