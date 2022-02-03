package virtual

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"path"

	"github.com/pelletier/go-toml/v2"
	"github.com/russross/blackfriday/v2"
)

// pipeFile is a file with an adjusted reader.
type pipeFile struct {
	fs.File               // Underling file
	reader  io.ReadCloser // Main ReadCloser to use
}

// Stat returns information about the file.
func (f pipeFile) Stat() (fs.FileInfo, error) {
	return f.File.Stat()
}

// Read reads up to len(b) bytes from the File. It returns the number of bytes read
// and any error encountered. At end of file, Read returns 0, io.EOF.
func (f *pipeFile) Read(b []byte) (int, error) {
	return f.reader.Read(b)
}

// Close closes the file. Cached files are in memory, so this function does nothing.
func (f *pipeFile) Close() error {
	f.reader.Close()
	return f.File.Close()
}

func (vfs *FS) markdownPipe(pathname string, f fs.File) (fs.File, error) {
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("markdownPipe: %w", err)
	}

	fm, r := extractFrontMatter(b)

	md := template.HTML(blackfriday.Run(r, blackfriday.WithExtensions(blackfriday.CommonExtensions|blackfriday.Footnotes)))

	var front FrontMatter
	if len(fm) > 0 {
		err = toml.Unmarshal(fm, &front)
		if err != nil {
			return nil, fmt.Errorf("markdownPipe: %w", err)
		}
	}

	// Check for redirect
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

	// Check date - don't render until date/time is passed
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
	rdr, wtr := io.Pipe()
	go func() {
		defer wtr.Close()
		err := tpl.ExecuteTemplate(wtr, templateName, data)
		if err != nil && !errors.Is(err, io.ErrClosedPipe) {
			log.Printf("Error executing template: %s", err)
		}
	}()

	return &pipeFile{File: f, reader: rdr}, nil
}

func (vfs *FS) imagePipe(pathname string, f fs.File) (fs.File, error) {
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

	// Check date - don't render until date/time is passed
	/*
		if time.Now().Before(data.FrontMatter.Date) {
			notFound(w, r)
			return
		}
	*/

	// Render the HTML template
	tpl := vfs.getTemplates()
	rdr, wtr := io.Pipe()
	go func() {
		defer wtr.Close()
		err := tpl.ExecuteTemplate(wtr, "image", data)
		if err != nil && !errors.Is(err, io.ErrClosedPipe) {
			log.Printf("Error executing template: %s", err)
		}
	}()

	return &pipeFile{File: f, reader: rdr}, nil
}
