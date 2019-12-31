package main

import (
	"errors"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/pelletier/go-toml"
	"github.com/russross/blackfriday/v2"
)

// frontMatter holds data scraped from a Markdown page.
type frontMatter struct {
	Title    string    `toml:"title" comment:"Title of this page"`
	Date     time.Time `toml:"date" comment:"Date the article appears"`
	Template string    `toml:"template" comment:"The name of the template to use"`
	Tags     []string  `toml:"tags" comment:"Tags to assign to this article"`
}

type Page struct {
	Name     string
	Filename string
}

type Section struct {
	Name  string
	Pages []Page
}

func (obj Section) Filename() string {
	if obj.Pages != nil && len(obj.Pages) > 0 {
		return obj.Pages[0].Filename
	}
	return ""
}

type Data struct {
	FrontMatter frontMatter
	Content     template.HTML
	Sections    []Section
	ActivePage  *Page
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
		var data = Data{
			Sections: []Section{
				{
					Name: "Home",
					Pages: []Page{
						{Name: "Index", Filename: "index"},
						{Name: "About", Filename: "about"},
					},
				},
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
		err = tpl.ExecuteTemplate(w, "default", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

// fmRegexp is the regular expression used to split out front matter.
var fmRegexp = regexp.MustCompile(`(?m)^\s*\+\+\+\s*$`)

// extractFrontMatter splits the front matter and Markdown content.
func extractFrontMatter(x []byte) (fm, r []byte) {
	subs := fmRegexp.Split(string(x), 3)
	if len(subs) != 3 {
		return nil, x
	}
	if s := strings.TrimSpace(subs[0]); len(s) > 0 {
		return nil, x
	}
	return []byte(strings.TrimSpace(subs[1])), []byte(strings.TrimSpace(subs[2]))
}
