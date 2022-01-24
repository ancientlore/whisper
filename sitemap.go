package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/golang/groupcache"
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
func sitemap(cacheBytes int64, cacheDuration time.Duration) http.HandlerFunc {
	type sitemapContent struct {
		ModTime time.Time
	}

	cache := groupcache.NewGroup("sitemap", cacheBytes, groupcache.GetterFunc(
		func(ctx context.Context, key string, dest groupcache.Sink) error {
			// key is only used for quantizing time
			files, modTime, err := loadSitemapFiles()
			if err != nil {
				return fmt.Errorf("sitemap: %w", err)
			}
			_, tplModTime := getTemplates()
			if tplModTime.After(modTime) {
				modTime = tplModTime
			}
			var (
				out  bytes.Buffer
				cont sitemapContent
			)
			enc := gob.NewEncoder(&out)
			err = enc.Encode(cont)
			if err != nil {
				return fmt.Errorf("sitemap: %w", err)
			}
			err = sitemapTpl.ExecuteTemplate(&out, "sitemap", files)
			if err != nil {
				return fmt.Errorf("sitemap: %w", err)
			}
			dest.SetBytes(out.Bytes())
			return nil
		}))

	return func(w http.ResponseWriter, r *http.Request) {
		if sitemapTpl == nil {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet || r.URL.Path != "/sitemap.txt" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		t := quantize(time.Now(), cacheDuration, "/sitemap.txt")
		var bv groupcache.ByteView
		err := cache.Get(context.Background(), strconv.FormatInt(t, 16), groupcache.ByteViewSink(&bv))
		if err != nil {
			http.Error(w, fmt.Sprintf("sitemap handler: %s", err), http.StatusInternalServerError)
			return
		}
		rdr := bv.Reader()
		crdr := &creader{Reader: rdr}
		dec := gob.NewDecoder(crdr)
		var c sitemapContent
		err = dec.Decode(&c)
		if err != nil {
			http.Error(w, fmt.Sprintf("sitemap handler: %s", err), http.StatusInternalServerError)
			return
		}
		// log.Printf("Read %d", crdr.Count())
		rdr = bv.SliceFrom(crdr.Count()).Reader()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.ServeContent(w, r, "sitemap.txt", c.ModTime, rdr)
	}
}

// loadSitemap reads the sitemap files.
func loadSitemapFiles() ([]string, time.Time, error) {
	var (
		result  []string
		maxTime time.Time
	)
	log.Printf("loadSitemapFiles: Walk: %q", ".")
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
		if info.Name() == "whisper.cfg" {
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
			if hasImageFolderPrefix(r) && hasImageExtension(r) {
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

// alreadyAdded checks if the last 4 entries already contain the given text.
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

// hasImageFolderPrefix checks if the entry is in an image folder.
func hasImageFolderPrefix(s string) bool {
	imageFolders := []string{"photos", "images", "pictures", "cartoons", "toons", `sketches`, `artwork`, `drawings`}
	for _, f := range imageFolders {
		if strings.HasPrefix(s, f) {
			return true
		}
	}
	return false
}

// hasImageExtension checks if the path ends in an image type.
func hasImageExtension(s string) bool {
	imageTypes := []string{".png", ".jpg", ".gif", ".jpeg"}
	for _, ext := range imageTypes {
		if strings.HasSuffix(s, ext) {
			return true
		}
	}
	return false
}
