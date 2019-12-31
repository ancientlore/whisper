package main

import (
	"net/http"
)

// sitemap is an http.HandlerFunc that renders the site map.
func sitemap(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
}
