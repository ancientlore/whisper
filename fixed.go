package main

import (
	"net/http"
	"os"
	"path"
)

func fixed(filename string) http.HandlerFunc {
	return http.FileServer(singleFileSystem(filename)).ServeHTTP
}

type singleFileSystem string

func (sfs singleFileSystem) Open(name string) (http.File, error) {
	_, name = path.Split(name)
	if name != string(sfs) {
		return nil, os.ErrNotExist
	}
	f, err := os.Open(name)
	return http.File(f), err
}
