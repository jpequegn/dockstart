package detector

import (
	"os"
	"path/filepath"
	"testing"
)

// TestServiceDetection_AllLanguages verifies that service detection works
// consistently across all language detectors.
func TestServiceDetection_AllLanguages(t *testing.T) {
	tests := []struct {
		name         string
		setupFunc    func(dir string) error
		wantPostgres bool
		wantRedis    bool
	}{
		{
			name: "node with pg",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
					"name": "test-app",
					"dependencies": {"pg": "^8.11.0"}
				}`), 0644)
			},
			wantPostgres: true,
			wantRedis:    false,
		},
		{
			name: "node with redis",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
					"name": "test-app",
					"dependencies": {"ioredis": "^5.0.0"}
				}`), 0644)
			},
			wantPostgres: false,
			wantRedis:    true,
		},
		{
			name: "node with both",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
					"name": "test-app",
					"dependencies": {
						"prisma": "^5.0.0",
						"bull": "^4.0.0"
					}
				}`), 0644)
			},
			wantPostgres: true,
			wantRedis:    true,
		},
		{
			name: "go with pgx",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "go.mod"), []byte(`
module test-app
go 1.21
require github.com/jackc/pgx/v5 v5.4.0
`), 0644)
			},
			wantPostgres: true,
			wantRedis:    false,
		},
		{
			name: "go with redis",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "go.mod"), []byte(`
module test-app
go 1.21
require github.com/redis/go-redis/v9 v9.0.0
`), 0644)
			},
			wantPostgres: false,
			wantRedis:    true,
		},
		{
			name: "go with both",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "go.mod"), []byte(`
module test-app
go 1.21
require (
	github.com/lib/pq v1.10.0
	github.com/go-redis/redis/v8 v8.11.0
)
`), 0644)
			},
			wantPostgres: true,
			wantRedis:    true,
		},
		{
			name: "python with psycopg2",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(`
[project]
name = "test-app"
dependencies = ["psycopg2-binary>=2.9.0"]
`), 0644)
			},
			wantPostgres: true,
			wantRedis:    false,
		},
		{
			name: "python with redis",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(`
[project]
name = "test-app"
dependencies = ["redis>=4.0.0"]
`), 0644)
			},
			wantPostgres: false,
			wantRedis:    true,
		},
		{
			name: "python with both",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(`
[project]
name = "test-app"
dependencies = ["asyncpg>=0.28.0", "aioredis>=2.0.0"]
`), 0644)
			},
			wantPostgres: true,
			wantRedis:    true,
		},
		{
			name: "rust with sqlx",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(`
[package]
name = "test-app"
version = "0.1.0"
edition = "2021"

[dependencies]
sqlx = { version = "0.7", features = ["postgres"] }
`), 0644)
			},
			wantPostgres: true,
			wantRedis:    false,
		},
		{
			name: "rust with redis",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(`
[package]
name = "test-app"
version = "0.1.0"
edition = "2021"

[dependencies]
redis = "0.24"
`), 0644)
			},
			wantPostgres: false,
			wantRedis:    true,
		},
		{
			name: "rust with both",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(`
[package]
name = "test-app"
version = "0.1.0"
edition = "2021"

[dependencies]
diesel = { version = "2.1", features = ["postgres"] }
deadpool-redis = "0.14"
`), 0644)
			},
			wantPostgres: true,
			wantRedis:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir, err := os.MkdirTemp("", "dockstart-service-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Setup project files
			if err := tt.setupFunc(tmpDir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			// Run detection
			registry := NewRegistry()
			detection, err := registry.DetectPrimary(tmpDir)
			if err != nil {
				t.Fatalf("Detection failed: %v", err)
			}
			if detection == nil {
				t.Fatal("Expected detection, got nil")
			}

			// Check PostgreSQL
			hasPostgres := containsService(detection.Services, "postgres")
			if hasPostgres != tt.wantPostgres {
				t.Errorf("postgres detection = %v, want %v (services: %v)",
					hasPostgres, tt.wantPostgres, detection.Services)
			}

			// Check Redis
			hasRedis := containsService(detection.Services, "redis")
			if hasRedis != tt.wantRedis {
				t.Errorf("redis detection = %v, want %v (services: %v)",
					hasRedis, tt.wantRedis, detection.Services)
			}
		})
	}
}

// TestServiceDetection_RequirementsTxt tests Python service detection from requirements.txt
func TestServiceDetection_RequirementsTxt(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dockstart-req-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create requirements.txt with services
	err = os.WriteFile(filepath.Join(tmpDir, "requirements.txt"), []byte(`
# Database
psycopg2>=2.9.0
sqlalchemy>=2.0.0

# Cache
redis>=4.0.0
celery>=5.3.0
`), 0644)
	if err != nil {
		t.Fatalf("Failed to write requirements.txt: %v", err)
	}

	registry := NewRegistry()
	detection, err := registry.DetectPrimary(tmpDir)
	if err != nil {
		t.Fatalf("Detection failed: %v", err)
	}

	if !containsService(detection.Services, "postgres") {
		t.Error("Should detect postgres from psycopg2")
	}
	if !containsService(detection.Services, "redis") {
		t.Error("Should detect redis from redis/celery packages")
	}
}

// TestServiceDetection_NoServices tests that projects without service dependencies
// return empty services list.
func TestServiceDetection_NoServices(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(dir string) error
	}{
		{
			name: "node without services",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
					"name": "test-app",
					"dependencies": {"express": "^4.18.0", "lodash": "^4.17.0"}
				}`), 0644)
			},
		},
		{
			name: "go without services",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "go.mod"), []byte(`
module test-app
go 1.21
require github.com/gin-gonic/gin v1.9.0
`), 0644)
			},
		},
		{
			name: "python without services",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(`
[project]
name = "test-app"
dependencies = ["fastapi>=0.100.0", "uvicorn>=0.23.0"]
`), 0644)
			},
		},
		{
			name: "rust without services",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(`
[package]
name = "test-app"
version = "0.1.0"
edition = "2021"

[dependencies]
actix-web = "4.0"
serde = { version = "1.0", features = ["derive"] }
`), 0644)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "dockstart-noservice-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if err := tt.setupFunc(tmpDir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			registry := NewRegistry()
			detection, err := registry.DetectPrimary(tmpDir)
			if err != nil {
				t.Fatalf("Detection failed: %v", err)
			}
			if detection == nil {
				t.Fatal("Expected detection, got nil")
			}

			if len(detection.Services) > 0 {
				t.Errorf("Expected no services, got %v", detection.Services)
			}
		})
	}
}
