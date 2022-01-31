package virtual

import (
	"fmt"
	"io/fs"
	"regexp"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// FrontMatter holds data scraped from a Markdown page.
type FrontMatter struct {
	Title    string    `toml:"title"`    // Title of this page
	Date     time.Time `toml:"date"`     // Date the article appears
	Template string    `toml:"template"` // The name of the template to use
	Tags     []string  `toml:"tags"`     // Tags to assign to this article
	Expires  Duration  `toml:"expires"`  // Use for pages that need an Expires header
	Redirect string    `toml:"redirect"` // Issue a redirect to another location
}

// fmRegexp is the regular expression used to split out front matter.
var fmRegexp = regexp.MustCompile(`(?m)^\s*\+\+\+\s*$`)

// extractFrontMatter splits the front matter and Markdown content.
func extractFrontMatter(x []byte) (fm, r []byte) {
	subs := fmRegexp.Split(string(x), 3)
	if len(subs) != 3 {
		return nil, x
	}
	if s := strings.TrimSpace(subs[0]); len(s) > 0 {
		return nil, x
	}
	return []byte(strings.TrimSpace(subs[1])), []byte(strings.TrimSpace(subs[2]))
}

// readFrontMatter extracts and unmarshals front matter from the given file.
func (vfs *FS) readFrontMatter(name string, fm *FrontMatter) error {
	b, err := fs.ReadFile(vfs.fs, name)
	if err != nil {
		return fmt.Errorf("readFrontMatter: %w", err)
	}
	fmb, _ := extractFrontMatter(b)
	err = toml.Unmarshal(fmb, fm)
	if err != nil {
		return fmt.Errorf("readFrontMatter: %w", err)
	}
	return nil
}
