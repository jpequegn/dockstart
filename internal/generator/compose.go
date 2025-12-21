package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jpequegn/dockstart/internal/models"
)

// ServiceConfig holds configuration for a single Docker Compose service.
type ServiceConfig struct {
	// Name is the service name (e.g., "postgres", "redis")
	Name string
}

// LogSidecarComposeConfig holds configuration for the log aggregator sidecar.
type LogSidecarComposeConfig struct {
	// Enabled indicates whether to include the log sidecar
	Enabled bool

	// LogFormat is the detected log format ("json", "text", "unknown")
	LogFormat string

	// LoggingLibraries is the list of detected logging libraries
	LoggingLibraries []string
}

// WorkerSidecarConfig holds configuration for the background worker sidecar.
type WorkerSidecarConfig struct {
	// Enabled indicates whether to include the worker sidecar
	Enabled bool

	// Command is the command to start the worker process
	Command string

	// QueueLibraries is the list of detected queue libraries
	QueueLibraries []string
}

// BackupSidecarComposeConfig holds configuration for the backup sidecar.
type BackupSidecarComposeConfig struct {
	// Enabled indicates whether to include the backup sidecar
	Enabled bool

	// Schedule is the cron schedule for backups
	Schedule string

	// RetentionDays is the number of days to keep backups
	RetentionDays int

	// HasPostgres indicates if PostgreSQL backup is needed
	HasPostgres bool

	// HasMySQL indicates if MySQL backup is needed
	HasMySQL bool

	// HasRedis indicates if Redis backup is needed
	HasRedis bool

	// HasSQLite indicates if SQLite backup is needed
	HasSQLite bool

	// NeedsDockerSocket indicates if docker socket access is required
	NeedsDockerSocket bool
}

// FileProcessorSidecarComposeConfig holds configuration for the file processor sidecar.
type FileProcessorSidecarComposeConfig struct {
	// Enabled indicates whether to include the file processor sidecar
	Enabled bool

	// FileUploadLibraries is the list of detected file upload libraries
	FileUploadLibraries []string

	// UploadPath is the detected upload path from the app
	UploadPath string

	// ProcessImages enables image processing (resize, thumbnails)
	ProcessImages bool

	// ProcessDocuments enables document processing (PDF text extraction)
	ProcessDocuments bool

	// ProcessVideo enables video processing (thumbnails, previews)
	ProcessVideo bool

	// MemoryLimit is the memory limit for the processor container (e.g., "512M")
	MemoryLimit string

	// CPULimit is the CPU limit for the processor container (e.g., "0.5")
	CPULimit string
}

// MetricsSidecarComposeConfig holds configuration for the Prometheus + Grafana metrics stack.
type MetricsSidecarComposeConfig struct {
	// Enabled indicates whether to include the metrics sidecar
	Enabled bool

	// MetricsLibraries is the list of detected metrics libraries
	MetricsLibraries []string

	// MetricsPort is the port where the app exposes /metrics
	MetricsPort int

	// MetricsPath is the path to the metrics endpoint
	MetricsPath string

	// PrometheusPort is the external port for Prometheus (default: 9090)
	PrometheusPort int

	// GrafanaPort is the external port for Grafana (default: 3001)
	GrafanaPort int

	// HasWorker indicates if a worker service exists
	HasWorker bool

	// HasPostgres indicates if Postgres exporter should be included
	HasPostgres bool

	// HasRedis indicates if Redis exporter should be included
	HasRedis bool

	// RetentionDays is the number of days to retain metrics (default: 7)
	RetentionDays int
}

// ComposeConfig holds the configuration for generating docker-compose.yml.
type ComposeConfig struct {
	// Name is the project name (used for database names, etc.)
	Name string

	// Services is a list of additional services to include
	Services []ServiceConfig

	// LogSidecar holds configuration for the log aggregator sidecar
	LogSidecar LogSidecarComposeConfig

	// WorkerSidecar holds configuration for the background worker sidecar
	WorkerSidecar WorkerSidecarConfig

	// BackupSidecar holds configuration for the database backup sidecar
	BackupSidecar BackupSidecarComposeConfig

	// FileProcessorSidecar holds configuration for the file processor sidecar
	FileProcessorSidecar FileProcessorSidecarComposeConfig

	// MetricsSidecar holds configuration for the Prometheus + Grafana metrics stack
	MetricsSidecar MetricsSidecarComposeConfig
}

// ComposeGenerator generates docker-compose.yml files.
type ComposeGenerator struct{}

// NewComposeGenerator creates a new compose generator.
func NewComposeGenerator() *ComposeGenerator {
	return &ComposeGenerator{}
}

// Generate creates a docker-compose.yml file from a Detection.
// The file is written to .devcontainer/docker-compose.yml.
func (g *ComposeGenerator) Generate(detection *models.Detection, projectPath string, projectName string) error {
	config := g.buildConfig(detection, projectName)

	// Create .devcontainer directory (may already exist)
	devcontainerDir := filepath.Join(projectPath, ".devcontainer")
	if err := os.MkdirAll(devcontainerDir, 0755); err != nil {
		return fmt.Errorf("failed to create .devcontainer directory: %w", err)
	}

	// Generate docker-compose.yml content
	content, err := g.render(config)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Write to file
	outputPath := filepath.Join(devcontainerDir, "docker-compose.yml")
	if err := os.WriteFile(outputPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write docker-compose.yml: %w", err)
	}

	return nil
}

// GenerateContent returns the generated docker-compose.yml content without writing to disk.
// Useful for dry-run mode.
func (g *ComposeGenerator) GenerateContent(detection *models.Detection, projectName string) ([]byte, error) {
	config := g.buildConfig(detection, projectName)
	return g.render(config)
}

// buildConfig creates a ComposeConfig from a Detection.
func (g *ComposeGenerator) buildConfig(detection *models.Detection, projectName string) *ComposeConfig {
	config := &ComposeConfig{
		Name:     projectName,
		Services: make([]ServiceConfig, 0, len(detection.Services)),
	}

	// Convert detected services to ServiceConfig
	for _, service := range detection.Services {
		config.Services = append(config.Services, ServiceConfig{
			Name: service,
		})
	}

	// Configure log sidecar if structured logging is detected
	if detection.HasStructuredLogging() {
		config.LogSidecar = LogSidecarComposeConfig{
			Enabled:          true,
			LogFormat:        detection.LogFormat,
			LoggingLibraries: detection.LoggingLibraries,
		}
	}

	// Configure worker sidecar if queue libraries are detected
	if detection.NeedsWorker() {
		config.WorkerSidecar = WorkerSidecarConfig{
			Enabled:        true,
			Command:        detection.WorkerCommand,
			QueueLibraries: detection.QueueLibraries,
		}

		// Auto-add Redis if a Redis-based queue library is detected
		// but Redis wasn't detected as a direct dependency
		if needsRedis(detection.QueueLibraries) && !hasService(config.Services, "redis") {
			config.Services = append(config.Services, ServiceConfig{
				Name: "redis",
			})
		}
	}

	// Configure backup sidecar if any database services are detected
	hasPostgres := hasService(config.Services, "postgres")
	hasMySQL := hasService(config.Services, "mysql")
	hasRedis := hasService(config.Services, "redis")

	if hasPostgres || hasMySQL || hasRedis {
		config.BackupSidecar = BackupSidecarComposeConfig{
			Enabled:           true,
			Schedule:          "0 3 * * *", // Daily at 3 AM
			RetentionDays:     7,
			HasPostgres:       hasPostgres,
			HasMySQL:          hasMySQL,
			HasRedis:          hasRedis,
			HasSQLite:         false, // SQLite detection not implemented yet
			NeedsDockerSocket: hasRedis, // Redis backup uses docker cp
		}
	}

	// Configure file processor sidecar if file upload libraries are detected
	if detection.NeedsFileProcessor() {
		uploadPath := detection.UploadPath
		if uploadPath == "" {
			uploadPath = "/uploads"
		}

		config.FileProcessorSidecar = FileProcessorSidecarComposeConfig{
			Enabled:             true,
			FileUploadLibraries: detection.FileUploadLibraries,
			UploadPath:          uploadPath,
			ProcessImages:       true,  // Enable by default
			ProcessDocuments:    false, // Disabled by default
			ProcessVideo:        false, // Disabled by default
			MemoryLimit:         "512M",
			CPULimit:            "0.5",
		}
	}

	// Configure metrics sidecar if metrics libraries are detected
	if detection.NeedsMetrics() {
		metricsPort := detection.GetMetricsPort()
		if detection.MetricsPort > 0 {
			metricsPort = detection.MetricsPort
		}

		metricsPath := detection.GetMetricsPath()
		if detection.MetricsPath != "" {
			metricsPath = detection.MetricsPath
		}

		config.MetricsSidecar = MetricsSidecarComposeConfig{
			Enabled:          true,
			MetricsLibraries: detection.MetricsLibraries,
			MetricsPort:      metricsPort,
			MetricsPath:      metricsPath,
			PrometheusPort:   9090,
			GrafanaPort:      3001,
			HasWorker:        detection.NeedsWorker(),
			HasPostgres:      hasPostgres,
			HasRedis:         hasRedis,
			RetentionDays:    7,
		}
	}

	return config
}

// redisBasedQueueLibraries contains queue libraries that require Redis as a broker.
var redisBasedQueueLibraries = map[string]bool{
	// Node.js
	"bull":      true,
	"bullmq":    true,
	"bee-queue": true,
	// Go
	"asynq": true,
	"rmq":   true,
	// Python
	"rq":  true,
	"arq": true,
	// Note: celery can use Redis but also supports other brokers
	// Rust
	"sidekiq": true,
}

// needsRedis checks if any detected queue library requires Redis.
func needsRedis(queueLibraries []string) bool {
	for _, lib := range queueLibraries {
		if redisBasedQueueLibraries[lib] {
			return true
		}
	}
	return false
}

// hasService checks if a service is already in the list.
func hasService(services []ServiceConfig, name string) bool {
	for _, s := range services {
		if s.Name == name {
			return true
		}
	}
	return false
}

// render executes the template with the given config.
func (g *ComposeGenerator) render(config *ComposeConfig) ([]byte, error) {
	tmpl, err := loadTemplate("docker-compose.yml.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}
