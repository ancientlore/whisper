package main

import (
	"errors"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/pelletier/go-toml"
	"github.com/russross/blackfriday/v2"
)

// pageInfo has information about the current page.
type pageInfo struct {
	Path     string
	Filename string
}

// data is what is passed to makedown templates.
type data struct {
	FrontMatter frontMatter
	Page        pageInfo
	Content     template.HTML
}

// markdown is an http.HandlerFunc that renders Markdown files into HTML using templates.
func markdown(defaultHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// split path and file
		d, fn := path.Split(r.URL.Path)
		if fn == "" {
			fn = "index.md"
		}
		// check if extension is non-empty and not Markdown
		ext := path.Ext(fn)
		if ext != "" && ext != ".md" {
			defaultHandler.ServeHTTP(w, r)
			return
		}
		// Prepare markdown physical file name
		d = strings.TrimPrefix(d, "/")
		if !strings.HasSuffix(fn, ".md") {
			fn += ".md"
		}
		fn = path.Join(d, fn)
		// See if the markdown file exists
		s, err := os.Stat(fn)
		if errors.Is(err, os.ErrNotExist) {
			defaultHandler.ServeHTTP(w, r)
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Read the markdown content
		x, err := ioutil.ReadFile(fn)
		if errors.Is(err, os.ErrNotExist) {
			notFound(w, r)
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// extract front matter
		fm, rst := extractFrontMatter(x)
		// format the markdown
		y := blackfriday.Run(rst)
		// prepare template data
		_, bn := path.Split(r.URL.Path)
		var data = data{
			Page: pageInfo{
				Path:     r.URL.Path,
				Filename: bn,
			},
			Content: template.HTML(y),
		}
		if len(fm) > 0 {
			err = toml.Unmarshal(fm, &data.FrontMatter)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		// Check date - don't render until date/time is passed
		if time.Now().Before(data.FrontMatter.Date) {
			notFound(w, r)
			return
		}
		// Set headers
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Last-Modified", s.ModTime().Format(time.RFC1123))
		// Render the HTML template
		templateName := "default"
		if data.FrontMatter.Template != "" {
			templateName = data.FrontMatter.Template
		}
		err = tpl.ExecuteTemplate(w, templateName, data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}
