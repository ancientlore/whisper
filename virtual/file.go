package virtual

import (
	"fmt"
	"io/fs"
	"strings"
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
func (f *virtualFile) ReadDir(n int) ([]fs.DirEntry, error) {
	rdf, ok := f.File.(fs.ReadDirFile)
	if !ok {
		return nil, &fs.PathError{Op: "readdir", Err: fmt.Errorf("Not a directory: %w", fs.ErrInvalid)}
	}

	// todo: need to honor the value of n
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
				vEntries = append(vEntries, virtualDirEntry{virtualFileInfo: virtualFileInfo{name: newNm, FileInfo: info}})
				added[newNm] = true
			}
		case hasImageExtension(nm): // TODO: Need to honor && hasImageFolderPrefix(name):
			info, err := entry.Info()
			if err != nil {
				return nil, err
			}
			a := strings.Split(nm, ".")
			newNm := strings.TrimSuffix(nm, "."+a[len(a)-1])
			if _, ok := added[newNm]; !ok {
				vEntries = append(vEntries, virtualDirEntry{virtualFileInfo: virtualFileInfo{name: newNm, FileInfo: info}})
				added[newNm] = true
			}
			vEntries = append(vEntries, entry)
		case nm == "sitemap.txt":
			info, err := entry.Info()
			if err != nil {
				return nil, err
			}
			vEntries = append(vEntries, virtualDirEntry{virtualFileInfo: virtualFileInfo{name: nm, FileInfo: info}})
		default:
			vEntries = append(vEntries, entry)
		}
	}
	return vEntries, nil
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
