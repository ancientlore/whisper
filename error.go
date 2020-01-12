package main

import (
	"log"
	"net/http"
	"path"
)

// serverError is a handler for rendering our error page if defined.
func serverError(w http.ResponseWriter, r *http.Request, errMsg string) {
	errTpl := tpl.Lookup("error")
	if errTpl == nil {
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}
	var d data
	d.FrontMatter.Title = "Server Error"
	d.Page.Path, d.Page.Filename = path.Split(r.URL.Path)
	d.Message = errMsg
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusInternalServerError)
	err := errTpl.Execute(w, d)
	if err != nil {
		log.Printf("serverError: %s", err)
	}
}
