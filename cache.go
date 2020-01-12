package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"html/template"
	"net/url"
	"strconv"
	"time"

	"github.com/golang/groupcache"
)

var (
	peers                 *groupcache.HTTPPool
	markdownCache         *groupcache.Group
	markdownCacheDuration time.Duration
)

func initGroupCache() {
	me := "http://127.0.0.1"
	peers = groupcache.NewHTTPPool(me)
	// Whenever peers change:
	// peers.Set("http://10.0.0.1", "http://10.0.0.2", "http://10.0.0.3")
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
