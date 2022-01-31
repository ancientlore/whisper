package virtual

import (
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"path"
	"strings"
	"time"
)

//go:embed default.html
var defaultTemplate string

// pageInfo has information about the current page.
type pageInfo struct {
	Path     string // path from URL
	Filename string // end portion (file) from URL
}

// Pathname joins the path and filename.
func (p pageInfo) Pathname() string {
	return path.Join(p.Path, p.Filename)
}

// data is what is passed to markdown templates.
type data struct {
	FrontMatter FrontMatter   // front matter from Markdown file or defaults
	Page        pageInfo      // information aboout current page
	Content     template.HTML // rendered Markdown
}

// getTemplates returns the templates and last time they were modified.
func (vfs *FS) getTemplates() *template.Template {
	vfs.tplMutex.RLock()
	defer vfs.tplMutex.RUnlock()
	return vfs.tpl
}

// loadTemplates loads and parses the HTML templates, returning true if custom templates were found.
func (vfs *FS) loadTemplates() (bool, error) {
	var err error
	funcMap := template.FuncMap{
		"dir":         vfs.dir,
		"sortbyname":  sortByName,
		"sortbytime":  sortByTime,
		"match":       match,
		"filter":      filter,
		"join":        path.Join,
		"ext":         path.Ext,
		"prev":        prev,
		"next":        next,
		"reverse":     reverse,
		"trimsuffix":  strings.TrimSuffix,
		"trimprefix":  strings.TrimPrefix,
		"trimspace":   strings.TrimSpace,
		"markdown":    vfs.md,
		"frontmatter": vfs.fm,
		"now":         time.Now,
	}
	vfs.tplMutex.Lock()
	defer vfs.tplMutex.Unlock()
	// Check if we are using default templates
	fi, err := fs.Stat(vfs.fs, "template")
	if errors.Is(err, fs.ErrNotExist) || (err == nil && !fi.IsDir()) {
		tpl, err := template.New("whisper").Funcs(funcMap).Parse(defaultTemplate)
		if err != nil {
			return false, fmt.Errorf("loadTemplates: %w", err)
		}
		vfs.tpl = tpl
		return false, nil
	}
	// use custom templates
	tpl, err := template.New("whisper").Funcs(funcMap).ParseFS(vfs.fs, "template/*.html")
	if err != nil {
		return true, fmt.Errorf("loadTemplates: %w", err)
	}
	vfs.tpl = tpl
	return true, nil
}
