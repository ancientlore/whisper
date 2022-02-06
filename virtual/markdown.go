package virtual

import (
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"path"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/russross/blackfriday/v2"
)

// pathToMarkdown takes a URL path and converts it into the path to the associated Markdown file.
func pathToMarkdown(filename string) string {
	// check for folder - if so, add index.md
	if strings.HasSuffix(filename, "/") {
		filename += "index.md"
	}
	filename = path.Clean(filename)
	// removing leading / so we find it on the file system
	filename = strings.TrimPrefix(filename, "/")
	// make sure the extension is present
	if path.Ext(filename) == "" {
		filename += ".md"
	}
	return filename
}

// renderMarkdown renders the markdown for the given file and returns the frontmatter.
func (vfs *FS) renderMarkdown(filename string) (*FrontMatter, template.HTML, time.Time, error) {
	var (
		fmData  FrontMatter
		md      template.HTML
		modTime time.Time
	)
	filename = pathToMarkdown(filename)
	s, err := fs.Stat(vfs.fs, filename)
	if err != nil {
		return nil, "", modTime, fmt.Errorf("renderMarkdown: %w", err)
	}
	b, err := fs.ReadFile(vfs.fs, filename)
	if err != nil {
		return nil, "", modTime, fmt.Errorf("renderMarkdown: %w", err)
	}
	fm, r := extractFrontMatter(b)
	md = template.HTML(blackfriday.Run(r, blackfriday.WithExtensions(blackfriday.CommonExtensions|blackfriday.Footnotes)))
	if len(fm) > 0 {
		err = toml.Unmarshal(fm, &fmData)
		if err != nil {
			return nil, "", modTime, fmt.Errorf("renderMarkdown: %w", err)
		}
	}
	return &fmData, md, s.ModTime(), nil
}

// md convert the given markdown file to HTML and is used in templates.
func (vfs *FS) md(filename string) template.HTML {
	_, md, _, err := vfs.renderMarkdown(filename)
	if err != nil {
		log.Printf("md: %s", err)
		return ""
	}
	return md
}

// fm returns front matter for the given file and is used in templates.
func (vfs *FS) fm(filename string) *FrontMatter {
	fm, _, _, err := vfs.renderMarkdown(filename)
	if err != nil {
		log.Printf("fm: %s", err)
		return nil
	}
	return fm
}
