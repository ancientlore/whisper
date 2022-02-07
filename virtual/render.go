package virtual

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"path"
	"sort"
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
			if d.Name() != "index.html" && d.Name() != "404.html" && d.Name() != "500.html" {
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

func (vfs *FS) newDirectory(f fs.File, pathname string) (fs.File, error) {
	rdf, ok := f.(fs.ReadDirFile)
	if !ok {
		return nil, &fs.PathError{Op: "open", Err: fmt.Errorf("Not a directory: %w", fs.ErrInvalid)}
	}

	fi, err := rdf.Stat()
	if err != nil {
		return nil, err
	}

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
			newNm := strings.TrimSuffix(nm, ".md") + ".html"
			if _, ok := added[newNm]; !ok {
				// TODO: info doesn't have the right size because data will be transformed
				vEntries = append(vEntries, fs.FileInfoToDirEntry(fileInfo{nm: newNm, sz: info.Size(), md: info.Mode(), mt: info.ModTime()}))
				added[newNm] = true
			}
		case hasImageExtension(nm) && hasImageFolderPrefix(pathname):
			info, err := entry.Info()
			if err != nil {
				return nil, err
			}
			a := strings.Split(nm, ".")
			newNm := strings.TrimSuffix(nm, "."+a[len(a)-1]) + ".html"
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

	return &virtualDir{
		fi: fileInfo{
			nm: fi.Name(),
			sz: fi.Size(),
			md: fi.Mode(),
			mt: fi.ModTime(),
		},
		entries: vEntries,
		pos:     0,
	}, nil
}
