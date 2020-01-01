package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/facebookgo/flagenv"
	cache "github.com/victorspringer/http-cache"
	"github.com/victorspringer/http-cache/adapter/memory"
)

// tpl stores the site's HTML templates.
var tpl *template.Template

// main is where it all begins.
func main() {
	// Setup flags
	var (
		fPort              = flag.Int("port", 8080, "Port to listen on.")
		fReadTimeout       = flag.Duration("readtimeout", 5*time.Second, "HTTP server read timeout.")
		fReadHeaderTimeout = flag.Duration("readheadertimeout", 3*time.Second, "HTTP server read header timeout.")
		fWriteTimeout      = flag.Duration("writetimeout", 10*time.Second, "HTTP server write timeout.")
		fRoot              = flag.String("root", ".", "Root of web site.")
		fCacheTTL          = flag.Duration("cachettl", 10*time.Minute, "Cache TTL.")
		fCacheSize         = flag.Int("cachesize", 1000, "Cache capacity (number of items).")
	)
	flag.Parse()
	flagenv.Parse()

	// Create HTTP server
	var srv = http.Server{
		Addr:              fmt.Sprintf(":%d", *fPort),
		ReadTimeout:       *fReadTimeout,
		WriteTimeout:      *fWriteTimeout,
		ReadHeaderTimeout: *fReadHeaderTimeout,
	}

	// Switch to site folder
	err := os.Chdir(*fRoot)
	if err != nil {
		log.Printf("Cannot switch to root %q: %s", *fRoot, err)
		os.Exit(1)
	}

	// Parse templates
	funcMap := template.FuncMap{
		// The name "title" is what the function will be called in the template text.
		"dir":  dir,
		"join": path.Join,
	}
	tpl, err = template.New("whisper").Funcs(funcMap).ParseGlob("template/*.html")
	if err != nil {
		log.Printf("Cannot parse templates: %s", err)
		os.Exit(2)
	}

	// Initialize HTTP ache
	memcache, err := memory.NewAdapter(
		memory.AdapterWithAlgorithm(memory.LRU),
		memory.AdapterWithCapacity(*fCacheSize),
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	cacheClient, err := cache.NewClient(
		cache.ClientWithAdapter(memcache),
		cache.ClientWithTTL(*fCacheTTL),
		// cache.ClientWithRefreshKey("opn"),
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Setup handlers
	http.Handle("/template/", http.HandlerFunc(notFound))
	http.Handle("/", cacheClient.Middleware(gziphandler.GzipHandler(markdown(existsHandler(http.FileServer(specialFileHidingFileSystem{http.Dir(".")}))))))

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
	}()

	// Listen for requests
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Printf("HTTP server: %v", err)
	} else {
		log.Print("Goodbye.")
	}
}
