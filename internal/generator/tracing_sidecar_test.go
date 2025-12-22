// Package generator provides code generation for devcontainer files.
package generator

import (
	"testing"

	"github.com/jpequegn/dockstart/internal/models"
)

func TestDefaultTracingConfig(t *testing.T) {
	config := DefaultTracingConfig()

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"TracingProtocol", config.TracingProtocol, "otlp"},
		{"JaegerUIPort", config.JaegerUIPort, 16686},
		{"OTLPGRPCPort", config.OTLPGRPCPort, 4317},
		{"OTLPHTTPPort", config.OTLPHTTPPort, 4318},
		{"JaegerAgentPort", config.JaegerAgentPort, 6831},
		{"MaxTraces", config.MaxTraces, 10000},
		{"SamplingRate", config.SamplingRate, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("DefaultTracingConfig().%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestTracingSidecarGenerator_ShouldGenerate(t *testing.T) {
	gen := NewTracingSidecarGenerator()

	tests := []struct {
		name      string
		detection *models.Detection
		expected  bool
	}{
		{
			name: "with tracing library",
			detection: &models.Detection{
				Language:         "nodejs",
				TracingLibraries: []string{"@opentelemetry/sdk-node"},
			},
			expected: true,
		},
		{
			name: "without tracing library",
			detection: &models.Detection{
				Language:         "nodejs",
				TracingLibraries: nil,
			},
			expected: false,
		},
		{
			name: "empty tracing library slice",
			detection: &models.Detection{
				Language:         "go",
				TracingLibraries: []string{},
			},
			expected: false,
		},
		{
			name: "multiple tracing libraries",
			detection: &models.Detection{
				Language:         "go",
				TracingLibraries: []string{"go.opentelemetry.io/otel", "go.opentelemetry.io/otel/exporters/otlp/otlptrace"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.ShouldGenerate(tt.detection)
			if result != tt.expected {
				t.Errorf("ShouldGenerate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTracingSidecarGenerator_BuildConfig(t *testing.T) {
	gen := NewTracingSidecarGenerator()

	tests := []struct {
		name        string
		detection   *models.Detection
		projectName string
		checkConfig func(*testing.T, *TracingSidecarConfig)
	}{
		{
			name: "uses detection values",
			detection: &models.Detection{
				Language:         "nodejs",
				TracingLibraries: []string{"@opentelemetry/sdk-node"},
				TracingProtocol:  "otlp",
			},
			projectName: "myproject",
			checkConfig: func(t *testing.T, config *TracingSidecarConfig) {
				if config.ProjectName != "myproject" {
					t.Errorf("ProjectName = %q, want %q", config.ProjectName, "myproject")
				}
				if config.Language != "nodejs" {
					t.Errorf("Language = %q, want %q", config.Language, "nodejs")
				}
				if config.TracingProtocol != "otlp" {
					t.Errorf("TracingProtocol = %q, want %q", config.TracingProtocol, "otlp")
				}
			},
		},
		{
			name: "uses default protocol when unknown",
			detection: &models.Detection{
				Language:         "go",
				TracingLibraries: []string{"go.opentelemetry.io/otel"},
				TracingProtocol:  "unknown",
			},
			projectName: "goapp",
			checkConfig: func(t *testing.T, config *TracingSidecarConfig) {
				if config.TracingProtocol != "otlp" {
					t.Errorf("TracingProtocol = %q, want %q (should default to otlp)", config.TracingProtocol, "otlp")
				}
			},
		},
		{
			name: "uses default protocol when empty",
			detection: &models.Detection{
				Language:         "python",
				TracingLibraries: []string{"opentelemetry-sdk"},
				TracingProtocol:  "",
			},
			projectName: "pyapp",
			checkConfig: func(t *testing.T, config *TracingSidecarConfig) {
				if config.TracingProtocol != "otlp" {
					t.Errorf("TracingProtocol = %q, want %q (should default to otlp)", config.TracingProtocol, "otlp")
				}
			},
		},
		{
			name: "preserves jaeger protocol",
			detection: &models.Detection{
				Language:         "rust",
				TracingLibraries: []string{"opentelemetry-jaeger"},
				TracingProtocol:  "jaeger",
			},
			projectName: "rustapp",
			checkConfig: func(t *testing.T, config *TracingSidecarConfig) {
				if config.TracingProtocol != "jaeger" {
					t.Errorf("TracingProtocol = %q, want %q", config.TracingProtocol, "jaeger")
				}
			},
		},
		{
			name: "preserves zipkin protocol",
			detection: &models.Detection{
				Language:         "nodejs",
				TracingLibraries: []string{"zipkin"},
				TracingProtocol:  "zipkin",
			},
			projectName: "zipkinapp",
			checkConfig: func(t *testing.T, config *TracingSidecarConfig) {
				if config.TracingProtocol != "zipkin" {
					t.Errorf("TracingProtocol = %q, want %q", config.TracingProtocol, "zipkin")
				}
			},
		},
		{
			name: "stores tracing libraries",
			detection: &models.Detection{
				Language:         "nodejs",
				TracingLibraries: []string{"@opentelemetry/sdk-node", "@opentelemetry/auto-instrumentations-node"},
				TracingProtocol:  "otlp",
			},
			projectName: "multilib",
			checkConfig: func(t *testing.T, config *TracingSidecarConfig) {
				if len(config.TracingLibraries) != 2 {
					t.Errorf("TracingLibraries length = %d, want 2", len(config.TracingLibraries))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := gen.BuildConfig(tt.detection, tt.projectName)
			tt.checkConfig(t, config)

			// Always check defaults are set
			if config.JaegerUIPort != 16686 {
				t.Errorf("JaegerUIPort = %d, want %d", config.JaegerUIPort, 16686)
			}
			if config.OTLPGRPCPort != 4317 {
				t.Errorf("OTLPGRPCPort = %d, want %d", config.OTLPGRPCPort, 4317)
			}
			if config.OTLPHTTPPort != 4318 {
				t.Errorf("OTLPHTTPPort = %d, want %d", config.OTLPHTTPPort, 4318)
			}
			if config.SamplingRate != 1.0 {
				t.Errorf("SamplingRate = %f, want %f", config.SamplingRate, 1.0)
			}
		})
	}
}

func TestTracingSidecarConfig_GetOTLPEndpoint(t *testing.T) {
	config := DefaultTracingConfig()
	endpoint := config.GetOTLPEndpoint()

	if endpoint != "http://jaeger:4318" {
		t.Errorf("GetOTLPEndpoint() = %q, want %q", endpoint, "http://jaeger:4318")
	}
}

func TestTracingSidecarConfig_GetOTLPProtocol(t *testing.T) {
	config := DefaultTracingConfig()
	protocol := config.GetOTLPProtocol()

	if protocol != "http/protobuf" {
		t.Errorf("GetOTLPProtocol() = %q, want %q", protocol, "http/protobuf")
	}
}

func TestTracingSidecarConfig_GetSamplerType(t *testing.T) {
	tests := []struct {
		name         string
		samplingRate float64
		expected     string
	}{
		{
			name:         "100% sampling",
			samplingRate: 1.0,
			expected:     "always_on",
		},
		{
			name:         "greater than 100% sampling",
			samplingRate: 1.5,
			expected:     "always_on",
		},
		{
			name:         "50% sampling",
			samplingRate: 0.5,
			expected:     "parentbased_traceidratio",
		},
		{
			name:         "0% sampling",
			samplingRate: 0.0,
			expected:     "parentbased_traceidratio",
		},
		{
			name:         "99% sampling",
			samplingRate: 0.99,
			expected:     "parentbased_traceidratio",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &TracingSidecarConfig{SamplingRate: tt.samplingRate}
			result := config.GetSamplerType()
			if result != tt.expected {
				t.Errorf("GetSamplerType() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestTracingSidecarConfig_NeedsJaegerEnv(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		expected bool
	}{
		{
			name:     "OTLP protocol",
			protocol: "otlp",
			expected: false,
		},
		{
			name:     "Jaeger protocol",
			protocol: "jaeger",
			expected: true,
		},
		{
			name:     "Zipkin protocol",
			protocol: "zipkin",
			expected: false,
		},
		{
			name:     "Unknown protocol",
			protocol: "unknown",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &TracingSidecarConfig{TracingProtocol: tt.protocol}
			result := config.NeedsJaegerEnv()
			if result != tt.expected {
				t.Errorf("NeedsJaegerEnv() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNewTracingSidecarGenerator(t *testing.T) {
	gen := NewTracingSidecarGenerator()
	if gen == nil {
		t.Error("NewTracingSidecarGenerator() returned nil")
	}
}

func TestTracingSidecarAllLanguages(t *testing.T) {
	gen := NewTracingSidecarGenerator()

	tests := []struct {
		name        string
		language    string
		libraries   []string
		protocol    string
	}{
		{
			name:      "nodejs",
			language:  "nodejs",
			libraries: []string{"@opentelemetry/sdk-node"},
			protocol:  "otlp",
		},
		{
			name:      "go",
			language:  "go",
			libraries: []string{"go.opentelemetry.io/otel"},
			protocol:  "otlp",
		},
		{
			name:      "python",
			language:  "python",
			libraries: []string{"opentelemetry-sdk"},
			protocol:  "otlp",
		},
		{
			name:      "rust",
			language:  "rust",
			libraries: []string{"opentelemetry"},
			protocol:  "otlp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detection := &models.Detection{
				Language:         tt.language,
				TracingLibraries: tt.libraries,
				TracingProtocol:  tt.protocol,
			}

			if !gen.ShouldGenerate(detection) {
				t.Error("ShouldGenerate() should return true for language with tracing")
			}

			config := gen.BuildConfig(detection, "testproject")
			if config.Language != tt.language {
				t.Errorf("Language = %q, want %q", config.Language, tt.language)
			}
			if config.TracingProtocol != tt.protocol {
				t.Errorf("TracingProtocol = %q, want %q", config.TracingProtocol, tt.protocol)
			}
		})
	}
}
