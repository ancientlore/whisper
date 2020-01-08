package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"html/template"
	"io"
	"net/url"
	"strconv"
	"time"

	"github.com/golang/groupcache"
)

type cachedDir struct {
	Files   []file
	ModTime time.Time
}

var (
	peers                   *groupcache.HTTPPool
	dirListingCache         *groupcache.Group
	dirListingCacheDuration time.Duration
	markdownCache           *groupcache.Group
	markdownCacheDuration   time.Duration
	templateCache           *groupcache.Group
	templateCacheDuration   time.Duration
)

func initGroupCache() {
	me := "http://127.0.0.1"
	peers = groupcache.NewHTTPPool(me)
	// Whenever peers change:
	// peers.Set("http://10.0.0.1", "http://10.0.0.2", "http://10.0.0.3")
}

func initReadDirCache(cacheBytes int64, cacheDuration time.Duration) {
	dirListingCacheDuration = cacheDuration
	dirListingCache = groupcache.NewGroup("readDir", cacheBytes, groupcache.GetterFunc(
		func(ctx context.Context, key string, dest groupcache.Sink) error {
			q, err := url.ParseQuery(key)
			if err != nil {
				return fmt.Errorf("dirListing group: %w", err)
			}
			// log.Printf("Calling readDir %s", q.Encode())
			var (
				d   cachedDir
				buf bytes.Buffer
			)
			d.Files, d.ModTime, err = readDir(q.Get("folderpath"))
			if err != nil {
				return fmt.Errorf("dirListing group: %w", err)
			}
			enc := gob.NewEncoder(&buf)
			err = enc.Encode(d)
			if err != nil {
				return fmt.Errorf("dirListing group: %w", err)
			}
			dest.SetBytes(buf.Bytes())
			return nil
		}))
}

func cachedReadDir(folderPath string) ([]file, time.Time, error) {
	var (
		data []byte
		q    = make(url.Values, 2)
		d    cachedDir
	)
	q.Set("folderpath", folderPath)
	t := quantize(time.Now(), dirListingCacheDuration, folderPath)
	q.Set("t", strconv.FormatInt(t, 10))
	// log.Printf("cachedReadDir %s", q.Encode())
	err := dirListingCache.Get(context.Background(), q.Encode(), groupcache.AllocatingByteSliceSink(&data))
	if err != nil {
		return nil, d.ModTime, fmt.Errorf("cachedReadDir: %w", err)
	}
	dec := gob.NewDecoder(bytes.NewReader(data))
	err = dec.Decode(&d)
	if err != nil {
		return nil, d.ModTime, fmt.Errorf("cachedReadDir: %w", err)
	}
	return d.Files, d.ModTime, nil
}

type cachedMarkdown struct {
	FrontMatter *frontMatter
	Content     template.HTML
	ModTime     time.Time
}

func initMarkdownCache(cacheBytes int64, cacheDuration time.Duration) {
	markdownCacheDuration = cacheDuration
	markdownCache = groupcache.NewGroup("renderMarkdown", cacheBytes, groupcache.GetterFunc(
		func(ctx context.Context, key string, dest groupcache.Sink) error {
			q, err := url.ParseQuery(key)
			if err != nil {
				return fmt.Errorf("renderMarkdown group: %w", err)
			}
			// log.Printf("Calling renderMarkdown %s", q.Encode())
			var (
				d   cachedMarkdown
				buf bytes.Buffer
			)
			d.FrontMatter, d.Content, d.ModTime, err = renderMarkdown(q.Get("filename"))
			if err != nil {
				return fmt.Errorf("renderMarkdown group: %w", err)
			}
			enc := gob.NewEncoder(&buf)
			err = enc.Encode(d)
			if err != nil {
				return fmt.Errorf("renderMarkdown group: %w", err)
			}
			dest.SetBytes(buf.Bytes())
			return nil
		}))
}

func cachedRenderMarkdown(filename string) (*frontMatter, template.HTML, time.Time, error) {
	var (
		data []byte
		q    = make(url.Values, 2)
		d    cachedMarkdown
	)
	q.Set("filename", filename)
	t := quantize(time.Now(), markdownCacheDuration, filename)
	q.Set("t", strconv.FormatInt(t, 10))
	// log.Printf("cachedRenderMarkdown %s", q.Encode())
	err := markdownCache.Get(context.Background(), q.Encode(), groupcache.AllocatingByteSliceSink(&data))
	if err != nil {
		return nil, "", d.ModTime, fmt.Errorf("cachedRenderMarkdown: %w", err)
	}
	dec := gob.NewDecoder(bytes.NewReader(data))
	err = dec.Decode(&d)
	if err != nil {
		return nil, "", d.ModTime, fmt.Errorf("cachedRenderMarkdown: %w", err)
	}
	return d.FrontMatter, d.Content, d.ModTime, nil
}

type ctxKey string

func initTemplateCache(cacheBytes int64, cacheDuration time.Duration) {
	templateCacheDuration = cacheDuration
	templateCache = groupcache.NewGroup("executeTemplate", cacheBytes, groupcache.GetterFunc(
		func(ctx context.Context, key string, dest groupcache.Sink) error {
			q, err := url.ParseQuery(key)
			if err != nil {
				return fmt.Errorf("executeTemplate group: %w", err)
			}
			// log.Printf("Calling executeTemplate %s", q.Encode())
			var (
				buf bytes.Buffer
			)
			data := ctx.Value(ctxKey("data")).(data)
			// log.Print(data)
			err = tpl.ExecuteTemplate(&buf, q.Get("templateName"), data)
			if err != nil {
				return fmt.Errorf("executeTemplate group: %w", err)
			}

			dest.SetBytes(buf.Bytes())
			return nil
		}))
}

func cachedExecuteTemplate(w io.Writer, name string, dat interface{}) error {
	var (
		buf groupcache.ByteView
		q   = make(url.Values, 3)
	)
	q.Set("filename", dat.(data).Page.Filename)
	t := quantize(time.Now(), templateCacheDuration, dat.(data).Page.Filename)
	q.Set("t", strconv.FormatInt(t, 10))
	q.Set("templateName", name)
	ctx := context.WithValue(context.Background(), ctxKey("data"), dat)
	// log.Printf("cachedExecuteTemplate %s", q.Encode())
	err := templateCache.Get(ctx, q.Encode(), groupcache.ByteViewSink(&buf))
	if err != nil {
		return fmt.Errorf("cachedExecuteTemplate: %w", err)
	}
	_, err = buf.WriteTo(w)
	if err != nil {
		return fmt.Errorf("cachedExecuteTemplate: %w", err)
	}
	return nil
}
