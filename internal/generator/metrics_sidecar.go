// Package generator provides code generation for devcontainer files.
package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jpequegn/dockstart/internal/models"
)

// MetricsSidecarConfig holds configuration for generating Prometheus + Grafana sidecar files.
type MetricsSidecarConfig struct {
	// ProjectName is the name of the project
	ProjectName string

	// Language is the detected programming language
	Language string

	// MetricsPort is the port where /metrics is exposed
	MetricsPort int

	// MetricsPath is the path to the metrics endpoint
	MetricsPath string

	// ScrapeInterval is the global Prometheus scrape interval (default: 30s)
	ScrapeInterval string

	// AppScrapeInterval is the scrape interval for the app (default: 15s)
	AppScrapeInterval string

	// HasWorker indicates if a worker service exists
	HasWorker bool

	// WorkerMetricsPort is the metrics port for the worker
	WorkerMetricsPort int

	// HasPostgres indicates if Postgres is detected
	HasPostgres bool

	// HasRedis indicates if Redis is detected
	HasRedis bool

	// GrafanaPort is the port to expose Grafana on (default: 3001)
	GrafanaPort int

	// PrometheusPort is the port to expose Prometheus on (default: 9090)
	PrometheusPort int

	// RetentionDays is the number of days to retain metrics (default: 7)
	RetentionDays int
}

// DefaultMetricsConfig returns a MetricsSidecarConfig with sensible defaults.
func DefaultMetricsConfig() *MetricsSidecarConfig {
	return &MetricsSidecarConfig{
		ScrapeInterval:    "30s",
		AppScrapeInterval: "15s",
		MetricsPath:       "/metrics",
		MetricsPort:       3000,
		GrafanaPort:       3001,
		PrometheusPort:    9090,
		RetentionDays:     7,
	}
}

// MetricsSidecarGenerator generates Prometheus + Grafana configuration files.
type MetricsSidecarGenerator struct{}

// NewMetricsSidecarGenerator creates a new metrics sidecar generator.
func NewMetricsSidecarGenerator() *MetricsSidecarGenerator {
	return &MetricsSidecarGenerator{}
}

// GeneratePrometheusConfig generates the prometheus.yml content.
func (g *MetricsSidecarGenerator) GeneratePrometheusConfig(config *MetricsSidecarConfig) ([]byte, error) {
	tmpl, err := loadTemplate("prometheus.yml.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to load prometheus template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return nil, fmt.Errorf("failed to execute prometheus template: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateGrafanaDatasource generates the Grafana datasource provisioning file.
func (g *MetricsSidecarGenerator) GenerateGrafanaDatasource(config *MetricsSidecarConfig) ([]byte, error) {
	tmpl, err := loadTemplate("grafana/datasources/prometheus.yml.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to load datasource template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return nil, fmt.Errorf("failed to execute datasource template: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateGrafanaDashboardProvider generates the Grafana dashboard provider file.
func (g *MetricsSidecarGenerator) GenerateGrafanaDashboardProvider(config *MetricsSidecarConfig) ([]byte, error) {
	tmpl, err := loadTemplate("grafana/dashboards/provider.yml.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to load dashboard provider template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return nil, fmt.Errorf("failed to execute dashboard provider template: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateAppDashboard generates the application metrics dashboard JSON.
func (g *MetricsSidecarGenerator) GenerateAppDashboard(config *MetricsSidecarConfig) ([]byte, error) {
	tmpl, err := loadTemplate("grafana/dashboards/app-metrics.json.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to load app dashboard template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return nil, fmt.Errorf("failed to execute app dashboard template: %w", err)
	}

	return buf.Bytes(), nil
}

// Generate creates all Prometheus and Grafana configuration files.
func (g *MetricsSidecarGenerator) Generate(detection *models.Detection, outputPath, projectName string) error {
	config := g.buildConfig(detection, projectName)

	devcontainerDir := filepath.Join(outputPath, ".devcontainer")

	// Create prometheus directory
	prometheusDir := filepath.Join(devcontainerDir, "prometheus")
	if err := os.MkdirAll(prometheusDir, 0755); err != nil {
		return fmt.Errorf("failed to create prometheus directory: %w", err)
	}

	// Create grafana provisioning directories
	grafanaDatasourcesDir := filepath.Join(devcontainerDir, "grafana", "provisioning", "datasources")
	grafanaDashboardsDir := filepath.Join(devcontainerDir, "grafana", "provisioning", "dashboards")
	if err := os.MkdirAll(grafanaDatasourcesDir, 0755); err != nil {
		return fmt.Errorf("failed to create grafana datasources directory: %w", err)
	}
	if err := os.MkdirAll(grafanaDashboardsDir, 0755); err != nil {
		return fmt.Errorf("failed to create grafana dashboards directory: %w", err)
	}

	// Generate prometheus.yml
	prometheusConfig, err := g.GeneratePrometheusConfig(config)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(prometheusDir, "prometheus.yml"), prometheusConfig, 0644); err != nil {
		return fmt.Errorf("failed to write prometheus.yml: %w", err)
	}

	// Generate Grafana datasource
	datasource, err := g.GenerateGrafanaDatasource(config)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(grafanaDatasourcesDir, "prometheus.yml"), datasource, 0644); err != nil {
		return fmt.Errorf("failed to write grafana datasource: %w", err)
	}

	// Generate Grafana dashboard provider
	provider, err := g.GenerateGrafanaDashboardProvider(config)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(grafanaDashboardsDir, "provider.yml"), provider, 0644); err != nil {
		return fmt.Errorf("failed to write grafana dashboard provider: %w", err)
	}

	// Generate app metrics dashboard
	dashboard, err := g.GenerateAppDashboard(config)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(grafanaDashboardsDir, "app-metrics.json"), dashboard, 0644); err != nil {
		return fmt.Errorf("failed to write app-metrics dashboard: %w", err)
	}

	return nil
}

// buildConfig creates a MetricsSidecarConfig from detection results.
func (g *MetricsSidecarGenerator) buildConfig(detection *models.Detection, projectName string) *MetricsSidecarConfig {
	config := DefaultMetricsConfig()
	config.ProjectName = projectName
	config.Language = detection.Language

	// Set metrics port and path from detection
	if detection.MetricsPort > 0 {
		config.MetricsPort = detection.MetricsPort
	} else {
		config.MetricsPort = detection.GetMetricsPort()
	}

	if detection.MetricsPath != "" {
		config.MetricsPath = detection.MetricsPath
	} else {
		config.MetricsPath = detection.GetMetricsPath()
	}

	// Check for worker
	config.HasWorker = detection.NeedsWorker()
	if config.HasWorker {
		config.WorkerMetricsPort = config.MetricsPort
	}

	// Check for services
	config.HasPostgres = detection.HasService("postgres")
	config.HasRedis = detection.HasService("redis")

	return config
}

// ShouldGenerate returns true if metrics sidecar should be generated.
func (g *MetricsSidecarGenerator) ShouldGenerate(detection *models.Detection) bool {
	// Generate metrics stack if metrics libraries are detected
	return detection.NeedsMetrics()
}
