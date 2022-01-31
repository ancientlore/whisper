package virtual

import (
	"html/template"
	"io/fs"
	"sync"
)

// FS provides a virtual view of the file system suitable for serving Markdown
// and other files in a web format.
type FS struct {
	fs       fs.FS
	tpl      *template.Template
	tplMutex sync.RWMutex
}

// New
func New(innerFS fs.FS) *FS {
	return &FS{
		fs: innerFS,
	}
}

// Open opens the named file.
//
// When Open returns an error, it should be of type *fs.PathError
// with the Op field set to "open", the Path field set to name,
// and the Err field describing the problem.
//
// Open should reject attempts to open names that do not satisfy
// fs.ValidPath(name), returning a *PathError with Err set to
// ErrInvalid or ErrNotExist.
func (vfs *FS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	if isHiddenFile(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}

	return vfs.fs.Open(name)
}

// ReadDir reads the named directory
// and returns a list of directory entries sorted by filename.
func (vfs *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, err := fs.ReadDir(vfs.fs, name)
	if err != nil {
		return nil, err
	}
	if len(entries) > 0 {
		e := make([]fs.DirEntry, 0, len(entries))
		for i := range entries {
			if !(name == "." && isHiddenFile(entries[i].Name())) && !containsSpecialFile(entries[i].Name()) {
				e = append(e, entries[i])
			}
		}
		entries = e
	}
	return entries, nil
}
