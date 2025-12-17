// Package detector provides language and service detection for projects.
package detector

import (
	"github.com/jpequegn/dockstart/internal/models"
)

// Detector is the interface that all language detectors must implement.
// Each detector is responsible for identifying a specific language/runtime.
type Detector interface {
	// Name returns the detector's identifier (e.g., "node", "go", "python")
	Name() string

	// Detect analyzes the given path and returns a Detection if the language is found.
	// Returns nil if the language is not detected, or an error if something went wrong.
	Detect(path string) (*models.Detection, error)
}

// DetectorRegistry holds all registered detectors and orchestrates detection.
type DetectorRegistry struct {
	detectors []Detector
}

// NewRegistry creates a new detector registry with default detectors.
func NewRegistry() *DetectorRegistry {
	return &DetectorRegistry{
		detectors: []Detector{
			NewNodeDetector(),
			// TODO: Add more detectors (Go, Python, Rust)
		},
	}
}

// Register adds a detector to the registry.
func (r *DetectorRegistry) Register(d Detector) {
	r.detectors = append(r.detectors, d)
}

// DetectAll runs all registered detectors and returns all detections.
// Results are sorted by confidence (highest first).
func (r *DetectorRegistry) DetectAll(path string) ([]*models.Detection, error) {
	var detections []*models.Detection

	for _, detector := range r.detectors {
		detection, err := detector.Detect(path)
		if err != nil {
			// Log error but continue with other detectors
			continue
		}
		if detection != nil {
			detections = append(detections, detection)
		}
	}

	// Sort by confidence (highest first)
	sortByConfidence(detections)

	return detections, nil
}

// DetectPrimary runs all detectors and returns the most confident detection.
// Returns nil if no language is detected.
func (r *DetectorRegistry) DetectPrimary(path string) (*models.Detection, error) {
	detections, err := r.DetectAll(path)
	if err != nil {
		return nil, err
	}

	if len(detections) == 0 {
		return nil, nil
	}

	return detections[0], nil
}

// sortByConfidence sorts detections by confidence score (highest first).
func sortByConfidence(detections []*models.Detection) {
	// Simple bubble sort (good enough for small lists)
	for i := 0; i < len(detections); i++ {
		for j := i + 1; j < len(detections); j++ {
			if detections[j].Confidence > detections[i].Confidence {
				detections[i], detections[j] = detections[j], detections[i]
			}
		}
	}
}
