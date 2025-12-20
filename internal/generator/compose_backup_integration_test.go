package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jpequegn/dockstart/internal/models"
)

// TestComposeGenerator_BackupSidecar_Integration tests the full integration
// of backup sidecar generation with docker-compose.yml.
func TestComposeGenerator_BackupSidecar_Integration(t *testing.T) {
	tests := []struct {
		name           string
		detection      *models.Detection
		projectName    string
		wantBackup     bool
		wantInYAML     []string
		dontWant       []string
		wantInEnv      []string
		wantVolumes    []string
		wantDependsOn  []string
	}{
		{
			name: "postgres project - full backup sidecar",
			detection: &models.Detection{
				Language:   "node",
				Version:    "20",
				Services:   []string{"postgres"},
				Confidence: 1.0,
			},
			projectName: "myapp",
			wantBackup:  true,
			wantInYAML: []string{
				"db-backup:",
				"Dockerfile.backup",
				"./backups:/backup",
			},
			wantInEnv: []string{
				"DB_HOST=postgres",
				"DB_USER=postgres",
				"DB_PASSWORD=postgres",
				"DB_NAME=myapp_dev",
				"RETENTION_DAYS=7",
			},
			wantVolumes:   []string{"backups:"},
			wantDependsOn: []string{"- postgres"},
		},
		{
			name: "redis project - backup with docker socket",
			detection: &models.Detection{
				Language:   "python",
				Version:    "3.11",
				Services:   []string{"redis"},
				Confidence: 1.0,
			},
			projectName: "cache-app",
			wantBackup:  true,
			wantInYAML: []string{
				"db-backup:",
				"/var/run/docker.sock:/var/run/docker.sock:ro",
			},
			wantInEnv: []string{
				"REDIS_HOST=redis",
				"REDIS_PORT=6379",
			},
			wantVolumes:   []string{"backups:"},
			wantDependsOn: []string{"- redis"},
		},
		{
			name: "postgres + redis - multi-database backup",
			detection: &models.Detection{
				Language:   "go",
				Version:    "1.23",
				Services:   []string{"postgres", "redis"},
				Confidence: 1.0,
			},
			projectName: "fullstack",
			wantBackup:  true,
			wantInYAML: []string{
				"db-backup:",
				"Dockerfile.backup",
				"/var/run/docker.sock:/var/run/docker.sock:ro",
			},
			wantInEnv: []string{
				"DB_HOST=postgres",
				"DB_NAME=fullstack_dev",
				"REDIS_HOST=redis",
				"RETENTION_DAYS=7",
			},
			wantVolumes:   []string{"backups:", "postgres-data:", "redis-data:"},
			wantDependsOn: []string{"- postgres", "- redis"},
		},
		{
			name: "no database - no backup sidecar",
			detection: &models.Detection{
				Language:   "node",
				Version:    "20",
				Services:   []string{},
				Confidence: 1.0,
			},
			projectName: "static-app",
			wantBackup:  false,
			dontWant: []string{
				"db-backup:",
				"Dockerfile.backup",
				"backups:",
				"RETENTION_DAYS",
			},
		},
		{
			name: "mysql project - mysql backup config",
			detection: &models.Detection{
				Language:   "rust",
				Version:    "1.75",
				Services:   []string{"mysql"},
				Confidence: 1.0,
			},
			projectName: "rust-api",
			wantBackup:  true,
			wantInYAML: []string{
				"db-backup:",
			},
			wantInEnv: []string{
				"DB_HOST=mysql",
				"DB_USER=root",
				"DB_PASSWORD=mysql",
				"DB_NAME=rust-api_dev",
			},
			dontWant: []string{
				"/var/run/docker.sock",
			},
		},
	}

	gen := NewComposeGenerator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := gen.GenerateContent(tt.detection, tt.projectName)
			if err != nil {
				t.Fatalf("GenerateContent() error = %v", err)
			}

			yaml := string(content)

			// Check if backup sidecar presence matches expectation
			hasBackup := strings.Contains(yaml, "db-backup:")
			if hasBackup != tt.wantBackup {
				t.Errorf("Backup sidecar presence = %v, want %v", hasBackup, tt.wantBackup)
			}

			// Check required strings
			for _, want := range tt.wantInYAML {
				if !strings.Contains(yaml, want) {
					t.Errorf("YAML should contain %q", want)
				}
			}

			// Check environment variables
			for _, env := range tt.wantInEnv {
				if !strings.Contains(yaml, env) {
					t.Errorf("YAML should contain env var %q", env)
				}
			}

			// Check volumes
			for _, vol := range tt.wantVolumes {
				if !strings.Contains(yaml, vol) {
					t.Errorf("YAML should contain volume %q", vol)
				}
			}

			// Check depends_on
			for _, dep := range tt.wantDependsOn {
				if !strings.Contains(yaml, dep) {
					t.Errorf("YAML should contain dependency %q", dep)
				}
			}

			// Check unwanted strings
			for _, dontWant := range tt.dontWant {
				if strings.Contains(yaml, dontWant) {
					t.Errorf("YAML should NOT contain %q", dontWant)
				}
			}
		})
	}
}

// TestBackupSidecar_FileGeneration_Integration tests that all backup files
// are generated correctly when Generate is called.
func TestBackupSidecar_FileGeneration_Integration(t *testing.T) {
	tests := []struct {
		name        string
		detection   *models.Detection
		projectName string
		wantFiles   []string
		dontWant    []string
	}{
		{
			name: "postgres project generates all backup files",
			detection: &models.Detection{
				Language: "node",
				Version:  "20",
				Services: []string{"postgres"},
			},
			projectName: "pg-project",
			wantFiles: []string{
				".devcontainer/Dockerfile.backup",
				".devcontainer/crontab",
				".devcontainer/entrypoint.sh",
				".devcontainer/scripts/backup.sh",
				".devcontainer/scripts/backup-postgres.sh",
				".devcontainer/scripts/restore-postgres.sh",
				".devcontainer/backups/.gitkeep",
			},
			dontWant: []string{
				".devcontainer/scripts/backup-mysql.sh",
				".devcontainer/scripts/backup-redis.sh",
			},
		},
		{
			name: "redis project generates redis backup files",
			detection: &models.Detection{
				Language: "python",
				Version:  "3.11",
				Services: []string{"redis"},
			},
			projectName: "redis-project",
			wantFiles: []string{
				".devcontainer/Dockerfile.backup",
				".devcontainer/scripts/backup-redis.sh",
				".devcontainer/scripts/restore-redis.sh",
			},
			dontWant: []string{
				".devcontainer/scripts/backup-postgres.sh",
				".devcontainer/scripts/backup-mysql.sh",
			},
		},
		{
			name: "multi-db project generates all db backup files",
			detection: &models.Detection{
				Language: "go",
				Version:  "1.23",
				Services: []string{"postgres", "redis"},
			},
			projectName: "multi-db",
			wantFiles: []string{
				".devcontainer/Dockerfile.backup",
				".devcontainer/scripts/backup-postgres.sh",
				".devcontainer/scripts/backup-redis.sh",
				".devcontainer/scripts/restore-postgres.sh",
				".devcontainer/scripts/restore-redis.sh",
			},
		},
		{
			name: "no database - no backup files",
			detection: &models.Detection{
				Language: "node",
				Version:  "20",
				Services: []string{},
			},
			projectName: "no-db",
			wantFiles:   []string{},
			dontWant: []string{
				".devcontainer/Dockerfile.backup",
				".devcontainer/scripts/backup.sh",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir, err := os.MkdirTemp("", "dockstart-backup-integration-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Generate backup sidecar files
			backupGen := NewBackupSidecarGenerator()
			err = backupGen.Generate(tt.detection, tmpDir, tt.projectName)
			if err != nil {
				t.Fatalf("BackupSidecarGenerator.Generate() error = %v", err)
			}

			// Check expected files exist
			for _, wantFile := range tt.wantFiles {
				filePath := filepath.Join(tmpDir, wantFile)
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Errorf("Expected file %s to exist", wantFile)
				}
			}

			// Check unwanted files don't exist
			for _, dontWant := range tt.dontWant {
				filePath := filepath.Join(tmpDir, dontWant)
				if _, err := os.Stat(filePath); !os.IsNotExist(err) {
					t.Errorf("File %s should NOT exist", dontWant)
				}
			}
		})
	}
}

// TestBackupSidecar_DockerfileContent_Integration tests that Dockerfile.backup
// contains correct database clients based on detection.
func TestBackupSidecar_DockerfileContent_Integration(t *testing.T) {
	tests := []struct {
		name       string
		detection  *models.Detection
		wantParts  []string
		dontWant   []string
	}{
		{
			name: "postgres only - pg client installed",
			detection: &models.Detection{
				Language: "node",
				Version:  "20",
				Services: []string{"postgres"},
			},
			wantParts: []string{
				"FROM alpine:3.19",
				"postgresql16-client",
				"supercronic",
			},
			dontWant: []string{
				"mysql-client",
				"redis",
			},
		},
		{
			name: "all databases - all clients installed",
			detection: &models.Detection{
				Language: "go",
				Version:  "1.23",
				Services: []string{"postgres", "mysql", "redis"},
			},
			wantParts: []string{
				"postgresql16-client",
				"mysql-client",
				"redis",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "dockstart-dockerfile-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			backupGen := NewBackupSidecarGenerator()
			err = backupGen.Generate(tt.detection, tmpDir, "test-project")
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			dockerfilePath := filepath.Join(tmpDir, ".devcontainer", "Dockerfile.backup")
			content, err := os.ReadFile(dockerfilePath)
			if err != nil {
				t.Fatalf("Failed to read Dockerfile.backup: %v", err)
			}

			dockerfile := string(content)

			for _, want := range tt.wantParts {
				if !strings.Contains(dockerfile, want) {
					t.Errorf("Dockerfile should contain %q", want)
				}
			}

			for _, dontWant := range tt.dontWant {
				if strings.Contains(dockerfile, dontWant) {
					t.Errorf("Dockerfile should NOT contain %q", dontWant)
				}
			}
		})
	}
}

// TestBackupSidecar_WithWorkerSidecar_Integration tests backup sidecar
// works correctly when combined with worker sidecar.
func TestBackupSidecar_WithWorkerSidecar_Integration(t *testing.T) {
	gen := NewComposeGenerator()

	detection := &models.Detection{
		Language:       "node",
		Version:        "20",
		Services:       []string{"postgres", "redis"},
		QueueLibraries: []string{"bullmq"},
		WorkerCommand:  "npm run worker",
	}

	content, err := gen.GenerateContent(detection, "fullstack-app")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	yaml := string(content)

	// Verify both sidecars are present
	expectedParts := []string{
		// Worker sidecar
		"worker:",
		"npm run worker",
		"WORKER_CONCURRENCY=2",
		// Backup sidecar
		"db-backup:",
		"Dockerfile.backup",
		"RETENTION_DAYS=7",
		// Database services
		"postgres:",
		"redis:",
		// Volumes
		"postgres-data:",
		"redis-data:",
		"backups:",
	}

	for _, part := range expectedParts {
		if !strings.Contains(yaml, part) {
			t.Errorf("YAML should contain %q", part)
		}
	}
}

// TestBackupSidecar_WithLogSidecar_Integration tests backup sidecar
// works correctly when combined with log sidecar.
func TestBackupSidecar_WithLogSidecar_Integration(t *testing.T) {
	gen := NewComposeGenerator()

	detection := &models.Detection{
		Language:         "go",
		Version:          "1.23",
		Services:         []string{"postgres"},
		LoggingLibraries: []string{"zap"},
		LogFormat:        "json",
	}

	content, err := gen.GenerateContent(detection, "observability-app")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	yaml := string(content)

	// Verify all sidecars are present
	expectedParts := []string{
		// Log sidecar
		"fluent-bit:",
		"fluent/fluent-bit:latest",
		"driver: fluentd",
		// Backup sidecar
		"db-backup:",
		"Dockerfile.backup",
		"DB_HOST=postgres",
		// Database service
		"postgres:",
		// Volumes
		"postgres-data:",
		"fluent-bit-logs:",
		"backups:",
	}

	for _, part := range expectedParts {
		if !strings.Contains(yaml, part) {
			t.Errorf("YAML should contain %q", part)
		}
	}
}

// TestBackupSidecar_AllSidecars_Integration tests that all three sidecars
// (log, worker, backup) can coexist in the same compose file.
func TestBackupSidecar_AllSidecars_Integration(t *testing.T) {
	gen := NewComposeGenerator()

	detection := &models.Detection{
		Language:         "node",
		Version:          "20",
		Services:         []string{"postgres", "redis"},
		LoggingLibraries: []string{"pino"},
		LogFormat:        "json",
		QueueLibraries:   []string{"bullmq"},
		WorkerCommand:    "npm run worker",
	}

	content, err := gen.GenerateContent(detection, "mega-app")
	if err != nil {
		t.Fatalf("GenerateContent() error = %v", err)
	}

	yaml := string(content)

	// Count sidecars
	wantServices := []string{
		"app:",
		"worker:",
		"postgres:",
		"redis:",
		"fluent-bit:",
		"db-backup:",
	}

	for _, svc := range wantServices {
		if !strings.Contains(yaml, svc) {
			t.Errorf("YAML should contain service %q", svc)
		}
	}

	// Verify volumes for all sidecars
	wantVolumes := []string{
		"postgres-data:",
		"redis-data:",
		"fluent-bit-logs:",
		"backups:",
	}

	for _, vol := range wantVolumes {
		if !strings.Contains(yaml, vol) {
			t.Errorf("YAML should contain volume %q", vol)
		}
	}
}

// TestBuildComposeConfig_BackupSidecar tests that buildConfig correctly
// configures the backup sidecar based on detection.
func TestBuildComposeConfig_BackupSidecar(t *testing.T) {
	gen := NewComposeGenerator()

	tests := []struct {
		name             string
		detection        *models.Detection
		wantEnabled      bool
		wantHasPostgres  bool
		wantHasMySQL     bool
		wantHasRedis     bool
		wantDockerSocket bool
	}{
		{
			name: "postgres only",
			detection: &models.Detection{
				Language: "node",
				Services: []string{"postgres"},
			},
			wantEnabled:      true,
			wantHasPostgres:  true,
			wantHasMySQL:     false,
			wantHasRedis:     false,
			wantDockerSocket: false,
		},
		{
			name: "redis only - needs docker socket",
			detection: &models.Detection{
				Language: "python",
				Services: []string{"redis"},
			},
			wantEnabled:      true,
			wantHasPostgres:  false,
			wantHasMySQL:     false,
			wantHasRedis:     true,
			wantDockerSocket: true,
		},
		{
			name: "postgres + redis",
			detection: &models.Detection{
				Language: "go",
				Services: []string{"postgres", "redis"},
			},
			wantEnabled:      true,
			wantHasPostgres:  true,
			wantHasMySQL:     false,
			wantHasRedis:     true,
			wantDockerSocket: true,
		},
		{
			name: "no databases",
			detection: &models.Detection{
				Language: "node",
				Services: []string{},
			},
			wantEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := gen.buildConfig(tt.detection, "test-app")

			if config.BackupSidecar.Enabled != tt.wantEnabled {
				t.Errorf("BackupSidecar.Enabled = %v, want %v",
					config.BackupSidecar.Enabled, tt.wantEnabled)
			}

			if tt.wantEnabled {
				if config.BackupSidecar.HasPostgres != tt.wantHasPostgres {
					t.Errorf("BackupSidecar.HasPostgres = %v, want %v",
						config.BackupSidecar.HasPostgres, tt.wantHasPostgres)
				}
				if config.BackupSidecar.HasMySQL != tt.wantHasMySQL {
					t.Errorf("BackupSidecar.HasMySQL = %v, want %v",
						config.BackupSidecar.HasMySQL, tt.wantHasMySQL)
				}
				if config.BackupSidecar.HasRedis != tt.wantHasRedis {
					t.Errorf("BackupSidecar.HasRedis = %v, want %v",
						config.BackupSidecar.HasRedis, tt.wantHasRedis)
				}
				if config.BackupSidecar.NeedsDockerSocket != tt.wantDockerSocket {
					t.Errorf("BackupSidecar.NeedsDockerSocket = %v, want %v",
						config.BackupSidecar.NeedsDockerSocket, tt.wantDockerSocket)
				}
				if config.BackupSidecar.Schedule != "0 3 * * *" {
					t.Errorf("BackupSidecar.Schedule = %v, want default",
						config.BackupSidecar.Schedule)
				}
				if config.BackupSidecar.RetentionDays != 7 {
					t.Errorf("BackupSidecar.RetentionDays = %v, want 7",
						config.BackupSidecar.RetentionDays)
				}
			}
		})
	}
}
