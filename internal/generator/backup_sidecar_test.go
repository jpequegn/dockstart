package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jpequegn/dockstart/internal/models"
)

func TestBackupSidecarGenerator_GenerateDockerfile(t *testing.T) {
	tests := []struct {
		name      string
		config    *BackupSidecarConfig
		wantParts []string
		dontWant  []string
	}{
		{
			name: "postgres only",
			config: &BackupSidecarConfig{
				HasPostgres:   true,
				HasMySQL:      false,
				HasRedis:      false,
				Schedule:      "0 3 * * *",
				RetentionDays: 7,
			},
			wantParts: []string{
				"FROM alpine:3.19",
				"postgresql16-client",
				"supercronic",
				"backup.sh",
			},
			dontWant: []string{
				"mysql-client",
				"redis",
			},
		},
		{
			name: "mysql only",
			config: &BackupSidecarConfig{
				HasPostgres:   false,
				HasMySQL:      true,
				HasRedis:      false,
				Schedule:      "0 3 * * *",
				RetentionDays: 7,
			},
			wantParts: []string{
				"FROM alpine:3.19",
				"mysql-client",
				"supercronic",
			},
			dontWant: []string{
				"postgresql16-client",
			},
		},
		{
			name: "all databases",
			config: &BackupSidecarConfig{
				HasPostgres:   true,
				HasMySQL:      true,
				HasRedis:      true,
				HasSQLite:     true,
				Schedule:      "0 3 * * *",
				RetentionDays: 7,
			},
			wantParts: []string{
				"postgresql16-client",
				"mysql-client",
				"redis",
				"sqlite",
			},
		},
	}

	g := NewBackupSidecarGenerator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := g.GenerateDockerfile(tt.config)
			if err != nil {
				t.Fatalf("GenerateDockerfile failed: %v", err)
			}

			contentStr := string(content)

			for _, want := range tt.wantParts {
				if !strings.Contains(contentStr, want) {
					t.Errorf("Expected content to contain %q", want)
				}
			}

			for _, dontWant := range tt.dontWant {
				if strings.Contains(contentStr, dontWant) {
					t.Errorf("Expected content NOT to contain %q", dontWant)
				}
			}
		})
	}
}

func TestBackupSidecarGenerator_GenerateBackupScript(t *testing.T) {
	tests := []struct {
		name      string
		config    *BackupSidecarConfig
		wantParts []string
		dontWant  []string
	}{
		{
			name: "postgres only",
			config: &BackupSidecarConfig{
				HasPostgres: true,
				HasMySQL:    false,
				HasRedis:    false,
			},
			wantParts: []string{
				"#!/bin/bash",
				"PostgreSQL backup",
				"backup-postgres.sh",
			},
			dontWant: []string{
				"MySQL backup",
				"Redis backup",
			},
		},
		{
			name: "all databases",
			config: &BackupSidecarConfig{
				HasPostgres: true,
				HasMySQL:    true,
				HasRedis:    true,
			},
			wantParts: []string{
				"PostgreSQL backup",
				"MySQL backup",
				"Redis backup",
			},
		},
	}

	g := NewBackupSidecarGenerator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := g.GenerateBackupScript(tt.config)
			if err != nil {
				t.Fatalf("GenerateBackupScript failed: %v", err)
			}

			contentStr := string(content)

			for _, want := range tt.wantParts {
				if !strings.Contains(contentStr, want) {
					t.Errorf("Expected content to contain %q", want)
				}
			}

			for _, dontWant := range tt.dontWant {
				if strings.Contains(contentStr, dontWant) {
					t.Errorf("Expected content NOT to contain %q", dontWant)
				}
			}
		})
	}
}

func TestBackupSidecarGenerator_GenerateCrontab(t *testing.T) {
	g := NewBackupSidecarGenerator()

	config := &BackupSidecarConfig{
		Schedule:      "0 3 * * *",
		RetentionDays: 7,
	}

	content, err := g.GenerateCrontab(config)
	if err != nil {
		t.Fatalf("GenerateCrontab failed: %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "0 3 * * *") {
		t.Error("Expected crontab to contain schedule")
	}

	if !strings.Contains(contentStr, "/usr/local/bin/backup.sh") {
		t.Error("Expected crontab to contain backup.sh command")
	}
}

func TestBackupSidecarGenerator_GenerateEntrypoint(t *testing.T) {
	tests := []struct {
		name      string
		config    *BackupSidecarConfig
		wantParts []string
		dontWant  []string
	}{
		{
			name: "postgres only",
			config: &BackupSidecarConfig{
				HasPostgres: true,
				HasMySQL:    false,
				HasRedis:    false,
			},
			wantParts: []string{
				"#!/bin/bash",
				"pg_isready",
				"PostgreSQL",
			},
			dontWant: []string{
				"mysqladmin",
				"redis-cli",
			},
		},
		{
			name: "all databases",
			config: &BackupSidecarConfig{
				HasPostgres: true,
				HasMySQL:    true,
				HasRedis:    true,
			},
			wantParts: []string{
				"pg_isready",
				"mysqladmin",
				"redis-cli",
			},
		},
	}

	g := NewBackupSidecarGenerator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := g.GenerateEntrypoint(tt.config)
			if err != nil {
				t.Fatalf("GenerateEntrypoint failed: %v", err)
			}

			contentStr := string(content)

			for _, want := range tt.wantParts {
				if !strings.Contains(contentStr, want) {
					t.Errorf("Expected content to contain %q", want)
				}
			}

			for _, dontWant := range tt.dontWant {
				if strings.Contains(contentStr, dontWant) {
					t.Errorf("Expected content NOT to contain %q", dontWant)
				}
			}
		})
	}
}

func TestBackupSidecarGenerator_Generate(t *testing.T) {
	g := NewBackupSidecarGenerator()

	tmpDir, err := os.MkdirTemp("", "dockstart-backup-sidecar-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	detection := &models.Detection{
		Language: "node",
		Version:  "20",
		Services: []string{"postgres", "redis"},
	}

	if err := g.Generate(detection, tmpDir, "myproject"); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check Dockerfile.backup exists
	dockerfilePath := filepath.Join(tmpDir, ".devcontainer", "Dockerfile.backup")
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		t.Errorf("Expected Dockerfile.backup at %s", dockerfilePath)
	}

	// Check crontab exists
	crontabPath := filepath.Join(tmpDir, ".devcontainer", "crontab")
	if _, err := os.Stat(crontabPath); os.IsNotExist(err) {
		t.Errorf("Expected crontab at %s", crontabPath)
	}

	// Check entrypoint exists
	entrypointPath := filepath.Join(tmpDir, ".devcontainer", "entrypoint.sh")
	if _, err := os.Stat(entrypointPath); os.IsNotExist(err) {
		t.Errorf("Expected entrypoint.sh at %s", entrypointPath)
	}

	// Check backup.sh exists
	backupScriptPath := filepath.Join(tmpDir, ".devcontainer", "scripts", "backup.sh")
	if _, err := os.Stat(backupScriptPath); os.IsNotExist(err) {
		t.Errorf("Expected backup.sh at %s", backupScriptPath)
	}

	// Check postgres scripts exist
	pgBackupPath := filepath.Join(tmpDir, ".devcontainer", "scripts", "backup-postgres.sh")
	if _, err := os.Stat(pgBackupPath); os.IsNotExist(err) {
		t.Errorf("Expected backup-postgres.sh at %s", pgBackupPath)
	}

	// Check redis scripts exist
	redisBackupPath := filepath.Join(tmpDir, ".devcontainer", "scripts", "backup-redis.sh")
	if _, err := os.Stat(redisBackupPath); os.IsNotExist(err) {
		t.Errorf("Expected backup-redis.sh at %s", redisBackupPath)
	}

	// Check backups directory exists
	backupsDir := filepath.Join(tmpDir, ".devcontainer", "backups")
	if _, err := os.Stat(backupsDir); os.IsNotExist(err) {
		t.Errorf("Expected backups directory at %s", backupsDir)
	}
}

func TestBackupSidecarGenerator_Generate_NoDatabases(t *testing.T) {
	g := NewBackupSidecarGenerator()

	tmpDir, err := os.MkdirTemp("", "dockstart-backup-sidecar-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	detection := &models.Detection{
		Language: "node",
		Version:  "20",
		Services: []string{}, // No databases
	}

	if err := g.Generate(detection, tmpDir, "myproject"); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should not create Dockerfile.backup when no databases
	dockerfilePath := filepath.Join(tmpDir, ".devcontainer", "Dockerfile.backup")
	if _, err := os.Stat(dockerfilePath); !os.IsNotExist(err) {
		t.Errorf("Should not create Dockerfile.backup when no databases")
	}
}

func TestBackupSidecarGenerator_ShouldGenerate(t *testing.T) {
	g := NewBackupSidecarGenerator()

	tests := []struct {
		name      string
		detection *models.Detection
		want      bool
	}{
		{
			name: "with postgres",
			detection: &models.Detection{
				Services: []string{"postgres"},
			},
			want: true,
		},
		{
			name: "with mysql",
			detection: &models.Detection{
				Services: []string{"mysql"},
			},
			want: true,
		},
		{
			name: "with redis",
			detection: &models.Detection{
				Services: []string{"redis"},
			},
			want: true,
		},
		{
			name: "with no databases",
			detection: &models.Detection{
				Services: []string{},
			},
			want: false,
		},
		{
			name: "with non-database service",
			detection: &models.Detection{
				Services: []string{"elasticsearch"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := g.ShouldGenerate(tt.detection)
			if got != tt.want {
				t.Errorf("ShouldGenerate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComposeConfig_BackupSidecar(t *testing.T) {
	tests := []struct {
		name       string
		detection  *models.Detection
		wantBackup bool
	}{
		{
			name: "postgres enables backup",
			detection: &models.Detection{
				Language: "node",
				Services: []string{"postgres"},
			},
			wantBackup: true,
		},
		{
			name: "redis enables backup",
			detection: &models.Detection{
				Language: "node",
				Services: []string{"redis"},
			},
			wantBackup: true,
		},
		{
			name: "no database - no backup",
			detection: &models.Detection{
				Language: "node",
				Services: []string{},
			},
			wantBackup: false,
		},
	}

	g := NewComposeGenerator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := g.GenerateContent(tt.detection, "test-project")
			if err != nil {
				t.Fatalf("GenerateContent failed: %v", err)
			}

			contentStr := string(content)
			hasBackup := strings.Contains(contentStr, "db-backup:")

			if hasBackup != tt.wantBackup {
				t.Errorf("Backup sidecar presence = %v, want %v", hasBackup, tt.wantBackup)
			}
		})
	}
}
