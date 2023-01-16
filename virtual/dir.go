package virtual

import (
	"errors"
	"io/fs"
	"log"
	"path"
	"sort"
	"strings"
)

// File holds data about a page endpoint.
type File struct {
	FrontMatter FrontMatter
	Filename    string
}

// dir returns a sorted slice of files and is used in templates.
func (vfs *FS) dir(folderpath string) []File {
	folderpath = "./" + strings.TrimPrefix(folderpath, "/")
	folderpath = path.Clean(folderpath)
	entries, err := fs.ReadDir(vfs, folderpath)
	if err != nil {
		log.Printf("dir: %s", err)
		return nil
	}
	f := make([]File, 0, len(entries))
	for _, entry := range entries {
		if entry.Name() != "index.html" && entry.Name() != "404.html" && entry.Name() != "500.html" {
			fm := FrontMatter{
				Title: strings.TrimSuffix(entry.Name(), path.Ext(entry.Name())),
			}
			fi, err := entry.Info()
			if err == nil {
				fm.Date = fi.ModTime().Local()
			}
			if !entry.IsDir() && path.Ext(entry.Name()) == ".html" {
				err = vfs.readFrontMatter(path.Join(folderpath, strings.TrimSuffix(entry.Name(), ".html")+".md"), &fm)
				if err != nil {
					if !errors.Is(err, fs.ErrNotExist) {
						log.Printf("readDir: %s", err)
					} else if hasImageFolderPrefix(folderpath) {
						extensions := []string{".png", ".jpg", ".gif", ".webp", ".jpeg"}
						newNm := strings.TrimSuffix(entry.Name(), path.Ext(entry.Name()))
						// find file with matching extension
						for _, ext := range extensions {
							_, err = fs.Stat(vfs, path.Join(folderpath, newNm+ext))
							if err == nil {
								fm.OriginalFile = newNm + ext
								break
							}
						}
					}
				}
			}
			f = append(f, File{FrontMatter: fm, Filename: entry.Name()})
		}
	}
	return f
}

// sortByTime sorts the files by the time in reverse order
func sortByTime(f []File) []File {
	sort.Slice(f, func(i, j int) bool { return f[j].FrontMatter.Date.Before(f[i].FrontMatter.Date) })
	return f
}

// sortByName sorts the files by the time in reverse order
func sortByName(f []File) []File {
	sort.Slice(f, func(i, j int) bool { return f[j].Filename < f[i].Filename })
	return f
}

// reverse reverses the order of the file list.
func reverse(f []File) []File {
	j := len(f) - 1
	for i := 0; i < len(f)/2; i++ {
		f[i], f[j] = f[j], f[i]
		j--
	}
	return f
}

// filter trims out non-matching files based on name.
func filter(f []File, pat ...string) []File {
	var r []File
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
func next(f []File, current string) *File {
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
func prev(f []File, current string) *File {
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
