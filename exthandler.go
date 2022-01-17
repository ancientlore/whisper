package main

import (
	"bytes"
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

// extHandler is an http.HandlerFunc that renders general files into HTML using templates.
// extensions should include the dot.
func extHandler(defaultHandler http.Handler, defaultExpiry time.Duration, extensions []string, templateName string) http.Handler {
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
		// Unlike the markdown handler, we don't want to consider a file with extension
		// for images because those need to be rendered by the default one.
		// clean up path
		fn := pathToFile(r.URL.Path)
		// check if extension is non-empty and not Markdown
		ext := path.Ext(fn)
		if ext != "" && ext != ".md" {
			defaultHandler.ServeHTTP(w, r)
			return
		}
		var (
			front    *frontMatter
			y        template.HTML
			modTime  time.Time
			err      error
			foundMD  bool
			foundExt string
		)
		if ext == ".md" || ext == "" {
			// Read the markdown content and front matter
			front, y, modTime, err = cachedRenderMarkdown(fn)
			if err == nil {
				foundMD = true
				// Check for redirect
				if front.Redirect != "" {
					http.Redirect(w, r, front.Redirect, http.StatusFound)
					return
				}
			} else if !errors.Is(err, os.ErrNotExist) {
				log.Printf("extHandler: %s", err)
				serverError(w, r, err.Error())
				return
			} else if ext == ".md" {
				// user was looking for .md file specifically but it doesn't exist
				notFound(w, r)
				return
			}
			if ext == ".md" {
				fn = strings.TrimSuffix(fn, ".md")
			}
		}
		for _, e := range extensions {
			s, err := os.Stat(fn + e)
			if errors.Is(err, os.ErrNotExist) {
				continue
			} else if err != nil {
				log.Printf("extHandler: %s", err)
				serverError(w, r, err.Error())
				return
			}
			modTime = s.ModTime()
			front = &frontMatter{
				Title: strings.TrimSuffix(s.Name(), e),
				Date:  s.ModTime(),
			}
			foundExt = e
			break
		}
		if !foundMD && foundExt == "" {
			defaultHandler.ServeHTTP(w, r)
			return
		}
		// prepare template data
		p, bn := path.Split(fn)
		var data = data{
			FrontMatter: *front,
			Page: pageInfo{
				Path:     "/" + p,
				Filename: bn + foundExt,
			},
			Content: template.HTML(y),
		}
		// Check date - don't render until date/time is passed
		if time.Now().Before(data.FrontMatter.Date) {
			notFound(w, r)
			return
		}
		// Render the HTML template
		tplName := templateName
		if data.FrontMatter.Template != "" {
			tplName = data.FrontMatter.Template
		}
		var out bytes.Buffer
		tpl, tplModTime := getTemplates()
		if tplModTime.After(modTime) {
			modTime = tplModTime
		}
		err = tpl.ExecuteTemplate(&out, tplName, data)
		if err != nil {
			log.Printf("extHandler: %s", err)
			serverError(w, r, err.Error())
			return
		}
		// Set headers
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Length", strconv.Itoa(out.Len()))
		expiry := defaultExpiry
		if data.FrontMatter.Expires != 0 {
			expiry = time.Duration(data.FrontMatter.Expires)
		}
		if expiry != 0 {
			w.Header().Set("Expires", time.Now().Add(expiry).In(gmtZone).Format(time.RFC1123))
		}
		// w.Header().Set("Last-Modified", s.ModTime().Format(time.RFC1123))
		// http.ServeContent(w, r, "", modTime, bytes.NewReader(out.Bytes()))
		_, err = w.Write(out.Bytes())
		if err != nil {
			log.Printf("markdown: %s", err)
		}
	})
}

// pathToFile takes a URL path and converts it into the path to the associated file.
func pathToFile(filename string) string {
	// check for folder - if so, add index.md
	if strings.HasSuffix(filename, "/") {
		filename += "index.md"
	}
	filename = path.Clean(filename)
	// removing leading / so we find it on the file system
	filename = strings.TrimPrefix(filename, "/")
	return filename
}

/*
type stringSlice []string

func (a stringSlice) Contains(s string) bool {
	for _, x := range a {
		if x == s {
			return true
		}
	}
	return false
}
*/
