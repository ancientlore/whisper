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

	"github.com/russross/blackfriday/v2"
)

type Page struct {
	Name     string
	Filename string
}

type Section struct {
	Name   string
	Pages  []Page
	Active bool
}

func (obj Section) Filename() string {
	if obj.Pages != nil && len(obj.Pages) > 0 {
		return obj.Pages[0].Filename
	}
	return ""
}

type Data struct {
	Sections      []Section
	Content       template.HTML
	ActiveSection *Section
	ActivePage    *Page
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
	s, err := os.Stat(path.Join(d, fn))
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
	y := blackfriday.Run(x)
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
	err = tpl.ExecuteTemplate(w, "default", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
