package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)

// file holds data about a page endpoint.
type file struct {
	FrontMatter frontMatter
	Filename    string
}

// filesByTime is a sorted list of files.
type filesByTime []file

// Len is part of sort.Interface.
func (f filesByTime) Len() int {
	return len(f)
}

// Swap is part of sort.Interface.
func (f filesByTime) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (f filesByTime) Less(i, j int) bool {
	return !f[i].FrontMatter.Date.Before(f[j].FrontMatter.Date)
}

// filesByName enabled sorting by file name.
type filesByName []file

// Len is part of sort.Interface.
func (f filesByName) Len() int {
	return len(f)
}

// Swap is part of sort.Interface.
func (f filesByName) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (f filesByName) Less(i, j int) bool {
	return strings.Compare(f[i].Filename, f[j].Filename) > 0
}

// dir returns a sorted slice of files and is used in templates.
func dir(folderpath string) []file {
	f, _, err := cachedReadDir(folderpath)
	if err != nil {
		log.Printf("dir: %s", err)
		return nil
	}
	return f
}

// sortByName sorts the files by the time in reverse order
func sortByTime(f []file) []file {
	sort.Sort(filesByTime(f))
	return f
}

// sortByName sorts the files by the time in reverse order
func sortByName(f []file) []file {
	sort.Sort(filesByName(f))
	return f
}

// reverse reverses the order of the file list.
func reverse(f []file) []file {
	j := len(f) - 1
	for i := 0; i < len(f)/2; i++ {
		f[i], f[j] = f[j], f[i]
		j--
	}
	return f
}

// filter trims out non-matching files based on name.
func filter(f []file, pat ...string) []file {
	var r []file
	for i := range f {
		if match(f[i].Filename, pat...) {
			r = append(r, f[i])
		}
	}
	return r
}

// match uses path.Match to test for a match.
func match(s string, pat ...string) bool {
	for i := range pat {
		b, err := path.Match(pat[i], s)
		if err != nil {
			log.Printf("match: %s", err)
		}
		if b {
			return true
		}
	}
	return false
}

// next returns the previous file in the list.
func next(f []file, current string) *file {
	for i := range f {
		if f[i].Filename == current {
			if i > 0 {
				return &f[i-1]
			}
			return nil
		}
	}
	return nil
}

// prev returns the previous file in the list.
func prev(f []file, current string) *file {
	for i := range f {
		if f[i].Filename == current {
			if i < len(f)-1 {
				return &f[i+1]
			}
			return nil
		}
	}
	return nil
}

// readDir returns a sorted slice of files and the max modification time of those files.
func readDir(folderpath string) ([]file, time.Time, error) {
	var maxTime time.Time
	folderpath = "./" + strings.TrimPrefix(folderpath, "/")
	folderpath = path.Clean(folderpath)
	f, err := os.Open(folderpath)
	if err != nil {
		return nil, maxTime, fmt.Errorf("readDir: %w", err)
	}
	defer f.Close()
	arr, err := f.Readdir(0)
	if err != nil {
		return nil, maxTime, fmt.Errorf("readDir: %w", err)
	}
	var r []file
	for _, fi := range arr {
		if !fi.IsDir() && !containsSpecialFile(fi.Name()) && fi.Name() != "index.md" && fi.Name() != "whisper.cfg" {
			itm := file{
				Filename: fi.Name(),
				FrontMatter: frontMatter{
					Title: strings.TrimSuffix(fi.Name(), path.Ext(fi.Name())),
					Date:  fi.ModTime(),
				},
			}
			if strings.HasSuffix(itm.Filename, ".md") {
				itm.Filename = strings.TrimSuffix(itm.Filename, ".md")
				itm.FrontMatter.Title = strings.TrimSuffix(itm.FrontMatter.Title, ".md")
				err = readFrontMatter(path.Join(folderpath, fi.Name()), &itm.FrontMatter)
				if err != nil {
					log.Printf("readDir: %s", err)
				}
			}
			if itm.FrontMatter.Date.Before(time.Now()) {
				if fi.ModTime().After(maxTime) {
					maxTime = fi.ModTime()
				}
				r = append(r, itm)
			}
		}
	}
	return r, maxTime, nil
}
