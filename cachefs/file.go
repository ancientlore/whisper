package cachefs

import (
	"fmt"
	"io"
	"io/fs"
	"math"
	"time"
)

// file represents a cached file
type file struct {
	Data []byte     // Contents of the file
	FI   fileInfo   // Metadata about the file
	pos  int        // Current position
	Dirs []dirEntry // Directory entries
}

// Stat returns information about the file.
func (f file) Stat() (fs.FileInfo, error) {
	return f.FI, nil
}

// Read reads up to len(b) bytes from the File. It returns the number of bytes read
// and any error encountered. At end of file, Read returns 0, io.EOF.
func (f *file) Read(b []byte) (int, error) {
	if f.pos >= len(f.Data) {
		return 0, io.EOF
	}
	n := copy(b, f.Data[f.pos:])
	f.pos += n
	return n, nil
}

// Seek sets the offset for the next Read or Write to offset, interpreted according
// to whence: io.SeekStart means relative to the start of the file, io.SeekCurrent
// means relative to the current offset, and io.SeekEnd means relative to the end.
// Seek returns the new offset relative to the start of the file and an error, if any.
//
// Seeking to an offset before the start of the file is an error. Seeking to any
// positive offset is legal, but the behavior of subsequent I/O operations on the
// underlying object is implementation-dependent.
func (f *file) Seek(offset int64, whence int) (int64, error) {
	// Check for dir
	if f.FI.IsDir() {
		return 0, fmt.Errorf("Cannot Seek on a directory")
	}
	// Sanitize offset
	if offset > math.MaxInt || offset < math.MinInt {
		return int64(f.pos), fmt.Errorf("Offset value too large for Seek: %d", offset)
	}

	// Calculate new position
	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = int64(f.pos) + offset
	case io.SeekEnd:
		newPos = int64(len(f.Data)) + offset
	default:
		return int64(f.pos), fmt.Errorf("Invalid whence value for Seek: %d", whence)
	}

	// Check sanity of new position
	if newPos < 0 {
		return int64(f.pos), fmt.Errorf("Cannot Seek before start of file: %d", newPos)
	}
	if newPos > int64(len(f.Data)) {
		newPos = int64(len(f.Data))
	}

	// Finalize and return
	f.pos = int(newPos)
	return int64(f.pos), nil
}

// Close closes the file. Cached files are in memory, so this function does nothing.
func (f *file) Close() error {
	return nil // nothing to do
}

// fileInfo holds the metadata about the cached file
type fileInfo struct {
	Nm string
	Sz int64
	Md fs.FileMode
	Mt time.Time
}

// Name returns the base name of the file.
func (fi fileInfo) Name() string {
	return fi.Nm
}

// Size returns the length in bytes for regular files; system-dependent for others.
func (fi fileInfo) Size() int64 {
	return fi.Sz
}

// Mode returns the file mode bits of the file.
func (fi fileInfo) Mode() fs.FileMode {
	return fi.Md
}

// ModTime returns the modification time of the file.
func (fi fileInfo) ModTime() time.Time {
	return fi.Mt
}

// IsDir is an abbreviation for Mode().IsDir().
func (fi fileInfo) IsDir() bool {
	return fi.Md.IsDir()
}

// Sys always returns nil because the cached file does not keep this information.
func (fi fileInfo) Sys() interface{} {
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
func (f *file) ReadDir(n int) ([]fs.DirEntry, error) {
	if !f.FI.IsDir() {
		return nil, &fs.PathError{Op: "readdir", Path: f.FI.Name(), Err: fmt.Errorf("Not a directory: %w", fs.ErrInvalid)}
	}

	if n <= 0 {
		dest := make([]fs.DirEntry, len(f.Dirs))
		for i := range f.Dirs {
			dest[i] = f.Dirs[i]
		}
		return dest, nil
	}

	max := len(f.Dirs) - f.pos
	if n < max {
		max = n
	}
	if max == 0 {
		return nil, io.EOF
	}
	dest := make([]fs.DirEntry, max)
	for i := f.pos; i < f.pos+max; i++ {
		dest[i-f.pos] = f.Dirs[i]
	}
	f.pos += max
	return dest, nil
}

// dirEntry is a special version of fileInfo to represent directory entries.
// It is lightweight in that it isn't as filled out as if you called Stat
// on the file itself.
type dirEntry struct {
	FI fileInfo
}

// IsDir reports whether the entry describes a directory.
func (di dirEntry) IsDir() bool {
	return di.FI.IsDir()
}

// Type returns the type bits for the entry.
// The type bits are a subset of the usual FileMode bits, those returned by the FileMode.Type method.
func (di dirEntry) Type() fs.FileMode {
	return di.FI.Mode().Type()
}

// Info returns the FileInfo for the file or subdirectory described by the entry.
// The returned info is from the time of the directory read and does not contain
// values for ModTime(), Sys(), or Size(). Additionally, the Mode() bits only
// contain the type. This is done to prevent additional reads on the file system
// when a directory is filled out.
func (di dirEntry) Info() (fs.FileInfo, error) {
	return di.FI, nil
}

// Name returns the name of the file (or subdirectory) described by the entry.
// This name is only the final element of the path (the base name), not the entire path.
// For example, Name would return "hello.go" not "home/gopher/hello.go".
func (di dirEntry) Name() string {
	return di.FI.Name()
}
