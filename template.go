package main

import (
	"html/template"
	"path"
)

// tpl stores the site's HTML templates.
var tpl *template.Template

func loadTemplates() error {
	var err error
	funcMap := template.FuncMap{
		"dir":  dir,
		"join": path.Join,
		"ext":  path.Ext,
	}
	tpl, err = template.New("whisper").Funcs(funcMap).ParseGlob("template/*.html")
	return err
}
