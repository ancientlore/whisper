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
	"syscall"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/facebookgo/flagenv"
	"github.com/golang/groupcache"
)

var (
	gmtZone *time.Location
	peers   *groupcache.HTTPPool
)

// main is where it all begins. 😀
func main() {
	// Setup flags
	var (
		fPort              = flag.Int("port", 8080, "Port to listen on.")
		fReadTimeout       = flag.Duration("readtimeout", 10*time.Second, "HTTP server read timeout.")
		fReadHeaderTimeout = flag.Duration("readheadertimeout", 5*time.Second, "HTTP server read header timeout.")
		fWriteTimeout      = flag.Duration("writetimeout", 30*time.Second, "HTTP server write timeout.")
		fRoot              = flag.String("root", ".", "Root of web site.")
		fCacheDuration     = flag.Duration("cacheduration", time.Minute, "How long to cache content.")
		fExpires           = flag.Duration("expires", 0, "Default expires header.")
		fStaticExpires     = flag.Duration("staticexpires", 0, "Default expires header for static content.")
	)
	flag.Parse()
	flagenv.Parse()

	// init GMT time zone
	err := initGMT()
	if err != nil {
		log.Printf("Cannot load GMT, using UTC instead: %s", err)
	} else {
		log.Print("Loaded GMT zone.")
	}

	// Create HTTP server
	var srv = http.Server{
		Addr:              fmt.Sprintf(":%d", *fPort),
		ReadTimeout:       *fReadTimeout,
		WriteTimeout:      *fWriteTimeout,
		ReadHeaderTimeout: *fReadHeaderTimeout,
	}

	// Switch to site folder
	err = os.Chdir(*fRoot)
	if err != nil {
		log.Printf("Cannot switch to root %q: %s", *fRoot, err)
		os.Exit(1)
	}
	log.Printf("Changed to %q directory.", *fRoot)

	// Parse templates
	custom, err := loadTemplates()
	if err != nil {
		log.Printf("Cannot parse templates: %s", err)
		os.Exit(2)
	}
	if !custom {
		log.Print("ERROR: No template folder found; using default templates.")
	}
	tpl, mt := getTemplates()
	log.Printf("Loaded templates: %s", tpl.DefinedTemplates())
	log.Printf("Templates last modified: %s", mt.In(gmtZone).Format(time.RFC1123))

	// Parse sitemap template
	ok, err := loadSitemapTemplate()
	if err != nil {
		log.Printf("Unable to load sitemap.txt template: %s", err)
		os.Exit(3)
	}
	if !ok {
		log.Print("No sitemap.txt template found.")
	} else {
		log.Print("Loaded sitemap.txt template.")
	}

	// initialize cache
	initGroupCache()
	initReadDirCache(2*1024*1024, *fCacheDuration)
	initMarkdownCache(2*1024*1024, *fCacheDuration)
	initTemplateCache(2*1024*1024, *fCacheDuration)
	log.Print("Initialized cache.")

	// Setup handlers
	http.Handle("/template/", gziphandler.GzipHandler(http.HandlerFunc(notFound)))
	http.Handle("/sitemap.txt", gziphandler.GzipHandler(http.HandlerFunc(sitemap(1024*1024, *fCacheDuration))))
	imageTypes := []string{".png", ".jpg", ".gif", ".jpeg"}
	imageHandler := gziphandler.GzipHandler(extHandler(existsHandler(http.FileServer(http.Dir(".")), *fStaticExpires), *fExpires, imageTypes, "image"))
	imageFolders := []string{"photos", "images", "pictures", "cartoons", "toons", `sketches`, `artwork`, `drawings`}
	for _, folder := range imageFolders {
		http.Handle("/"+folder+"/", imageHandler)
	}
	http.Handle("/", gziphandler.GzipHandler(markdown(existsHandler(http.FileServer(http.Dir(".")), *fStaticExpires), *fExpires)))
	log.Print("Created handlers.")

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
	log.Print("Listening for requests.")
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Printf("HTTP server: %v", err)
	} else {
		log.Print("Goodbye.")
	}
}

// initGMT initialized the GMT zone used in headers.
func initGMT() error {
	var err error
	gmtZone, err = time.LoadLocation("GMT")
	if err != nil {
		gmtZone = time.UTC
	}
	return err
}

// initGroupCache initializes our group cache.
func initGroupCache() {
	me := "http://127.0.0.1"
	peers = groupcache.NewHTTPPool(me)
	// Whenever peers change:
	// peers.Set("http://10.0.0.1", "http://10.0.0.2", "http://10.0.0.3")
}
