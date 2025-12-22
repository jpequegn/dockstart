// Package generator provides code generation for devcontainer files.
package generator

import (
	"github.com/jpequegn/dockstart/internal/models"
)

// TracingSidecarConfig holds configuration for generating Jaeger tracing sidecar.
type TracingSidecarConfig struct {
	// ProjectName is the name of the project
	ProjectName string

	// Language is the detected programming language
	Language string

	// TracingProtocol is the detected tracing protocol (otlp, jaeger, zipkin)
	TracingProtocol string

	// TracingLibraries is the list of detected tracing libraries
	TracingLibraries []string

	// JaegerUIPort is the port for the Jaeger UI (default: 16686)
	JaegerUIPort int

	// OTLPGRPCPort is the port for OTLP gRPC collector (default: 4317)
	OTLPGRPCPort int

	// OTLPHTTPPort is the port for OTLP HTTP collector (default: 4318)
	OTLPHTTPPort int

	// JaegerAgentPort is the port for Jaeger agent (default: 6831)
	JaegerAgentPort int

	// MaxTraces is the maximum number of traces to keep in memory (default: 10000)
	MaxTraces int

	// SamplingRate is the trace sampling rate 0.0-1.0 (default: 1.0 for dev)
	SamplingRate float64
}

// DefaultTracingConfig returns a TracingSidecarConfig with sensible defaults.
func DefaultTracingConfig() *TracingSidecarConfig {
	return &TracingSidecarConfig{
		TracingProtocol: "otlp",
		JaegerUIPort:    16686,
		OTLPGRPCPort:    4317,
		OTLPHTTPPort:    4318,
		JaegerAgentPort: 6831,
		MaxTraces:       10000,
		SamplingRate:    1.0, // 100% sampling for development
	}
}

// TracingSidecarGenerator generates Jaeger configuration for docker-compose.
type TracingSidecarGenerator struct{}

// NewTracingSidecarGenerator creates a new tracing sidecar generator.
func NewTracingSidecarGenerator() *TracingSidecarGenerator {
	return &TracingSidecarGenerator{}
}

// BuildConfig creates a TracingSidecarConfig from detection results.
func (g *TracingSidecarGenerator) BuildConfig(detection *models.Detection, projectName string) *TracingSidecarConfig {
	config := DefaultTracingConfig()
	config.ProjectName = projectName
	config.Language = detection.Language
	config.TracingLibraries = detection.TracingLibraries

	// Use detected protocol or default to OTLP
	if detection.TracingProtocol != "" && detection.TracingProtocol != "unknown" {
		config.TracingProtocol = detection.TracingProtocol
	}

	return config
}

// ShouldGenerate returns true if tracing sidecar should be generated.
func (g *TracingSidecarGenerator) ShouldGenerate(detection *models.Detection) bool {
	return detection.NeedsTracing()
}

// GetOTLPEndpoint returns the OTLP endpoint URL for the given protocol preference.
func (c *TracingSidecarConfig) GetOTLPEndpoint() string {
	// OTLP HTTP endpoint (used by default for most SDKs)
	return "http://jaeger:4318"
}

// GetOTLPProtocol returns the OTLP protocol string.
func (c *TracingSidecarConfig) GetOTLPProtocol() string {
	return "http/protobuf"
}

// GetSamplerType returns the OTEL sampler type.
func (c *TracingSidecarConfig) GetSamplerType() string {
	if c.SamplingRate >= 1.0 {
		return "always_on"
	}
	return "parentbased_traceidratio"
}

// NeedsJaegerEnv returns true if legacy Jaeger environment variables are needed.
func (c *TracingSidecarConfig) NeedsJaegerEnv() bool {
	return c.TracingProtocol == "jaeger"
}
