package main

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
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
}

// tplSite stores the site's HTML templates.
var tplSite *template.Template

// tplLastModTime stores the last time the templates were modified.
var tplLastModTime time.Time

// getTemplates returns the templates and last time they were modified.
func getTemplates() (*template.Template, time.Time) {
	return tplSite, tplLastModTime
}

// loadTemplates loads and parses the HTML templates, returning true if custom templates were found.
func loadTemplates() (bool, error) {
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
		"now":         time.Now,
	}
	// Check if we are using default templates
	fi, err := os.Stat("template")
	if errors.Is(err, os.ErrNotExist) || (err == nil && !fi.IsDir()) {
		tplSite, err = template.New("whisper").Funcs(funcMap).Parse(defaultTemplate)
		if err != nil {
			return false, fmt.Errorf("loadTemplates: %w", err)
		}
		// tplLasModTime stays set to zero
		return false, nil
	}
	// use custom templates
	tplSite, err = template.New("whisper").Funcs(funcMap).ParseGlob("template/*.html")
	if err != nil {
		return true, fmt.Errorf("loadTemplates: %w", err)
	}
	tplLastModTime, err = getTplModTime("template/*.html")
	if err != nil {
		return true, fmt.Errorf("loadTemplates: %w", err)
	}
	return true, nil
}

// getTplModTime gets the latest modification time of the templates.
func getTplModTime(glob string) (time.Time, error) {
	var maxTime time.Time
	files, err := filepath.Glob(glob)
	if err != nil {
		return maxTime, fmt.Errorf("getTplModTime: %w", err)
	}
	for _, file := range files {
		log.Printf("getTplModTime: Stat: %q", file)
		fi, err := os.Stat(file)
		if err != nil {
			return maxTime, fmt.Errorf("getTplModTime: %w", err)
		}
		if !fi.IsDir() {
			if maxTime.Before(fi.ModTime()) {
				maxTime = fi.ModTime()
			}
		}
	}
	return maxTime, nil
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
