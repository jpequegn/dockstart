// Package generator provides code generation for devcontainer files.
package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jpequegn/dockstart/internal/models"
)

func TestComposeConfig_MetricsSidecar(t *testing.T) {
	gen := NewComposeGenerator()

	tests := []struct {
		name           string
		detection      *models.Detection
		checkConfig    func(*testing.T, *ComposeConfig)
	}{
		{
			name: "metrics library enables metrics sidecar",
			detection: &models.Detection{
				Language:         "nodejs",
				MetricsLibraries: []string{"prom-client"},
				MetricsPort:      3000,
				MetricsPath:      "/metrics",
			},
			checkConfig: func(t *testing.T, config *ComposeConfig) {
				if !config.MetricsSidecar.Enabled {
					t.Error("MetricsSidecar.Enabled should be true when metrics libraries detected")
				}
				if config.MetricsSidecar.MetricsPort != 3000 {
					t.Errorf("MetricsSidecar.MetricsPort = %d, want 3000", config.MetricsSidecar.MetricsPort)
				}
				if config.MetricsSidecar.MetricsPath != "/metrics" {
					t.Errorf("MetricsSidecar.MetricsPath = %q, want /metrics", config.MetricsSidecar.MetricsPath)
				}
				if config.MetricsSidecar.PrometheusPort != 9090 {
					t.Errorf("MetricsSidecar.PrometheusPort = %d, want 9090", config.MetricsSidecar.PrometheusPort)
				}
				if config.MetricsSidecar.GrafanaPort != 3001 {
					t.Errorf("MetricsSidecar.GrafanaPort = %d, want 3001", config.MetricsSidecar.GrafanaPort)
				}
			},
		},
		{
			name: "no metrics library disables metrics sidecar",
			detection: &models.Detection{
				Language:         "nodejs",
				MetricsLibraries: nil,
			},
			checkConfig: func(t *testing.T, config *ComposeConfig) {
				if config.MetricsSidecar.Enabled {
					t.Error("MetricsSidecar.Enabled should be false when no metrics libraries detected")
				}
			},
		},
		{
			name: "metrics with postgres enables postgres exporter",
			detection: &models.Detection{
				Language:         "nodejs",
				MetricsLibraries: []string{"prom-client"},
				Services:         []string{"postgres"},
			},
			checkConfig: func(t *testing.T, config *ComposeConfig) {
				if !config.MetricsSidecar.Enabled {
					t.Error("MetricsSidecar.Enabled should be true")
				}
				if !config.MetricsSidecar.HasPostgres {
					t.Error("MetricsSidecar.HasPostgres should be true")
				}
			},
		},
		{
			name: "metrics with redis enables redis exporter",
			detection: &models.Detection{
				Language:         "nodejs",
				MetricsLibraries: []string{"prom-client"},
				Services:         []string{"redis"},
			},
			checkConfig: func(t *testing.T, config *ComposeConfig) {
				if !config.MetricsSidecar.Enabled {
					t.Error("MetricsSidecar.Enabled should be true")
				}
				if !config.MetricsSidecar.HasRedis {
					t.Error("MetricsSidecar.HasRedis should be true")
				}
			},
		},
		{
			name: "metrics with worker enables worker in config",
			detection: &models.Detection{
				Language:         "nodejs",
				MetricsLibraries: []string{"prom-client"},
				QueueLibraries:   []string{"bull"},
			},
			checkConfig: func(t *testing.T, config *ComposeConfig) {
				if !config.MetricsSidecar.Enabled {
					t.Error("MetricsSidecar.Enabled should be true")
				}
				if !config.MetricsSidecar.HasWorker {
					t.Error("MetricsSidecar.HasWorker should be true")
				}
			},
		},
		{
			name: "go project with prometheus client",
			detection: &models.Detection{
				Language:         "go",
				MetricsLibraries: []string{"prometheus/client_golang"},
				MetricsPort:      8080,
				MetricsPath:      "/metrics",
			},
			checkConfig: func(t *testing.T, config *ComposeConfig) {
				if !config.MetricsSidecar.Enabled {
					t.Error("MetricsSidecar.Enabled should be true")
				}
				if config.MetricsSidecar.MetricsPort != 8080 {
					t.Errorf("MetricsSidecar.MetricsPort = %d, want 8080", config.MetricsSidecar.MetricsPort)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := gen.buildConfig(tt.detection, "testproject")
			tt.checkConfig(t, config)
		})
	}
}

func TestComposeGenerator_MetricsServices(t *testing.T) {
	gen := NewComposeGenerator()

	tests := []struct {
		name          string
		detection     *models.Detection
		expectedParts []string
		unexpectedParts []string
	}{
		{
			name: "metrics enabled generates prometheus and grafana",
			detection: &models.Detection{
				Language:         "nodejs",
				MetricsLibraries: []string{"prom-client"},
				MetricsPort:      3000,
				MetricsPath:      "/metrics",
			},
			expectedParts: []string{
				"prometheus:",
				"image: prom/prometheus:latest",
				"grafana:",
				"image: grafana/grafana:latest",
				"prometheus-data:",
				"grafana-data:",
				"GF_SECURITY_ADMIN_PASSWORD=admin",
				"prometheus.scrape=true",
				"prometheus.port=3000",
				"prometheus.path=/metrics",
			},
			unexpectedParts: []string{
				"postgres-exporter:",
				"redis-exporter:",
			},
		},
		{
			name: "metrics with postgres includes postgres exporter",
			detection: &models.Detection{
				Language:         "nodejs",
				MetricsLibraries: []string{"prom-client"},
				Services:         []string{"postgres"},
			},
			expectedParts: []string{
				"prometheus:",
				"grafana:",
				"postgres-exporter:",
				"quay.io/prometheuscommunity/postgres-exporter:latest",
			},
		},
		{
			name: "metrics with redis includes redis exporter",
			detection: &models.Detection{
				Language:         "nodejs",
				MetricsLibraries: []string{"prom-client"},
				Services:         []string{"redis"},
			},
			expectedParts: []string{
				"prometheus:",
				"grafana:",
				"redis-exporter:",
				"oliver006/redis_exporter:latest",
			},
		},
		{
			name: "metrics with worker includes worker dependency",
			detection: &models.Detection{
				Language:         "nodejs",
				MetricsLibraries: []string{"prom-client"},
				QueueLibraries:   []string{"bull"},
			},
			expectedParts: []string{
				"prometheus:",
				"depends_on:",
				"- worker",
			},
		},
		{
			name: "no metrics libraries no prometheus",
			detection: &models.Detection{
				Language: "nodejs",
			},
			unexpectedParts: []string{
				"prometheus:",
				"grafana:",
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

func TestEndToEndMetricsGeneration(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "dockstart-metrics-e2e-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	detection := &models.Detection{
		Language:         "nodejs",
		Version:          "20",
		MetricsLibraries: []string{"prom-client"},
		MetricsPort:      3000,
		MetricsPath:      "/metrics",
		Services:         []string{"postgres", "redis"},
		QueueLibraries:   []string{"bull"},
	}

	projectName := "e2e-test"

	// Generate compose file
	composeGen := NewComposeGenerator()
	err = composeGen.Generate(detection, tmpDir, projectName)
	if err != nil {
		t.Fatalf("ComposeGenerator.Generate() error = %v", err)
	}

	// Generate metrics sidecar files
	metricsGen := NewMetricsSidecarGenerator()
	err = metricsGen.Generate(detection, tmpDir, projectName)
	if err != nil {
		t.Fatalf("MetricsSidecarGenerator.Generate() error = %v", err)
	}

	// Verify all expected files exist
	expectedFiles := []string{
		".devcontainer/docker-compose.yml",
		".devcontainer/prometheus/prometheus.yml",
		".devcontainer/grafana/provisioning/datasources/prometheus.yml",
		".devcontainer/grafana/provisioning/dashboards/provider.yml",
		".devcontainer/grafana/provisioning/dashboards/app-metrics.json",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file not created: %s", file)
		}
	}

	// Verify docker-compose.yml content
	composeContent, err := os.ReadFile(filepath.Join(tmpDir, ".devcontainer/docker-compose.yml"))
	if err != nil {
		t.Fatalf("Failed to read docker-compose.yml: %v", err)
	}

	composeStr := string(composeContent)
	requiredContent := []string{
		"prometheus:",
		"grafana:",
		"postgres-exporter:",
		"redis-exporter:",
		"worker:",
		"prometheus-data:",
		"grafana-data:",
	}

	for _, content := range requiredContent {
		if !strings.Contains(composeStr, content) {
			t.Errorf("docker-compose.yml missing required content: %q", content)
		}
	}

	// Verify prometheus.yml content
	prometheusContent, err := os.ReadFile(filepath.Join(tmpDir, ".devcontainer/prometheus/prometheus.yml"))
	if err != nil {
		t.Fatalf("Failed to read prometheus.yml: %v", err)
	}

	prometheusStr := string(prometheusContent)
	prometheusRequired := []string{
		"job_name: 'e2e-test'",
		"job_name: 'e2e-test-worker'",
		"job_name: 'postgres'",
		"job_name: 'redis'",
	}

	for _, content := range prometheusRequired {
		if !strings.Contains(prometheusStr, content) {
			t.Errorf("prometheus.yml missing required content: %q", content)
		}
	}
}

func TestDevcontainerGenerator_MetricsPorts(t *testing.T) {
	gen := NewDevcontainerGenerator()

	tests := []struct {
		name         string
		detection    *models.Detection
		expectPorts  []int
	}{
		{
			name: "metrics adds prometheus and grafana ports",
			detection: &models.Detection{
				Language:         "node",
				Version:          "20",
				MetricsLibraries: []string{"prom-client"},
			},
			expectPorts: []int{3000, 9090, 3001}, // app + prometheus + grafana
		},
		{
			name: "no metrics no extra ports",
			detection: &models.Detection{
				Language: "node",
				Version:  "20",
			},
			expectPorts: []int{3000}, // just app
		},
		{
			name: "metrics with postgres adds exporter port",
			detection: &models.Detection{
				Language:         "node",
				Version:          "20",
				MetricsLibraries: []string{"prom-client"},
				Services:         []string{"postgres"},
			},
			expectPorts: []int{3000, 5432, 9090, 3001}, // app + postgres + prometheus + grafana
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

func TestDevcontainerGenerator_UseComposeWithMetrics(t *testing.T) {
	gen := NewDevcontainerGenerator()

	tests := []struct {
		name       string
		detection  *models.Detection
		expectUseCompose bool
	}{
		{
			name: "metrics enables compose",
			detection: &models.Detection{
				Language:         "node",
				Version:          "20",
				MetricsLibraries: []string{"prom-client"},
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
			name: "services enable compose",
			detection: &models.Detection{
				Language: "node",
				Version:  "20",
				Services: []string{"postgres"},
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
