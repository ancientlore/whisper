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
	"github.com/pelletier/go-toml/v2"
)

var (
	gmtZone *time.Location
	peers   *groupcache.HTTPPool
)

type config struct {
	Expires       duration          `toml:"expires"`
	StaticExpires duration          `toml:"staticexpires"`
	Headers       map[string]string `toml:"headers"`
}

// main is where it all begins. 😀
func main() {
	// Setup flags
	var (
		fPort              = flag.Int("port", 8080, "Port to listen on.")
		fReadTimeout       = flag.Duration("readtimeout", 10*time.Second, "HTTP server read timeout.")
		fReadHeaderTimeout = flag.Duration("readheadertimeout", 5*time.Second, "HTTP server read header timeout.")
		fWriteTimeout      = flag.Duration("writetimeout", 30*time.Second, "HTTP server write timeout.")
		fRoot              = flag.String("root", ".", "Root of web site.")
		fCacheDuration     = flag.Duration("cacheduration", 5*time.Minute, "How long to cache content.")
		fExpires           = flag.Duration("expires", 0, "Default expires header.")
		fStaticExpires     = flag.Duration("staticexpires", 0, "Default expires header for static content.")
		fWaitForFiles      = flag.Bool("wait", false, "Wait for files to appear in root folder before starting up.")
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

	// If requested, wait for files to show up in root folder, up to 60 seconds
	if *fWaitForFiles {
		var dir []string
		for i := 0; i < 60; i++ {
			d, err := os.ReadDir(*fRoot)
			if err != nil {
				log.Printf("os.Dir: %s", err)
			} else if len(d) > 0 {
				for _, entry := range d {
					if entry.IsDir() {
						dir = append(dir, entry.Name()+"/")
					} else {
						dir = append(dir, entry.Name())
					}
				}
				log.Printf("Found files %v", dir)
				break
			}
			if i%10 == 0 {
				log.Print("Waiting for files...")
			}
			time.Sleep(time.Second)
		}
		if len(dir) == 0 {
			log.Printf("No files in root folder")
			os.Exit(6)
		}
	}

	// Switch to site folder
	err = os.Chdir(*fRoot)
	if err != nil {
		log.Printf("Cannot switch to root %q: %s", *fRoot, err)
		os.Exit(1)
	}
	log.Printf("Changed to %q directory.", *fRoot)

	// load config
	var cfg config
	cfgBytes, err := os.ReadFile("whisper.cfg")
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("Cannot read config file: %s", err)
			os.Exit(4)
		}
		log.Print("No config file found.")
	} else {
		err = toml.Unmarshal(cfgBytes, &cfg)
		if err != nil {
			log.Printf("Cannot parse config file: %s", err)
			os.Exit(5)
		}
		log.Printf("Read config file: %+v", cfg)
	}
	if *fExpires != 0 {
		cfg.Expires = duration(*fExpires)
	}
	if *fStaticExpires != 0 {
		cfg.StaticExpires = duration(*fStaticExpires)
	}

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
	http.Handle("/template/", gziphandler.GzipHandler(headerHandler(http.HandlerFunc(notFound), cfg.Headers)))
	http.Handle("/whisper.cfg", gziphandler.GzipHandler(headerHandler(http.HandlerFunc(notFound), cfg.Headers)))
	http.Handle("/sitemap.txt", gziphandler.GzipHandler(headerHandler(http.HandlerFunc(sitemap(1024*1024, *fCacheDuration)), cfg.Headers)))
	imageTypes := []string{".png", ".jpg", ".gif", ".jpeg"}
	imageHandler := gziphandler.GzipHandler(headerHandler(extHandler(existsHandler(http.FileServer(http.Dir(".")), time.Duration(cfg.StaticExpires)), time.Duration(cfg.Expires), imageTypes, "image"), cfg.Headers))
	imageFolders := []string{"photos", "images", "pictures", "cartoons", "toons", `sketches`, `artwork`, `drawings`}
	for _, folder := range imageFolders {
		http.Handle("/"+folder+"/", imageHandler)
	}
	http.Handle("/", gziphandler.GzipHandler(headerHandler(markdown(existsHandler(http.FileServer(http.Dir(".")), time.Duration(cfg.StaticExpires)), time.Duration(cfg.Expires)), cfg.Headers)))
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

func headerHandler(h http.Handler, hdrs map[string]string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range hdrs {
			w.Header().Set(k, v)
		}
		h.ServeHTTP(w, r)
	})
}
