package web

import (
	"io/fs"
	"net/http"
)

// ErrorHandler captures 404 and 500 errors and serves /404.html or /500.html from the file system.
func ErrorHandler(h http.Handler, fsys fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writer := &responseWriter{
			ResponseWriter: w,
			fsys:           fsys,
		}
		h.ServeHTTP(writer, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	fsys    fs.FS
	noWrite bool
	err     error
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if w.noWrite {
		return len(b), w.err
	}
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) WriteHeader(statusCode int) {
	var (
		b    []byte
		err  error
		file string
	)
	if statusCode == http.StatusNotFound {
		file = "404.html"
	} else if statusCode == http.StatusInternalServerError {
		file = "500.html"
	}
	if file != "" {
		// special processing of response
		b, err = fs.ReadFile(w.fsys, file)
		if err == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Del("X-Content-Type-Options")
			w.ResponseWriter.WriteHeader(statusCode)
			w.noWrite = true
			_, w.err = w.ResponseWriter.Write(b)
			return
		}
	}
	// normal processing
	w.ResponseWriter.WriteHeader(statusCode)
}
