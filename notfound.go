package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// notFound is a handler for rendering our 404 page.
func notFound(w http.ResponseWriter, r *http.Request) {
	tpl, _ := getTemplates()
	notfoundTpl := tpl.Lookup("notfound")
	if notfoundTpl == nil {
		http.NotFound(w, r)
		return
	}
	var d data
	d.FrontMatter.Title = "Not Found"
	d.Page.Path, d.Page.Filename = "/", "404 Not Found"
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusNotFound)
	// err := notfoundTpl.Execute(w, d)
	err := cachedExecuteTemplate(w, "notfound", d)
	if err != nil {
		log.Printf("notFound: %s", err)
	}
}

// exists will pretest for file existence and render a 404 if the file is not found.
func existsHandler(defaultHandler http.Handler, defaultExpiry time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if containsSpecialFile(r.URL.Path) {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		_, err := os.Stat(strings.TrimPrefix(r.URL.Path, "/"))
		if errors.Is(err, os.ErrNotExist) {
			notFound(w, r)
			return
		} else if err != nil {
			log.Printf("existsHandler: %s", err)
			serverError(w, r, err.Error())
			return
		}
		if defaultExpiry != 0 {
			w.Header().Set("Expires", time.Now().Add(defaultExpiry).In(gmtZone).Format(time.RFC1123))
		}
		defaultHandler.ServeHTTP(w, r)
	})
}
