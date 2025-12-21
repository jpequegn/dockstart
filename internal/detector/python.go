package detector

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/jpequegn/dockstart/internal/models"
)

// PythonDetector detects Python projects by analyzing pyproject.toml or requirements.txt.
type PythonDetector struct{}

// NewPythonDetector creates a new Python detector.
func NewPythonDetector() *PythonDetector {
	return &PythonDetector{}
}

// Name returns the detector identifier.
func (d *PythonDetector) Name() string {
	return "python"
}

// pyprojectTOML represents the structure of a pyproject.toml file.
// We only parse the fields we care about.
type pyprojectTOML struct {
	Project struct {
		Name            string   `toml:"name"`
		RequiresPython  string   `toml:"requires-python"`
		Dependencies    []string `toml:"dependencies"`
		OptionalDeps    map[string][]string `toml:"optional-dependencies"`
	} `toml:"project"`
	Tool struct {
		Poetry struct {
			Name         string            `toml:"name"`
			Dependencies map[string]interface{} `toml:"dependencies"`
			DevDeps      map[string]interface{} `toml:"dev-dependencies"`
		} `toml:"poetry"`
	} `toml:"tool"`
}

// Detect analyzes the path for a Python project.
// It looks for pyproject.toml or requirements.txt and extracts version and service information.
func (d *PythonDetector) Detect(path string) (*models.Detection, error) {
	// Try pyproject.toml first (modern Python)
	pyprojectPath := filepath.Join(path, "pyproject.toml")
	if _, err := os.Stat(pyprojectPath); err == nil {
		return d.detectFromPyproject(pyprojectPath)
	}

	// Fall back to requirements.txt
	requirementsPath := filepath.Join(path, "requirements.txt")
	if _, err := os.Stat(requirementsPath); err == nil {
		return d.detectFromRequirements(requirementsPath)
	}

	// Not a Python project
	return nil, nil
}

// detectFromPyproject parses pyproject.toml for Python project info.
func (d *PythonDetector) detectFromPyproject(path string) (*models.Detection, error) {
	var config pyprojectTOML
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, err
	}

	// Collect all dependencies and extract package names
	var deps []string
	for _, dep := range config.Project.Dependencies {
		deps = append(deps, d.extractPackageName(dep))
	}
	for _, optDeps := range config.Project.OptionalDeps {
		for _, dep := range optDeps {
			deps = append(deps, d.extractPackageName(dep))
		}
	}

	// Also check Poetry format (Poetry uses map keys as package names)
	for dep := range config.Tool.Poetry.Dependencies {
		if dep != "python" { // Skip python version specifier
			deps = append(deps, dep)
		}
	}
	for dep := range config.Tool.Poetry.DevDeps {
		deps = append(deps, dep)
	}

	loggingLibs, logFormat := d.detectLogging(deps)
	queueLibs, workerCmd := d.detectQueue(deps, config.Project.Name, config.Tool.Poetry.Name)
	uploadLibs, uploadPath := d.detectFileUpload(deps, filepath.Dir(path))
	metricsLibs, metricsPort, metricsPath := d.detectMetrics(deps)

	detection := &models.Detection{
		Language:            "python",
		Version:             d.extractVersion(config),
		Services:            d.detectServicesFromDeps(deps),
		Confidence:          d.calculateConfidencePyproject(config),
		LoggingLibraries:    loggingLibs,
		LogFormat:           logFormat,
		QueueLibraries:      queueLibs,
		WorkerCommand:       workerCmd,
		FileUploadLibraries: uploadLibs,
		UploadPath:          uploadPath,
		MetricsLibraries:    metricsLibs,
		MetricsPort:         metricsPort,
		MetricsPath:         metricsPath,
	}

	return detection, nil
}

// extractPackageName extracts the package name from a dependency string.
// Examples: "redis>=4.0.0" -> "redis", "psycopg2-binary" -> "psycopg2-binary"
func (d *PythonDetector) extractPackageName(dep string) string {
	// Match package name (before any version specifier)
	re := regexp.MustCompile(`^([a-zA-Z0-9_-]+)`)
	if matches := re.FindStringSubmatch(dep); matches != nil {
		return strings.ToLower(matches[1])
	}
	return strings.ToLower(dep)
}

// detectFromRequirements parses requirements.txt for Python project info.
func (d *PythonDetector) detectFromRequirements(path string) (*models.Detection, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var deps []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Extract package name (before any version specifier)
		// e.g., "psycopg2>=2.9.0" -> "psycopg2"
		re := regexp.MustCompile(`^([a-zA-Z0-9_-]+)`)
		if matches := re.FindStringSubmatch(line); matches != nil {
			deps = append(deps, strings.ToLower(matches[1]))
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	loggingLibs, logFormat := d.detectLogging(deps)
	queueLibs, workerCmd := d.detectQueue(deps, "", "")
	uploadLibs, uploadPath := d.detectFileUpload(deps, filepath.Dir(path))
	metricsLibs, metricsPort, metricsPath := d.detectMetrics(deps)

	detection := &models.Detection{
		Language:            "python",
		Version:             "3.11", // Default when not specified
		Services:            d.detectServicesFromDeps(deps),
		Confidence:          0.6, // Lower confidence without pyproject.toml
		LoggingLibraries:    loggingLibs,
		LogFormat:           logFormat,
		QueueLibraries:      queueLibs,
		WorkerCommand:       workerCmd,
		FileUploadLibraries: uploadLibs,
		UploadPath:          uploadPath,
		MetricsLibraries:    metricsLibs,
		MetricsPort:         metricsPort,
		MetricsPath:         metricsPath,
	}

	return detection, nil
}

// extractVersion extracts the Python version from pyproject.toml.
func (d *PythonDetector) extractVersion(config pyprojectTOML) string {
	// Try project.requires-python first
	if config.Project.RequiresPython != "" {
		return d.parseVersionConstraint(config.Project.RequiresPython)
	}

	// Try Poetry python dependency
	if pythonVer, ok := config.Tool.Poetry.Dependencies["python"]; ok {
		if verStr, ok := pythonVer.(string); ok {
			return d.parseVersionConstraint(verStr)
		}
	}

	// Default to Python 3.11 (current stable)
	return "3.11"
}

// parseVersionConstraint extracts the major.minor version from a constraint.
// Examples: ">=3.10" -> "3.10", "^3.11" -> "3.11", ">=3.9,<4.0" -> "3.9"
func (d *PythonDetector) parseVersionConstraint(constraint string) string {
	// Match version pattern like 3.10, 3.11, etc.
	re := regexp.MustCompile(`(\d+\.\d+)`)
	match := re.FindString(constraint)
	if match != "" {
		return match
	}
	return "3.11" // Default
}

// detectServicesFromDeps identifies backing services from dependencies.
func (d *PythonDetector) detectServicesFromDeps(deps []string) []string {
	var services []string

	// PostgreSQL indicators
	postgresPackages := []string{
		"psycopg2", "psycopg2-binary", "psycopg",
		"asyncpg", "sqlalchemy", "django",
		"databases", "tortoise-orm", "piccolo",
	}

	// Redis indicators
	redisPackages := []string{
		"redis", "aioredis", "redis-py",
		"celery", "rq", "dramatiq",
	}

	for _, dep := range deps {
		depLower := strings.ToLower(dep)

		// Check PostgreSQL
		for _, pkg := range postgresPackages {
			if depLower == pkg || strings.HasPrefix(depLower, pkg+"-") {
				if !containsService(services, "postgres") {
					services = append(services, "postgres")
				}
				break
			}
		}

		// Check Redis
		for _, pkg := range redisPackages {
			if depLower == pkg || strings.HasPrefix(depLower, pkg+"-") {
				if !containsService(services, "redis") {
					services = append(services, "redis")
				}
				break
			}
		}
	}

	return services
}

// calculateConfidencePyproject determines how confident we are in the detection.
func (d *PythonDetector) calculateConfidencePyproject(config pyprojectTOML) float64 {
	confidence := 0.7 // Base confidence for having pyproject.toml

	// Higher confidence if project name is specified
	if config.Project.Name != "" || config.Tool.Poetry.Name != "" {
		confidence += 0.1
	}

	// Higher confidence if Python version is specified
	if config.Project.RequiresPython != "" {
		confidence += 0.1
	}

	// Higher confidence if dependencies exist
	if len(config.Project.Dependencies) > 0 || len(config.Tool.Poetry.Dependencies) > 0 {
		confidence += 0.1
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// GetVSCodeExtensions returns recommended VS Code extensions for Python.
func (d *PythonDetector) GetVSCodeExtensions() []string {
	return []string{
		"ms-python.python",
		"ms-python.vscode-pylance",
	}
}

// detectLogging identifies structured logging libraries from Python dependencies.
// Returns the list of detected libraries and the inferred log format.
func (d *PythonDetector) detectLogging(deps []string) ([]string, string) {
	var libraries []string
	logFormat := "unknown"

	// Structured logging libraries that output JSON by default
	jsonLoggers := map[string]string{
		"structlog":           "structlog",
		"python-json-logger":  "python-json-logger",
		"json-logging":        "json-logging",
		"pythonjsonlogger":    "python-json-logger",
	}

	// Structured JSON loggers
	jsonStructuredLoggers := map[string]string{
		"eliot":               "eliot",
	}

	// Logging libraries that typically output text by default
	textLoggers := map[string]string{
		"loguru":              "loguru",
		"logbook":             "logbook",
		"twiggy":              "twiggy",
	}

	// Logging utilities/formatters
	loggingUtils := map[string]string{
		"coloredlogs":         "coloredlogs",
		"rich":                "rich",
	}

	for _, dep := range deps {
		depLower := strings.ToLower(dep)

		// Check JSON loggers first
		for pkg, name := range jsonLoggers {
			if depLower == pkg {
				libraries = append(libraries, name)
				logFormat = "json"
				break
			}
		}

		// Check structured JSON loggers
		for pkg, name := range jsonStructuredLoggers {
			if depLower == pkg {
				libraries = append(libraries, name)
				logFormat = "json"
				break
			}
		}

		// Check text loggers
		for pkg, name := range textLoggers {
			if depLower == pkg {
				libraries = append(libraries, name)
				if logFormat == "unknown" {
					logFormat = "text"
				}
				break
			}
		}

		// Check logging utilities
		for pkg, name := range loggingUtils {
			if depLower == pkg {
				libraries = append(libraries, name)
				break
			}
		}
	}

	return libraries, logFormat
}

// detectQueue identifies job queue/worker libraries from Python dependencies.
// Returns the list of detected libraries and the inferred worker command.
func (d *PythonDetector) detectQueue(deps []string, projectName, poetryName string) ([]string, string) {
	var libraries []string
	workerCmd := ""

	// Queue libraries that require a worker process
	queuePackages := map[string]string{
		"celery":   "celery",
		"rq":       "rq",
		"dramatiq": "dramatiq",
		"huey":     "huey",
		"arq":      "arq",
		"taskiq":   "taskiq",
	}

	for _, dep := range deps {
		depLower := strings.ToLower(dep)

		for pkg, name := range queuePackages {
			if depLower == pkg {
				libraries = append(libraries, name)
				break
			}
		}
	}

	// If queue libraries detected, set appropriate worker command
	if len(libraries) > 0 {
		// Determine app name for worker command
		appName := "app"
		if projectName != "" {
			appName = projectName
		} else if poetryName != "" {
			appName = poetryName
		}

		// Set worker command based on detected library
		// Priority: celery > dramatiq > rq > huey > arq > taskiq
		for _, lib := range libraries {
			switch lib {
			case "celery":
				workerCmd = "celery -A " + appName + " worker"
				return libraries, workerCmd
			case "dramatiq":
				workerCmd = "dramatiq " + appName
				return libraries, workerCmd
			case "rq":
				workerCmd = "rq worker"
				return libraries, workerCmd
			case "huey":
				workerCmd = "huey_consumer " + appName + ".huey"
				return libraries, workerCmd
			case "arq":
				workerCmd = "arq " + appName + ".WorkerSettings"
				return libraries, workerCmd
			case "taskiq":
				workerCmd = "taskiq worker " + appName + ":broker"
				return libraries, workerCmd
			}
		}
	}

	return libraries, workerCmd
}

// detectFileUpload identifies file upload libraries from Python dependencies.
// Returns the list of detected libraries and the inferred upload path.
func (d *PythonDetector) detectFileUpload(deps []string, projectPath string) ([]string, string) {
	var libraries []string
	uploadPath := ""

	// File upload libraries
	uploadPackages := map[string]string{
		"python-multipart": "python-multipart",
		"aiofiles":         "aiofiles",
		"starlette":        "starlette",
		"werkzeug":         "werkzeug",
	}

	// Web frameworks with file upload support
	webFrameworks := map[string]string{
		"fastapi":  "fastapi",
		"flask":    "flask",
		"django":   "django",
		"starlite": "starlite",
		"litestar": "litestar",
	}

	hasWebFramework := false

	for _, dep := range deps {
		depLower := strings.ToLower(dep)

		// Check upload libraries
		for pkg, name := range uploadPackages {
			if depLower == pkg {
				libraries = append(libraries, name)
				break
			}
		}

		// Check web frameworks
		for pkg := range webFrameworks {
			if depLower == pkg || strings.HasPrefix(depLower, pkg+"[") {
				hasWebFramework = true
				break
			}
		}
	}

	// If we have upload libraries or a web framework, check for uploads directory
	if len(libraries) > 0 || hasWebFramework {
		uploadPath = d.findUploadPath(projectPath)
	}

	return libraries, uploadPath
}

// findUploadPath attempts to find the upload directory for Python projects.
func (d *PythonDetector) findUploadPath(projectPath string) string {
	// Common upload directory names
	commonDirs := []string{
		"uploads",
		"upload",
		"files",
		"media",
		"media/uploads",
		"static/uploads",
	}

	for _, dir := range commonDirs {
		fullPath := filepath.Join(projectPath, dir)
		if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
			return dir
		}
	}

	return ""
}

// detectMetrics identifies Prometheus metrics libraries from Python dependencies.
// Returns the list of detected libraries, the metrics port, and the metrics path.
func (d *PythonDetector) detectMetrics(deps []string) ([]string, int, string) {
	var libraries []string
	metricsPort := 0  // 0 means use default
	metricsPath := "" // Empty means use default "/metrics"

	// Prometheus client libraries for Python
	metricsPackages := map[string]string{
		"prometheus-client":                  "prometheus-client",
		"prometheus_client":                  "prometheus-client",
		"prometheus-fastapi-instrumentator":  "prometheus-fastapi-instrumentator",
		"prometheus-flask-exporter":          "prometheus-flask-exporter",
		"django-prometheus":                  "django-prometheus",
		"starlette-prometheus":               "starlette-prometheus",
		"opentelemetry-exporter-prometheus":  "opentelemetry-prometheus",
		"aioprometheus":                      "aioprometheus",
	}

	for _, dep := range deps {
		depLower := strings.ToLower(dep)

		for pkg, name := range metricsPackages {
			if depLower == pkg || strings.ReplaceAll(depLower, "_", "-") == pkg {
				// Avoid duplicates
				found := false
				for _, lib := range libraries {
					if lib == name {
						found = true
						break
					}
				}
				if !found {
					libraries = append(libraries, name)
				}
				break
			}
		}
	}

	// If metrics libraries detected, default port is 8000 (Python/FastAPI standard)
	// and path is "/metrics"
	if len(libraries) > 0 {
		metricsPort = 8000
		metricsPath = "/metrics"
	}

	return libraries, metricsPort, metricsPath
}
