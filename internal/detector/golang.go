package detector

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jpequegn/dockstart/internal/models"
)

// GoDetector detects Go projects by analyzing go.mod files.
type GoDetector struct{}

// NewGoDetector creates a new Go detector.
func NewGoDetector() *GoDetector {
	return &GoDetector{}
}

// Name returns the detector identifier.
func (d *GoDetector) Name() string {
	return "go"
}

// goMod represents parsed information from a go.mod file.
type goMod struct {
	Module   string
	Version  string
	Requires []string
}

// Detect analyzes the path for a Go project.
// It looks for go.mod and extracts version and dependency information.
func (d *GoDetector) Detect(path string) (*models.Detection, error) {
	goModPath := filepath.Join(path, "go.mod")

	// Check if go.mod exists
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		return nil, nil // Not a Go project
	}

	// Parse go.mod
	mod, err := d.parseGoMod(goModPath)
	if err != nil {
		return nil, err
	}

	loggingLibs, logFormat := d.detectLogging(mod)
	queueLibs, workerCmd := d.detectQueue(mod)
	uploadLibs, uploadPath := d.detectFileUpload(mod, path)

	detection := &models.Detection{
		Language:            "go",
		Version:             mod.Version,
		Services:            d.detectServices(mod),
		Confidence:          d.calculateConfidence(mod),
		LoggingLibraries:    loggingLibs,
		LogFormat:           logFormat,
		QueueLibraries:      queueLibs,
		WorkerCommand:       workerCmd,
		FileUploadLibraries: uploadLibs,
		UploadPath:          uploadPath,
	}

	return detection, nil
}

// parseGoMod reads and parses a go.mod file.
// go.mod format:
//
//	module github.com/user/project
//	go 1.21
//	require (
//	    github.com/some/dep v1.0.0
//	)
func (d *GoDetector) parseGoMod(path string) (*goMod, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	mod := &goMod{
		Version: "1.21", // Default version
	}

	// Regex patterns
	moduleRe := regexp.MustCompile(`^module\s+(.+)$`)
	goVersionRe := regexp.MustCompile(`^go\s+(\d+\.\d+)`)
	requireRe := regexp.MustCompile(`^\s*([a-zA-Z0-9._/-]+)\s+v`)

	inRequireBlock := false
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// Parse module name
		if matches := moduleRe.FindStringSubmatch(line); matches != nil {
			mod.Module = matches[1]
			continue
		}

		// Parse Go version
		if matches := goVersionRe.FindStringSubmatch(line); matches != nil {
			mod.Version = matches[1]
			continue
		}

		// Track require block
		if strings.HasPrefix(line, "require (") {
			inRequireBlock = true
			continue
		}
		if line == ")" && inRequireBlock {
			inRequireBlock = false
			continue
		}

		// Parse single-line require
		if strings.HasPrefix(line, "require ") && !strings.Contains(line, "(") {
			// Single require line: require github.com/foo/bar v1.0.0
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				mod.Requires = append(mod.Requires, parts[1])
			}
			continue
		}

		// Parse dependencies in require block
		if inRequireBlock {
			if matches := requireRe.FindStringSubmatch(line); matches != nil {
				mod.Requires = append(mod.Requires, matches[1])
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return mod, nil
}

// detectServices identifies backing services from Go dependencies.
func (d *GoDetector) detectServices(mod *goMod) []string {
	var services []string

	// PostgreSQL indicators
	postgresPatterns := []string{
		"github.com/jackc/pgx",
		"github.com/lib/pq",
		"gorm.io/driver/postgres",
		"github.com/go-pg/pg",
		"entgo.io/ent",
	}

	// Redis indicators
	redisPatterns := []string{
		"github.com/redis/go-redis",
		"github.com/go-redis/redis",
		"github.com/gomodule/redigo",
	}

	for _, req := range mod.Requires {
		// Check PostgreSQL
		for _, pattern := range postgresPatterns {
			if strings.HasPrefix(req, pattern) {
				if !containsService(services, "postgres") {
					services = append(services, "postgres")
				}
				break
			}
		}

		// Check Redis
		for _, pattern := range redisPatterns {
			if strings.HasPrefix(req, pattern) {
				if !containsService(services, "redis") {
					services = append(services, "redis")
				}
				break
			}
		}
	}

	return services
}

// containsService checks if a service is already in the list.
func containsService(services []string, service string) bool {
	for _, s := range services {
		if s == service {
			return true
		}
	}
	return false
}

// detectLogging identifies structured logging libraries from Go dependencies.
// Returns the list of detected libraries and the inferred log format.
func (d *GoDetector) detectLogging(mod *goMod) ([]string, string) {
	var libraries []string
	logFormat := "unknown"

	// Structured logging libraries that output JSON by default
	jsonLoggers := map[string]string{
		"go.uber.org/zap":                "zap",
		"github.com/rs/zerolog":          "zerolog",
		"log/slog":                       "slog",
		"golang.org/x/exp/slog":          "slog",
	}

	// Logging libraries that typically output text by default
	textLoggers := map[string]string{
		"github.com/sirupsen/logrus":     "logrus",
		"github.com/apex/log":            "apex-log",
		"github.com/inconshreveable/log15": "log15",
		"github.com/go-kit/log":          "go-kit-log",
		"github.com/hashicorp/go-hclog":  "hclog",
	}

	for _, req := range mod.Requires {
		// Check JSON loggers first
		for pattern, name := range jsonLoggers {
			if strings.HasPrefix(req, pattern) {
				libraries = append(libraries, name)
				logFormat = "json"
				break
			}
		}

		// Check text loggers
		for pattern, name := range textLoggers {
			if strings.HasPrefix(req, pattern) {
				libraries = append(libraries, name)
				if logFormat == "unknown" {
					logFormat = "text"
				}
				break
			}
		}
	}

	return libraries, logFormat
}

// calculateConfidence determines how confident we are in the detection.
func (d *GoDetector) calculateConfidence(mod *goMod) float64 {
	confidence := 0.6 // Base confidence for having go.mod

	// Higher confidence if module path is specified
	if mod.Module != "" {
		confidence += 0.2
	}

	// Higher confidence if explicit Go version is specified
	if mod.Version != "1.21" { // Not using default
		confidence += 0.1
	}

	// Higher confidence if dependencies exist
	if len(mod.Requires) > 0 {
		confidence += 0.1
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// GetVSCodeExtensions returns recommended VS Code extensions for Go.
func (d *GoDetector) GetVSCodeExtensions() []string {
	return []string{
		"golang.go",
	}
}

// detectQueue identifies job queue/worker libraries from Go dependencies.
// Returns the list of detected libraries and the inferred worker command.
func (d *GoDetector) detectQueue(mod *goMod) ([]string, string) {
	var libraries []string
	workerCmd := ""

	// Queue libraries that require a worker process
	queuePatterns := map[string]string{
		"github.com/hibiken/asynq":           "asynq",
		"github.com/RichardKnop/machinery":   "machinery",
		"github.com/gocraft/work":            "gocraft-work",
		"github.com/adjust/rmq":              "rmq",
		"github.com/gocelery/gocelery":       "gocelery",
	}

	for _, req := range mod.Requires {
		for pattern, name := range queuePatterns {
			if strings.HasPrefix(req, pattern) {
				libraries = append(libraries, name)
				break
			}
		}
	}

	// If queue libraries detected, set default worker command
	// Go workers typically use the same binary with a flag or subcommand
	if len(libraries) > 0 {
		// Extract binary name from module path
		binaryName := "app"
		if mod.Module != "" {
			parts := strings.Split(mod.Module, "/")
			binaryName = parts[len(parts)-1]
		}
		workerCmd = "./" + binaryName + " worker"
	}

	return libraries, workerCmd
}

// detectFileUpload identifies file upload handling from Go dependencies.
// Returns the list of detected libraries and the inferred upload path.
func (d *GoDetector) detectFileUpload(mod *goMod, projectPath string) ([]string, string) {
	var libraries []string
	uploadPath := ""

	// File upload/multipart handling patterns
	uploadPatterns := map[string]string{
		"github.com/gin-contrib/static": "gin-static",
		"github.com/h2non/filetype":     "filetype",
		"github.com/gabriel-vasile/mimetype": "mimetype",
	}

	// Web frameworks that have built-in multipart support
	webFrameworks := map[string]string{
		"github.com/gin-gonic/gin":     "gin",
		"github.com/labstack/echo":     "echo",
		"github.com/gofiber/fiber":     "fiber",
		"github.com/go-chi/chi":        "chi",
		"github.com/gorilla/mux":       "gorilla",
	}

	hasWebFramework := false

	for _, req := range mod.Requires {
		// Check explicit upload libraries
		for pattern, name := range uploadPatterns {
			if strings.HasPrefix(req, pattern) {
				libraries = append(libraries, name)
				break
			}
		}

		// Check for web frameworks (they all support multipart/form-data)
		for pattern := range webFrameworks {
			if strings.HasPrefix(req, pattern) {
				hasWebFramework = true
				break
			}
		}
	}

	// If we have a web framework, check for uploads directory as hint
	if hasWebFramework || len(libraries) > 0 {
		uploadPath = d.findUploadPath(projectPath)
		// If uploads directory exists, mark as having file upload capability
		if uploadPath != "" && len(libraries) == 0 {
			libraries = append(libraries, "multipart")
		}
	}

	return libraries, uploadPath
}

// findUploadPath attempts to find the upload directory for Go projects.
func (d *GoDetector) findUploadPath(projectPath string) string {
	// Common upload directory names
	commonDirs := []string{
		"uploads",
		"upload",
		"files",
		"static/uploads",
		"public/uploads",
		"assets/uploads",
	}

	for _, dir := range commonDirs {
		fullPath := filepath.Join(projectPath, dir)
		if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
			return dir
		}
	}

	return ""
}
