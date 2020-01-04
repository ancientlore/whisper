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

// files is a sorted list of files.
type files []file

// Len is part of sort.Interface.
func (f files) Len() int {
	return len(f)
}

// Swap is part of sort.Interface.
func (f files) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (f files) Less(i, j int) bool {
	return !f[i].FrontMatter.Date.Before(f[j].FrontMatter.Date)
}

// dir returns a sorted slice of files and is used in templates.
func dir(folderpath string) []file {
	f, _, err := readDir(folderpath)
	if err != nil {
		log.Printf("dir: %s", err)
		return nil
	}
	return f
}

// readDir returns a sorted slice of files and the max modification time of those files.
func readDir(folderpath string) ([]file, time.Time, error) {
	var maxTime time.Time
	folderpath = "./" + strings.TrimPrefix(folderpath, "/")
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
		if !fi.IsDir() && !containsSpecialFile(fi.Name()) && fi.Name() != "index.md" {
			itm := file{
				Filename: fi.Name(),
				FrontMatter: frontMatter{
					Title: fi.Name(),
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
	sort.Sort(files(r))
	return r, maxTime, nil
}
