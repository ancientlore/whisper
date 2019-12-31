package main

import (
	"net/http"
	"os"
	"path"
)

// fixed returns a handler that serves only the given file from the file system.
func fixed(filename string) http.HandlerFunc {
	return http.FileServer(singleFileSystem(filename)).ServeHTTP
}

// singleFileSystem implements http.FileSystem, serving only a single file.
type singleFileSystem string

// Open opens the file.
func (sfs singleFileSystem) Open(name string) (http.File, error) {
	_, name = path.Split(name)
	if name != string(sfs) {
		return nil, os.ErrNotExist
	}
	f, err := os.Open(name)
	return http.File(f), err
}
