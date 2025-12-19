package generator

import (
	"strings"
	"testing"

	"github.com/jpequegn/dockstart/internal/models"
	"gopkg.in/yaml.v3"
)

// TestWorkerSidecar tests worker sidecar generation in docker-compose.yml.
func TestWorkerSidecar(t *testing.T) {
	tests := []struct {
		name        string
		detection   *models.Detection
		projectName string
		wantParts   []string
		dontWant    []string
	}{
		{
			name: "node with bull generates worker",
			detection: &models.Detection{
				Language:       "node",
				Version:        "20",
				Services:       []string{"redis"},
				QueueLibraries: []string{"bull"},
				WorkerCommand:  "npm run worker",
			},
			projectName: "node-bull-app",
			wantParts: []string{
				"worker:",
				"command: npm run worker",
				"depends_on:",
				"- app",
				"- redis",
				"WORKER_CONCURRENCY=2",
				"NODE_ENV=development",
				"REDIS_URL=redis://redis:6379",
				"restart: unless-stopped",
			},
			dontWant: []string{
				"fluent-bit:",
			},
		},
		{
			name: "python with celery generates worker",
			detection: &models.Detection{
				Language:       "python",
				Version:        "3.11",
				Services:       []string{"redis", "postgres"},
				QueueLibraries: []string{"celery"},
				WorkerCommand:  "celery -A myapp worker",
			},
			projectName: "python-celery-app",
			wantParts: []string{
				"worker:",
				"command: celery -A myapp worker",
				"depends_on:",
				"- app",
				"- redis",
				"- postgres",
				"DATABASE_URL=postgres://postgres:postgres@postgres:5432/python-celery-app_dev",
				"restart: unless-stopped",
			},
			dontWant: []string{
				"fluent-bit:",
			},
		},
		{
			name: "go with asynq generates worker",
			detection: &models.Detection{
				Language:       "go",
				Version:        "1.21",
				Services:       []string{"redis"},
				QueueLibraries: []string{"asynq"},
				WorkerCommand:  "./app worker",
			},
			projectName: "go-asynq-app",
			wantParts: []string{
				"worker:",
				"command: ./app worker",
				"- app",
				"- redis",
				"restart: unless-stopped",
			},
			dontWant: []string{
				"fluent-bit:",
				"postgres:",
			},
		},
		{
			name: "rust with apalis generates worker",
			detection: &models.Detection{
				Language:       "rust",
				Version:        "1.75",
				Services:       []string{"postgres"},
				QueueLibraries: []string{"apalis"},
				WorkerCommand:  "./myworker worker",
			},
			projectName: "rust-apalis-app",
			wantParts: []string{
				"worker:",
				"command: ./myworker worker",
				"- app",
				"- postgres",
				"DATABASE_URL",
				"restart: unless-stopped",
			},
			dontWant: []string{
				"fluent-bit:",
				"redis:",
			},
		},
		{
			name: "no queue library - no worker",
			detection: &models.Detection{
				Language:       "node",
				Version:        "20",
				Services:       []string{"postgres"},
				QueueLibraries: nil,
				WorkerCommand:  "",
			},
			projectName: "node-simple-app",
			wantParts: []string{
				"app:",
				"postgres:",
			},
			dontWant: []string{
				"worker:",
				"WORKER_CONCURRENCY",
			},
		},
		{
			name: "worker with log sidecar",
			detection: &models.Detection{
				Language:         "node",
				Version:          "20",
				Services:         []string{"redis"},
				QueueLibraries:   []string{"bullmq"},
				WorkerCommand:    "npm run worker",
				LoggingLibraries: []string{"pino"},
				LogFormat:        "json",
			},
			projectName: "node-full-app",
			wantParts: []string{
				"worker:",
				"command: npm run worker",
				"fluent-bit:",
				"tag: worker.node-full-app",
				"tag: app.node-full-app",
			},
			dontWant: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewComposeGenerator()
			content, err := g.GenerateContent(tt.detection, tt.projectName)
			if err != nil {
				t.Fatalf("GenerateContent failed: %v", err)
			}

			contentStr := string(content)

			// Check expected parts are present
			for _, want := range tt.wantParts {
				if !strings.Contains(contentStr, want) {
					t.Errorf("Expected content to contain %q, but it doesn't.\nContent:\n%s", want, contentStr)
				}
			}

			// Check unwanted parts are absent
			for _, dontWant := range tt.dontWant {
				if strings.Contains(contentStr, dontWant) {
					t.Errorf("Expected content NOT to contain %q, but it does.\nContent:\n%s", dontWant, contentStr)
				}
			}
		})
	}
}

// TestWorkerSidecar_ValidYAML tests that generated compose files are valid YAML.
func TestWorkerSidecar_ValidYAML(t *testing.T) {
	tests := []struct {
		name        string
		detection   *models.Detection
		projectName string
	}{
		{
			name: "worker with redis",
			detection: &models.Detection{
				Language:       "node",
				Version:        "20",
				Services:       []string{"redis"},
				QueueLibraries: []string{"bull"},
				WorkerCommand:  "npm run worker",
			},
			projectName: "test-app",
		},
		{
			name: "worker with postgres and redis",
			detection: &models.Detection{
				Language:       "python",
				Version:        "3.11",
				Services:       []string{"postgres", "redis"},
				QueueLibraries: []string{"celery"},
				WorkerCommand:  "celery -A app worker",
			},
			projectName: "celery-app",
		},
		{
			name: "worker with log sidecar",
			detection: &models.Detection{
				Language:         "node",
				Version:          "20",
				Services:         []string{"redis"},
				QueueLibraries:   []string{"bullmq"},
				WorkerCommand:    "npm run worker",
				LoggingLibraries: []string{"pino"},
				LogFormat:        "json",
			},
			projectName: "full-stack-app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewComposeGenerator()
			content, err := g.GenerateContent(tt.detection, tt.projectName)
			if err != nil {
				t.Fatalf("GenerateContent failed: %v", err)
			}

			// Try to parse as YAML
			var parsed map[string]interface{}
			if err := yaml.Unmarshal(content, &parsed); err != nil {
				t.Errorf("Generated content is not valid YAML: %v\nContent:\n%s", err, string(content))
			}

			// Verify worker service exists in parsed YAML
			if services, ok := parsed["services"].(map[string]interface{}); ok {
				if _, hasWorker := services["worker"]; !hasWorker {
					t.Error("Expected 'worker' service in parsed YAML")
				}
			} else {
				t.Error("Expected 'services' key in parsed YAML")
			}
		})
	}
}

// TestWorkerSidecar_DependsOn tests that worker depends_on is correctly ordered.
func TestWorkerSidecar_DependsOn(t *testing.T) {
	detection := &models.Detection{
		Language:       "node",
		Version:        "20",
		Services:       []string{"postgres", "redis"},
		QueueLibraries: []string{"bull"},
		WorkerCommand:  "npm run worker",
	}

	g := NewComposeGenerator()
	content, err := g.GenerateContent(detection, "test-app")
	if err != nil {
		t.Fatalf("GenerateContent failed: %v", err)
	}

	// Parse as YAML
	var parsed map[string]interface{}
	if err := yaml.Unmarshal(content, &parsed); err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	services := parsed["services"].(map[string]interface{})
	worker := services["worker"].(map[string]interface{})
	dependsOn := worker["depends_on"].([]interface{})

	// Check that app is first in depends_on
	if len(dependsOn) < 1 || dependsOn[0] != "app" {
		t.Errorf("Expected 'app' to be first in depends_on, got %v", dependsOn)
	}

	// Check that all services are in depends_on
	expectedDeps := map[string]bool{"app": false, "postgres": false, "redis": false}
	for _, dep := range dependsOn {
		if depStr, ok := dep.(string); ok {
			expectedDeps[depStr] = true
		}
	}
	for dep, found := range expectedDeps {
		if !found {
			t.Errorf("Expected %q to be in depends_on", dep)
		}
	}
}

// TestWorkerSidecar_BuildContext tests that worker uses same Dockerfile as app.
func TestWorkerSidecar_BuildContext(t *testing.T) {
	detection := &models.Detection{
		Language:       "go",
		Version:        "1.21",
		Services:       []string{"redis"},
		QueueLibraries: []string{"asynq"},
		WorkerCommand:  "./app worker",
	}

	g := NewComposeGenerator()
	content, err := g.GenerateContent(detection, "test-app")
	if err != nil {
		t.Fatalf("GenerateContent failed: %v", err)
	}

	// Parse as YAML
	var parsed map[string]interface{}
	if err := yaml.Unmarshal(content, &parsed); err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	services := parsed["services"].(map[string]interface{})

	// Get app build config
	app := services["app"].(map[string]interface{})
	appBuild := app["build"].(map[string]interface{})

	// Get worker build config
	worker := services["worker"].(map[string]interface{})
	workerBuild := worker["build"].(map[string]interface{})

	// Verify both use same build context and dockerfile
	if appBuild["context"] != workerBuild["context"] {
		t.Errorf("Expected worker build context to match app, got app=%v worker=%v",
			appBuild["context"], workerBuild["context"])
	}
	if appBuild["dockerfile"] != workerBuild["dockerfile"] {
		t.Errorf("Expected worker dockerfile to match app, got app=%v worker=%v",
			appBuild["dockerfile"], workerBuild["dockerfile"])
	}
}
