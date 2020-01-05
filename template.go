package main

import (
	"fmt"
	"html/template"
	"path"
	"strings"
)

// tpl stores the site's HTML templates.
var tpl *template.Template

// loadTemplates loads and parses the HTML templates.
func loadTemplates() error {
	var err error
	funcMap := template.FuncMap{
		"dir":         dir,
		"sortbyname":  sortByName,
		"sortbytime":  sortByTime,
		"match":       match,
		"filter":      filter,
		"join":        path.Join,
		"ext":         path.Ext,
		"prev":        prev,
		"next":        next,
		"reverse":     reverse,
		"trimsuffix":  strings.TrimSuffix,
		"trimprefix":  strings.TrimPrefix,
		"trimspace":   strings.TrimSpace,
		"markdown":    md,
		"frontmatter": fm,
	}
	tpl, err = template.New("whisper").Funcs(funcMap).ParseGlob("template/*.html")
	if err != nil {
		return fmt.Errorf("loadTemplates: %w", err)
	}
	return nil
}
