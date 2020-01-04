package main

import (
	"bytes"
	"errors"
	"html/template"
	"io/ioutil"
	"log"
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

var gmtZone *time.Location

func init() {
	var err error
	gmtZone, err = time.LoadLocation("GMT")
	if err != nil {
		log.Printf("Cannot load GMT, using UTC instead: %s", err)
		gmtZone = time.UTC
	}
}

// markdown is an http.HandlerFunc that renders Markdown files into HTML using templates.
func markdown(defaultHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// log.Print(r.URL.Path)
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
		if ext != ".md" {
			fn += ".md"
		}
		fn = path.Join(d, fn)
		// See if the markdown file exists
		s, err := os.Stat(fn)
		if errors.Is(err, os.ErrNotExist) {
			defaultHandler.ServeHTTP(w, r)
			return
		} else if err != nil {
			log.Printf("markdown: %s", err)
			serverError(w, r, err.Error())
			return
		}
		// Read the markdown content
		x, err := ioutil.ReadFile(fn)
		if errors.Is(err, os.ErrNotExist) {
			notFound(w, r)
			return
		} else if err != nil {
			log.Printf("markdown: %s", err)
			serverError(w, r, err.Error())
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
				log.Printf("markdown: %s", err)
				serverError(w, r, err.Error())
				return
			}
		}
		// Check date - don't render until date/time is passed
		if time.Now().Before(data.FrontMatter.Date) {
			notFound(w, r)
			return
		}
		// Render the HTML template
		templateName := "default"
		if data.FrontMatter.Template != "" {
			templateName = data.FrontMatter.Template
		}
		var out bytes.Buffer
		err = tpl.ExecuteTemplate(&out, templateName, data)
		if err != nil {
			log.Printf("markdown: %s", err)
			serverError(w, r, err.Error())
			return
		}
		// Set headers
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if data.FrontMatter.Expires != 0 {
			w.Header().Set("Expires", time.Now().Add(data.FrontMatter.Expires).In(gmtZone).Format(time.RFC1123))
		}
		// w.Header().Set("Last-Modified", s.ModTime().Format(time.RFC1123))
		http.ServeContent(w, r, "", s.ModTime(), bytes.NewReader(out.Bytes()))
	})
}
