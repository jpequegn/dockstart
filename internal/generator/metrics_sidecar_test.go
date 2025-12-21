// Package generator provides code generation for devcontainer files.
package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jpequegn/dockstart/internal/models"
)

func TestDefaultMetricsConfig(t *testing.T) {
	config := DefaultMetricsConfig()

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"ScrapeInterval", config.ScrapeInterval, "30s"},
		{"AppScrapeInterval", config.AppScrapeInterval, "15s"},
		{"MetricsPath", config.MetricsPath, "/metrics"},
		{"MetricsPort", config.MetricsPort, 3000},
		{"GrafanaPort", config.GrafanaPort, 3001},
		{"PrometheusPort", config.PrometheusPort, 9090},
		{"RetentionDays", config.RetentionDays, 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("DefaultMetricsConfig().%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestGeneratePrometheusConfig(t *testing.T) {
	gen := NewMetricsSidecarGenerator()

	tests := []struct {
		name           string
		config         *MetricsSidecarConfig
		expectedParts  []string
		unexpectedParts []string
	}{
		{
			name: "basic config",
			config: &MetricsSidecarConfig{
				ProjectName:       "testapp",
				ScrapeInterval:    "30s",
				AppScrapeInterval: "15s",
				MetricsPort:       3000,
				MetricsPath:       "/metrics",
			},
			expectedParts: []string{
				"scrape_interval: 30s",
				"job_name: 'testapp'",
				"targets: ['app:3000']",
				"metrics_path: /metrics",
			},
			unexpectedParts: []string{
				"job_name: 'testapp-worker'",
				"postgres-exporter",
				"redis-exporter",
			},
		},
		{
			name: "with worker",
			config: &MetricsSidecarConfig{
				ProjectName:       "myapp",
				ScrapeInterval:    "30s",
				AppScrapeInterval: "15s",
				MetricsPort:       8080,
				MetricsPath:       "/metrics",
				HasWorker:         true,
				WorkerMetricsPort: 8080,
			},
			expectedParts: []string{
				"job_name: 'myapp'",
				"job_name: 'myapp-worker'",
				"targets: ['worker:8080']",
			},
		},
		{
			name: "with postgres",
			config: &MetricsSidecarConfig{
				ProjectName:       "dbapp",
				ScrapeInterval:    "30s",
				AppScrapeInterval: "15s",
				MetricsPort:       3000,
				MetricsPath:       "/metrics",
				HasPostgres:       true,
			},
			expectedParts: []string{
				"job_name: 'postgres'",
				"targets: ['postgres-exporter:9187']",
			},
		},
		{
			name: "with redis",
			config: &MetricsSidecarConfig{
				ProjectName:       "cacheapp",
				ScrapeInterval:    "30s",
				AppScrapeInterval: "15s",
				MetricsPort:       3000,
				MetricsPath:       "/metrics",
				HasRedis:          true,
			},
			expectedParts: []string{
				"job_name: 'redis'",
				"targets: ['redis-exporter:9121']",
			},
		},
		{
			name: "full config",
			config: &MetricsSidecarConfig{
				ProjectName:       "fullapp",
				ScrapeInterval:    "60s",
				AppScrapeInterval: "30s",
				MetricsPort:       9000,
				MetricsPath:       "/api/metrics",
				HasWorker:         true,
				WorkerMetricsPort: 9000,
				HasPostgres:       true,
				HasRedis:          true,
			},
			expectedParts: []string{
				"scrape_interval: 60s",
				"job_name: 'fullapp'",
				"targets: ['app:9000']",
				"metrics_path: /api/metrics",
				"job_name: 'fullapp-worker'",
				"job_name: 'postgres'",
				"job_name: 'redis'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := gen.GeneratePrometheusConfig(tt.config)
			if err != nil {
				t.Fatalf("GeneratePrometheusConfig() error = %v", err)
			}

			content := string(result)
			for _, part := range tt.expectedParts {
				if !strings.Contains(content, part) {
					t.Errorf("GeneratePrometheusConfig() missing expected content: %q", part)
				}
			}

			for _, part := range tt.unexpectedParts {
				if strings.Contains(content, part) {
					t.Errorf("GeneratePrometheusConfig() contains unexpected content: %q", part)
				}
			}
		})
	}
}

func TestGenerateGrafanaDatasource(t *testing.T) {
	gen := NewMetricsSidecarGenerator()

	config := &MetricsSidecarConfig{
		ProjectName:    "testapp",
		PrometheusPort: 9090,
	}

	result, err := gen.GenerateGrafanaDatasource(config)
	if err != nil {
		t.Fatalf("GenerateGrafanaDatasource() error = %v", err)
	}

	content := string(result)
	expectedParts := []string{
		"apiVersion: 1",
		"name: Prometheus",
		"type: prometheus",
		"url: http://prometheus:9090",
		"isDefault: true",
	}

	for _, part := range expectedParts {
		if !strings.Contains(content, part) {
			t.Errorf("GenerateGrafanaDatasource() missing expected content: %q", part)
		}
	}
}

func TestGenerateGrafanaDashboardProvider(t *testing.T) {
	gen := NewMetricsSidecarGenerator()

	config := &MetricsSidecarConfig{
		ProjectName: "testapp",
	}

	result, err := gen.GenerateGrafanaDashboardProvider(config)
	if err != nil {
		t.Fatalf("GenerateGrafanaDashboardProvider() error = %v", err)
	}

	content := string(result)
	expectedParts := []string{
		"apiVersion: 1",
		"name: 'dockstart'",
		"folder: 'testapp'",
		"type: file",
		"path: /etc/grafana/provisioning/dashboards",
	}

	for _, part := range expectedParts {
		if !strings.Contains(content, part) {
			t.Errorf("GenerateGrafanaDashboardProvider() missing expected content: %q", part)
		}
	}
}

func TestGenerateAppDashboard(t *testing.T) {
	gen := NewMetricsSidecarGenerator()

	config := &MetricsSidecarConfig{
		ProjectName: "myapp",
		Language:    "go",
	}

	result, err := gen.GenerateAppDashboard(config)
	if err != nil {
		t.Fatalf("GenerateAppDashboard() error = %v", err)
	}

	content := string(result)
	expectedParts := []string{
		`"title": "myapp - Application Metrics"`,
		`"uid": "myapp-app-metrics"`,
		`"tags": ["myapp", "dockstart", "go"]`,
		`"title": "Request Rate"`,
		`"title": "Response Time Percentiles"`,
		`"title": "Error Rate (5xx)"`,
		`"title": "Requests by Status Code"`,
		`job=\"myapp\"`,
		`"type": "prometheus"`,
	}

	for _, part := range expectedParts {
		if !strings.Contains(content, part) {
			t.Errorf("GenerateAppDashboard() missing expected content: %q", part)
		}
	}
}

func TestMetricsSidecarGenerator_Generate(t *testing.T) {
	gen := NewMetricsSidecarGenerator()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "dockstart-metrics-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	detection := &models.Detection{
		Language:         "nodejs",
		MetricsPort:      3000,
		MetricsPath:      "/metrics",
		MetricsLibraries: []string{"prom-client"},
		Services:         []string{"postgres", "redis"},
		QueueLibraries:   []string{"bull"}, // This makes NeedsWorker() return true
	}

	err = gen.Generate(detection, tmpDir, "testproject")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Check all expected files exist
	expectedFiles := []string{
		".devcontainer/prometheus/prometheus.yml",
		".devcontainer/grafana/provisioning/datasources/prometheus.yml",
		".devcontainer/grafana/provisioning/dashboards/provider.yml",
		".devcontainer/grafana/provisioning/dashboards/app-metrics.json",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Generate() did not create expected file: %s", file)
		}
	}

	// Verify prometheus.yml content
	prometheusContent, err := os.ReadFile(filepath.Join(tmpDir, ".devcontainer/prometheus/prometheus.yml"))
	if err != nil {
		t.Fatalf("Failed to read prometheus.yml: %v", err)
	}
	prometheusStr := string(prometheusContent)
	if !strings.Contains(prometheusStr, "job_name: 'testproject'") {
		t.Error("prometheus.yml missing project job")
	}
	if !strings.Contains(prometheusStr, "job_name: 'testproject-worker'") {
		t.Error("prometheus.yml missing worker job")
	}
	if !strings.Contains(prometheusStr, "postgres-exporter") {
		t.Error("prometheus.yml missing postgres-exporter")
	}
	if !strings.Contains(prometheusStr, "redis-exporter") {
		t.Error("prometheus.yml missing redis-exporter")
	}

	// Verify dashboard content
	dashboardContent, err := os.ReadFile(filepath.Join(tmpDir, ".devcontainer/grafana/provisioning/dashboards/app-metrics.json"))
	if err != nil {
		t.Fatalf("Failed to read app-metrics.json: %v", err)
	}
	dashboardStr := string(dashboardContent)
	if !strings.Contains(dashboardStr, `"title": "testproject - Application Metrics"`) {
		t.Error("app-metrics.json missing correct title")
	}
}

func TestMetricsSidecarGenerator_ShouldGenerate(t *testing.T) {
	gen := NewMetricsSidecarGenerator()

	tests := []struct {
		name      string
		detection *models.Detection
		expected  bool
	}{
		{
			name: "with metrics library",
			detection: &models.Detection{
				Language:         "nodejs",
				MetricsLibraries: []string{"prom-client"},
			},
			expected: true,
		},
		{
			name: "without metrics library",
			detection: &models.Detection{
				Language:         "nodejs",
				MetricsLibraries: nil,
			},
			expected: false,
		},
		{
			name: "empty metrics library slice",
			detection: &models.Detection{
				Language:         "go",
				MetricsLibraries: []string{},
			},
			expected: false,
		},
		{
			name: "multiple metrics libraries",
			detection: &models.Detection{
				Language:         "go",
				MetricsLibraries: []string{"prometheus/client_golang", "prometheus/promhttp"},
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

func TestMetricsSidecarGenerator_buildConfig(t *testing.T) {
	gen := NewMetricsSidecarGenerator()

	tests := []struct {
		name        string
		detection   *models.Detection
		projectName string
		checkConfig func(*testing.T, *MetricsSidecarConfig)
	}{
		{
			name: "uses detection values",
			detection: &models.Detection{
				Language:    "nodejs",
				MetricsPort: 8080,
				MetricsPath: "/api/metrics",
			},
			projectName: "myproject",
			checkConfig: func(t *testing.T, config *MetricsSidecarConfig) {
				if config.ProjectName != "myproject" {
					t.Errorf("ProjectName = %q, want %q", config.ProjectName, "myproject")
				}
				if config.Language != "nodejs" {
					t.Errorf("Language = %q, want %q", config.Language, "nodejs")
				}
				if config.MetricsPort != 8080 {
					t.Errorf("MetricsPort = %d, want %d", config.MetricsPort, 8080)
				}
				if config.MetricsPath != "/api/metrics" {
					t.Errorf("MetricsPath = %q, want %q", config.MetricsPath, "/api/metrics")
				}
			},
		},
		{
			name: "uses defaults when detection values are zero",
			detection: &models.Detection{
				Language:         "go",
				MetricsPort:      0,
				MetricsPath:      "",
				MetricsLibraries: []string{"prometheus/client_golang"},
			},
			projectName: "goapp",
			checkConfig: func(t *testing.T, config *MetricsSidecarConfig) {
				// Should use defaults from GetMetricsPort/GetMetricsPath
				if config.MetricsPort == 0 {
					t.Error("MetricsPort should not be 0")
				}
				if config.MetricsPath == "" {
					t.Error("MetricsPath should not be empty")
				}
			},
		},
		{
			name: "detects services",
			detection: &models.Detection{
				Language: "nodejs",
				Services: []string{"postgres", "redis", "elasticsearch"},
			},
			projectName: "serviceapp",
			checkConfig: func(t *testing.T, config *MetricsSidecarConfig) {
				if !config.HasPostgres {
					t.Error("HasPostgres should be true")
				}
				if !config.HasRedis {
					t.Error("HasRedis should be true")
				}
			},
		},
		{
			name: "detects worker",
			detection: &models.Detection{
				Language:       "nodejs",
				QueueLibraries: []string{"bull"}, // NeedsWorker() returns true when queue libs exist
			},
			projectName: "workerapp",
			checkConfig: func(t *testing.T, config *MetricsSidecarConfig) {
				if !config.HasWorker {
					t.Error("HasWorker should be true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := gen.buildConfig(tt.detection, tt.projectName)
			tt.checkConfig(t, config)

			// Always check defaults are set
			if config.ScrapeInterval != "30s" {
				t.Errorf("ScrapeInterval = %q, want %q", config.ScrapeInterval, "30s")
			}
			if config.GrafanaPort != 3001 {
				t.Errorf("GrafanaPort = %d, want %d", config.GrafanaPort, 3001)
			}
			if config.PrometheusPort != 9090 {
				t.Errorf("PrometheusPort = %d, want %d", config.PrometheusPort, 9090)
			}
		})
	}
}
