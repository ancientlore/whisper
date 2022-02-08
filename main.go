package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/ancientlore/whisper/cachefs"
	"github.com/ancientlore/whisper/virtual"
	"github.com/ancientlore/whisper/web"
	"github.com/facebookgo/flagenv"
	"github.com/golang/groupcache"
)

// main is where it all begins. 😀
func main() {
	// Setup flags
	var (
		fPort              = flag.Int("port", 8080, "Port to listen on.")
		fReadTimeout       = flag.Duration("readtimeout", 10*time.Second, "HTTP server read timeout.")
		fReadHeaderTimeout = flag.Duration("readheadertimeout", 5*time.Second, "HTTP server read header timeout.")
		fWriteTimeout      = flag.Duration("writetimeout", 30*time.Second, "HTTP server write timeout.")
		fIdleTimeout       = flag.Duration("idletimeout", 60*time.Second, "HTTP server keep-alive timeout.")
		fRoot              = flag.String("root", ".", "Root of web site.")
		fCacheSize         = flag.Int("cachesize", 0, "Cache size in MB.")
		fCacheDuration     = flag.Duration("cacheduration", 0, "How long to cache content.")
		fExpires           = flag.Duration("expires", 0, "Default expires header.")
		fStaticExpires     = flag.Duration("staticexpires", 0, "Default expires header for static content.")
		fWaitForFiles      = flag.Bool("wait", false, "Wait for files to appear in root folder before starting up.")
	)
	flag.Parse()
	flagenv.Parse()

	// If requested, wait for files to show up in root folder, up to 60 seconds
	if *fWaitForFiles {
		err := waitForFiles(*fRoot)
		if err != nil {
			log.Print(err)
			os.Exit(1)
		}
	}

	// Setup groupcache (in this example with no peers)
	groupcache.RegisterPeerPicker(func() groupcache.PeerPicker { return groupcache.NoPeers{} })

	// Create the virtual file system
	virtualFileSystem, err := virtual.New(os.DirFS(*fRoot))
	if err != nil {
		log.Print(err)
		os.Exit(2)
	}

	// get the config
	cfg, err := virtualFileSystem.Config()
	if err != nil {
		log.Print(err)
		os.Exit(3)
	}

	// Apply config overrides
	if *fExpires != 0 {
		cfg.Expires = virtual.Duration(*fExpires)
	}
	if *fStaticExpires != 0 {
		cfg.StaticExpires = virtual.Duration(*fStaticExpires)
	}
	if *fCacheSize != 0 {
		cfg.CacheSize = *fCacheSize
	}
	if *fCacheDuration != 0 {
		cfg.CacheDuration = virtual.Duration(*fCacheDuration)
	}
	if cfg.CacheSize <= 0 {
		cfg.CacheSize = 1 // need a default
	}
	log.Printf("Expires: %s", cfg.Expires.String())
	log.Printf("Static Expires: %s", cfg.StaticExpires.String())
	log.Printf("Cache Size: %dMB", cfg.CacheSize)
	log.Printf("Cache Duration: %s", cfg.CacheDuration.String())

	// Create the cached file system
	cachedFileSystem := cachefs.New(virtualFileSystem, &cachefs.Config{GroupName: "whisper", SizeInBytes: int64(cfg.CacheSize) * 1024 * 1024, Duration: time.Duration(cfg.CacheDuration)})

	// create handler
	handler := web.HeaderHandler(
		web.ExpiresHandler(
			gziphandler.GzipHandler(
				web.ErrorHandler(
					http.FileServer(
						http.FS(cachedFileSystem),
					),
					cachedFileSystem,
				),
			),
			time.Duration(cfg.Expires),
			time.Duration(cfg.StaticExpires),
		),
		cfg.Headers)

	// Create HTTP server
	var srv = http.Server{
		Addr:              fmt.Sprintf(":%d", *fPort),
		ReadTimeout:       *fReadTimeout,
		WriteTimeout:      *fWriteTimeout,
		ReadHeaderTimeout: *fReadHeaderTimeout,
		IdleTimeout:       *fIdleTimeout,
		Handler:           handler,
	}

	// Start cache monitor
	monc := stats("whisper")

	// Create signal handler for graceful shutdown
	go func() {
		sigint := make(chan os.Signal, 1)

		// interrupt signal sent from terminal
		signal.Notify(sigint, os.Interrupt)
		// sigterm signal sent from kubernetes
		signal.Notify(sigint, syscall.SIGTERM)

		<-sigint

		// We received an interrupt signal, shut down.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			// Error from closing listeners, or context timeout:
			log.Printf("HTTP server Shutdown: %v", err)
		}
		close(monc)
	}()

	// Listen for requests
	log.Print("Listening for requests.")
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Printf("HTTP server: %v", err)
	} else {
		log.Print("Goodbye.")
	}
}

func stats(groupName string) chan<- bool {
	c := make(chan bool)
	g := groupcache.GetGroup(groupName)
	if g != nil {
		go func() {
			t := time.NewTicker(5 * time.Minute)
			for {
				select {
				case _, ok := <-c:
					if !ok {
						return
					}
				case <-t.C:
					s := g.CacheStats(groupcache.HotCache)
					log.Printf("Hot Cache  %#v", s)
					s = g.CacheStats(groupcache.MainCache)
					log.Printf("Main Cache %#v", s)
				}
			}
		}()
	}
	return c
}

func waitForFiles(pathname string) error {
	foundFiles := false
	for i := 0; i < 60; i++ {
		d, err := os.ReadDir(pathname)
		if err != nil {
			log.Printf("os.Dir: %s", err)
		} else if len(d) > 0 {
			var dir []string
			var hasError bool
			for _, entry := range d {
				if entry.IsDir() {
					dir = append(dir, entry.Name()+"/")
				} else {
					dir = append(dir, entry.Name())
					if entry.Name() == "cpln-error.txt" {
						hasError = true
						errData, err := os.ReadFile(path.Join(pathname, "cpln-error.txt"))
						log.Printf("cpln-error.txt: %s %v", errData, err)
					}
				}
			}
			log.Printf("Found files %v", dir)
			if !hasError {
				foundFiles = true
				break
			}
		}
		if i%10 == 0 {
			log.Print("Waiting for files...")
		}
		time.Sleep(time.Second)
	}
	if !foundFiles {
		return fmt.Errorf("No files found in %q", pathname)
	}
	return nil
}
