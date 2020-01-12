package main

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

var sitemapTpl *template.Template

// loadSitemapTemplate loads the /sitemap.txt template,
// returning true if it exists.
func loadSitemapTemplate() (bool, error) {
	var err error
	sitemapTpl, err = template.New("sitemap").ParseFiles("./sitemap.txt")
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

// sitemap is an http.HandlerFunc that renders the site map.
func sitemap(w http.ResponseWriter, r *http.Request) {
	if sitemapTpl == nil {
		notFound(w, r)
		return
	}
	files, modTime, err := loadSitemapFiles()
	if err != nil {
		log.Printf("sitemap: %s", err)
		serverError(w, r, err.Error())
		return
	}
	_, tplModTime := getTemplates()
	if tplModTime.After(modTime) {
		modTime = tplModTime
	}
	var out bytes.Buffer
	err = sitemapTpl.ExecuteTemplate(&out, "sitemap", files)
	if err != nil {
		log.Printf("sitemap: %s", err)
		serverError(w, r, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, "sitemap.txt", modTime, bytes.NewReader(out.Bytes()))
}

// loadSitemap reads the sitemap files.
func loadSitemapFiles() ([]string, time.Time, error) {
	var (
		result  []string
		maxTime time.Time
	)
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("loadSitemap: %s", err)
			return err
		}
		if info.IsDir() {
			if info.Name() == "." {
				return nil
			}
			if containsSpecialFile(path) || info.Name() == "template" {
				return filepath.SkipDir
			}
			return nil
		}
		if containsSpecialFile(path) {
			return nil
		}
		var fm frontMatter
		if info.Name() == "index.md" {
			err = readFrontMatter(path, &fm)
			if err != nil {
				log.Printf("loadSitemapFiles: %s", err)
			}
			path = strings.TrimSuffix(path, "index.md")
		}
		if strings.HasSuffix(path, ".md") {
			err = readFrontMatter(path, &fm)
			if err != nil {
				log.Printf("loadSitemapFiles: %s", err)
			}
			path = strings.TrimSuffix(path, ".md")
		}
		if info.ModTime().After(maxTime) {
			maxTime = info.ModTime()
		}
		if fm.Date.Before(time.Now()) {
			r := filepath.ToSlash(path)
			if !alreadyAdded(result, r) {
				result = append(result, r)
			}
			if hasImageFolderPrefix(r) && hasImageExtention(r) {
				ext := filepath.Ext(r)
				r = strings.TrimSuffix(r, ext)
				if !alreadyAdded(result, r) {
					result = append(result, r)
				}
			}
		}
		return nil
	})
	return result, maxTime, err
}

func alreadyAdded(arr []string, r string) bool {
	c := 0
	for i := len(arr) - 1; i >= 0 && c < 4; i-- {
		if arr[i] == r {
			return true
		}
		c++
	}
	return false
}

func hasImageFolderPrefix(s string) bool {
	imageFolders := []string{"photos", "images", "pictures", "cartoons", "toons", `sketches`, `artwork`, `drawings`}
	for _, f := range imageFolders {
		if strings.HasPrefix(s, f) {
			return true
		}
	}
	return false
}

func hasImageExtention(s string) bool {
	imageTypes := []string{".png", ".jpg", ".gif", ".jpeg"}
	for _, ext := range imageTypes {
		if strings.HasSuffix(s, ext) {
			return true
		}
	}
	return false
}
