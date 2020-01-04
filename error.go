package main

import (
	"bytes"
	"log"
	"net/http"
	"path"
	"strings"
)

// errData adds a message to the template data.
type errData struct {
	data
	Message string
}

// serverError is a handler for rendering our error page if defined.
func serverError(w http.ResponseWriter, r *http.Request, errMsg string) {
	var d errData
	d.FrontMatter.Title = "Server Error"
	d.Page.Path, d.Page.Filename = path.Split(r.URL.Path)
	d.Message = errMsg
	var out bytes.Buffer
	err := tpl.ExecuteTemplate(&out, "error", d)
	if err != nil {
		// This is hacky but template package doesn't make it nice to see the error type
		if !strings.HasSuffix(err.Error(), "is undefined") {
			log.Printf("serverError: %s", err)
		}
		http.Error(w, errMsg, http.StatusInternalServerError)
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
