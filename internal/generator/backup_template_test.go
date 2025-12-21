package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/jpequegn/dockstart/internal/models"
)

// TestBackupDockerfileGeneration tests Dockerfile.backup template rendering
// with various database combinations as specified in issue #41.
func TestBackupDockerfileGeneration(t *testing.T) {
	tests := []struct {
		name        string
		hasPostgres bool
		hasMySQL    bool
		hasRedis    bool
		hasSQLite   bool
		wantTools   []string
		dontWant    []string
	}{
		{
			name:        "postgres only",
			hasPostgres: true,
			hasMySQL:    false,
			hasRedis:    false,
			hasSQLite:   false,
			wantTools:   []string{"postgresql16-client"},
			dontWant:    []string{"mysql-client", "redis", "sqlite"},
		},
		{
			name:        "redis only",
			hasPostgres: false,
			hasMySQL:    false,
			hasRedis:    true,
			hasSQLite:   false,
			wantTools:   []string{"redis"},
			dontWant:    []string{"postgresql16-client", "mysql-client", "sqlite"},
		},
		{
			name:        "mysql only",
			hasPostgres: false,
			hasMySQL:    true,
			hasRedis:    false,
			hasSQLite:   false,
			wantTools:   []string{"mysql-client"},
			dontWant:    []string{"postgresql16-client", "redis", "sqlite"},
		},
		{
			name:        "postgres and redis",
			hasPostgres: true,
			hasMySQL:    false,
			hasRedis:    true,
			hasSQLite:   false,
			wantTools:   []string{"postgresql16-client", "redis"},
			dontWant:    []string{"mysql-client", "sqlite"},
		},
		{
			name:        "all databases",
			hasPostgres: true,
			hasMySQL:    true,
			hasRedis:    true,
			hasSQLite:   true,
			wantTools:   []string{"postgresql16-client", "mysql-client", "redis", "sqlite"},
			dontWant:    []string{},
		},
	}

	g := NewBackupSidecarGenerator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &BackupSidecarConfig{
				HasPostgres:   tt.hasPostgres,
				HasMySQL:      tt.hasMySQL,
				HasRedis:      tt.hasRedis,
				HasSQLite:     tt.hasSQLite,
				Schedule:      "0 3 * * *",
				RetentionDays: 7,
			}

			content, err := g.GenerateDockerfile(config)
			if err != nil {
				t.Fatalf("GenerateDockerfile failed: %v", err)
			}

			dockerfile := string(content)

			// Verify correct packages are installed
			for _, tool := range tt.wantTools {
				if !strings.Contains(dockerfile, tool) {
					t.Errorf("Expected Dockerfile to contain %q", tool)
				}
			}

			// Verify unwanted packages are NOT installed
			for _, tool := range tt.dontWant {
				if strings.Contains(dockerfile, tool) {
					t.Errorf("Dockerfile should NOT contain %q", tool)
				}
			}

			// Verify essential components are always present
			essentials := []string{
				"FROM alpine:3.19",
				"supercronic",
				"ENTRYPOINT",
				"CMD",
			}
			for _, essential := range essentials {
				if !strings.Contains(dockerfile, essential) {
					t.Errorf("Dockerfile missing essential: %q", essential)
				}
			}
		})
	}
}

// TestBackupScriptGeneration tests backup script generation for each database
// as specified in issue #41.
func TestBackupScriptGeneration(t *testing.T) {
	g := NewBackupGenerator()

	tests := []struct {
		name       string
		dbType     string
		wantParts  []string
		dontWant   []string
	}{
		{
			name:   "postgres backup script has pg_dump",
			dbType: "postgres",
			wantParts: []string{
				"pg_dump",
				"DB_HOST",
				"DB_USER",
				"DB_PASSWORD",
				"gzip",
				".sql.gz",
			},
		},
		{
			name:   "mysql backup script has mysqldump",
			dbType: "mysql",
			wantParts: []string{
				"mysqldump",
				"--single-transaction",
				"gzip",
				".sql.gz",
			},
		},
		{
			name:   "redis backup script has BGSAVE",
			dbType: "redis",
			wantParts: []string{
				"redis-cli",
				"BGSAVE",
				"docker cp",
				".rdb",
			},
		},
		{
			name:   "sqlite backup script has VACUUM INTO",
			dbType: "sqlite",
			wantParts: []string{
				"VACUUM INTO",
				".db.gz",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := models.DefaultBackupConfig(tt.dbType, tt.dbType)
			config.DatabaseName = "testdb"
			config.DatabaseUser = "testuser"
			config.DatabasePassword = "testpass"

			script, err := g.GenerateBackupScript(config)
			if err != nil {
				t.Fatalf("GenerateBackupScript failed: %v", err)
			}

			scriptStr := string(script)
			for _, want := range tt.wantParts {
				if !strings.Contains(scriptStr, want) {
					t.Errorf("Script should contain %q", want)
				}
			}
		})
	}
}

// TestRestoreScriptGeneration tests restore script generation for each database.
func TestRestoreScriptGeneration(t *testing.T) {
	g := NewBackupGenerator()

	tests := []struct {
		name      string
		dbType    string
		wantParts []string
	}{
		{
			name:   "postgres restore script has psql",
			dbType: "postgres",
			wantParts: []string{
				"psql",
				"gunzip",
				"WARNING",
			},
		},
		{
			name:   "mysql restore script has mysql",
			dbType: "mysql",
			wantParts: []string{
				"mysql",
				"gunzip",
				"WARNING",
			},
		},
		{
			name:   "redis restore script has docker commands",
			dbType: "redis",
			wantParts: []string{
				"docker stop",
				"docker cp",
				"docker start",
				"WARNING",
			},
		},
		{
			name:   "sqlite restore script has gunzip",
			dbType: "sqlite",
			wantParts: []string{
				"gunzip",
				"docker stop",
				"docker start",
				"WARNING",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := models.DefaultBackupConfig(tt.dbType, tt.dbType)
			config.DatabaseName = "testdb"
			config.DatabaseUser = "testuser"
			config.DatabasePassword = "testpass"

			script, err := g.GenerateRestoreScript(config)
			if err != nil {
				t.Fatalf("GenerateRestoreScript failed: %v", err)
			}

			scriptStr := string(script)
			for _, want := range tt.wantParts {
				if !strings.Contains(scriptStr, want) {
					t.Errorf("Script should contain %q", want)
				}
			}
		})
	}
}

// TestCrontabGeneration tests crontab generation with various schedules.
func TestCrontabGeneration(t *testing.T) {
	g := NewBackupSidecarGenerator()

	tests := []struct {
		name          string
		schedule      string
		retentionDays int
		wantParts     []string
	}{
		{
			name:          "default daily at 3am",
			schedule:      "0 3 * * *",
			retentionDays: 7,
			wantParts: []string{
				"0 3 * * *",
				"/usr/local/bin/backup.sh",
			},
		},
		{
			name:          "hourly schedule",
			schedule:      "0 * * * *",
			retentionDays: 3,
			wantParts: []string{
				"0 * * * *",
				"/usr/local/bin/backup.sh",
			},
		},
		{
			name:          "every 6 hours",
			schedule:      "0 */6 * * *",
			retentionDays: 14,
			wantParts: []string{
				"0 */6 * * *",
				"/usr/local/bin/backup.sh",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &BackupSidecarConfig{
				Schedule:      tt.schedule,
				RetentionDays: tt.retentionDays,
			}

			content, err := g.GenerateCrontab(config)
			if err != nil {
				t.Fatalf("GenerateCrontab failed: %v", err)
			}

			crontab := string(content)
			for _, want := range tt.wantParts {
				if !strings.Contains(crontab, want) {
					t.Errorf("Crontab should contain %q", want)
				}
			}
		})
	}
}

// TestShellScriptsAreValid validates that generated shell scripts have valid syntax.
func TestShellScriptsAreValid(t *testing.T) {
	// Skip if bash is not available
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available, skipping shell validation")
	}

	g := NewBackupSidecarGenerator()

	// Test backup.sh syntax
	t.Run("backup.sh syntax", func(t *testing.T) {
		config := &BackupSidecarConfig{
			HasPostgres:   true,
			HasMySQL:      true,
			HasRedis:      true,
			Schedule:      "0 3 * * *",
			RetentionDays: 7,
		}

		content, err := g.GenerateBackupScript(config)
		if err != nil {
			t.Fatalf("GenerateBackupScript failed: %v", err)
		}

		// Write to temp file and validate syntax
		tmpFile, err := os.CreateTemp("", "backup-*.sh")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.Write(content); err != nil {
			t.Fatalf("Failed to write script: %v", err)
		}
		tmpFile.Close()

		// Validate bash syntax with -n flag
		cmd := exec.Command("bash", "-n", tmpFile.Name())
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("Invalid bash syntax: %s\n%s", err, output)
		}
	})

	// Test entrypoint.sh syntax
	t.Run("entrypoint.sh syntax", func(t *testing.T) {
		config := &BackupSidecarConfig{
			HasPostgres:   true,
			HasRedis:      true,
			Schedule:      "0 3 * * *",
			RetentionDays: 7,
		}

		content, err := g.GenerateEntrypoint(config)
		if err != nil {
			t.Fatalf("GenerateEntrypoint failed: %v", err)
		}

		tmpFile, err := os.CreateTemp("", "entrypoint-*.sh")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.Write(content); err != nil {
			t.Fatalf("Failed to write script: %v", err)
		}
		tmpFile.Close()

		cmd := exec.Command("bash", "-n", tmpFile.Name())
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("Invalid bash syntax: %s\n%s", err, output)
		}
	})
}

// TestDockerfileHasValidSyntax checks basic Dockerfile syntax validity.
func TestDockerfileHasValidSyntax(t *testing.T) {
	g := NewBackupSidecarGenerator()

	config := &BackupSidecarConfig{
		HasPostgres:   true,
		HasMySQL:      true,
		HasRedis:      true,
		HasSQLite:     true,
		Schedule:      "0 3 * * *",
		RetentionDays: 7,
	}

	content, err := g.GenerateDockerfile(config)
	if err != nil {
		t.Fatalf("GenerateDockerfile failed: %v", err)
	}

	dockerfile := string(content)

	// Check essential Dockerfile instructions are present
	requiredInstructions := []string{
		"FROM ",
		"RUN ",
		"COPY ",
		"VOLUME ",
		"HEALTHCHECK ",
		"ENTRYPOINT ",
		"CMD ",
	}

	for _, instruction := range requiredInstructions {
		if !strings.Contains(dockerfile, instruction) {
			t.Errorf("Dockerfile missing instruction: %s", instruction)
		}
	}

	// Verify FROM is the first non-comment instruction
	lines := strings.Split(dockerfile, "\n")
	foundFrom := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "FROM ") {
			foundFrom = true
			break
		} else {
			t.Error("First instruction should be FROM")
			break
		}
	}
	if !foundFrom {
		t.Error("Dockerfile missing FROM instruction")
	}

	// Verify no syntax errors in RUN commands (basic check)
	runPattern := regexp.MustCompile(`RUN\s+(.+)`)
	matches := runPattern.FindAllStringSubmatch(dockerfile, -1)
	for _, match := range matches {
		if len(match) > 1 {
			// Check for unclosed quotes
			cmd := match[1]
			if strings.Count(cmd, "'")%2 != 0 || strings.Count(cmd, "\"")%2 != 0 {
				t.Errorf("RUN command has unclosed quotes: %s", cmd)
			}
		}
	}
}

// TestComposeWithBackupSidecar tests that docker-compose.yml correctly includes
// backup sidecar configuration as specified in issue #41.
func TestComposeWithBackupSidecar(t *testing.T) {
	g := NewComposeGenerator()

	detection := &models.Detection{
		Language: "node",
		Version:  "20",
		Services: []string{"postgres"},
	}

	content, err := g.GenerateContent(detection, "myapp")
	if err != nil {
		t.Fatalf("GenerateContent failed: %v", err)
	}

	yaml := string(content)

	// Verify backup service is added
	if !strings.Contains(yaml, "db-backup:") {
		t.Error("Compose should contain db-backup service")
	}

	// Verify backups volume is added
	if !strings.Contains(yaml, "backups:") {
		t.Error("Compose should contain backups volume")
	}

	// Verify depends_on is correct
	if !strings.Contains(yaml, "depends_on:") {
		t.Error("Compose should contain depends_on")
	}

	// Verify backup service references Dockerfile.backup
	if !strings.Contains(yaml, "Dockerfile.backup") {
		t.Error("Compose should reference Dockerfile.backup")
	}

	// Verify environment variables are set
	expectedEnvVars := []string{
		"DB_HOST=postgres",
		"RETENTION_DAYS",
	}
	for _, env := range expectedEnvVars {
		if !strings.Contains(yaml, env) {
			t.Errorf("Compose should contain env var: %s", env)
		}
	}
}

// TestEndToEndBackupGeneration tests the complete generation flow.
func TestEndToEndBackupGeneration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dockstart-e2e-backup-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	detection := &models.Detection{
		Language: "node",
		Version:  "20",
		Services: []string{"postgres", "redis"},
	}

	// Generate compose file
	composeGen := NewComposeGenerator()
	if err := composeGen.Generate(detection, tmpDir, "e2e-test"); err != nil {
		t.Fatalf("Compose generate failed: %v", err)
	}

	// Generate backup sidecar files
	backupGen := NewBackupSidecarGenerator()
	if err := backupGen.Generate(detection, tmpDir, "e2e-test"); err != nil {
		t.Fatalf("Backup sidecar generate failed: %v", err)
	}

	// Verify all expected files exist
	expectedFiles := []string{
		".devcontainer/docker-compose.yml",
		".devcontainer/Dockerfile.backup",
		".devcontainer/crontab",
		".devcontainer/entrypoint.sh",
		".devcontainer/scripts/backup.sh",
		".devcontainer/scripts/backup-postgres.sh",
		".devcontainer/scripts/backup-redis.sh",
		".devcontainer/scripts/restore-postgres.sh",
		".devcontainer/scripts/restore-redis.sh",
		".devcontainer/backups/.gitkeep",
	}

	for _, file := range expectedFiles {
		filePath := filepath.Join(tmpDir, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Expected file to exist: %s", file)
		}
	}

	// Verify compose file contains backup service
	composeContent, err := os.ReadFile(filepath.Join(tmpDir, ".devcontainer/docker-compose.yml"))
	if err != nil {
		t.Fatalf("Failed to read docker-compose.yml: %v", err)
	}
	if !strings.Contains(string(composeContent), "db-backup:") {
		t.Error("Compose file should contain db-backup service")
	}

	// Verify Dockerfile.backup has correct database clients
	dockerfileContent, err := os.ReadFile(filepath.Join(tmpDir, ".devcontainer/Dockerfile.backup"))
	if err != nil {
		t.Fatalf("Failed to read Dockerfile.backup: %v", err)
	}
	dockerfileStr := string(dockerfileContent)
	if !strings.Contains(dockerfileStr, "postgresql16-client") {
		t.Error("Dockerfile.backup should contain postgresql16-client")
	}
	if !strings.Contains(dockerfileStr, "redis") {
		t.Error("Dockerfile.backup should contain redis")
	}

	// Verify scripts are executable
	scriptsDir := filepath.Join(tmpDir, ".devcontainer/scripts")
	entries, err := os.ReadDir(scriptsDir)
	if err != nil {
		t.Fatalf("Failed to read scripts dir: %v", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sh") {
			info, err := entry.Info()
			if err != nil {
				t.Fatalf("Failed to get file info: %v", err)
			}
			if info.Mode()&0111 == 0 {
				t.Errorf("Script %s should be executable", entry.Name())
			}
		}
	}
}

// TestBackupWithAllDatabaseCombinations tests all possible database combinations.
func TestBackupWithAllDatabaseCombinations(t *testing.T) {
	databases := []string{"postgres", "mysql", "redis"}

	// Generate all possible combinations (2^3 = 8, minus empty set = 7)
	for i := 1; i < 8; i++ {
		var services []string
		if i&1 != 0 {
			services = append(services, "postgres")
		}
		if i&2 != 0 {
			services = append(services, "mysql")
		}
		if i&4 != 0 {
			services = append(services, "redis")
		}

		testName := strings.Join(services, "+")
		t.Run(testName, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "dockstart-combo-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			detection := &models.Detection{
				Language: "node",
				Version:  "20",
				Services: services,
			}

			backupGen := NewBackupSidecarGenerator()
			if err := backupGen.Generate(detection, tmpDir, "combo-test"); err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			// Verify Dockerfile.backup exists
			dockerfilePath := filepath.Join(tmpDir, ".devcontainer/Dockerfile.backup")
			content, err := os.ReadFile(dockerfilePath)
			if err != nil {
				t.Fatalf("Failed to read Dockerfile.backup: %v", err)
			}

			dockerfileStr := string(content)

			// Verify correct database clients are installed
			for _, db := range databases {
				hasDB := false
				for _, s := range services {
					if s == db {
						hasDB = true
						break
					}
				}

				var clientName string
				switch db {
				case "postgres":
					clientName = "postgresql16-client"
				case "mysql":
					clientName = "mysql-client"
				case "redis":
					clientName = "redis"
				}

				if hasDB {
					if !strings.Contains(dockerfileStr, clientName) {
						t.Errorf("Dockerfile should contain %s for %s", clientName, db)
					}
				}
			}

			// Verify backup scripts exist for each database
			for _, db := range services {
				scriptPath := filepath.Join(tmpDir, ".devcontainer/scripts", "backup-"+db+".sh")
				if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
					t.Errorf("Expected backup script for %s", db)
				}
			}
		})
	}
}
