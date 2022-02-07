/*
virtual implements a "virtual" view over a fs.FS that makes it suitable for serving Markdown
and other files in a web format. It includes a template system and helpers for presenting
a nice static web view and an easy-to-maintain format.

A special file "whisper.cfg" at the root exposes settings you can use via the Config() function.
This file is hidden from view.

A special folder "template" at the root holds HTML templates should you want to customize. At
minimum, a template called "default" is required for handling Markdown files, and a template
called "image" is required for handling image files.

Hidden files and folders (those starting with ".") are ignored.

If the above conditions are not met, then the file is provided as-is from the underline file system.

Special File Handling

When an endpoint like "/foo/bar.html" is called and it does not exist, the virtual file system first looks for
a Markdown file named "/foo/bar.md". If present, a "virtual" file "/foo/bar.html" is presented that will
render the underlying Markdown file into HTML. In this case, the underlying Markdowndown file,
"/foo/bar.md", is hidden from view outside of the file system. By default, a template called "default"
is used to render the Markdown, unless the front matter of the file specifies a different template.

If a Markdown file is not found, the system will look for an image file (PNG, JPG, and GIF). If an image
file is found, a virtual file "/foo/bar.html" is created that will render an HTML file using the "image" template.
The underlying image file is not hidden, because it needs to be served for the HTML. Note that this
special image handling only happens when the top-level folder is one of the following:

	"photos", "images", "pictures", "cartoons", "toons", "sketches", "artwork", "drawings"

Site Map

If a file in the root names "sitemap.txt" is present, it will be run as template that can list the files
of the site map. This allows you to customize what your site map looks like. The site map receives only
the list of file names as a slice of strings.

Front Matter

Markdown files may contain front matter which is in TOML format. The front matter is delimited by "+++"" at
the start and end. For example:

    +++
    # This is my front matter
    title = "My glorious page"
    +++
    # This is my Heading
    This is my [Markdown](https://en.wikipedia.org/wiki/Markdown).

Front matter may include:

	Name       Type                  Description
	---------  -----------------     -----------------------------------------
	title         string             Title of page
	date          time               Publish date
	tags          array of strings   Tags for the articles (not used yet)
	template      string             Override the template to render this file
	redirect      string             You can use this to issue an HTML meta-tag redirect
	originalfile  string             The original filename (markdown or image)

Templates

The system uses standard Go templates from the `html/template` package, and includes two default templates,
"default" and "image". Templates are stored in the "template" top-level folder with the extension ".html".

Templates are passed page information (virtual.PageInfo), front matter (virtual.FrontMatter), and rendered HTML from
Markdown (template.HTML), and can use these data elements in their processing. Template also make
the following helper functions available:

	dir(path string) []virtual.File
		Return the contents of the given folder, excluding special files and subfolders
	sortbyname([]virtual.File) []virtual.File
		Sort by name (reverse)
	sortbytime([]virtual.File) []virtual.File
		Sort by time (reverse)
	match(string, ...string) bool
		Match string against file patterns
	filter([]virtual.File, ...string) []virtual.File
		Filter list against file patterns
	join(parts ...string) string
		The same as path.Join
	ext(path string) string
		The same as path.Ext
	prev([]virtual.File, string) *virtual.File
		Find the previous file based on Filename
	next([]virtual.File, string) *virtual.File
		Find the next file based on Filename
	reverse([]virtual.File) []virtual.File
		Reverse the list
	trimsuffix(string, string) string
		The same as strings.TrimSuffix
	trimprefix(string, string) string
		The same as strings.TrimPrefix
	trimspace(string) string
		The same as strings.TrimSpace
	markdown(string) template.HTML
		Render Markdown file into HTML
	frontmatter(string) *virtual.FrontMatter
		Read front matter from file
	now() time.Time
		Current time

Index Files

Most web servers will want to provide an "index.html" file to handle folder roots (like "/articles"). This is
handled automatically when using things like http.FileServer if you simply create an "index.md" file to
render the "index.html" in the folder.

Errors

To assist web implementations that want to serve a custom file for 404 or 500 errors, you can create
a 404.md and 500.md files in the root of the file system. Although the file system will expose these
files through the normal "fs" package operations, the sitemap and "dir" template function will not
show them, making it straightforward to design your site. The web implementation can request 404.html
or 500.html to be served when the file system returns an error or fs.ErrNotExist.
*/
package virtual

import (
	"errors"
	"html/template"
	"io/fs"
	"path"
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
func New(innerFS fs.FS) (*FS, error) {
	var vfs = FS{
		fs: innerFS,
	}
	_, err := vfs.loadTemplates()
	if err != nil {
		return nil, err
	}

	return &vfs, nil
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
		if errors.Is(err, fs.ErrNotExist) && path.Ext(name) == ".html" {
			extensions := []string{".md", ".png", ".jpg", ".git", ".jpeg"}
			// if it's not in an image folder, only check markdown files
			if !hasImageFolderPrefix(name) {
				extensions = extensions[:1]
			}
			newNm := strings.TrimSuffix(name, path.Ext(name))
			// find file with matching extension
			for _, ext := range extensions {
				f, err2 := vfs.fs.Open(newNm + ext)
				if err2 == nil {
					// match found, so return a virtual file
					defer f.Close()
					if ext == ".md" {
						return vfs.newMarkdownFile(f, newNm+".html")
					} else {
						return vfs.newImageFile(f, newNm+".html")
					}
				}
			}
		}
		// no matching underlying file; return error from opening the underlying file
		return f, err
	}
	// check for directory
	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	// Directories need to be virtual so that we don't
	// accidentally pick up the wrong ReadDir implementation.
	if fi.IsDir() {
		// don't close f because it will be used for ReadDir
		return &virtualDir{File: f, path: name}, nil
	}
	// The sitemap file, if present, needs to be handled as a virtual
	// file to process the template.
	if name == "sitemap.txt" {
		defer f.Close()
		return vfs.newSitemapFile(f, name)
	}
	return f, nil
}
