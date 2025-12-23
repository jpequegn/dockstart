// Package generator provides code generation for devcontainer files.
package generator

import (
	"testing"

	"github.com/jpequegn/dockstart/internal/models"
	"gopkg.in/yaml.v3"
)

// TestTracingComposeValidYAML verifies that generated docker-compose.yml with tracing is valid YAML.
func TestTracingComposeValidYAML(t *testing.T) {
	gen := NewComposeGenerator()

	tests := []struct {
		name      string
		detection *models.Detection
	}{
		{
			name: "nodejs with otlp tracing",
			detection: &models.Detection{
				Language:         "nodejs",
				Version:          "20",
				TracingLibraries: []string{"@opentelemetry/sdk-node"},
				TracingProtocol:  "otlp",
			},
		},
		{
			name: "go with otlp tracing",
			detection: &models.Detection{
				Language:         "go",
				Version:          "1.21",
				TracingLibraries: []string{"go.opentelemetry.io/otel"},
				TracingProtocol:  "otlp",
			},
		},
		{
			name: "python with jaeger protocol",
			detection: &models.Detection{
				Language:         "python",
				Version:          "3.12",
				TracingLibraries: []string{"jaeger-client"},
				TracingProtocol:  "jaeger",
			},
		},
		{
			name: "rust with zipkin protocol",
			detection: &models.Detection{
				Language:         "rust",
				Version:          "1.75",
				TracingLibraries: []string{"opentelemetry-zipkin"},
				TracingProtocol:  "zipkin",
			},
		},
		{
			name: "tracing with services",
			detection: &models.Detection{
				Language:         "nodejs",
				Version:          "20",
				TracingLibraries: []string{"@opentelemetry/sdk-node"},
				TracingProtocol:  "otlp",
				Services:         []string{"postgres", "redis"},
			},
		},
		{
			name: "tracing with worker",
			detection: &models.Detection{
				Language:         "nodejs",
				Version:          "20",
				TracingLibraries: []string{"@opentelemetry/sdk-node"},
				TracingProtocol:  "otlp",
				QueueLibraries:   []string{"bull"},
				WorkerCommand:    "npm run worker",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := gen.GenerateContent(tt.detection, "testproject")
			if err != nil {
				t.Fatalf("GenerateContent() error = %v", err)
			}

			// Verify valid YAML
			var parsed map[string]interface{}
			if err := yaml.Unmarshal(content, &parsed); err != nil {
				t.Errorf("Generated docker-compose.yml is not valid YAML: %v\nContent:\n%s", err, string(content))
			}

			// Verify required top-level keys
			if _, ok := parsed["services"]; !ok {
				t.Error("docker-compose.yml missing 'services' section")
			}
		})
	}
}

// TestTracingComposeStructure verifies docker-compose.yml has correct structure for tracing.
func TestTracingComposeStructure(t *testing.T) {
	gen := NewComposeGenerator()

	detection := &models.Detection{
		Language:         "nodejs",
		Version:          "20",
		TracingLibraries: []string{"@opentelemetry/sdk-node"},
		TracingProtocol:  "otlp",
	}

	content, err := gen.GenerateContent(detection, "myapp")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	// Parse the generated YAML
	var parsed map[string]interface{}
	if err := yaml.Unmarshal(content, &parsed); err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	// Get services section
	services, ok := parsed["services"].(map[string]interface{})
	if !ok {
		t.Fatal("Could not get services section")
	}

	// Verify jaeger service exists
	jaeger, ok := services["jaeger"].(map[string]interface{})
	if !ok {
		t.Fatal("jaeger service not found in compose")
	}

	// Verify jaeger image
	image, ok := jaeger["image"].(string)
	if !ok || image != "jaegertracing/all-in-one:latest" {
		t.Errorf("jaeger image = %q, want jaegertracing/all-in-one:latest", image)
	}

	// Verify jaeger has ports
	ports, ok := jaeger["ports"].([]interface{})
	if !ok {
		t.Fatal("jaeger service missing ports")
	}

	// Check for required ports
	requiredPorts := map[string]bool{
		"4317:4317":   false, // OTLP gRPC
		"4318:4318":   false, // OTLP HTTP
		"16686:16686": false, // Web UI
	}
	for _, p := range ports {
		port := p.(string)
		if _, exists := requiredPorts[port]; exists {
			requiredPorts[port] = true
		}
	}
	for port, found := range requiredPorts {
		if !found {
			t.Errorf("jaeger missing required port: %s", port)
		}
	}

	// Verify jaeger has healthcheck
	healthcheck, ok := jaeger["healthcheck"].(map[string]interface{})
	if !ok {
		t.Fatal("jaeger service missing healthcheck")
	}

	// Verify healthcheck has required fields
	if test, ok := healthcheck["test"].([]interface{}); !ok || len(test) == 0 {
		t.Error("jaeger healthcheck missing test command")
	}
	if _, ok := healthcheck["interval"]; !ok {
		t.Error("jaeger healthcheck missing interval")
	}
	if _, ok := healthcheck["timeout"]; !ok {
		t.Error("jaeger healthcheck missing timeout")
	}
	if _, ok := healthcheck["retries"]; !ok {
		t.Error("jaeger healthcheck missing retries")
	}

	// Verify jaeger has environment
	env, ok := jaeger["environment"].([]interface{})
	if !ok {
		t.Fatal("jaeger service missing environment")
	}

	foundOTLPEnabled := false
	for _, e := range env {
		if envStr, ok := e.(string); ok && envStr == "COLLECTOR_OTLP_ENABLED=true" {
			foundOTLPEnabled = true
			break
		}
	}
	if !foundOTLPEnabled {
		t.Error("jaeger missing COLLECTOR_OTLP_ENABLED=true environment variable")
	}
}

// TestTracingAppEnvironmentVariables verifies app service has correct OTEL environment variables.
func TestTracingAppEnvironmentVariables(t *testing.T) {
	gen := NewComposeGenerator()

	detection := &models.Detection{
		Language:         "nodejs",
		Version:          "20",
		TracingLibraries: []string{"@opentelemetry/sdk-node"},
		TracingProtocol:  "otlp",
	}

	content, err := gen.GenerateContent(detection, "myapp")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	// Parse the generated YAML
	var parsed map[string]interface{}
	if err := yaml.Unmarshal(content, &parsed); err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	// Get services section
	services, ok := parsed["services"].(map[string]interface{})
	if !ok {
		t.Fatal("Could not get services section")
	}

	// Get app service
	app, ok := services["app"].(map[string]interface{})
	if !ok {
		t.Fatal("app service not found in compose")
	}

	// Get environment variables
	env, ok := app["environment"].([]interface{})
	if !ok {
		t.Fatal("app service missing environment")
	}

	// Required OTEL environment variables
	requiredEnvVars := map[string]bool{
		"OTEL_SERVICE_NAME=myapp":                        false,
		"OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318": false,
		"OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf":      false,
		"OTEL_TRACES_SAMPLER=always_on":                  false,
	}

	for _, e := range env {
		envStr, ok := e.(string)
		if !ok {
			continue
		}
		if _, exists := requiredEnvVars[envStr]; exists {
			requiredEnvVars[envStr] = true
		}
	}

	for envVar, found := range requiredEnvVars {
		if !found {
			t.Errorf("app service missing required OTEL environment variable: %s", envVar)
		}
	}
}

// TestTracingWorkerEnvironmentVariables verifies worker service has correct OTEL environment variables.
func TestTracingWorkerEnvironmentVariables(t *testing.T) {
	gen := NewComposeGenerator()

	detection := &models.Detection{
		Language:         "nodejs",
		Version:          "20",
		TracingLibraries: []string{"@opentelemetry/sdk-node"},
		TracingProtocol:  "otlp",
		QueueLibraries:   []string{"bull"},
		WorkerCommand:    "npm run worker",
	}

	content, err := gen.GenerateContent(detection, "myapp")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	// Parse the generated YAML
	var parsed map[string]interface{}
	if err := yaml.Unmarshal(content, &parsed); err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	// Get services section
	services, ok := parsed["services"].(map[string]interface{})
	if !ok {
		t.Fatal("Could not get services section")
	}

	// Get worker service
	worker, ok := services["worker"].(map[string]interface{})
	if !ok {
		t.Fatal("worker service not found in compose")
	}

	// Get environment variables
	env, ok := worker["environment"].([]interface{})
	if !ok {
		t.Fatal("worker service missing environment")
	}

	// Required OTEL environment variables for worker
	requiredEnvVars := map[string]bool{
		"OTEL_SERVICE_NAME=myapp-worker":                 false,
		"OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318": false,
		"OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf":      false,
		"OTEL_TRACES_SAMPLER=always_on":                  false,
	}

	for _, e := range env {
		envStr, ok := e.(string)
		if !ok {
			continue
		}
		if _, exists := requiredEnvVars[envStr]; exists {
			requiredEnvVars[envStr] = true
		}
	}

	for envVar, found := range requiredEnvVars {
		if !found {
			t.Errorf("worker service missing required OTEL environment variable: %s", envVar)
		}
	}

	// Verify worker service name differs from app
	hasAppName := false
	hasWorkerName := false
	for _, e := range env {
		envStr, ok := e.(string)
		if !ok {
			continue
		}
		if envStr == "OTEL_SERVICE_NAME=myapp" {
			hasAppName = true
		}
		if envStr == "OTEL_SERVICE_NAME=myapp-worker" {
			hasWorkerName = true
		}
	}

	if hasAppName {
		t.Error("worker should have different OTEL_SERVICE_NAME than app")
	}
	if !hasWorkerName {
		t.Error("worker should have OTEL_SERVICE_NAME=myapp-worker")
	}
}

// TestTracingJaegerRestartPolicy verifies jaeger has correct restart policy.
func TestTracingJaegerRestartPolicy(t *testing.T) {
	gen := NewComposeGenerator()

	detection := &models.Detection{
		Language:         "nodejs",
		Version:          "20",
		TracingLibraries: []string{"@opentelemetry/sdk-node"},
		TracingProtocol:  "otlp",
	}

	content, err := gen.GenerateContent(detection, "testapp")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	// Parse the generated YAML
	var parsed map[string]interface{}
	if err := yaml.Unmarshal(content, &parsed); err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	// Get services section
	services, ok := parsed["services"].(map[string]interface{})
	if !ok {
		t.Fatal("Could not get services section")
	}

	// Get jaeger service
	jaeger, ok := services["jaeger"].(map[string]interface{})
	if !ok {
		t.Fatal("jaeger service not found in compose")
	}

	// Verify restart policy
	restart, ok := jaeger["restart"].(string)
	if !ok || restart != "unless-stopped" {
		t.Errorf("jaeger restart = %q, want unless-stopped", restart)
	}
}

// TestTracingFullObservabilityStack verifies tracing works with metrics and logging sidecars.
func TestTracingFullObservabilityStack(t *testing.T) {
	gen := NewComposeGenerator()

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
		Services:         []string{"postgres", "redis"},
	}

	content, err := gen.GenerateContent(detection, "fullstack")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	// Parse the generated YAML
	var parsed map[string]interface{}
	if err := yaml.Unmarshal(content, &parsed); err != nil {
		t.Fatalf("Failed to parse YAML: %v\nContent:\n%s", err, string(content))
	}

	// Get services section
	services, ok := parsed["services"].(map[string]interface{})
	if !ok {
		t.Fatal("Could not get services section")
	}

	// Verify all observability services exist
	requiredServices := []string{"app", "jaeger", "prometheus", "grafana", "fluent-bit", "postgres", "redis"}
	for _, svc := range requiredServices {
		if _, ok := services[svc]; !ok {
			t.Errorf("missing required service: %s", svc)
		}
	}
}

// TestTracingProtocolVariations verifies compose generation for all protocol variations.
func TestTracingProtocolVariations(t *testing.T) {
	gen := NewComposeGenerator()

	tests := []struct {
		name     string
		protocol string
	}{
		{"otlp", "otlp"},
		{"jaeger", "jaeger"},
		{"zipkin", "zipkin"},
		{"unknown_defaults_to_otlp", "unknown"},
		{"empty_defaults_to_otlp", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detection := &models.Detection{
				Language:         "nodejs",
				Version:          "20",
				TracingLibraries: []string{"@opentelemetry/sdk-node"},
				TracingProtocol:  tt.protocol,
			}

			content, err := gen.GenerateContent(detection, "prototest")
			if err != nil {
				t.Fatalf("GenerateContent() error = %v", err)
			}

			// Verify valid YAML
			var parsed map[string]interface{}
			if err := yaml.Unmarshal(content, &parsed); err != nil {
				t.Errorf("Generated compose is not valid YAML: %v", err)
			}

			// Verify jaeger service exists for all protocols
			services, ok := parsed["services"].(map[string]interface{})
			if !ok {
				t.Fatal("Could not get services section")
			}
			if _, ok := services["jaeger"]; !ok {
				t.Error("jaeger service should exist regardless of protocol")
			}
		})
	}
}

// TestTracingSidecarConfigMethods verifies TracingSidecarConfig helper methods.
func TestTracingSidecarConfigMethods(t *testing.T) {
	t.Run("GetOTLPEndpoint returns HTTP endpoint", func(t *testing.T) {
		config := DefaultTracingConfig()
		endpoint := config.GetOTLPEndpoint()
		expected := "http://jaeger:4318"
		if endpoint != expected {
			t.Errorf("GetOTLPEndpoint() = %q, want %q", endpoint, expected)
		}
	})

	t.Run("GetOTLPProtocol returns http/protobuf", func(t *testing.T) {
		config := DefaultTracingConfig()
		protocol := config.GetOTLPProtocol()
		expected := "http/protobuf"
		if protocol != expected {
			t.Errorf("GetOTLPProtocol() = %q, want %q", protocol, expected)
		}
	})

	t.Run("GetSamplerType returns always_on for 100% sampling", func(t *testing.T) {
		config := &TracingSidecarConfig{SamplingRate: 1.0}
		sampler := config.GetSamplerType()
		if sampler != "always_on" {
			t.Errorf("GetSamplerType() = %q, want always_on", sampler)
		}
	})

	t.Run("GetSamplerType returns parentbased_traceidratio for partial sampling", func(t *testing.T) {
		config := &TracingSidecarConfig{SamplingRate: 0.5}
		sampler := config.GetSamplerType()
		if sampler != "parentbased_traceidratio" {
			t.Errorf("GetSamplerType() = %q, want parentbased_traceidratio", sampler)
		}
	})

	t.Run("NeedsJaegerEnv returns true for jaeger protocol", func(t *testing.T) {
		config := &TracingSidecarConfig{TracingProtocol: "jaeger"}
		if !config.NeedsJaegerEnv() {
			t.Error("NeedsJaegerEnv() should return true for jaeger protocol")
		}
	})

	t.Run("NeedsJaegerEnv returns false for otlp protocol", func(t *testing.T) {
		config := &TracingSidecarConfig{TracingProtocol: "otlp"}
		if config.NeedsJaegerEnv() {
			t.Error("NeedsJaegerEnv() should return false for otlp protocol")
		}
	})
}

// TestTracingComposeWithCustomPorts verifies that default ports are correctly applied.
func TestTracingComposeWithDefaultPorts(t *testing.T) {
	gen := NewComposeGenerator()

	detection := &models.Detection{
		Language:         "nodejs",
		Version:          "20",
		TracingLibraries: []string{"@opentelemetry/sdk-node"},
		TracingProtocol:  "otlp",
	}

	config := gen.buildConfig(detection, "porttest")

	// Verify default ports
	if config.TracingSidecar.JaegerUIPort != 16686 {
		t.Errorf("JaegerUIPort = %d, want 16686", config.TracingSidecar.JaegerUIPort)
	}
	if config.TracingSidecar.OTLPGRPCPort != 4317 {
		t.Errorf("OTLPGRPCPort = %d, want 4317", config.TracingSidecar.OTLPGRPCPort)
	}
	if config.TracingSidecar.OTLPHTTPPort != 4318 {
		t.Errorf("OTLPHTTPPort = %d, want 4318", config.TracingSidecar.OTLPHTTPPort)
	}
}
