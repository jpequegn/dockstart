// Package generator provides file generation for Docker development environments.
package generator

import (
	"embed"
	"text/template"
)

// templates embeds all template files at compile time.
// This means the templates are included in the binary - no external files needed.
//
//go:embed templates/*.tmpl templates/processor/*.tmpl
var templatesFS embed.FS

// loadTemplate loads and parses a template from the embedded filesystem.
func loadTemplate(name string) (*template.Template, error) {
	return template.ParseFS(templatesFS, "templates/"+name)
}
