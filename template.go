package main

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"os"
	"path"
	"strings"
)

// pageInfo has information about the current page.
type pageInfo struct {
	Path     string // path from URL
	Filename string // end portion (file) from URL
}

// Pathname joins the path and filename.
func (p pageInfo) Pathname() string {
	return path.Join(p.Path, p.Filename)
}

// data is what is passed to markdown templates.
type data struct {
	FrontMatter frontMatter   // front matter from Markdown file or defaults
	Page        pageInfo      // information aboout current page
	Content     template.HTML // rendered Markdown
	Message     string        // Passed to error or 404 templates
}

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
	fi, err := os.Stat("template")
	if errors.Is(err, os.ErrNotExist) || (err == nil && !fi.IsDir()) {
		log.Print("ERROR: No template folder found; using default templates.")
		tpl, err = template.New("whisper").Funcs(funcMap).Parse(defaultTemplate)
	} else {
		tpl, err = template.New("whisper").Funcs(funcMap).ParseGlob("template/*.html")
	}
	if err != nil {
		return fmt.Errorf("loadTemplates: %w", err)
	}
	return nil
}

const (
	defaultTemplate = `{{define "default"}}<!DOCTYPE html>
<html lang="en">
	<head>
		<meta charset="utf-8">
		<title>{{.FrontMatter.Title}}</title>
	</head>
	<body>
		{{.Content}}
		<hr/>
		<ul>{{ $p := .Page.Path}}{{range sortbyname (dir .Page.Path)}}
			<li><a href="{{join $p .Filename}}">{{if eq ".md" (ext .Filename)}}{{.FrontMatter.Title}}{{else}}{{.Filename}}{{end}}</a> {{.FrontMatter.Date.String}}</li>
		{{end}}</ul>
		<hr/>
		<a href="/">Home</a>
	</body>
</html>
{{end}}{{define "image"}}<!DOCTYPE html>
<html lang="en">
	<head>
		<meta charset="utf-8">
		<title>{{.FrontMatter.Title}}</title>
	</head>
	<body>		
		{{.Content}}
		<hr/>
		<h3>{{.FrontMatter.Title}}</h1>
		<img src="{{join .Page.Path .Page.Filename}}" alt="{{.FrontMatter.Title}}"/>
		<hr/>
		<ul>{{ $p := .Page.Path}}{{range sortbyname (dir .Page.Path)}}
			<li><a href="{{join $p .Filename}}">{{if eq ".md" (ext .Filename)}}{{.FrontMatter.Title}}{{else}}{{.Filename}}{{end}}</a> {{.FrontMatter.Date.String}}</li>
		{{end}}</ul>
		<hr/>
		<a href="/">Home</a>
	</body>
</html>
{{end}}`
)
