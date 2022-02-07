package virtual

import (
	"fmt"
	"io"
	"io/fs"
	"time"
)

// virtualFile represents a view of an underlying file.
type virtualFile struct {
	fi     fileInfo
	reader io.ReadSeeker // Main Reader to use
}

// Stat returns information about the file.
func (f virtualFile) Stat() (fs.FileInfo, error) {
	return f.fi, nil
}

// Read reads up to len(b) bytes from the File. It returns the number of bytes read
// and any error encountered. At end of file, Read returns 0, io.EOF.
func (f *virtualFile) Read(b []byte) (int, error) {
	return f.reader.Read(b)
}

// Close closes the file. Cached files are in memory, so this function does nothing.
func (f *virtualFile) Close() error {
	return nil
}

// Seek sets the offset for the next Read or Write to offset, interpreted according
// to whence: io.SeekStart means relative to the start of the file, io.SeekCurrent
// means relative to the current offset, and io.SeekEnd means relative to the end.
// Seek returns the new offset relative to the start of the file and an error, if any.
//
// Seeking to an offset before the start of the file is an error. Seeking to any
// positive offset is legal, but the behavior of subsequent I/O operations on the
// underlying object is implementation-dependent.
func (f *virtualFile) Seek(offset int64, whence int) (int64, error) {
	return f.reader.Seek(offset, whence)
}

// virtualDir specializes virtualFile with the ReadDir method.
type virtualDir struct {
	fi      fileInfo
	entries []fs.DirEntry
	pos     int
}

// Stat returns information about the file.
func (f virtualDir) Stat() (fs.FileInfo, error) {
	return f.fi, nil
}

// Read reads up to len(b) bytes from the File. It returns the number of bytes read
// and any error encountered. At end of file, Read returns 0, io.EOF.
func (f *virtualDir) Read(b []byte) (int, error) {
	return 0, io.EOF
}

// Close closes the file. Cached files are in memory, so this function does nothing.
func (f *virtualDir) Close() error {
	return nil
}

// ReadDir reads the contents of the directory and returns
// a slice of up to n DirEntry values in directory order.
// Subsequent calls on the same file will yield further DirEntry values.
//
// If n > 0, ReadDir returns at most n DirEntry structures.
// In this case, if ReadDir returns an empty slice, it will return
// a non-nil error explaining why.
// At the end of a directory, the error is io.EOF.
//
// If n <= 0, ReadDir returns all the DirEntry values from the directory
// in a single slice. In this case, if ReadDir succeeds (reads all the way
// to the end of the directory), it returns the slice and a nil error.
// If it encounters an error before the end of the directory,
// ReadDir returns the DirEntry list read until that point and a non-nil error.
func (f *virtualDir) ReadDir(n int) ([]fs.DirEntry, error) {
	if !f.fi.IsDir() {
		return nil, &fs.PathError{Op: "readdir", Path: f.fi.Name(), Err: fmt.Errorf("Not a directory: %w", fs.ErrInvalid)}
	}

	if n <= 0 {
		dest := make([]fs.DirEntry, len(f.entries))
		copy(dest, f.entries)
		return dest, nil
	}

	max := len(f.entries) - f.pos
	if n < max {
		max = n
	}
	if max == 0 {
		return nil, io.EOF
	}
	dest := make([]fs.DirEntry, max)
	copy(dest, f.entries[f.pos:])
	f.pos += max
	return dest, nil
}

// fileInfo holds the metadata about the file
type fileInfo struct {
	nm string
	sz int64
	md fs.FileMode
	mt time.Time
}

// Name returns the base name of the file.
func (fi fileInfo) Name() string {
	return fi.nm
}

// Size returns the length in bytes for regular files; system-dependent for others.
func (fi fileInfo) Size() int64 {
	return fi.sz
}

// Mode returns the file mode bits of the file.
func (fi fileInfo) Mode() fs.FileMode {
	return fi.md
}

// ModTime returns the modification time of the file.
func (fi fileInfo) ModTime() time.Time {
	return fi.mt
}

// IsDir is an abbreviation for Mode().IsDir().
func (fi fileInfo) IsDir() bool {
	return fi.md.IsDir()
}

// Sys always returns nil because the virtual file does not keep this information.
func (fi fileInfo) Sys() interface{} {
	return nil
}
