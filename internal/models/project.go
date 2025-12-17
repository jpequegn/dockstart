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
