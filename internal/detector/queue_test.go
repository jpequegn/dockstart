package detector

import (
	"os"
	"path/filepath"
	"testing"
)

// TestQueueDetection_Node tests queue library detection for Node.js projects.
func TestQueueDetection_Node(t *testing.T) {
	tests := []struct {
		name           string
		packageJSON    string
		wantLibraries  []string
		wantWorkerCmd  string
	}{
		{
			name: "bull queue",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"bull": "^4.0.0", "express": "^4.18.0"}
			}`,
			wantLibraries: []string{"bull"},
			wantWorkerCmd: "node worker.js",
		},
		{
			name: "bullmq queue",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"bullmq": "^4.0.0"}
			}`,
			wantLibraries: []string{"bullmq"},
			wantWorkerCmd: "node worker.js",
		},
		{
			name: "bull with worker script",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"bull": "^4.0.0"},
				"scripts": {"worker": "node src/worker.js"}
			}`,
			wantLibraries: []string{"bull"},
			wantWorkerCmd: "npm run worker",
		},
		{
			name: "bullmq with start:worker script",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"bullmq": "^4.0.0"},
				"scripts": {"start:worker": "node worker.js"}
			}`,
			wantLibraries: []string{"bullmq"},
			wantWorkerCmd: "npm run start:worker",
		},
		{
			name: "bee-queue",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"bee-queue": "^1.4.0"}
			}`,
			wantLibraries: []string{"bee-queue"},
			wantWorkerCmd: "node worker.js",
		},
		{
			name: "agenda",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"agenda": "^5.0.0"}
			}`,
			wantLibraries: []string{"agenda"},
			wantWorkerCmd: "node worker.js",
		},
		{
			name: "pg-boss",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"pg-boss": "^9.0.0"}
			}`,
			wantLibraries: []string{"pg-boss"},
			wantWorkerCmd: "node worker.js",
		},
		{
			name: "no queue library",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"express": "^4.18.0"}
			}`,
			wantLibraries: nil,
			wantWorkerCmd: "",
		},
		{
			name: "custom worker script name",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"bull": "^4.0.0"},
				"scripts": {"my-worker-process": "node worker.js"}
			}`,
			wantLibraries: []string{"bull"},
			wantWorkerCmd: "npm run my-worker-process",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "dockstart-queue-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(tt.packageJSON), 0644); err != nil {
				t.Fatalf("Failed to write package.json: %v", err)
			}

			d := NewNodeDetector()
			detection, err := d.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detection failed: %v", err)
			}
			if detection == nil {
				t.Fatal("Expected detection, got nil")
			}

			// Check queue libraries
			if len(tt.wantLibraries) == 0 {
				if len(detection.QueueLibraries) != 0 {
					t.Errorf("QueueLibraries = %v, want empty", detection.QueueLibraries)
				}
			} else {
				for _, lib := range tt.wantLibraries {
					if !detection.HasQueueLibrary(lib) {
						t.Errorf("Expected queue library %q to be detected, got %v", lib, detection.QueueLibraries)
					}
				}
			}

			// Check worker command
			if detection.WorkerCommand != tt.wantWorkerCmd {
				t.Errorf("WorkerCommand = %q, want %q", detection.WorkerCommand, tt.wantWorkerCmd)
			}

			// Check NeedsWorker
			if len(tt.wantLibraries) > 0 && !detection.NeedsWorker() {
				t.Error("Expected NeedsWorker() to return true")
			}
			if len(tt.wantLibraries) == 0 && detection.NeedsWorker() {
				t.Error("Expected NeedsWorker() to return false")
			}
		})
	}
}

// TestQueueDetection_Go tests queue library detection for Go projects.
func TestQueueDetection_Go(t *testing.T) {
	tests := []struct {
		name          string
		goMod         string
		wantLibraries []string
		wantWorkerCmd string
	}{
		{
			name: "asynq",
			goMod: `module github.com/user/myworker

go 1.21

require github.com/hibiken/asynq v0.24.0
`,
			wantLibraries: []string{"asynq"},
			wantWorkerCmd: "./myworker worker",
		},
		{
			name: "machinery",
			goMod: `module github.com/user/taskrunner

go 1.21

require github.com/RichardKnop/machinery/v2 v2.0.0
`,
			wantLibraries: []string{"machinery"},
			wantWorkerCmd: "./taskrunner worker",
		},
		{
			name: "gocraft-work",
			goMod: `module github.com/user/jobprocessor

go 1.21

require github.com/gocraft/work v0.5.1
`,
			wantLibraries: []string{"gocraft-work"},
			wantWorkerCmd: "./jobprocessor worker",
		},
		{
			name: "rmq",
			goMod: `module github.com/user/queueapp

go 1.21

require github.com/adjust/rmq/v5 v5.0.0
`,
			wantLibraries: []string{"rmq"},
			wantWorkerCmd: "./queueapp worker",
		},
		{
			name: "gocelery",
			goMod: `module github.com/user/celerygo

go 1.21

require github.com/gocelery/gocelery v0.5.1
`,
			wantLibraries: []string{"gocelery"},
			wantWorkerCmd: "./celerygo worker",
		},
		{
			name: "no queue library",
			goMod: `module github.com/user/webapp

go 1.21

require github.com/gin-gonic/gin v1.9.1
`,
			wantLibraries: nil,
			wantWorkerCmd: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "dockstart-queue-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(tt.goMod), 0644); err != nil {
				t.Fatalf("Failed to write go.mod: %v", err)
			}

			d := NewGoDetector()
			detection, err := d.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detection failed: %v", err)
			}
			if detection == nil {
				t.Fatal("Expected detection, got nil")
			}

			// Check queue libraries
			if len(tt.wantLibraries) == 0 {
				if len(detection.QueueLibraries) != 0 {
					t.Errorf("QueueLibraries = %v, want empty", detection.QueueLibraries)
				}
			} else {
				for _, lib := range tt.wantLibraries {
					if !detection.HasQueueLibrary(lib) {
						t.Errorf("Expected queue library %q to be detected, got %v", lib, detection.QueueLibraries)
					}
				}
			}

			// Check worker command
			if detection.WorkerCommand != tt.wantWorkerCmd {
				t.Errorf("WorkerCommand = %q, want %q", detection.WorkerCommand, tt.wantWorkerCmd)
			}
		})
	}
}

// TestQueueDetection_Python tests queue library detection for Python projects.
func TestQueueDetection_Python(t *testing.T) {
	tests := []struct {
		name           string
		pyprojectTOML  string
		wantLibraries  []string
		wantWorkerCmd  string
	}{
		{
			name: "celery",
			pyprojectTOML: `[project]
name = "myapp"
dependencies = ["celery>=5.0.0", "redis>=4.0.0"]
`,
			wantLibraries: []string{"celery"},
			wantWorkerCmd: "celery -A myapp worker",
		},
		{
			name: "rq (Redis Queue)",
			pyprojectTOML: `[project]
name = "taskapp"
dependencies = ["rq>=1.0.0"]
`,
			wantLibraries: []string{"rq"},
			wantWorkerCmd: "rq worker",
		},
		{
			name: "dramatiq",
			pyprojectTOML: `[project]
name = "dramaapp"
dependencies = ["dramatiq>=1.0.0"]
`,
			wantLibraries: []string{"dramatiq"},
			wantWorkerCmd: "dramatiq dramaapp",
		},
		{
			name: "huey",
			pyprojectTOML: `[project]
name = "hueyapp"
dependencies = ["huey>=2.0.0"]
`,
			wantLibraries: []string{"huey"},
			wantWorkerCmd: "huey_consumer hueyapp.huey",
		},
		{
			name: "arq",
			pyprojectTOML: `[project]
name = "arqapp"
dependencies = ["arq>=0.20.0"]
`,
			wantLibraries: []string{"arq"},
			wantWorkerCmd: "arq arqapp.WorkerSettings",
		},
		{
			name: "taskiq",
			pyprojectTOML: `[project]
name = "taskiqapp"
dependencies = ["taskiq>=0.5.0"]
`,
			wantLibraries: []string{"taskiq"},
			wantWorkerCmd: "taskiq worker taskiqapp:broker",
		},
		{
			name: "no queue library",
			pyprojectTOML: `[project]
name = "webapp"
dependencies = ["flask>=2.0.0"]
`,
			wantLibraries: nil,
			wantWorkerCmd: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "dockstart-queue-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if err := os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte(tt.pyprojectTOML), 0644); err != nil {
				t.Fatalf("Failed to write pyproject.toml: %v", err)
			}

			d := NewPythonDetector()
			detection, err := d.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detection failed: %v", err)
			}
			if detection == nil {
				t.Fatal("Expected detection, got nil")
			}

			// Check queue libraries
			if len(tt.wantLibraries) == 0 {
				if len(detection.QueueLibraries) != 0 {
					t.Errorf("QueueLibraries = %v, want empty", detection.QueueLibraries)
				}
			} else {
				for _, lib := range tt.wantLibraries {
					if !detection.HasQueueLibrary(lib) {
						t.Errorf("Expected queue library %q to be detected, got %v", lib, detection.QueueLibraries)
					}
				}
			}

			// Check worker command
			if detection.WorkerCommand != tt.wantWorkerCmd {
				t.Errorf("WorkerCommand = %q, want %q", detection.WorkerCommand, tt.wantWorkerCmd)
			}
		})
	}
}

// TestQueueDetection_Python_Requirements tests queue detection with requirements.txt.
func TestQueueDetection_Python_Requirements(t *testing.T) {
	tests := []struct {
		name           string
		requirements   string
		wantLibraries  []string
		wantWorkerCmd  string
	}{
		{
			name:          "celery",
			requirements:  "celery>=5.0.0\nredis>=4.0.0\n",
			wantLibraries: []string{"celery"},
			wantWorkerCmd: "celery -A app worker",
		},
		{
			name:          "rq",
			requirements:  "rq>=1.0.0\n",
			wantLibraries: []string{"rq"},
			wantWorkerCmd: "rq worker",
		},
		{
			name:          "no queue",
			requirements:  "flask>=2.0.0\n",
			wantLibraries: nil,
			wantWorkerCmd: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "dockstart-queue-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if err := os.WriteFile(filepath.Join(tmpDir, "requirements.txt"), []byte(tt.requirements), 0644); err != nil {
				t.Fatalf("Failed to write requirements.txt: %v", err)
			}

			d := NewPythonDetector()
			detection, err := d.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detection failed: %v", err)
			}
			if detection == nil {
				t.Fatal("Expected detection, got nil")
			}

			// Check queue libraries
			if len(tt.wantLibraries) == 0 {
				if len(detection.QueueLibraries) != 0 {
					t.Errorf("QueueLibraries = %v, want empty", detection.QueueLibraries)
				}
			} else {
				for _, lib := range tt.wantLibraries {
					if !detection.HasQueueLibrary(lib) {
						t.Errorf("Expected queue library %q to be detected, got %v", lib, detection.QueueLibraries)
					}
				}
			}

			// Check worker command
			if detection.WorkerCommand != tt.wantWorkerCmd {
				t.Errorf("WorkerCommand = %q, want %q", detection.WorkerCommand, tt.wantWorkerCmd)
			}
		})
	}
}

// TestQueueDetection_MultipleLibraries tests detection when multiple queue libraries are present.
func TestQueueDetection_MultipleLibraries(t *testing.T) {
	tests := []struct {
		name             string
		packageJSON      string
		wantLibraryCount int
		wantWorkerCmd    string
	}{
		{
			name: "bull and bullmq together",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"bull": "^4.0.0", "bullmq": "^4.0.0"}
			}`,
			wantLibraryCount: 2,
			wantWorkerCmd:    "node worker.js",
		},
		{
			name: "bull and agenda together",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"bull": "^4.0.0", "agenda": "^5.0.0"},
				"scripts": {"worker": "node src/worker.js"}
			}`,
			wantLibraryCount: 2,
			wantWorkerCmd:    "npm run worker",
		},
		{
			name: "three queue libraries",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"bull": "^4.0.0", "bullmq": "^4.0.0", "bee-queue": "^1.4.0"}
			}`,
			wantLibraryCount: 3,
			wantWorkerCmd:    "node worker.js",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "dockstart-queue-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(tt.packageJSON), 0644); err != nil {
				t.Fatalf("Failed to write package.json: %v", err)
			}

			d := NewNodeDetector()
			detection, err := d.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detection failed: %v", err)
			}
			if detection == nil {
				t.Fatal("Expected detection, got nil")
			}

			// Check number of queue libraries detected
			if len(detection.QueueLibraries) != tt.wantLibraryCount {
				t.Errorf("QueueLibraries count = %d, want %d (got: %v)",
					len(detection.QueueLibraries), tt.wantLibraryCount, detection.QueueLibraries)
			}

			// Check worker command
			if detection.WorkerCommand != tt.wantWorkerCmd {
				t.Errorf("WorkerCommand = %q, want %q", detection.WorkerCommand, tt.wantWorkerCmd)
			}

			// Verify NeedsWorker returns true
			if !detection.NeedsWorker() {
				t.Error("Expected NeedsWorker() to return true with multiple queue libraries")
			}
		})
	}
}

// TestQueueDetection_EdgeCases tests edge cases in queue detection.
func TestQueueDetection_EdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		packageJSON   string
		wantLibraries []string
		wantWorkerCmd string
	}{
		{
			name: "queue in devDependencies",
			packageJSON: `{
				"name": "test-app",
				"devDependencies": {"bullmq": "^4.0.0"}
			}`,
			wantLibraries: []string{"bullmq"},
			wantWorkerCmd: "node worker.js",
		},
		{
			name: "queue with special character script name",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"bull": "^4.0.0"},
				"scripts": {"worker:dev": "node src/worker.js"}
			}`,
			wantLibraries: []string{"bull"},
			wantWorkerCmd: "npm run worker:dev",
		},
		{
			name: "empty dependencies",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {}
			}`,
			wantLibraries: nil,
			wantWorkerCmd: "",
		},
		{
			name: "only unrelated dependencies",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"lodash": "^4.0.0", "axios": "^1.0.0"}
			}`,
			wantLibraries: nil,
			wantWorkerCmd: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "dockstart-queue-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(tt.packageJSON), 0644); err != nil {
				t.Fatalf("Failed to write package.json: %v", err)
			}

			d := NewNodeDetector()
			detection, err := d.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detection failed: %v", err)
			}
			if detection == nil {
				t.Fatal("Expected detection, got nil")
			}

			// Check queue libraries
			if len(tt.wantLibraries) == 0 {
				if len(detection.QueueLibraries) != 0 {
					t.Errorf("QueueLibraries = %v, want empty", detection.QueueLibraries)
				}
			} else {
				for _, lib := range tt.wantLibraries {
					if !detection.HasQueueLibrary(lib) {
						t.Errorf("Expected queue library %q, got %v", lib, detection.QueueLibraries)
					}
				}
			}

			// Check worker command
			if detection.WorkerCommand != tt.wantWorkerCmd {
				t.Errorf("WorkerCommand = %q, want %q", detection.WorkerCommand, tt.wantWorkerCmd)
			}
		})
	}
}

// TestQueueDetection_Rust tests queue library detection for Rust projects.
func TestQueueDetection_Rust(t *testing.T) {
	tests := []struct {
		name          string
		cargoTOML     string
		wantLibraries []string
		wantWorkerCmd string
	}{
		{
			name: "sidekiq",
			cargoTOML: `[package]
name = "myworker"
version = "0.1.0"
edition = "2021"

[dependencies]
sidekiq = "0.1"
`,
			wantLibraries: []string{"sidekiq"},
			wantWorkerCmd: "./myworker worker",
		},
		{
			name: "apalis",
			cargoTOML: `[package]
name = "jobrunner"
version = "0.1.0"
edition = "2021"

[dependencies]
apalis = "0.4"
`,
			wantLibraries: []string{"apalis"},
			wantWorkerCmd: "./jobrunner worker",
		},
		{
			name: "lapin (RabbitMQ)",
			cargoTOML: `[package]
name = "rabbitmqworker"
version = "0.1.0"
edition = "2021"

[dependencies]
lapin = "2.0"
`,
			wantLibraries: []string{"lapin"},
			wantWorkerCmd: "./rabbitmqworker worker",
		},
		{
			name: "faktory",
			cargoTOML: `[package]
name = "faktoryworker"
version = "0.1.0"
edition = "2021"

[dependencies]
faktory = "0.12"
`,
			wantLibraries: []string{"faktory"},
			wantWorkerCmd: "./faktoryworker worker",
		},
		{
			name: "no queue library",
			cargoTOML: `[package]
name = "webapp"
version = "0.1.0"
edition = "2021"

[dependencies]
axum = "0.7"
`,
			wantLibraries: nil,
			wantWorkerCmd: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "dockstart-queue-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if err := os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte(tt.cargoTOML), 0644); err != nil {
				t.Fatalf("Failed to write Cargo.toml: %v", err)
			}

			d := NewRustDetector()
			detection, err := d.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detection failed: %v", err)
			}
			if detection == nil {
				t.Fatal("Expected detection, got nil")
			}

			// Check queue libraries
			if len(tt.wantLibraries) == 0 {
				if len(detection.QueueLibraries) != 0 {
					t.Errorf("QueueLibraries = %v, want empty", detection.QueueLibraries)
				}
			} else {
				for _, lib := range tt.wantLibraries {
					if !detection.HasQueueLibrary(lib) {
						t.Errorf("Expected queue library %q to be detected, got %v", lib, detection.QueueLibraries)
					}
				}
			}

			// Check worker command
			if detection.WorkerCommand != tt.wantWorkerCmd {
				t.Errorf("WorkerCommand = %q, want %q", detection.WorkerCommand, tt.wantWorkerCmd)
			}
		})
	}
}
