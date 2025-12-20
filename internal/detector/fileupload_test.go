package detector

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFileUploadDetection_Node tests file upload library detection for Node.js projects.
func TestFileUploadDetection_Node(t *testing.T) {
	tests := []struct {
		name           string
		packageJSON    string
		createUploads  bool
		wantLibraries  []string
		wantUploadPath string
	}{
		{
			name: "multer",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"multer": "^1.4.5", "express": "^4.18.0"}
			}`,
			wantLibraries:  []string{"multer"},
			wantUploadPath: "",
		},
		{
			name: "multer with uploads directory",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"multer": "^1.4.5"}
			}`,
			createUploads:  true,
			wantLibraries:  []string{"multer"},
			wantUploadPath: "uploads",
		},
		{
			name: "formidable",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"formidable": "^3.5.0"}
			}`,
			wantLibraries:  []string{"formidable"},
			wantUploadPath: "",
		},
		{
			name: "busboy",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"busboy": "^1.6.0"}
			}`,
			wantLibraries:  []string{"busboy"},
			wantUploadPath: "",
		},
		{
			name: "express-fileupload",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"express-fileupload": "^1.4.0"}
			}`,
			wantLibraries:  []string{"express-fileupload"},
			wantUploadPath: "",
		},
		{
			name: "multiparty",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"multiparty": "^4.2.0"}
			}`,
			wantLibraries:  []string{"multiparty"},
			wantUploadPath: "",
		},
		{
			name: "no upload library",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"express": "^4.18.0"}
			}`,
			wantLibraries:  nil,
			wantUploadPath: "",
		},
		{
			name: "multiple upload libraries",
			packageJSON: `{
				"name": "test-app",
				"dependencies": {"multer": "^1.4.5", "formidable": "^3.5.0"}
			}`,
			wantLibraries:  []string{"multer", "formidable"},
			wantUploadPath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "dockstart-upload-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(tt.packageJSON), 0644); err != nil {
				t.Fatalf("Failed to write package.json: %v", err)
			}

			// Create uploads directory if needed
			if tt.createUploads {
				if err := os.Mkdir(filepath.Join(tmpDir, "uploads"), 0755); err != nil {
					t.Fatalf("Failed to create uploads dir: %v", err)
				}
			}

			d := NewNodeDetector()
			detection, err := d.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detection failed: %v", err)
			}
			if detection == nil {
				t.Fatal("Expected detection, got nil")
			}

			// Check file upload libraries
			if len(tt.wantLibraries) == 0 {
				if len(detection.FileUploadLibraries) != 0 {
					t.Errorf("FileUploadLibraries = %v, want empty", detection.FileUploadLibraries)
				}
			} else {
				for _, lib := range tt.wantLibraries {
					if !detection.HasFileUploadLibrary(lib) {
						t.Errorf("Expected file upload library %q to be detected, got %v", lib, detection.FileUploadLibraries)
					}
				}
			}

			// Check upload path
			if detection.UploadPath != tt.wantUploadPath {
				t.Errorf("UploadPath = %q, want %q", detection.UploadPath, tt.wantUploadPath)
			}

			// Check NeedsFileProcessor
			if len(tt.wantLibraries) > 0 && !detection.NeedsFileProcessor() {
				t.Error("Expected NeedsFileProcessor() to return true")
			}
			if len(tt.wantLibraries) == 0 && detection.NeedsFileProcessor() {
				t.Error("Expected NeedsFileProcessor() to return false")
			}
		})
	}
}

// TestFileUploadDetection_Go tests file upload library detection for Go projects.
func TestFileUploadDetection_Go(t *testing.T) {
	tests := []struct {
		name           string
		goMod          string
		createUploads  bool
		wantLibraries  []string
		wantUploadPath string
	}{
		{
			name: "gin-static",
			goMod: `module github.com/user/myapp

go 1.21

require github.com/gin-contrib/static v1.1.0
`,
			wantLibraries:  []string{"gin-static"},
			wantUploadPath: "",
		},
		{
			name: "filetype detection library",
			goMod: `module github.com/user/myapp

go 1.21

require github.com/h2non/filetype v1.1.3
`,
			wantLibraries:  []string{"filetype"},
			wantUploadPath: "",
		},
		{
			name: "mimetype detection library",
			goMod: `module github.com/user/myapp

go 1.21

require github.com/gabriel-vasile/mimetype v1.4.2
`,
			wantLibraries:  []string{"mimetype"},
			wantUploadPath: "",
		},
		{
			name: "gin framework with uploads dir",
			goMod: `module github.com/user/myapp

go 1.21

require github.com/gin-gonic/gin v1.9.1
`,
			createUploads:  true,
			wantLibraries:  []string{"multipart"},
			wantUploadPath: "uploads",
		},
		{
			name: "echo framework with uploads dir",
			goMod: `module github.com/user/myapp

go 1.21

require github.com/labstack/echo/v4 v4.11.0
`,
			createUploads:  true,
			wantLibraries:  []string{"multipart"},
			wantUploadPath: "uploads",
		},
		{
			name: "fiber framework with uploads dir",
			goMod: `module github.com/user/myapp

go 1.21

require github.com/gofiber/fiber/v2 v2.50.0
`,
			createUploads:  true,
			wantLibraries:  []string{"multipart"},
			wantUploadPath: "uploads",
		},
		{
			name: "web framework without uploads dir",
			goMod: `module github.com/user/myapp

go 1.21

require github.com/gin-gonic/gin v1.9.1
`,
			createUploads:  false,
			wantLibraries:  nil,
			wantUploadPath: "",
		},
		{
			name: "no web framework",
			goMod: `module github.com/user/myapp

go 1.21

require github.com/spf13/cobra v1.7.0
`,
			wantLibraries:  nil,
			wantUploadPath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "dockstart-upload-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(tt.goMod), 0644); err != nil {
				t.Fatalf("Failed to write go.mod: %v", err)
			}

			// Create uploads directory if needed
			if tt.createUploads {
				if err := os.Mkdir(filepath.Join(tmpDir, "uploads"), 0755); err != nil {
					t.Fatalf("Failed to create uploads dir: %v", err)
				}
			}

			d := NewGoDetector()
			detection, err := d.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detection failed: %v", err)
			}
			if detection == nil {
				t.Fatal("Expected detection, got nil")
			}

			// Check file upload libraries
			if len(tt.wantLibraries) == 0 {
				if len(detection.FileUploadLibraries) != 0 {
					t.Errorf("FileUploadLibraries = %v, want empty", detection.FileUploadLibraries)
				}
			} else {
				for _, lib := range tt.wantLibraries {
					if !detection.HasFileUploadLibrary(lib) {
						t.Errorf("Expected file upload library %q to be detected, got %v", lib, detection.FileUploadLibraries)
					}
				}
			}

			// Check upload path
			if detection.UploadPath != tt.wantUploadPath {
				t.Errorf("UploadPath = %q, want %q", detection.UploadPath, tt.wantUploadPath)
			}
		})
	}
}

// TestFileUploadDetection_Python tests file upload library detection for Python projects.
func TestFileUploadDetection_Python(t *testing.T) {
	tests := []struct {
		name           string
		pyprojectTOML  string
		createUploads  bool
		wantLibraries  []string
		wantUploadPath string
	}{
		{
			name: "python-multipart",
			pyprojectTOML: `[project]
name = "myapp"
dependencies = ["python-multipart>=0.0.5", "fastapi>=0.100.0"]
`,
			wantLibraries:  []string{"python-multipart"},
			wantUploadPath: "",
		},
		{
			name: "aiofiles",
			pyprojectTOML: `[project]
name = "myapp"
dependencies = ["aiofiles>=23.0.0"]
`,
			wantLibraries:  []string{"aiofiles"},
			wantUploadPath: "",
		},
		{
			name: "starlette",
			pyprojectTOML: `[project]
name = "myapp"
dependencies = ["starlette>=0.30.0"]
`,
			wantLibraries:  []string{"starlette"},
			wantUploadPath: "",
		},
		{
			name: "werkzeug (Flask dependency)",
			pyprojectTOML: `[project]
name = "myapp"
dependencies = ["werkzeug>=3.0.0"]
`,
			wantLibraries:  []string{"werkzeug"},
			wantUploadPath: "",
		},
		{
			name: "fastapi with uploads directory",
			pyprojectTOML: `[project]
name = "myapp"
dependencies = ["fastapi>=0.100.0"]
`,
			createUploads:  true,
			wantLibraries:  nil,
			wantUploadPath: "uploads",
		},
		{
			name: "flask with media directory",
			pyprojectTOML: `[project]
name = "myapp"
dependencies = ["flask>=3.0.0"]
`,
			createUploads:  true,
			wantLibraries:  nil,
			wantUploadPath: "uploads",
		},
		{
			name: "django with media directory",
			pyprojectTOML: `[project]
name = "myapp"
dependencies = ["django>=4.2.0"]
`,
			createUploads:  true,
			wantLibraries:  nil,
			wantUploadPath: "uploads",
		},
		{
			name: "no web framework or upload library",
			pyprojectTOML: `[project]
name = "myapp"
dependencies = ["requests>=2.31.0"]
`,
			wantLibraries:  nil,
			wantUploadPath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "dockstart-upload-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if err := os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte(tt.pyprojectTOML), 0644); err != nil {
				t.Fatalf("Failed to write pyproject.toml: %v", err)
			}

			// Create uploads directory if needed
			if tt.createUploads {
				if err := os.Mkdir(filepath.Join(tmpDir, "uploads"), 0755); err != nil {
					t.Fatalf("Failed to create uploads dir: %v", err)
				}
			}

			d := NewPythonDetector()
			detection, err := d.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detection failed: %v", err)
			}
			if detection == nil {
				t.Fatal("Expected detection, got nil")
			}

			// Check file upload libraries
			if len(tt.wantLibraries) == 0 {
				if len(detection.FileUploadLibraries) != 0 {
					t.Errorf("FileUploadLibraries = %v, want empty", detection.FileUploadLibraries)
				}
			} else {
				for _, lib := range tt.wantLibraries {
					if !detection.HasFileUploadLibrary(lib) {
						t.Errorf("Expected file upload library %q to be detected, got %v", lib, detection.FileUploadLibraries)
					}
				}
			}

			// Check upload path
			if detection.UploadPath != tt.wantUploadPath {
				t.Errorf("UploadPath = %q, want %q", detection.UploadPath, tt.wantUploadPath)
			}
		})
	}
}

// TestFileUploadDetection_Python_Requirements tests file upload detection with requirements.txt.
func TestFileUploadDetection_Python_Requirements(t *testing.T) {
	tests := []struct {
		name           string
		requirements   string
		createUploads  bool
		wantLibraries  []string
		wantUploadPath string
	}{
		{
			name:          "python-multipart",
			requirements:  "python-multipart>=0.0.5\nfastapi>=0.100.0\n",
			wantLibraries: []string{"python-multipart"},
		},
		{
			name:          "aiofiles",
			requirements:  "aiofiles>=23.0.0\n",
			wantLibraries: []string{"aiofiles"},
		},
		{
			name:           "flask with uploads",
			requirements:   "flask>=3.0.0\n",
			createUploads:  true,
			wantLibraries:  nil,
			wantUploadPath: "uploads",
		},
		{
			name:          "no upload library",
			requirements:  "requests>=2.31.0\n",
			wantLibraries: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "dockstart-upload-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if err := os.WriteFile(filepath.Join(tmpDir, "requirements.txt"), []byte(tt.requirements), 0644); err != nil {
				t.Fatalf("Failed to write requirements.txt: %v", err)
			}

			// Create uploads directory if needed
			if tt.createUploads {
				if err := os.Mkdir(filepath.Join(tmpDir, "uploads"), 0755); err != nil {
					t.Fatalf("Failed to create uploads dir: %v", err)
				}
			}

			d := NewPythonDetector()
			detection, err := d.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detection failed: %v", err)
			}
			if detection == nil {
				t.Fatal("Expected detection, got nil")
			}

			// Check file upload libraries
			if len(tt.wantLibraries) == 0 {
				if len(detection.FileUploadLibraries) != 0 {
					t.Errorf("FileUploadLibraries = %v, want empty", detection.FileUploadLibraries)
				}
			} else {
				for _, lib := range tt.wantLibraries {
					if !detection.HasFileUploadLibrary(lib) {
						t.Errorf("Expected file upload library %q to be detected, got %v", lib, detection.FileUploadLibraries)
					}
				}
			}

			// Check upload path
			if detection.UploadPath != tt.wantUploadPath {
				t.Errorf("UploadPath = %q, want %q", detection.UploadPath, tt.wantUploadPath)
			}
		})
	}
}

// TestFileUploadDetection_Rust tests file upload library detection for Rust projects.
func TestFileUploadDetection_Rust(t *testing.T) {
	tests := []struct {
		name           string
		cargoTOML      string
		createUploads  bool
		wantLibraries  []string
		wantUploadPath string
	}{
		{
			name: "actix-multipart",
			cargoTOML: `[package]
name = "myapp"
version = "0.1.0"
edition = "2021"

[dependencies]
actix-multipart = "0.6"
actix-web = "4"
`,
			wantLibraries:  []string{"actix-multipart"},
			wantUploadPath: "",
		},
		{
			name: "multer (rust)",
			cargoTOML: `[package]
name = "myapp"
version = "0.1.0"
edition = "2021"

[dependencies]
multer = "2.0"
`,
			wantLibraries:  []string{"multer"},
			wantUploadPath: "",
		},
		{
			name: "axum-extra",
			cargoTOML: `[package]
name = "myapp"
version = "0.1.0"
edition = "2021"

[dependencies]
axum-extra = "0.9"
axum = "0.7"
`,
			wantLibraries:  []string{"axum-extra"},
			wantUploadPath: "",
		},
		{
			name: "actix-web with uploads directory",
			cargoTOML: `[package]
name = "myapp"
version = "0.1.0"
edition = "2021"

[dependencies]
actix-web = "4"
`,
			createUploads:  true,
			wantLibraries:  []string{"multipart"},
			wantUploadPath: "uploads",
		},
		{
			name: "axum with uploads directory",
			cargoTOML: `[package]
name = "myapp"
version = "0.1.0"
edition = "2021"

[dependencies]
axum = "0.7"
`,
			createUploads:  true,
			wantLibraries:  []string{"multipart"},
			wantUploadPath: "uploads",
		},
		{
			name: "rocket with uploads directory",
			cargoTOML: `[package]
name = "myapp"
version = "0.1.0"
edition = "2021"

[dependencies]
rocket = "0.5"
`,
			createUploads:  true,
			wantLibraries:  []string{"multipart"},
			wantUploadPath: "uploads",
		},
		{
			name: "web framework without uploads directory",
			cargoTOML: `[package]
name = "myapp"
version = "0.1.0"
edition = "2021"

[dependencies]
axum = "0.7"
`,
			createUploads:  false,
			wantLibraries:  nil,
			wantUploadPath: "",
		},
		{
			name: "no web framework",
			cargoTOML: `[package]
name = "myapp"
version = "0.1.0"
edition = "2021"

[dependencies]
clap = "4"
`,
			wantLibraries:  nil,
			wantUploadPath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "dockstart-upload-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if err := os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte(tt.cargoTOML), 0644); err != nil {
				t.Fatalf("Failed to write Cargo.toml: %v", err)
			}

			// Create uploads directory if needed
			if tt.createUploads {
				if err := os.Mkdir(filepath.Join(tmpDir, "uploads"), 0755); err != nil {
					t.Fatalf("Failed to create uploads dir: %v", err)
				}
			}

			d := NewRustDetector()
			detection, err := d.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detection failed: %v", err)
			}
			if detection == nil {
				t.Fatal("Expected detection, got nil")
			}

			// Check file upload libraries
			if len(tt.wantLibraries) == 0 {
				if len(detection.FileUploadLibraries) != 0 {
					t.Errorf("FileUploadLibraries = %v, want empty", detection.FileUploadLibraries)
				}
			} else {
				for _, lib := range tt.wantLibraries {
					if !detection.HasFileUploadLibrary(lib) {
						t.Errorf("Expected file upload library %q to be detected, got %v", lib, detection.FileUploadLibraries)
					}
				}
			}

			// Check upload path
			if detection.UploadPath != tt.wantUploadPath {
				t.Errorf("UploadPath = %q, want %q", detection.UploadPath, tt.wantUploadPath)
			}
		})
	}
}

// TestFileUploadDetection_UploadDirectoryVariants tests detection of various upload directory names.
func TestFileUploadDetection_UploadDirectoryVariants(t *testing.T) {
	uploadDirs := []string{
		"uploads",
		"upload",
		"files",
		"public/uploads",
		"static/uploads",
	}

	for _, dir := range uploadDirs {
		t.Run("directory_"+dir, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "dockstart-upload-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Create package.json with multer
			packageJSON := `{
				"name": "test-app",
				"dependencies": {"multer": "^1.4.5"}
			}`
			if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644); err != nil {
				t.Fatalf("Failed to write package.json: %v", err)
			}

			// Create the upload directory (with parent if needed)
			uploadPath := filepath.Join(tmpDir, dir)
			if err := os.MkdirAll(uploadPath, 0755); err != nil {
				t.Fatalf("Failed to create upload dir %s: %v", dir, err)
			}

			d := NewNodeDetector()
			detection, err := d.Detect(tmpDir)
			if err != nil {
				t.Fatalf("Detection failed: %v", err)
			}
			if detection == nil {
				t.Fatal("Expected detection, got nil")
			}

			// Check that upload path was detected
			if detection.UploadPath != dir {
				t.Errorf("UploadPath = %q, want %q", detection.UploadPath, dir)
			}
		})
	}
}

// TestFileUploadDetection_ModelHelpers tests the Detection model helper methods.
func TestFileUploadDetection_ModelHelpers(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dockstart-upload-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	packageJSON := `{
		"name": "test-app",
		"dependencies": {"multer": "^1.4.5", "formidable": "^3.5.0"}
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatalf("Failed to write package.json: %v", err)
	}

	d := NewNodeDetector()
	detection, err := d.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detection failed: %v", err)
	}

	// Test HasFileUploadLibrary
	if !detection.HasFileUploadLibrary("multer") {
		t.Error("HasFileUploadLibrary('multer') should return true")
	}
	if !detection.HasFileUploadLibrary("formidable") {
		t.Error("HasFileUploadLibrary('formidable') should return true")
	}
	if detection.HasFileUploadLibrary("busboy") {
		t.Error("HasFileUploadLibrary('busboy') should return false")
	}

	// Test AddFileUploadLibrary (should not add duplicates)
	originalLen := len(detection.FileUploadLibraries)
	detection.AddFileUploadLibrary("multer")
	if len(detection.FileUploadLibraries) != originalLen {
		t.Error("AddFileUploadLibrary should not add duplicate")
	}

	// Test adding new library
	detection.AddFileUploadLibrary("busboy")
	if !detection.HasFileUploadLibrary("busboy") {
		t.Error("AddFileUploadLibrary should add new library")
	}

	// Test NeedsFileProcessor
	if !detection.NeedsFileProcessor() {
		t.Error("NeedsFileProcessor should return true when upload libraries exist")
	}
}
