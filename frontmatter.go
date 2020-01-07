package main

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
	"time"

	"github.com/pelletier/go-toml"
)

// frontMatter holds data scraped from a Markdown page.
type frontMatter struct {
	Title    string        `toml:"title" comment:"Title of this page"`
	Date     time.Time     `toml:"date" comment:"Date the article appears"`
	Template string        `toml:"template" comment:"The name of the template to use"`
	Tags     []string      `toml:"tags" comment:"Tags to assign to this article"`
	Expires  time.Duration `toml:"expires" comment:"Use for pages that need an Expires header"`
	Redirect string        `toml:"redirect" comment:"Issue a redirect to another location"`
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
func readFrontMatter(name string, fm *frontMatter) error {
	b, err := ioutil.ReadFile(name)
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
