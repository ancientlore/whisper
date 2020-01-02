package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var sitemapTpl *template.Template

func loadSitemap() {
	var err error
	sitemapTpl, err = template.New("sitemap").ParseFiles("./sitemap.txt")
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("Unable to load sitemap.txt template: %s", err)
			os.Exit(3)
		}
		log.Print("No sitemap.txt template found.")
		return
	}
	log.Print("Loaded sitemap.txt template.")
}

// sitemap is an http.HandlerFunc that renders the site map.
func sitemap(w http.ResponseWriter, r *http.Request) {
	if sitemapTpl == nil {
		notFound(w, r)
		return
	}
	files, err := loadSitemapFiles()
	if err != nil {
		log.Printf("sitemap: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	err = sitemapTpl.ExecuteTemplate(w, "sitemap", files)
	if err != nil {
		log.Printf("sitemap: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// loadSitemap reads the sitemap files.
func loadSitemapFiles() ([]string, error) {
	var result []string
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
		if info.Name() == "index.md" {
			path = strings.TrimSuffix(path, "index.md")
		}
		if strings.HasSuffix(path, ".md") {
			path = strings.TrimSuffix(path, ".md")
		}
		result = append(result, filepath.ToSlash(path))
		return nil
	})
	return result, err
}
