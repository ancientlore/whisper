package main

import (
	"bytes"
	"errors"
	"fmt"
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

// initGMT initialized the GMT zone used in headers.
func initGMT() error {
	var err error
	gmtZone, err = time.LoadLocation("GMT")
	if err != nil {
		gmtZone = time.UTC
	}
	return err
}

// markdown is an http.HandlerFunc that renders Markdown files into HTML using templates.
func markdown(defaultHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		// log.Print(r.URL.Path)
		if containsSpecialFile(r.URL.Path) {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		// clean up path
		fn := pathToMarkdown(r.URL.Path)
		// check if extension is non-empty and not Markdown
		ext := path.Ext(fn)
		if ext != "" && ext != ".md" {
			defaultHandler.ServeHTTP(w, r)
			return
		}
		// Read the markdown content and front matter
		front, y, modTime, err := renderMarkdown(fn)
		if errors.Is(err, os.ErrNotExist) {
			notFound(w, r)
			return
		} else if err != nil {
			log.Printf("markdown: %s", err)
			serverError(w, r, err.Error())
			return
		}
		// prepare template data
		p, bn := path.Split(r.URL.Path)
		var data = data{
			FrontMatter: *front,
			Page: pageInfo{
				Path:     p,
				Filename: bn,
			},
			Content: template.HTML(y),
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
		http.ServeContent(w, r, "", modTime, bytes.NewReader(out.Bytes()))
	})
}

// pathToMarkdown takes a URL path and converts it into the path to the associated Markdown file.
func pathToMarkdown(filename string) string {
	// check for folder - if so, add index.md
	if strings.HasSuffix(filename, "/") {
		filename += "index.md"
	}
	// removing leading / so we find it on the file system
	filename = strings.TrimPrefix(filename, "/")
	// make sure the extension is present
	if path.Ext(filename) == "" {
		filename += ".md"
	}
	return filename
}

// renderMarkdown renders the markdown for the given file and returns the frontmatter.
func renderMarkdown(filename string) (*frontMatter, template.HTML, time.Time, error) {
	var (
		fmData  frontMatter
		md      template.HTML
		modTime time.Time
	)
	filename = pathToMarkdown(filename)
	s, err := os.Stat(filename)
	if err != nil {
		return nil, "", modTime, fmt.Errorf("renderMarkdown: %w", err)
	}
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, "", modTime, fmt.Errorf("renderMarkdown: %w", err)
	}
	fm, r := extractFrontMatter(b)
	md = template.HTML(blackfriday.Run(r))
	if len(fm) > 0 {
		err = toml.Unmarshal(fm, &fmData)
		if err != nil {
			return nil, "", modTime, fmt.Errorf("renderMarkdown: %w", err)
		}
	}
	return &fmData, md, s.ModTime(), nil
}

// md convert the given markdown file to HTML and is used in templates.
func md(filename string) template.HTML {
	_, md, _, err := renderMarkdown(filename)
	if err != nil {
		log.Printf("md: %s", err)
		return ""
	}
	return md
}

// fm returns front matter for the given file and is used in templates.
func fm(filename string) *frontMatter {
	fm, _, _, err := renderMarkdown(filename)
	if err != nil {
		log.Printf("fm: %s", err)
		return nil
	}
	return fm
}
