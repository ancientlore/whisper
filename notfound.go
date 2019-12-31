package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// notFound is a handler for rendering our 404 page.
func notFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusNotFound)
	err := tpl.ExecuteTemplate(w, "notfound", nil)
	if err != nil {
		fmt.Fprintln(w, http.StatusText(http.StatusNotFound))
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
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defaultHandler.ServeHTTP(w, r)
	})
}
