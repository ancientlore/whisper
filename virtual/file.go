package virtual

import (
	"io/fs"
)

/*
Types of virtual files:

	Directory
	Index
	Sitemap
	Markdown
	Image


*/

// file represents a cached file
type virtualFile struct {
	fs.File

	name string // Virtual name of the file
}

// Stat returns information about the file.
func (f virtualFile) Stat() (fs.FileInfo, error) {
	var (
		fi  virtualFileInfo
		err error
	)
	fi.FileInfo, err = f.File.Stat()
	fi.name = f.name

	return fi, err
}

// Read reads up to len(b) bytes from the File. It returns the number of bytes read
// and any error encountered. At end of file, Read returns 0, io.EOF.
func (f *virtualFile) Read(b []byte) (int, error) {
	return f.File.Read(b)
}

// Close closes the file. Cached files are in memory, so this function does nothing.
func (f *virtualFile) Close() error {
	return f.File.Close()
}

// fileInfo holds the metadata about the cached file
type virtualFileInfo struct {
	fs.FileInfo
	name string
}

// Name returns the base name of the file.
func (fi virtualFileInfo) Name() string {
	return fi.name
}

// virtualDirEntry is a special version of fileInfo to represent directory entries.
// It is lightweight in that it isn't as filled out as if you called Stat
// on the file itself.
type virtualDirEntry struct {
	virtualFileInfo
}

// Type returns the type bits for the entry.
// The type bits are a subset of the usual FileMode bits, those returned by the FileMode.Type method.
func (di virtualDirEntry) Type() fs.FileMode {
	return di.virtualFileInfo.Mode().Type()
}

// Info returns the FileInfo for the file or subdirectory described by the entry.
// The returned info is from the time of the directory read.
func (di virtualDirEntry) Info() (fs.FileInfo, error) {
	return di.virtualFileInfo, nil
}
