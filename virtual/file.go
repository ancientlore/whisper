package virtual

import (
	"io/fs"
)

// virtualFile represents a view of an underlying file.
type virtualFile struct {
	fs.File        // Underling file
	name    string // Name of virtual file
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

// virtualFileInfo holds the metadata about the cached file but allows you to rename it.
type virtualFileInfo struct {
	fs.FileInfo        // Underlying file information
	name        string // Name of virtual file
}

// Name returns the base name of the file.
func (fi virtualFileInfo) Name() string {
	return fi.name
}

// virtualDirEntry is a special version of fileInfo to represent directory entries.
type virtualDirEntry struct {
	virtualFileInfo
}

// Type returns the type bits for the entry.
// The type bits are a subset of the usual FileMode bits, those returned by the FileMode.Type method.
func (di virtualDirEntry) Type() fs.FileMode {
	return di.virtualFileInfo.Mode().Type()
}

// Info returns the FileInfo for the file or subdirectory described by the entry.
func (di virtualDirEntry) Info() (fs.FileInfo, error) {
	return di.virtualFileInfo, nil
}
