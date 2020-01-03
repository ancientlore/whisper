package main

import (
	"bytes"
	"log"
	"net/http"
	"path"
)

// serverError is a handler for rendering our error page.
func serverError(w http.ResponseWriter, r *http.Request) {
	var d data
	d.FrontMatter.Title = "Server Error"
	d.Page.Path, d.Page.Filename = path.Split(r.URL.Path)
	var out bytes.Buffer
	err := tpl.ExecuteTemplate(&out, "error", d)
	if err != nil {
		log.Printf("serverError: %s", err)
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusInternalServerError)
	_, err = w.Write(out.Bytes())
	if err != nil {
		log.Printf("serverError: %s", err)
	}
}
