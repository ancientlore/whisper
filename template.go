package main

import (
	"fmt"
	"html/template"
	"path"
)

// tpl stores the site's HTML templates.
var tpl *template.Template

// loadTemplates loads and parses the HTML templates.
func loadTemplates() error {
	var err error
	funcMap := template.FuncMap{
		"dir":         dir,
		"join":        path.Join,
		"ext":         path.Ext,
		"markdown":    md,
		"frontmatter": fm,
	}
	tpl, err = template.New("whisper").Funcs(funcMap).ParseGlob("template/*.html")
	if err != nil {
		return fmt.Errorf("loadTemplates: %w", err)
	}
	return nil
}
