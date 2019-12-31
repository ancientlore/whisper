package main

import (
	"net/http"
	"os"
	"strings"
)

// containsSpecialFile reports whether name contains a path element starting with a period
// or is another kind of special file. The name is assumed to be a delimited by forward
// slashes, as guaranteed by the http.FileSystem interface.
func containsSpecialFile(name string) bool {
	parts := strings.Split(name, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ".") || strings.HasSuffix(part, ".toml") {
			return true
		}
	}
	return false
}

// specialFileHidingFile is the http.File use in specialFileHidingFileSystem.
// It is used to wrap the Readdir method of http.File so that we can
// remove files and directories that start with a period or are otherwise
// special from its output.
type specialFileHidingFile struct {
	http.File
}

// Readdir is a wrapper around the Readdir method of the embedded File
// that filters out all files that start with a period in their name.
func (f specialFileHidingFile) Readdir(n int) (fis []os.FileInfo, err error) {
	files, err := f.File.Readdir(n)
	for _, file := range files { // Filters out the dot files
		if !strings.HasPrefix(file.Name(), ".") && !strings.HasSuffix(file.Name(), ".toml") {
			fis = append(fis, file)
		}
	}
	return
}

// specialFileHidingFileSystem is an http.FileSystem that hides
// hidden "dot files" or other special files from being served.
type specialFileHidingFileSystem struct {
	http.FileSystem
}

// Open is a wrapper around the Open method of the embedded FileSystem
// that serves a 403 permission error when name has a file or directory
// with whose name starts with a period in its path or is otherwise special.
func (fs specialFileHidingFileSystem) Open(name string) (http.File, error) {
	if containsSpecialFile(name) { // If dot file, return 403 response
		return nil, os.ErrPermission
	}

	file, err := fs.FileSystem.Open(name)
	if err != nil {
		return nil, err
	}
	return specialFileHidingFile{file}, err
}
