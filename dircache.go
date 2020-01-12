package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
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
	dirListingCache         *groupcache.Group
	dirListingCacheDuration time.Duration
)

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
