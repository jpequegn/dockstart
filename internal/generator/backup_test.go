package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jpequegn/dockstart/internal/models"
)

func TestBackupGenerator_GenerateBackupScript(t *testing.T) {
	tests := []struct {
		name     string
		config   *models.BackupConfig
		wantParts []string
	}{
		{
			name: "postgres backup script",
			config: &models.BackupConfig{
				DatabaseType:     "postgres",
				ContainerName:    "postgres",
				DatabaseHost:     "postgres",
				DatabaseName:     "mydb",
				DatabaseUser:     "myuser",
				DatabasePassword: "secret",
				RetentionDays:    7,
			},
			wantParts: []string{
				"#!/bin/sh",
				"PostgreSQL Backup Script",
				"pg_dump",
				"--no-owner",
				"--clean",
				"--if-exists",
				"gzip",
				"postgres-",
				".sql.gz",
				"RETENTION_DAYS",
			},
		},
		{
			name: "mysql backup script",
			config: &models.BackupConfig{
				DatabaseType:     "mysql",
				ContainerName:    "mysql",
				DatabaseHost:     "mysql",
				DatabaseName:     "mydb",
				DatabaseUser:     "root",
				DatabasePassword: "secret",
				RetentionDays:    7,
			},
			wantParts: []string{
				"#!/bin/sh",
				"MySQL/MariaDB Backup Script",
				"mysqldump",
				"--single-transaction",
				"--routines",
				"--triggers",
				"gzip",
				"mysql-",
				".sql.gz",
			},
		},
		{
			name: "redis backup script",
			config: &models.BackupConfig{
				DatabaseType:  "redis",
				ContainerName: "redis",
				DatabaseHost:  "redis",
				RetentionDays: 3,
			},
			wantParts: []string{
				"#!/bin/sh",
				"Redis Backup Script",
				"redis-cli",
				"BGSAVE",
				"docker cp",
				"redis-",
				".rdb.gz",
			},
		},
		{
			name: "sqlite backup script",
			config: &models.BackupConfig{
				DatabaseType:  "sqlite",
				ContainerName: "app-db",
				DatabasePath:  "/data/app.db",
				AppContainer:  "app",
				RetentionDays: 7,
				StopContainer: true,
			},
			wantParts: []string{
				"#!/bin/sh",
				"SQLite Backup Script",
				"VACUUM INTO",
				"docker stop",
				"docker start",
				"app-db-",
				".db.gz",
			},
		},
	}

	g := NewBackupGenerator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := g.GenerateBackupScript(tt.config)
			if err != nil {
				t.Fatalf("GenerateBackupScript failed: %v", err)
			}

			contentStr := string(content)
			for _, want := range tt.wantParts {
				if !strings.Contains(contentStr, want) {
					t.Errorf("Expected content to contain %q\nGot:\n%s", want, contentStr)
				}
			}
		})
	}
}

func TestBackupGenerator_GenerateRestoreScript(t *testing.T) {
	tests := []struct {
		name      string
		config    *models.BackupConfig
		wantParts []string
	}{
		{
			name: "postgres restore script",
			config: &models.BackupConfig{
				DatabaseType:     "postgres",
				ContainerName:    "postgres",
				DatabaseHost:     "postgres",
				DatabaseName:     "mydb",
				DatabaseUser:     "myuser",
				DatabasePassword: "secret",
			},
			wantParts: []string{
				"#!/bin/sh",
				"PostgreSQL Restore Script",
				"psql",
				"gunzip -c",
				"WARNING",
			},
		},
		{
			name: "mysql restore script",
			config: &models.BackupConfig{
				DatabaseType:     "mysql",
				ContainerName:    "mysql",
				DatabaseHost:     "mysql",
				DatabaseName:     "mydb",
				DatabaseUser:     "root",
				DatabasePassword: "secret",
			},
			wantParts: []string{
				"#!/bin/sh",
				"MySQL/MariaDB Restore Script",
				"mysql",
				"gunzip -c",
				"WARNING",
			},
		},
		{
			name: "redis restore script",
			config: &models.BackupConfig{
				DatabaseType:  "redis",
				ContainerName: "redis",
				DatabaseHost:  "redis",
			},
			wantParts: []string{
				"#!/bin/sh",
				"Redis Restore Script",
				"docker stop",
				"docker cp",
				"docker start",
				"WARNING",
			},
		},
		{
			name: "sqlite restore script",
			config: &models.BackupConfig{
				DatabaseType:  "sqlite",
				ContainerName: "app-db",
				DatabasePath:  "/data/app.db",
				AppContainer:  "app",
			},
			wantParts: []string{
				"#!/bin/sh",
				"SQLite Restore Script",
				"docker stop",
				"docker start",
				"gunzip -c",
				"WARNING",
			},
		},
	}

	g := NewBackupGenerator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := g.GenerateRestoreScript(tt.config)
			if err != nil {
				t.Fatalf("GenerateRestoreScript failed: %v", err)
			}

			contentStr := string(content)
			for _, want := range tt.wantParts {
				if !strings.Contains(contentStr, want) {
					t.Errorf("Expected content to contain %q\nGot:\n%s", want, contentStr)
				}
			}
		})
	}
}

func TestBackupGenerator_Generate(t *testing.T) {
	g := NewBackupGenerator()

	tmpDir, err := os.MkdirTemp("", "dockstart-backup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &models.BackupConfig{
		DatabaseType:     "postgres",
		ContainerName:    "postgres",
		DatabaseHost:     "postgres",
		DatabaseName:     "mydb",
		DatabaseUser:     "myuser",
		DatabasePassword: "secret",
		RetentionDays:    7,
	}

	if err := g.Generate(config, tmpDir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check backup script exists
	backupPath := filepath.Join(tmpDir, "scripts", "backup-postgres.sh")
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("Expected backup script at %s", backupPath)
	}

	// Check restore script exists
	restorePath := filepath.Join(tmpDir, "scripts", "restore-postgres.sh")
	if _, err := os.Stat(restorePath); os.IsNotExist(err) {
		t.Errorf("Expected restore script at %s", restorePath)
	}

	// Check scripts are executable
	backupInfo, _ := os.Stat(backupPath)
	if backupInfo.Mode()&0111 == 0 {
		t.Errorf("Expected backup script to be executable")
	}
}

func TestDefaultBackupConfig(t *testing.T) {
	tests := []struct {
		dbType        string
		containerName string
		wantSchedule  string
		wantRetention int
		wantStop      bool
	}{
		{"postgres", "postgres", "0 3 * * *", 7, false},
		{"mysql", "mysql", "0 3 * * *", 7, false},
		{"redis", "redis", "0 3 * * *", 7, false},
		{"sqlite", "app-db", "0 3 * * *", 7, true},
	}

	for _, tt := range tests {
		t.Run(tt.dbType, func(t *testing.T) {
			config := models.DefaultBackupConfig(tt.dbType, tt.containerName)

			if config.Schedule != tt.wantSchedule {
				t.Errorf("Schedule = %q, want %q", config.Schedule, tt.wantSchedule)
			}
			if config.RetentionDays != tt.wantRetention {
				t.Errorf("RetentionDays = %d, want %d", config.RetentionDays, tt.wantRetention)
			}
			if config.StopContainer != tt.wantStop {
				t.Errorf("StopContainer = %v, want %v", config.StopContainer, tt.wantStop)
			}
		})
	}
}

func TestBackupConfig_GetBackupExtension(t *testing.T) {
	tests := []struct {
		dbType string
		want   string
	}{
		{"postgres", "sql.gz"},
		{"mysql", "sql.gz"},
		{"mariadb", "sql.gz"},
		{"redis", "rdb.gz"},
		{"sqlite", "db.gz"},
		{"unknown", "backup.gz"},
	}

	for _, tt := range tests {
		t.Run(tt.dbType, func(t *testing.T) {
			config := &models.BackupConfig{DatabaseType: tt.dbType}
			got := config.GetBackupExtension()
			if got != tt.want {
				t.Errorf("GetBackupExtension() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBackupConfig_NeedsDockerSocket(t *testing.T) {
	tests := []struct {
		name   string
		config *models.BackupConfig
		want   bool
	}{
		{
			name:   "postgres does not need docker socket",
			config: &models.BackupConfig{DatabaseType: "postgres"},
			want:   false,
		},
		{
			name:   "mysql does not need docker socket",
			config: &models.BackupConfig{DatabaseType: "mysql"},
			want:   false,
		},
		{
			name:   "redis needs docker socket for docker cp",
			config: &models.BackupConfig{DatabaseType: "redis"},
			want:   true,
		},
		{
			name:   "sqlite with stop needs docker socket",
			config: &models.BackupConfig{DatabaseType: "sqlite", StopContainer: true},
			want:   true,
		},
		{
			name:   "sqlite without stop does not need docker socket",
			config: &models.BackupConfig{DatabaseType: "sqlite", StopContainer: false},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.NeedsDockerSocket()
			if got != tt.want {
				t.Errorf("NeedsDockerSocket() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSupportedDatabaseTypes(t *testing.T) {
	types := SupportedDatabaseTypes()

	expected := []string{"postgres", "mysql", "redis", "sqlite"}
	if len(types) != len(expected) {
		t.Errorf("SupportedDatabaseTypes() returned %d types, want %d", len(types), len(expected))
	}

	for _, e := range expected {
		found := false
		for _, got := range types {
			if got == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected %q to be in SupportedDatabaseTypes()", e)
		}
	}
}

func TestIsSupported(t *testing.T) {
	tests := []struct {
		dbType string
		want   bool
	}{
		{"postgres", true},
		{"mysql", true},
		{"redis", true},
		{"sqlite", true},
		{"mongodb", false},
		{"cassandra", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.dbType, func(t *testing.T) {
			got := IsSupported(tt.dbType)
			if got != tt.want {
				t.Errorf("IsSupported(%q) = %v, want %v", tt.dbType, got, tt.want)
			}
		})
	}
}
