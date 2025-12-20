// Package generator provides code generation for devcontainer files.
package generator

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/jpequegn/dockstart/internal/models"
)

//go:embed templates/backup/*.tmpl
var backupTemplates embed.FS

// BackupGenerator generates database backup scripts.
type BackupGenerator struct {
	templates *template.Template
}

// NewBackupGenerator creates a new backup script generator.
func NewBackupGenerator() *BackupGenerator {
	tmpl := template.Must(template.ParseFS(backupTemplates, "templates/backup/*.tmpl"))
	return &BackupGenerator{templates: tmpl}
}

// GenerateBackupScript generates the backup script for the given database type.
func (g *BackupGenerator) GenerateBackupScript(config *models.BackupConfig) ([]byte, error) {
	templateName := fmt.Sprintf("%s-backup.sh.tmpl", config.DatabaseType)

	var buf bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buf, templateName, config); err != nil {
		return nil, fmt.Errorf("failed to execute backup template: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateRestoreScript generates the restore script for the given database type.
func (g *BackupGenerator) GenerateRestoreScript(config *models.BackupConfig) ([]byte, error) {
	templateName := fmt.Sprintf("%s-restore.sh.tmpl", config.DatabaseType)

	var buf bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buf, templateName, config); err != nil {
		return nil, fmt.Errorf("failed to execute restore template: %w", err)
	}

	return buf.Bytes(), nil
}

// Generate writes the backup and restore scripts to the target directory.
func (g *BackupGenerator) Generate(config *models.BackupConfig, targetDir string) error {
	scriptsDir := filepath.Join(targetDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		return fmt.Errorf("failed to create scripts directory: %w", err)
	}

	// Generate backup script
	backupContent, err := g.GenerateBackupScript(config)
	if err != nil {
		return err
	}

	backupPath := filepath.Join(scriptsDir, fmt.Sprintf("backup-%s.sh", config.DatabaseType))
	if err := os.WriteFile(backupPath, backupContent, 0755); err != nil {
		return fmt.Errorf("failed to write backup script: %w", err)
	}

	// Generate restore script
	restoreContent, err := g.GenerateRestoreScript(config)
	if err != nil {
		return err
	}

	restorePath := filepath.Join(scriptsDir, fmt.Sprintf("restore-%s.sh", config.DatabaseType))
	if err := os.WriteFile(restorePath, restoreContent, 0755); err != nil {
		return fmt.Errorf("failed to write restore script: %w", err)
	}

	return nil
}

// SupportedDatabaseTypes returns the list of supported database types.
func SupportedDatabaseTypes() []string {
	return []string{"postgres", "mysql", "redis", "sqlite"}
}

// IsSupported checks if a database type is supported for backup.
func IsSupported(dbType string) bool {
	for _, supported := range SupportedDatabaseTypes() {
		if supported == dbType {
			return true
		}
	}
	return false
}
