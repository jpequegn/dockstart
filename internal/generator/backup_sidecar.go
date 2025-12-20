// Package generator provides code generation for devcontainer files.
package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jpequegn/dockstart/internal/models"
)

// BackupSidecarConfig holds configuration for generating backup sidecar container files.
type BackupSidecarConfig struct {
	// HasPostgres indicates if PostgreSQL backup is needed
	HasPostgres bool

	// HasMySQL indicates if MySQL backup is needed
	HasMySQL bool

	// HasRedis indicates if Redis backup is needed
	HasRedis bool

	// HasSQLite indicates if SQLite backup is needed
	HasSQLite bool

	// Schedule is the cron schedule for backups
	Schedule string

	// RetentionDays is the number of days to keep backups
	RetentionDays int

	// ProjectName is the name of the project
	ProjectName string
}

// BackupSidecarGenerator generates backup sidecar container files.
type BackupSidecarGenerator struct{}

// NewBackupSidecarGenerator creates a new backup sidecar generator.
func NewBackupSidecarGenerator() *BackupSidecarGenerator {
	return &BackupSidecarGenerator{}
}

// GenerateDockerfile generates the Dockerfile.backup content.
func (g *BackupSidecarGenerator) GenerateDockerfile(config *BackupSidecarConfig) ([]byte, error) {
	tmpl, err := loadTemplate("Dockerfile.backup.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateBackupScript generates the main backup.sh script.
func (g *BackupSidecarGenerator) GenerateBackupScript(config *BackupSidecarConfig) ([]byte, error) {
	tmpl, err := loadTemplate("backup.sh.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateCrontab generates the crontab file.
func (g *BackupSidecarGenerator) GenerateCrontab(config *BackupSidecarConfig) ([]byte, error) {
	tmpl, err := loadTemplate("crontab.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateEntrypoint generates the entrypoint.sh script.
func (g *BackupSidecarGenerator) GenerateEntrypoint(config *BackupSidecarConfig) ([]byte, error) {
	tmpl, err := loadTemplate("entrypoint.backup.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// Generate writes all backup sidecar files to the target directory.
func (g *BackupSidecarGenerator) Generate(detection *models.Detection, projectPath string, projectName string) error {
	devcontainerDir := filepath.Join(projectPath, ".devcontainer")
	scriptsDir := filepath.Join(devcontainerDir, "scripts")

	// Create directories
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		return fmt.Errorf("failed to create scripts directory: %w", err)
	}

	// Determine which databases need backup
	config := &BackupSidecarConfig{
		HasPostgres:   detection.HasService("postgres"),
		HasMySQL:      detection.HasService("mysql"),
		HasRedis:      detection.HasService("redis"),
		HasSQLite:     false, // Not implemented yet
		Schedule:      "0 3 * * *",
		RetentionDays: 7,
		ProjectName:   projectName,
	}

	// If no databases, skip backup sidecar generation
	if !config.HasPostgres && !config.HasMySQL && !config.HasRedis && !config.HasSQLite {
		return nil
	}

	// Generate Dockerfile.backup
	dockerfile, err := g.GenerateDockerfile(config)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(devcontainerDir, "Dockerfile.backup"), dockerfile, 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile.backup: %w", err)
	}

	// Generate main backup.sh script
	backupScript, err := g.GenerateBackupScript(config)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(scriptsDir, "backup.sh"), backupScript, 0755); err != nil {
		return fmt.Errorf("failed to write backup.sh: %w", err)
	}

	// Generate crontab
	crontab, err := g.GenerateCrontab(config)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(devcontainerDir, "crontab"), crontab, 0644); err != nil {
		return fmt.Errorf("failed to write crontab: %w", err)
	}

	// Generate entrypoint
	entrypoint, err := g.GenerateEntrypoint(config)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(devcontainerDir, "entrypoint.sh"), entrypoint, 0755); err != nil {
		return fmt.Errorf("failed to write entrypoint.sh: %w", err)
	}

	// Generate database-specific backup/restore scripts using BackupGenerator
	backupGen := NewBackupGenerator()

	if config.HasPostgres {
		pgConfig := models.DefaultBackupConfig("postgres", "postgres")
		pgConfig.DatabaseName = projectName + "_dev"
		pgConfig.DatabaseUser = "postgres"
		pgConfig.DatabasePassword = "postgres"
		if err := backupGen.Generate(pgConfig, devcontainerDir); err != nil {
			return fmt.Errorf("failed to generate postgres backup scripts: %w", err)
		}
	}

	if config.HasMySQL {
		mysqlConfig := models.DefaultBackupConfig("mysql", "mysql")
		mysqlConfig.DatabaseName = projectName + "_dev"
		mysqlConfig.DatabaseUser = "root"
		mysqlConfig.DatabasePassword = "mysql"
		if err := backupGen.Generate(mysqlConfig, devcontainerDir); err != nil {
			return fmt.Errorf("failed to generate mysql backup scripts: %w", err)
		}
	}

	if config.HasRedis {
		redisConfig := models.DefaultBackupConfig("redis", "redis")
		if err := backupGen.Generate(redisConfig, devcontainerDir); err != nil {
			return fmt.Errorf("failed to generate redis backup scripts: %w", err)
		}
	}

	// Create backups directory
	backupsDir := filepath.Join(devcontainerDir, "backups")
	if err := os.MkdirAll(backupsDir, 0755); err != nil {
		return fmt.Errorf("failed to create backups directory: %w", err)
	}

	// Create .gitkeep in backups directory
	gitkeep := filepath.Join(backupsDir, ".gitkeep")
	if err := os.WriteFile(gitkeep, []byte{}, 0644); err != nil {
		return fmt.Errorf("failed to write .gitkeep: %w", err)
	}

	return nil
}

// ShouldGenerate checks if backup sidecar should be generated based on detection.
func (g *BackupSidecarGenerator) ShouldGenerate(detection *models.Detection) bool {
	return detection.HasService("postgres") ||
		detection.HasService("mysql") ||
		detection.HasService("redis")
}
