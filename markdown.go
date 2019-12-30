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

type FrontMatter struct {
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
	FrontMatter FrontMatter
	Content     template.HTML
	Sections    []Section
	ActivePage  *Page
}

func markdown(w http.ResponseWriter, r *http.Request) {
	d, fn := path.Split(r.URL.Path)
	if fn == "" {
		fn = "index.md"
	}
	d = strings.TrimPrefix(d, "/")
	if !strings.HasSuffix(fn, ".md") {
		fn += ".md"
	}
	fn = path.Join(d, fn)
	s, err := os.Stat(fn)
	if errors.Is(err, os.ErrNotExist) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	x, err := ioutil.ReadFile(fn)
	if errors.Is(err, os.ErrNotExist) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Last-Modified", s.ModTime().Format(time.RFC1123))
	fm, rst := extractFrontMatter(x)
	y := blackfriday.Run(rst)
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
	err = tpl.ExecuteTemplate(w, "default", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

var fmRegexp = regexp.MustCompile(`(?m)^\s*\+\+\+\s*$`)

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
