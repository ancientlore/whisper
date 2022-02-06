package virtual

import (
	"fmt"
	"io"
	"io/fs"
	"sort"
	"strings"
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
	fs.File // Underling file

	path string // directory path (needed in ReadDir)
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
	rdf, ok := f.File.(fs.ReadDirFile)
	if !ok {
		return nil, &fs.PathError{Op: "readdir", Err: fmt.Errorf("Not a directory: %w", fs.ErrInvalid)}
	}

	// TODO: need to honor the value of n
	entries, err := rdf.ReadDir(0)
	if err != nil {
		return nil, err
	}
	var vEntries []fs.DirEntry
	if len(entries) > 0 {
		vEntries = make([]fs.DirEntry, 0, len(entries))
	}
	added := make(map[string]bool)
	for _, entry := range entries {
		nm := entry.Name()
		switch {
		case containsSpecialFile(nm):
			continue
		case isHiddenFile(nm):
			continue
		case strings.HasSuffix(nm, ".md"):
			info, err := entry.Info()
			if err != nil {
				return nil, err
			}
			// new version hides the markdown
			newNm := strings.TrimSuffix(nm, ".md")
			if _, ok := added[newNm]; !ok {
				// TODO: info doesn't have the right size because data will be transformed
				vEntries = append(vEntries, fs.FileInfoToDirEntry(fileInfo{nm: newNm, sz: info.Size(), md: info.Mode(), mt: info.ModTime()}))
				added[newNm] = true
			}
		case hasImageExtension(nm) && hasImageFolderPrefix(f.path):
			info, err := entry.Info()
			if err != nil {
				return nil, err
			}
			a := strings.Split(nm, ".")
			newNm := strings.TrimSuffix(nm, "."+a[len(a)-1])
			if _, ok := added[newNm]; !ok {
				// TODO: info doesn't have the right size because data will be transformed
				vEntries = append(vEntries, fs.FileInfoToDirEntry(fileInfo{nm: newNm, sz: info.Size(), md: info.Mode(), mt: info.ModTime()}))
				added[newNm] = true
			}
			vEntries = append(vEntries, entry)
		case nm == "sitemap.txt":
			info, err := entry.Info()
			if err != nil {
				return nil, err
			}
			// TODO: info doesn't have the right size because data will be transformed
			vEntries = append(vEntries, fs.FileInfoToDirEntry(fileInfo{nm: nm, sz: info.Size(), md: info.Mode(), mt: info.ModTime()}))
		default:
			// Check name just in case of collisions
			if _, ok := added[nm]; !ok {
				vEntries = append(vEntries, entry)
				added[nm] = true
			}
		}
	}
	// Sort by filename
	sort.Slice(vEntries, func(i, j int) bool {
		return vEntries[i].Name() < vEntries[j].Name()
	})
	return vEntries, nil
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
