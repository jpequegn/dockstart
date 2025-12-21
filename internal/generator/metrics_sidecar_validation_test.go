// Package generator provides code generation for devcontainer files.
package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jpequegn/dockstart/internal/models"
	"gopkg.in/yaml.v3"
)

// TestPrometheusConfigValidYAML verifies that generated prometheus.yml is valid YAML
func TestPrometheusConfigValidYAML(t *testing.T) {
	gen := NewMetricsSidecarGenerator()

	tests := []struct {
		name   string
		config *MetricsSidecarConfig
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
		},
		{
			name: "full stack config",
			config: &MetricsSidecarConfig{
				ProjectName:       "fullapp",
				ScrapeInterval:    "60s",
				AppScrapeInterval: "30s",
				MetricsPort:       8080,
				MetricsPath:       "/api/metrics",
				HasWorker:         true,
				WorkerMetricsPort: 8080,
				HasPostgres:       true,
				HasRedis:          true,
			},
		},
		{
			name: "custom metrics path with special chars",
			config: &MetricsSidecarConfig{
				ProjectName:       "specialapp",
				ScrapeInterval:    "15s",
				AppScrapeInterval: "5s",
				MetricsPort:       9000,
				MetricsPath:       "/v1/api/metrics",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := gen.GeneratePrometheusConfig(tt.config)
			if err != nil {
				t.Fatalf("GeneratePrometheusConfig() error = %v", err)
			}

			// Verify valid YAML
			var parsed map[string]interface{}
			if err := yaml.Unmarshal(result, &parsed); err != nil {
				t.Errorf("Generated prometheus.yml is not valid YAML: %v", err)
			}

			// Verify required top-level keys
			if _, ok := parsed["global"]; !ok {
				t.Error("prometheus.yml missing 'global' section")
			}
			if _, ok := parsed["scrape_configs"]; !ok {
				t.Error("prometheus.yml missing 'scrape_configs' section")
			}
		})
	}
}

// TestPrometheusConfigStructure verifies prometheus.yml has correct structure
func TestPrometheusConfigStructure(t *testing.T) {
	gen := NewMetricsSidecarGenerator()

	config := &MetricsSidecarConfig{
		ProjectName:       "structuretest",
		ScrapeInterval:    "30s",
		AppScrapeInterval: "15s",
		MetricsPort:       3000,
		MetricsPath:       "/metrics",
		HasWorker:         true,
		WorkerMetricsPort: 3000,
		HasPostgres:       true,
		HasRedis:          true,
	}

	result, err := gen.GeneratePrometheusConfig(config)
	if err != nil {
		t.Fatalf("GeneratePrometheusConfig() error = %v", err)
	}

	var parsed struct {
		Global struct {
			ScrapeInterval     string `yaml:"scrape_interval"`
			EvaluationInterval string `yaml:"evaluation_interval"`
		} `yaml:"global"`
		ScrapeConfigs []struct {
			JobName       string `yaml:"job_name"`
			ScrapeInterval string `yaml:"scrape_interval,omitempty"`
			MetricsPath   string `yaml:"metrics_path,omitempty"`
			StaticConfigs []struct {
				Targets []string `yaml:"targets"`
			} `yaml:"static_configs"`
		} `yaml:"scrape_configs"`
	}

	if err := yaml.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Failed to parse prometheus.yml: %v", err)
	}

	// Verify global settings
	if parsed.Global.ScrapeInterval != "30s" {
		t.Errorf("Global scrape_interval = %q, want %q", parsed.Global.ScrapeInterval, "30s")
	}

	// Verify we have expected number of scrape configs
	expectedJobs := 5 // prometheus, app, worker, postgres, redis
	if len(parsed.ScrapeConfigs) != expectedJobs {
		t.Errorf("Expected %d scrape configs, got %d", expectedJobs, len(parsed.ScrapeConfigs))
	}

	// Verify job names
	jobNames := make(map[string]bool)
	for _, sc := range parsed.ScrapeConfigs {
		jobNames[sc.JobName] = true
	}

	expectedJobNames := []string{"prometheus", "structuretest", "structuretest-worker", "postgres", "redis"}
	for _, name := range expectedJobNames {
		if !jobNames[name] {
			t.Errorf("Missing job: %s", name)
		}
	}
}

// TestGrafanaDatasourceValidYAML verifies that grafana datasource is valid YAML
func TestGrafanaDatasourceValidYAML(t *testing.T) {
	gen := NewMetricsSidecarGenerator()

	config := &MetricsSidecarConfig{
		ProjectName:    "testapp",
		PrometheusPort: 9090,
	}

	result, err := gen.GenerateGrafanaDatasource(config)
	if err != nil {
		t.Fatalf("GenerateGrafanaDatasource() error = %v", err)
	}

	var parsed struct {
		APIVersion  int `yaml:"apiVersion"`
		Datasources []struct {
			Name      string `yaml:"name"`
			Type      string `yaml:"type"`
			URL       string `yaml:"url"`
			IsDefault bool   `yaml:"isDefault"`
		} `yaml:"datasources"`
	}

	if err := yaml.Unmarshal(result, &parsed); err != nil {
		t.Errorf("Generated grafana datasource is not valid YAML: %v", err)
	}

	if parsed.APIVersion != 1 {
		t.Errorf("apiVersion = %d, want 1", parsed.APIVersion)
	}

	if len(parsed.Datasources) == 0 {
		t.Error("No datasources defined")
	} else {
		ds := parsed.Datasources[0]
		if ds.Name != "Prometheus" {
			t.Errorf("Datasource name = %q, want %q", ds.Name, "Prometheus")
		}
		if ds.Type != "prometheus" {
			t.Errorf("Datasource type = %q, want %q", ds.Type, "prometheus")
		}
		if ds.URL != "http://prometheus:9090" {
			t.Errorf("Datasource URL = %q, want %q", ds.URL, "http://prometheus:9090")
		}
		if !ds.IsDefault {
			t.Error("Datasource should be default")
		}
	}
}

// TestGrafanaDashboardProviderValidYAML verifies dashboard provider is valid YAML
func TestGrafanaDashboardProviderValidYAML(t *testing.T) {
	gen := NewMetricsSidecarGenerator()

	config := &MetricsSidecarConfig{
		ProjectName: "testapp",
	}

	result, err := gen.GenerateGrafanaDashboardProvider(config)
	if err != nil {
		t.Fatalf("GenerateGrafanaDashboardProvider() error = %v", err)
	}

	var parsed struct {
		APIVersion int `yaml:"apiVersion"`
		Providers  []struct {
			Name    string `yaml:"name"`
			Folder  string `yaml:"folder"`
			Type    string `yaml:"type"`
			Options struct {
				Path string `yaml:"path"`
			} `yaml:"options"`
		} `yaml:"providers"`
	}

	if err := yaml.Unmarshal(result, &parsed); err != nil {
		t.Errorf("Generated dashboard provider is not valid YAML: %v", err)
	}

	if parsed.APIVersion != 1 {
		t.Errorf("apiVersion = %d, want 1", parsed.APIVersion)
	}

	if len(parsed.Providers) == 0 {
		t.Error("No providers defined")
	} else {
		p := parsed.Providers[0]
		if p.Name != "dockstart" {
			t.Errorf("Provider name = %q, want %q", p.Name, "dockstart")
		}
		if p.Folder != "testapp" {
			t.Errorf("Provider folder = %q, want %q", p.Folder, "testapp")
		}
		if p.Type != "file" {
			t.Errorf("Provider type = %q, want %q", p.Type, "file")
		}
	}
}

// TestAppDashboardValidJSON verifies that generated dashboard is valid JSON
func TestAppDashboardValidJSON(t *testing.T) {
	gen := NewMetricsSidecarGenerator()

	tests := []struct {
		name   string
		config *MetricsSidecarConfig
	}{
		{
			name: "nodejs app",
			config: &MetricsSidecarConfig{
				ProjectName: "nodeapp",
				Language:    "nodejs",
			},
		},
		{
			name: "go app",
			config: &MetricsSidecarConfig{
				ProjectName: "goapp",
				Language:    "go",
			},
		},
		{
			name: "python app",
			config: &MetricsSidecarConfig{
				ProjectName: "pythonapp",
				Language:    "python",
			},
		},
		{
			name: "rust app",
			config: &MetricsSidecarConfig{
				ProjectName: "rustapp",
				Language:    "rust",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := gen.GenerateAppDashboard(tt.config)
			if err != nil {
				t.Fatalf("GenerateAppDashboard() error = %v", err)
			}

			// Verify valid JSON
			var parsed map[string]interface{}
			if err := json.Unmarshal(result, &parsed); err != nil {
				t.Errorf("Generated dashboard is not valid JSON: %v", err)
			}
		})
	}
}

// TestAppDashboardStructure verifies dashboard has correct Grafana structure
func TestAppDashboardStructure(t *testing.T) {
	gen := NewMetricsSidecarGenerator()

	config := &MetricsSidecarConfig{
		ProjectName: "structuretest",
		Language:    "go",
	}

	result, err := gen.GenerateAppDashboard(config)
	if err != nil {
		t.Fatalf("GenerateAppDashboard() error = %v", err)
	}

	var dashboard struct {
		UID       string   `json:"uid"`
		Title     string   `json:"title"`
		Tags      []string `json:"tags"`
		Timezone  string   `json:"timezone"`
		Refresh   string   `json:"refresh"`
		Editable  bool     `json:"editable"`
		SchemaVersion int  `json:"schemaVersion"`
		Panels    []struct {
			ID         int    `json:"id"`
			Title      string `json:"title"`
			Type       string `json:"type"`
			GridPos    struct {
				X int `json:"x"`
				Y int `json:"y"`
				W int `json:"w"`
				H int `json:"h"`
			} `json:"gridPos"`
			Targets []struct {
				Expr string `json:"expr"`
			} `json:"targets"`
		} `json:"panels"`
	}

	if err := json.Unmarshal(result, &dashboard); err != nil {
		t.Fatalf("Failed to parse dashboard JSON: %v", err)
	}

	// Verify dashboard metadata
	if dashboard.UID != "structuretest-app-metrics" {
		t.Errorf("Dashboard UID = %q, want %q", dashboard.UID, "structuretest-app-metrics")
	}

	if dashboard.Title != "structuretest - Application Metrics" {
		t.Errorf("Dashboard Title = %q, want %q", dashboard.Title, "structuretest - Application Metrics")
	}

	// Verify tags include project name and language
	foundProject := false
	foundLanguage := false
	for _, tag := range dashboard.Tags {
		if tag == "structuretest" {
			foundProject = true
		}
		if tag == "go" {
			foundLanguage = true
		}
	}
	if !foundProject {
		t.Error("Dashboard tags should include project name")
	}
	if !foundLanguage {
		t.Error("Dashboard tags should include language")
	}

	// Verify panels
	if len(dashboard.Panels) < 4 {
		t.Errorf("Expected at least 4 panels, got %d", len(dashboard.Panels))
	}

	// Verify each panel has required fields
	for i, panel := range dashboard.Panels {
		if panel.Title == "" {
			t.Errorf("Panel %d missing title", i)
		}
		if panel.Type != "timeseries" {
			t.Errorf("Panel %d type = %q, want %q", i, panel.Type, "timeseries")
		}
		if len(panel.Targets) == 0 {
			t.Errorf("Panel %d has no targets", i)
		}
	}
}

// TestGenerateErrorHandling tests error paths in Generate function
func TestGenerateErrorHandling(t *testing.T) {
	gen := NewMetricsSidecarGenerator()

	t.Run("non-existent output path", func(t *testing.T) {
		detection := &models.Detection{
			Language:         "nodejs",
			MetricsLibraries: []string{"prom-client"},
		}

		// Try to generate in a non-existent nested path without write permissions
		err := gen.Generate(detection, "/nonexistent/path/that/doesnt/exist", "testproject")
		if err == nil {
			t.Error("Expected error for non-existent output path")
		}
	})
}

// TestMetricsSidecarAllLanguages tests generation for all supported languages
func TestMetricsSidecarAllLanguages(t *testing.T) {
	gen := NewMetricsSidecarGenerator()

	languages := []struct {
		name      string
		detection *models.Detection
	}{
		{
			name: "nodejs",
			detection: &models.Detection{
				Language:         "nodejs",
				MetricsLibraries: []string{"prom-client"},
				MetricsPort:      3000,
				MetricsPath:      "/metrics",
			},
		},
		{
			name: "go",
			detection: &models.Detection{
				Language:         "go",
				MetricsLibraries: []string{"prometheus/client_golang"},
				MetricsPort:      8080,
				MetricsPath:      "/metrics",
			},
		},
		{
			name: "python",
			detection: &models.Detection{
				Language:         "python",
				MetricsLibraries: []string{"prometheus-client"},
				MetricsPort:      8000,
				MetricsPath:      "/metrics",
			},
		},
		{
			name: "rust",
			detection: &models.Detection{
				Language:         "rust",
				MetricsLibraries: []string{"prometheus"},
				MetricsPort:      8080,
				MetricsPath:      "/metrics",
			},
		},
	}

	for _, lang := range languages {
		t.Run(lang.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "dockstart-metrics-lang-*")
			if err != nil {
				t.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			err = gen.Generate(lang.detection, tmpDir, lang.name+"-project")
			if err != nil {
				t.Fatalf("Generate() error for %s: %v", lang.name, err)
			}

			// Verify all files created
			expectedFiles := []string{
				".devcontainer/prometheus/prometheus.yml",
				".devcontainer/grafana/provisioning/datasources/prometheus.yml",
				".devcontainer/grafana/provisioning/dashboards/provider.yml",
				".devcontainer/grafana/provisioning/dashboards/app-metrics.json",
			}

			for _, file := range expectedFiles {
				path := filepath.Join(tmpDir, file)
				if _, err := os.Stat(path); os.IsNotExist(err) {
					t.Errorf("Missing file for %s: %s", lang.name, file)
				}
			}

			// Verify dashboard includes correct language tag
			dashboardContent, err := os.ReadFile(filepath.Join(tmpDir, ".devcontainer/grafana/provisioning/dashboards/app-metrics.json"))
			if err != nil {
				t.Fatalf("Failed to read dashboard: %v", err)
			}

			var dashboard map[string]interface{}
			if err := json.Unmarshal(dashboardContent, &dashboard); err != nil {
				t.Fatalf("Invalid dashboard JSON for %s: %v", lang.name, err)
			}

			tags, ok := dashboard["tags"].([]interface{})
			if !ok {
				t.Errorf("Dashboard for %s missing tags", lang.name)
			} else {
				foundLang := false
				for _, tag := range tags {
					if tag == lang.name {
						foundLang = true
						break
					}
				}
				if !foundLang {
					t.Errorf("Dashboard for %s should have language tag", lang.name)
				}
			}
		})
	}
}

// TestMetricsSidecarWithAllServices tests generation with various service combinations
func TestMetricsSidecarWithAllServices(t *testing.T) {
	gen := NewMetricsSidecarGenerator()

	tests := []struct {
		name             string
		projectName      string
		detection        *models.Detection
		expectedJobNames []string
	}{
		{
			name:        "postgres only",
			projectName: "pgapp",
			detection: &models.Detection{
				Language:         "nodejs",
				MetricsLibraries: []string{"prom-client"},
				Services:         []string{"postgres"},
			},
			expectedJobNames: []string{"pgapp", "postgres"},
		},
		{
			name:        "redis only",
			projectName: "redisapp",
			detection: &models.Detection{
				Language:         "nodejs",
				MetricsLibraries: []string{"prom-client"},
				Services:         []string{"redis"},
			},
			expectedJobNames: []string{"redisapp", "redis"},
		},
		{
			name:        "both postgres and redis",
			projectName: "bothapp",
			detection: &models.Detection{
				Language:         "nodejs",
				MetricsLibraries: []string{"prom-client"},
				Services:         []string{"postgres", "redis"},
			},
			expectedJobNames: []string{"bothapp", "postgres", "redis"},
		},
		{
			name:        "with worker",
			projectName: "workerapp",
			detection: &models.Detection{
				Language:         "nodejs",
				MetricsLibraries: []string{"prom-client"},
				QueueLibraries:   []string{"bull"},
			},
			expectedJobNames: []string{"workerapp", "workerapp-worker"},
		},
		{
			name:        "full stack",
			projectName: "fullapp",
			detection: &models.Detection{
				Language:         "nodejs",
				MetricsLibraries: []string{"prom-client"},
				Services:         []string{"postgres", "redis"},
				QueueLibraries:   []string{"bull"},
			},
			expectedJobNames: []string{"fullapp", "fullapp-worker", "postgres", "redis"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "dockstart-metrics-services-*")
			if err != nil {
				t.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			err = gen.Generate(tt.detection, tmpDir, tt.projectName)
			if err != nil {
				t.Fatalf("Generate() error: %v", err)
			}

			// Read and parse prometheus config
			promContent, err := os.ReadFile(filepath.Join(tmpDir, ".devcontainer/prometheus/prometheus.yml"))
			if err != nil {
				t.Fatalf("Failed to read prometheus.yml: %v", err)
			}

			var promConfig struct {
				ScrapeConfigs []struct {
					JobName string `yaml:"job_name"`
				} `yaml:"scrape_configs"`
			}

			if err := yaml.Unmarshal(promContent, &promConfig); err != nil {
				t.Fatalf("Failed to parse prometheus.yml: %v", err)
			}

			// Verify expected jobs exist
			jobNames := make(map[string]bool)
			for _, sc := range promConfig.ScrapeConfigs {
				jobNames[sc.JobName] = true
			}

			for _, expected := range tt.expectedJobNames {
				if !jobNames[expected] {
					t.Errorf("Missing expected job: %s", expected)
				}
			}
		})
	}
}

// TestDashboardPromQLQueries verifies that dashboard PromQL queries are well-formed
func TestDashboardPromQLQueries(t *testing.T) {
	gen := NewMetricsSidecarGenerator()

	config := &MetricsSidecarConfig{
		ProjectName: "querytest",
		Language:    "go",
	}

	result, err := gen.GenerateAppDashboard(config)
	if err != nil {
		t.Fatalf("GenerateAppDashboard() error = %v", err)
	}

	var dashboard struct {
		Panels []struct {
			Title   string `json:"title"`
			Targets []struct {
				Expr         string `json:"expr"`
				LegendFormat string `json:"legendFormat"`
			} `json:"targets"`
		} `json:"panels"`
	}

	if err := json.Unmarshal(result, &dashboard); err != nil {
		t.Fatalf("Failed to parse dashboard: %v", err)
	}

	for _, panel := range dashboard.Panels {
		for _, target := range panel.Targets {
			// Verify query contains job filter with project name
			if target.Expr == "" {
				t.Errorf("Panel %q has empty expression", panel.Title)
			}

			// All queries should reference the project job
			if target.Expr != "" && panel.Title != "" {
				// Verify query has balanced brackets
				openParen := 0
				openBracket := 0
				openBrace := 0
				for _, c := range target.Expr {
					switch c {
					case '(':
						openParen++
					case ')':
						openParen--
					case '[':
						openBracket++
					case ']':
						openBracket--
					case '{':
						openBrace++
					case '}':
						openBrace--
					}
				}
				if openParen != 0 {
					t.Errorf("Panel %q has unbalanced parentheses in query: %s", panel.Title, target.Expr)
				}
				if openBracket != 0 {
					t.Errorf("Panel %q has unbalanced brackets in query: %s", panel.Title, target.Expr)
				}
				if openBrace != 0 {
					t.Errorf("Panel %q has unbalanced braces in query: %s", panel.Title, target.Expr)
				}
			}
		}
	}
}
