package virtual

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"path"

	"github.com/pelletier/go-toml/v2"
	"github.com/russross/blackfriday/v2"
)

// renderFile is a specialization of virtualFile for customer rendering.
type renderFile struct {
	virtualFile

	reader io.ReadSeeker // Main Reader to use
	size   int64         // Length of data
}

// Stat returns a FileInfo describing the file.
func (f *renderFile) Stat() (fs.FileInfo, error) {
	fi, err := f.virtualFile.Stat()
	if err != nil {
		return nil, err
	}
	return renderFileInfo{FileInfo: fi, size: f.size}, nil
}

// Read reads up to len(b) bytes from the File. It returns the number of bytes read
// and any error encountered. At end of file, Read returns 0, io.EOF.
func (f *renderFile) Read(b []byte) (int, error) {
	return f.reader.Read(b)
}

// Seek sets the offset for the next Read or Write to offset, interpreted according
// to whence: io.SeekStart means relative to the start of the file, io.SeekCurrent
// means relative to the current offset, and io.SeekEnd means relative to the end.
// Seek returns the new offset relative to the start of the file and an error, if any.
//
// Seeking to an offset before the start of the file is an error. Seeking to any
// positive offset is legal, but the behavior of subsequent I/O operations on the
// underlying object is implementation-dependent.
func (f *renderFile) Seek(offset int64, whence int) (int64, error) {
	return f.reader.Seek(offset, whence)
}

// renderFileInfo holds the metadata about the file and allows you
// to customize the size, which is important for reporting the
// length of the rendered data.
type renderFileInfo struct {
	fs.FileInfo

	size int64 // Size of file data
}

// Size reports the length of the file.
func (rfi renderFileInfo) Size() int64 {
	return rfi.size
}

// newMarkdownFile reads the underlying markdown file, extracts the front matter,
// renders the markdown, and executes the specified template, returning the
// resulting renderFile.
func (vfs *FS) newMarkdownFile(f fs.File, pathname string) (fs.File, error) {
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("newMarkdownFile: %w", err)
	}

	fm, r := extractFrontMatter(b)

	var front FrontMatter
	if len(fm) > 0 {
		err = toml.Unmarshal(fm, &front)
		if err != nil {
			return nil, fmt.Errorf("newMarkdownFile: %w", err)
		}
	}

	md := template.HTML(blackfriday.Run(r, blackfriday.WithExtensions(blackfriday.CommonExtensions|blackfriday.Footnotes)))

	// TODO: Check for redirect
	/*
		if front.Redirect != "" {
			return
		}
	*/

	// prepare template data
	p, bn := path.Split(pathname)
	var data = data{
		FrontMatter: front,
		Page: PageInfo{
			Path:     p,
			Filename: bn,
		},
		Content: md,
	}

	// TODO: Check date - don't render until date/time is passed
	/*
		if time.Now().Before(data.FrontMatter.Date) {
			notFound(w, r)
			return
		}
	*/

	// Render the HTML template
	templateName := "default"
	if data.FrontMatter.Template != "" {
		templateName = data.FrontMatter.Template
	}
	tpl := vfs.getTemplates()
	var wtr bytes.Buffer
	err = tpl.ExecuteTemplate(&wtr, templateName, data)
	if err != nil {
		log.Printf("Error executing template: %s", err)
	}

	return &renderFile{
		virtualFile: virtualFile{
			File: f,
			name: bn,
		},
		reader: bytes.NewReader(wtr.Bytes()),
		size:   int64(wtr.Len()),
	}, nil
}

// newImageFile reads the underlying image file, creates front matter,
// and executes the specified template, returning the resulting
// renderFile.
func (vfs *FS) newImageFile(f fs.File, pathname string) (fs.File, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	// prepare template data
	p, bn := path.Split(pathname)
	var data = data{
		FrontMatter: FrontMatter{
			Title: bn,
			Date:  fi.ModTime(),
		},
		Page: PageInfo{
			Path:     p,
			Filename: fi.Name(),
		},
	}

	// TODO: Check date - don't render until date/time is passed
	/*
		if time.Now().Before(data.FrontMatter.Date) {
			notFound(w, r)
			return
		}
	*/

	// Render the HTML template
	tpl := vfs.getTemplates()
	var wtr bytes.Buffer
	err = tpl.ExecuteTemplate(&wtr, "image", data)
	if err != nil {
		log.Printf("Error executing template: %s", err)
	}

	return &renderFile{
		virtualFile: virtualFile{
			File: f,
			name: bn,
		},
		reader: bytes.NewReader(wtr.Bytes()),
		size:   int64(wtr.Len()),
	}, nil
}

// newSitemapFile parses the underlying text file as a template, reads the
// directory listing, and executes the template, returning the resulting
// renderFile.
func (vfs *FS) newSitemapFile(f fs.File, pathname string) (fs.File, error) {
	_, bn := path.Split(pathname)
	var wtr bytes.Buffer
	// TODO: implement
	wtr.WriteString("sitemap.txt")
	return &renderFile{
		virtualFile: virtualFile{
			File: f,
			name: bn,
		},
		reader: bytes.NewReader(wtr.Bytes()),
		size:   int64(wtr.Len()),
	}, nil
}
