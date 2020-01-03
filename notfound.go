package main

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
)

// notFound is a handler for rendering our 404 page.
func notFound(w http.ResponseWriter, r *http.Request) {
	var d data
	d.FrontMatter.Title = "Not Found"
	d.Page.Path, d.Page.Filename = path.Split(r.URL.Path)
	var out bytes.Buffer
	err := tpl.ExecuteTemplate(&out, "notfound", d)
	if err != nil {
		log.Printf("notFound: %s", err)
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusNotFound)
	_, err = w.Write(out.Bytes())
	if err != nil {
		log.Printf("notFound: %s", err)
	}
}

// exists will pretest for file existence and render a 404 if the file is not found.
func existsHandler(defaultHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := os.Stat(strings.TrimPrefix(r.URL.Path, "/"))
		if errors.Is(err, os.ErrNotExist) {
			notFound(w, r)
			return
		} else if err != nil {
			log.Printf("existsHandler: %s", err)
			serverError(w, r)
			return
		}
		defaultHandler.ServeHTTP(w, r)
	})
}
