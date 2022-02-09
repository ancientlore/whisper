package web

import (
	"net/http"
	"strings"
	"time"
)

var gmtZone *time.Location

func init() {
	var err error
	gmtZone, err = time.LoadLocation("GMT")
	if err != nil {
		gmtZone = time.UTC
	}
}

// HeaderHandler returns an http.Handler that adds the given headers to the response.
func HeaderHandler(h http.Handler, headers map[string]string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range headers {
			w.Header().Set(k, v)
		}
		h.ServeHTTP(w, r)
	})
}

// ExpiresHandler adds the expires header choosing expires for dynamic content
// and staticExpires for static content.
func ExpiresHandler(h http.Handler, expires, staticExpires time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expiry := staticExpires
		if strings.HasSuffix(r.URL.Path, "/") || strings.HasSuffix(r.URL.Path, ".html") || r.URL.Path == "/sitemap.txt" {
			expiry = expires
		}
		if expiry != 0 {
			w.Header().Set("Expires", time.Now().Add(expiry).In(gmtZone).Format(time.RFC1123))
		}
		h.ServeHTTP(w, r)
	})
}
