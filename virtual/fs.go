package virtual

import (
	"errors"
	"html/template"
	"io/fs"
	"strings"
	"sync"
)

// FS provides a virtual view of the file system suitable for serving Markdown
// and other files in a web format.
type FS struct {
	fs       fs.FS
	tpl      *template.Template
	tplMutex sync.RWMutex
}

// New returns a new FS that presents a virtual view of innerFS.
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
	/*
		What Open needs to do:
		- Ignore "whisper.cfg" at the root. When someone needs configuration, they should call Config()
		- Ignore the "template" directory at the root. These are for templates.
		- Ignore files and directories that start with ".".
		- When an endpoint like /foo/bar is called and bar is a folder, look for an index.md file and process with the specified template, or "default".
		- When an endpoint like /foo/bar is called that does not exist, look for files in this order:
			- bar.md - process the file using the default template (or named index in front matter) and return the HTML.
			- bar.png, bar.jpg, bar.gif, bar.jpeg - process the file using the "image" template.
		- If the file is named "sitemap.txt" in the root, process the sitemap template.
		- Otherwise serve the file as-is.
	*/
	// Make sure the path is valid per fs rules
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	// Don't show hidden or special files
	if isHiddenFile(name) || (name != "." && containsSpecialFile(name)) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}

	// open the file with the underlying file system
	f, err := vfs.fs.Open(name)
	if err != nil {
		// for files that don't exist, check for underlying matching files
		if errors.Is(err, fs.ErrNotExist) {
			extensions := []string{".md", ".png", ".jpg", ".git", ".jpeg"}
			// if it's not in an image folder, only check markdown files
			if !hasImageFolderPrefix(name) {
				extensions = extensions[:1]
			}
			// find file with matching extension
			for _, ext := range extensions {
				f, err2 := vfs.fs.Open(name + ext)
				if err2 == nil {
					// match found, so return a virtual file
					return &virtualFile{File: f, name: name}, nil
				}
			}
		}
		// no matching underlying file; return error from opening the underlying file
		return f, err
	}
	// check for directory
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	// Directories need to be virtual so that we don't
	// accidentally pick up the wrong ReadDir implementation.
	if fi.IsDir() {
		return &virtualFile{name: name, File: f}, nil
	}
	return f, nil
}

// ReadDir reads the named directory
// and returns a list of directory entries sorted by filename.
func (vfs *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, err := fs.ReadDir(vfs.fs, name)
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
		case hasImageExtension(nm) && hasImageFolderPrefix(name):
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
