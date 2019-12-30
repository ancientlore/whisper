package main

import (
	"errors"
	"net/http"
	"os"
)

func favicon(w http.ResponseWriter, r *http.Request) {
	_, err := os.Stat("static/favicon.ico")
	if errors.Is(err, os.ErrNotExist) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	http.Redirect(w, r, "static/favicon.ico", http.StatusPermanentRedirect)
}
