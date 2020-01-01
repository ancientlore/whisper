package main

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/pelletier/go-toml"
)

type file struct {
	FrontMatter frontMatter
	Filename    string
}

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

func dir(folderpath string) []file {
	folderpath = "./" + strings.TrimPrefix(folderpath, "/")
	f, err := os.Open(folderpath)
	if err != nil {
		log.Printf("dir: %s", err)
		return nil
	}
	defer f.Close()
	arr, err := f.Readdir(0)
	if err != nil {
		log.Printf("dir: %s", err)
		return nil
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
				err = readFrontMatter(path.Join(folderpath, fi.Name()), &itm.FrontMatter)
				if err != nil {
					log.Printf("readFrontMatter: %s", err)
				}
			}
			if itm.FrontMatter.Date.Before(time.Now()) {
				r = append(r, itm)
			}
		}
	}
	sort.Sort(files(r))
	return r
}

func readFrontMatter(name string, fm *frontMatter) error {
	b, err := ioutil.ReadFile(name)
	if err != nil {
		return err
	}
	fmb, _ := extractFrontMatter(b)
	err = toml.Unmarshal(fmb, fm)
	if err != nil {
		return err
	}
	return nil
}
