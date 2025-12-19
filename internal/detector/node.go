package detector

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jpequegn/dockstart/internal/models"
)

// NodeDetector detects Node.js projects by analyzing package.json.
type NodeDetector struct{}

// NewNodeDetector creates a new Node.js detector.
func NewNodeDetector() *NodeDetector {
	return &NodeDetector{}
}

// Name returns the detector identifier.
func (d *NodeDetector) Name() string {
	return "node"
}

// packageJSON represents the structure of a package.json file.
// We only parse the fields we care about.
type packageJSON struct {
	Name            string            `json:"name"`
	Engines         engines           `json:"engines"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

type engines struct {
	Node string `json:"node"`
}

// Detect analyzes the path for a Node.js project.
// It looks for package.json and extracts version and service information.
func (d *NodeDetector) Detect(path string) (*models.Detection, error) {
	packagePath := filepath.Join(path, "package.json")

	// Check if package.json exists
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		return nil, nil // Not a Node.js project
	}

	// Read and parse package.json
	data, err := os.ReadFile(packagePath)
	if err != nil {
		return nil, err
	}

	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	loggingLibs, logFormat := d.detectLogging(pkg)

	detection := &models.Detection{
		Language:         "node",
		Version:          d.extractVersion(pkg),
		Services:         d.detectServices(pkg),
		Confidence:       d.calculateConfidence(pkg),
		LoggingLibraries: loggingLibs,
		LogFormat:        logFormat,
	}

	return detection, nil
}

// extractVersion extracts the Node.js version from package.json.
// Priority: engines.node > inferred from dependencies > default
func (d *NodeDetector) extractVersion(pkg packageJSON) string {
	if pkg.Engines.Node != "" {
		// Parse version from engines.node (e.g., ">=18", "^20.0.0", "20.x")
		return parseVersionConstraint(pkg.Engines.Node)
	}

	// Default to Node 20 LTS if not specified
	return "20"
}

// parseVersionConstraint extracts the major version from a semver constraint.
// Examples: ">=18" -> "18", "^20.0.0" -> "20", "20.x" -> "20"
func parseVersionConstraint(constraint string) string {
	// Match the first number in the constraint
	re := regexp.MustCompile(`\d+`)
	match := re.FindString(constraint)
	if match != "" {
		return match
	}
	return "20" // Default
}

// detectServices identifies backing services from dependencies.
func (d *NodeDetector) detectServices(pkg packageJSON) []string {
	var services []string

	// Merge all dependencies for checking
	allDeps := make(map[string]string)
	for k, v := range pkg.Dependencies {
		allDeps[k] = v
	}
	for k, v := range pkg.DevDependencies {
		allDeps[k] = v
	}

	// PostgreSQL indicators
	postgresPackages := []string{"pg", "postgres", "postgresql", "prisma", "@prisma/client", "typeorm", "sequelize", "knex"}
	if hasAnyDep(allDeps, postgresPackages) {
		services = append(services, "postgres")
	}

	// Redis indicators
	redisPackages := []string{"redis", "ioredis", "@redis/client", "bull", "bullmq"}
	if hasAnyDep(allDeps, redisPackages) {
		services = append(services, "redis")
	}

	return services
}

// hasAnyDep checks if any of the given package names exist in dependencies.
func hasAnyDep(deps map[string]string, packages []string) bool {
	for _, pkg := range packages {
		if _, exists := deps[pkg]; exists {
			return true
		}
	}
	return false
}

// detectLogging identifies structured logging libraries from dependencies.
// Returns the list of detected libraries and the inferred log format.
func (d *NodeDetector) detectLogging(pkg packageJSON) ([]string, string) {
	var libraries []string
	logFormat := "unknown"

	// Merge all dependencies for checking
	allDeps := make(map[string]string)
	for k, v := range pkg.Dependencies {
		allDeps[k] = v
	}
	for k, v := range pkg.DevDependencies {
		allDeps[k] = v
	}

	// Structured logging libraries that output JSON by default
	jsonLoggers := map[string]string{
		"pino":      "pino",
		"bunyan":    "bunyan",
		"roarr":     "roarr",
		"bole":      "bole",
	}

	// Logging libraries that can be configured for JSON
	configurableLoggers := map[string]string{
		"winston":     "winston",
		"log4js":      "log4js",
		"loglevel":    "loglevel",
		"signale":     "signale",
	}

	// HTTP request loggers (often paired with other loggers)
	requestLoggers := map[string]string{
		"morgan":      "morgan",
		"express-winston": "express-winston",
	}

	// Check JSON loggers first (they set format to json)
	for dep, name := range jsonLoggers {
		if _, exists := allDeps[dep]; exists {
			libraries = append(libraries, name)
			logFormat = "json"
		}
	}

	// Check configurable loggers
	for dep, name := range configurableLoggers {
		if _, exists := allDeps[dep]; exists {
			libraries = append(libraries, name)
			if logFormat == "unknown" {
				logFormat = "text" // Default for configurable loggers
			}
		}
	}

	// Check request loggers
	for dep, name := range requestLoggers {
		if _, exists := allDeps[dep]; exists {
			libraries = append(libraries, name)
		}
	}

	return libraries, logFormat
}

// calculateConfidence determines how confident we are in the detection.
func (d *NodeDetector) calculateConfidence(pkg packageJSON) float64 {
	confidence := 0.5 // Base confidence for having package.json

	// Higher confidence if engines.node is specified
	if pkg.Engines.Node != "" {
		confidence += 0.3
	}

	// Higher confidence if name is specified
	if pkg.Name != "" {
		confidence += 0.1
	}

	// Higher confidence if dependencies exist
	if len(pkg.Dependencies) > 0 || len(pkg.DevDependencies) > 0 {
		confidence += 0.1
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// GetVSCodeExtensions returns recommended VS Code extensions for Node.js.
func (d *NodeDetector) GetVSCodeExtensions(pkg packageJSON) []string {
	extensions := []string{
		"dbaeumer.vscode-eslint",
	}

	// Check for TypeScript
	allDeps := make(map[string]string)
	for k, v := range pkg.Dependencies {
		allDeps[k] = v
	}
	for k, v := range pkg.DevDependencies {
		allDeps[k] = v
	}

	if _, hasTS := allDeps["typescript"]; hasTS {
		// TypeScript support is built into VS Code, no extra extension needed
	}

	// Check for Prettier
	if _, hasPrettier := allDeps["prettier"]; hasPrettier {
		extensions = append(extensions, "esbenp.prettier-vscode")
	}

	// Check for Prisma
	for dep := range allDeps {
		if strings.Contains(dep, "prisma") {
			extensions = append(extensions, "Prisma.prisma")
			break
		}
	}

	return extensions
}
