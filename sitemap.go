package main

import (
	"net/http"
)

func sitemap(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
}
