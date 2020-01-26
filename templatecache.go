package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"time"

	"github.com/golang/groupcache"
)

var (
	templateCache         *groupcache.Group
	templateCacheDuration time.Duration
)

// ctxKey is the type used to hold data passed to a template execution.
type ctxKey string

// initTemplateCache initializes the template cache of the given size and expiry.
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
			tpl, _ := getTemplates()
			err = tpl.ExecuteTemplate(&buf, q.Get("templateName"), data)
			if err != nil {
				return fmt.Errorf("executeTemplate group: %w", err)
			}

			dest.SetBytes(buf.Bytes())
			return nil
		}))
}

// cachedExecuteTemplate is a version of template.ExecuteTemplate that provides caching.
// dat is passed via a context key of type ctxKey, and it is assumed that dat does
// not change for a given template name and pathname. In most scenarious that would
// be a bad assumption, but the data we pass actually doesn't change in our case.
func cachedExecuteTemplate(w io.Writer, name string, dat interface{}) error {
	var (
		buf groupcache.ByteView
		q   = make(url.Values, 3)
	)
	q.Set("pathname", dat.(data).Page.Pathname())
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
