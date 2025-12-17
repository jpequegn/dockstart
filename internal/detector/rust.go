package detector

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/jpequegn/dockstart/internal/models"
)

// RustDetector detects Rust projects by analyzing Cargo.toml.
type RustDetector struct{}

// NewRustDetector creates a new Rust detector.
func NewRustDetector() *RustDetector {
	return &RustDetector{}
}

// Name returns the detector identifier.
func (d *RustDetector) Name() string {
	return "rust"
}

// cargoTOML represents the structure of a Cargo.toml file.
// We only parse the fields we care about.
type cargoTOML struct {
	Package struct {
		Name        string `toml:"name"`
		Version     string `toml:"version"`
		Edition     string `toml:"edition"`
		RustVersion string `toml:"rust-version"`
	} `toml:"package"`
	Dependencies    map[string]interface{} `toml:"dependencies"`
	DevDependencies map[string]interface{} `toml:"dev-dependencies"`
}

// Detect analyzes the path for a Rust project.
// It looks for Cargo.toml and extracts version and service information.
func (d *RustDetector) Detect(path string) (*models.Detection, error) {
	cargoPath := filepath.Join(path, "Cargo.toml")

	// Check if Cargo.toml exists
	if _, err := os.Stat(cargoPath); os.IsNotExist(err) {
		return nil, nil // Not a Rust project
	}

	// Parse Cargo.toml
	var config cargoTOML
	if _, err := toml.DecodeFile(cargoPath, &config); err != nil {
		return nil, err
	}

	// Collect all dependencies
	deps := d.collectDependencies(config)

	detection := &models.Detection{
		Language:   "rust",
		Version:    d.extractVersion(config),
		Services:   d.detectServices(deps),
		Confidence: d.calculateConfidence(config),
	}

	return detection, nil
}

// collectDependencies extracts all dependency names from Cargo.toml.
func (d *RustDetector) collectDependencies(config cargoTOML) []string {
	var deps []string

	// Add regular dependencies
	for dep := range config.Dependencies {
		deps = append(deps, dep)
	}

	// Add dev dependencies
	for dep := range config.DevDependencies {
		deps = append(deps, dep)
	}

	return deps
}

// extractVersion extracts the Rust version from Cargo.toml.
// Priority: rust-version > edition mapping > default
func (d *RustDetector) extractVersion(config cargoTOML) string {
	// Try rust-version field first (MSRV - Minimum Supported Rust Version)
	if config.Package.RustVersion != "" {
		return config.Package.RustVersion
	}

	// Map edition to approximate Rust version
	switch config.Package.Edition {
	case "2024":
		return "1.85" // Rust 2024 edition
	case "2021":
		return "1.75" // Stable Rust 2021 edition
	case "2018":
		return "1.31" // Rust 2018 edition
	case "2015":
		return "1.0" // Rust 2015 edition
	}

	// Default to current stable
	return "1.75"
}

// detectServices identifies backing services from Rust dependencies.
func (d *RustDetector) detectServices(deps []string) []string {
	var services []string

	// PostgreSQL indicators
	postgresPackages := []string{
		"sqlx",
		"diesel",
		"tokio-postgres",
		"postgres",
		"deadpool-postgres",
		"sea-orm",
		"cornucopia",
	}

	// Redis indicators
	redisPackages := []string{
		"redis",
		"deadpool-redis",
		"fred",
		"bb8-redis",
	}

	for _, dep := range deps {
		depLower := strings.ToLower(dep)

		// Check PostgreSQL
		for _, pkg := range postgresPackages {
			if depLower == pkg {
				if !containsService(services, "postgres") {
					services = append(services, "postgres")
				}
				break
			}
		}

		// Check Redis
		for _, pkg := range redisPackages {
			if depLower == pkg {
				if !containsService(services, "redis") {
					services = append(services, "redis")
				}
				break
			}
		}
	}

	return services
}

// calculateConfidence determines how confident we are in the detection.
func (d *RustDetector) calculateConfidence(config cargoTOML) float64 {
	confidence := 0.7 // Base confidence for having Cargo.toml

	// Higher confidence if package name is specified
	if config.Package.Name != "" {
		confidence += 0.1
	}

	// Higher confidence if edition is specified
	if config.Package.Edition != "" {
		confidence += 0.1
	}

	// Higher confidence if dependencies exist
	if len(config.Dependencies) > 0 {
		confidence += 0.1
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// GetVSCodeExtensions returns recommended VS Code extensions for Rust.
func (d *RustDetector) GetVSCodeExtensions() []string {
	return []string{
		"rust-lang.rust-analyzer",
	}
}
