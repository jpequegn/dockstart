// Package models contains shared data structures used across the application.
package models

// Detection represents the result of analyzing a project directory.
// It contains information about the detected language, version, and services.
type Detection struct {
	// Language is the primary programming language detected (e.g., "node", "go", "python", "rust")
	Language string

	// Version is the detected or inferred language version (e.g., "20", "1.23", "3.11")
	Version string

	// Services is a list of detected backing services (e.g., "postgres", "redis")
	Services []string

	// Confidence is a score from 0.0 to 1.0 indicating detection certainty
	// Higher values mean more confident detection (e.g., explicit version vs inferred)
	Confidence float64

	// LoggingLibraries is a list of detected structured logging libraries
	// (e.g., "winston", "pino" for Node.js, "zap", "zerolog" for Go)
	LoggingLibraries []string

	// LogFormat indicates the detected or inferred log format
	// Values: "json", "text", "unknown"
	LogFormat string

	// QueueLibraries is a list of detected job queue/worker libraries
	// (e.g., "bull", "bullmq" for Node.js, "celery" for Python)
	QueueLibraries []string

	// WorkerCommand is the detected or inferred command to start the worker
	// (e.g., "npm run worker", "celery -A app worker")
	WorkerCommand string

	// FileUploadLibraries is a list of detected file upload libraries
	// (e.g., "multer", "formidable" for Node.js, "python-multipart" for Python)
	FileUploadLibraries []string

	// UploadPath is the detected upload directory path (e.g., "/uploads", "uploads/")
	// Empty string if not detected
	UploadPath string

	// MetricsLibraries is a list of detected Prometheus metrics libraries
	// (e.g., "prom-client" for Node.js, "prometheus/client_golang" for Go)
	MetricsLibraries []string

	// MetricsPort is the detected or inferred port for the /metrics endpoint
	// Default: same as app port (e.g., 3000 for Node.js, 8080 for Go)
	MetricsPort int

	// MetricsPath is the detected or inferred path for the metrics endpoint
	// Default: "/metrics"
	MetricsPath string
}

// Project represents a fully analyzed project with all its detections.
type Project struct {
	// Path is the absolute path to the project directory
	Path string

	// Name is the project name (derived from directory or package.json/go.mod)
	Name string

	// Detection contains the primary language detection result
	Detection *Detection
}

// HasService checks if a specific service was detected.
func (d *Detection) HasService(service string) bool {
	for _, s := range d.Services {
		if s == service {
			return true
		}
	}
	return false
}

// AddService adds a service to the detection if not already present.
func (d *Detection) AddService(service string) {
	if !d.HasService(service) {
		d.Services = append(d.Services, service)
	}
}

// HasLoggingLibrary checks if a specific logging library was detected.
func (d *Detection) HasLoggingLibrary(library string) bool {
	for _, l := range d.LoggingLibraries {
		if l == library {
			return true
		}
	}
	return false
}

// AddLoggingLibrary adds a logging library to the detection if not already present.
func (d *Detection) AddLoggingLibrary(library string) {
	if !d.HasLoggingLibrary(library) {
		d.LoggingLibraries = append(d.LoggingLibraries, library)
	}
}

// HasStructuredLogging returns true if any structured logging library was detected.
func (d *Detection) HasStructuredLogging() bool {
	return len(d.LoggingLibraries) > 0
}

// HasQueueLibrary checks if a specific queue library was detected.
func (d *Detection) HasQueueLibrary(library string) bool {
	for _, l := range d.QueueLibraries {
		if l == library {
			return true
		}
	}
	return false
}

// AddQueueLibrary adds a queue library to the detection if not already present.
func (d *Detection) AddQueueLibrary(library string) {
	if !d.HasQueueLibrary(library) {
		d.QueueLibraries = append(d.QueueLibraries, library)
	}
}

// NeedsWorker returns true if any queue library was detected that requires a worker.
func (d *Detection) NeedsWorker() bool {
	return len(d.QueueLibraries) > 0
}

// HasFileUploadLibrary checks if a specific file upload library was detected.
func (d *Detection) HasFileUploadLibrary(library string) bool {
	for _, l := range d.FileUploadLibraries {
		if l == library {
			return true
		}
	}
	return false
}

// AddFileUploadLibrary adds a file upload library to the detection if not already present.
func (d *Detection) AddFileUploadLibrary(library string) {
	if !d.HasFileUploadLibrary(library) {
		d.FileUploadLibraries = append(d.FileUploadLibraries, library)
	}
}

// NeedsFileProcessor returns true if any file upload library was detected.
func (d *Detection) NeedsFileProcessor() bool {
	return len(d.FileUploadLibraries) > 0
}

// HasMetricsLibrary checks if a specific metrics library was detected.
func (d *Detection) HasMetricsLibrary(library string) bool {
	for _, l := range d.MetricsLibraries {
		if l == library {
			return true
		}
	}
	return false
}

// AddMetricsLibrary adds a metrics library to the detection if not already present.
func (d *Detection) AddMetricsLibrary(library string) {
	if !d.HasMetricsLibrary(library) {
		d.MetricsLibraries = append(d.MetricsLibraries, library)
	}
}

// NeedsMetrics returns true if any Prometheus metrics library was detected.
func (d *Detection) NeedsMetrics() bool {
	return len(d.MetricsLibraries) > 0
}

// GetMetricsPath returns the metrics endpoint path, defaulting to "/metrics".
func (d *Detection) GetMetricsPath() string {
	if d.MetricsPath != "" {
		return d.MetricsPath
	}
	return "/metrics"
}

// GetMetricsPort returns the metrics port, defaulting to the standard app port for the language.
func (d *Detection) GetMetricsPort() int {
	if d.MetricsPort != 0 {
		return d.MetricsPort
	}
	// Default ports by language
	switch d.Language {
	case "node":
		return 3000
	case "go":
		return 8080
	case "python":
		return 8000
	case "rust":
		return 8080
	default:
		return 3000
	}
}

// BackupConfig represents the configuration for database backup sidecar.
type BackupConfig struct {
	// DatabaseType is the type of database (postgres, mysql, redis, sqlite)
	DatabaseType string

	// ContainerName is the name of the database container
	ContainerName string

	// DatabaseHost is the hostname of the database (usually container name)
	DatabaseHost string

	// DatabaseName is the name of the database to backup
	DatabaseName string

	// DatabaseUser is the database user for authentication
	DatabaseUser string

	// DatabasePassword is the database password for authentication
	DatabasePassword string

	// DatabasePath is the path to the database file (SQLite only)
	DatabasePath string

	// AppContainer is the app container name (SQLite only, for stopping)
	AppContainer string

	// Schedule is the cron schedule for backups (default: "0 3 * * *")
	Schedule string

	// RetentionDays is the number of days to keep backups (default: 7)
	RetentionDays int

	// CompressionLevel is the gzip compression level 1-9 (default: 6)
	CompressionLevel int

	// StopContainer indicates if container should be stopped for backup (SQLite)
	StopContainer bool
}

// DefaultBackupConfig returns a BackupConfig with sensible defaults.
func DefaultBackupConfig(dbType, containerName string) *BackupConfig {
	return &BackupConfig{
		DatabaseType:     dbType,
		ContainerName:    containerName,
		DatabaseHost:     containerName,
		Schedule:         "0 3 * * *",
		RetentionDays:    7,
		CompressionLevel: 6,
		StopContainer:    dbType == "sqlite",
	}
}

// GetBackupExtension returns the file extension for the database type.
func (b *BackupConfig) GetBackupExtension() string {
	switch b.DatabaseType {
	case "postgres", "mysql", "mariadb":
		return "sql.gz"
	case "redis":
		return "rdb.gz"
	case "sqlite":
		return "db.gz"
	default:
		return "backup.gz"
	}
}

// NeedsDockerSocket returns true if the backup requires Docker socket access.
func (b *BackupConfig) NeedsDockerSocket() bool {
	// Redis uses docker cp, SQLite may need to stop containers
	return b.DatabaseType == "redis" || (b.DatabaseType == "sqlite" && b.StopContainer)
}
