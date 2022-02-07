package virtual

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"path"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/russross/blackfriday/v2"
)

// newMarkdownFile reads the underlying markdown file, extracts the front matter,
// renders the markdown, and executes the specified template, returning the
// resulting virtualFile.
func (vfs *FS) newMarkdownFile(f fs.File, pathname string) (fs.File, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("newMarkdownFile: %w", err)
	}

	b, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("newMarkdownFile: %w", err)
	}

	fm, r := extractFrontMatter(b)

	var front FrontMatter
	front.Date = fi.ModTime().Local()
	front.Template = "default"
	front.Title = strings.TrimSuffix(fi.Name(), path.Ext(fi.Name()))
	front.OriginalFile = fi.Name()
	if len(fm) > 0 {
		err = toml.Unmarshal(fm, &front)
		if err != nil {
			return nil, fmt.Errorf("newMarkdownFile: %w", err)
		}
	}

	md := template.HTML(blackfriday.Run(r, blackfriday.WithExtensions(blackfriday.CommonExtensions|blackfriday.Footnotes)))

	// prepare template data
	p, bn := path.Split(pathname)
	var data = data{
		FrontMatter: front,
		Page: PageInfo{
			Path:     "/" + p,
			Filename: bn,
		},
		Content: md,
	}

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

	return &virtualFile{
		fi: fileInfo{
			nm: bn,
			sz: int64(wtr.Len()),
			md: fi.Mode(),
			mt: fi.ModTime(),
		},
		reader: bytes.NewReader(wtr.Bytes()),
	}, nil
}

// newImageFile reads the underlying image file, creates front matter,
// and executes the specified template, returning the resulting
// virtualFile.
func (vfs *FS) newImageFile(f fs.File, pathname string) (fs.File, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	// prepare template data
	p, bn := path.Split(pathname)
	var data = data{
		FrontMatter: FrontMatter{
			Title:        strings.TrimSuffix(fi.Name(), path.Ext(fi.Name())),
			Date:         fi.ModTime().Local(),
			Template:     "image",
			OriginalFile: fi.Name(), // allows reference to image in template
		},
		Page: PageInfo{
			Path:     "/" + p,
			Filename: bn,
		},
	}

	// Render the HTML template
	tpl := vfs.getTemplates()
	var wtr bytes.Buffer
	err = tpl.ExecuteTemplate(&wtr, "image", data)
	if err != nil {
		log.Printf("Error executing template: %s", err)
	}

	return &virtualFile{
		fi: fileInfo{
			nm: bn,
			sz: int64(wtr.Len()),
			md: fi.Mode(),
			mt: fi.ModTime(),
		},
		reader: bytes.NewReader(wtr.Bytes()),
	}, nil
}

// newSitemapFile parses the underlying text file as a template, reads the
// directory listing, and executes the template, returning the resulting
// virtualFile.
func (vfs *FS) newSitemapFile(f fs.File, pathname string) (fs.File, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	sitemapTpl, err := template.New("sitemap").ParseFS(vfs.fs, pathname)
	if err != nil {
		return nil, err
	}

	var files []string
	err = fs.WalkDir(vfs, ".", func(path string, d fs.DirEntry, err error) error {
		if err == nil && path != "" {
			if path == "." {
				path = ""
			}
			if d.IsDir() && path != "" {
				path = path + "/"
			}
			if d.Name() != "index.html" {
				files = append(files, path)
			}
		}
		return nil
	})

	var wtr bytes.Buffer
	err = sitemapTpl.ExecuteTemplate(&wtr, "sitemap", files)
	if err != nil {
		wtr.Reset()
		log.Printf("sitemap: %s", err)
	}

	_, bn := path.Split(pathname)
	return &virtualFile{
		fi: fileInfo{
			nm: bn,
			sz: int64(wtr.Len()),
			md: fi.Mode(),
			mt: fi.ModTime(),
		},
		reader: bytes.NewReader(wtr.Bytes()),
	}, nil
}
