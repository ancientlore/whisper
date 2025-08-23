/*
whisper implements a simple web server aimed at small websites.

# Conventions

In general, whisper serves static content from the location it's found - making it easy to structure your site how you want.
There is special handling for certain content like Markdown files.

See the example folder for a sample site layout. In general, whisper uses conventions instead of configuration files.
Conventions used by this server include:

* The template folder holds HTML templates, using Go's html/template package. These templates are used for rendering content but never served directly.
* A sitemap.txt can be created as a template. See the example for details.
* The default page for a folder is a Markdown file called index.md.
* An optional whisper.cfg file holds settings should you want to preserve them.
* Files 404.md and 500.md can be provided for custom errors.

# Markdown

Web pages are generally written in Markdown and use HTML templates to render into the site. The default template to use is called "default"; you must
have a "default" template and an "image" template.  Templates are stored in the "template" folder.

NOTE: If no "template" folder is found, then default templates are loaded named "default" and "image". You probably don't want these because they are
extremely basic, but it's okay for just messing around and viewing Markdown locally.

Markdown may contain front matter which is in TOML format. The front matter is delimited by "+++"" at the start and end. For example:

	+++
	# This is my front matter
	title = "My glorious page"
	+++
	# This is my Heading
	This is my [Markdown](https://en.wikipedia.org/wiki/Markdown).

Front matter may include:

	Name         | Type             | Description
	-------------|------------------|------------------------------------------
	title        | string           | Title of page
	date         | time             | Publish date
	tags         | array of strings | Tags for the articles (not used yet)
	template     | string           | Override the template to render this file
	redirect     | duration         | Provide redirect info (not automated)
	originalfile | string           | Name of the base Markdown or image file

Front matter is used for sorting and constructing lists of articles.

# Templates

whisper uses standard Go templates from the "html/template" package. Templates are passed the following data:

	// FrontMatter holds data scraped from a Markdown page.
	type FrontMatter struct {
	    Title        string    `toml:"title"`        // Title of this page
	    Date         time.Time `toml:"date"`         // Date the article appears
	    Template     string    `toml:"template"`     // The name of the template to use
	    Tags         []string  `toml:"tags"`         // Tags to assign to this article
	    Redirect     string    `toml:"redirect"`     // Issue a redirect to another location
	    OriginalFile string    `toml:"originalfile"` // The original file (markdown or image)
	}

	// PageInfo has information about the current page.
	type PageInfo struct {
	    Path     string // path from URL
	    Filename string // end portion (file) from URL
	}

	// data is what is passed to markdown templates.
	type data struct {
	    FrontMatter FrontMatter   // front matter from Markdown file or defaults
	    Page        PageInfo      // information aboout current page
	    Content     template.HTML // rendered Markdown
	}

Page is information about the current page, and FrontMatter is the front-matter from the current Markdown file.
Content contains the HTML version of the Markdown file.

Functions are added to the template for your convenience.

	Function                          | Description
	----------------------------------|------------
	dir(path string) []File           | Return the contents of the given folder, excluding special files and subfolders.
	sortbyname([]File) []File         | Sort by name (reverse)
	sortbytime([]File) []File         | Sort by time (reverse)
	match(string, ...string) bool     | Match string against file patterns
	filter([]File, ...string) []File  | Filter list against file patterns
	join(parts ...string) string      | The same as path.Join
	ext(path string) string           | The same as path.Ext
	prev([]File, string) *File        | Find the previous file based on Filename
	next([]File, string) *File        | Find the next file based on Filename
	reverse([]File) []File            | Reverse the list
	trimsuffix(string, string) string | The same as strings.TrimSuffix
	trimprefix(string, string) string | The same as strings.TrimPrefix
	trimspace(string) string          | The same as strings.TrimSpace
	markdown(string) template.HTML    | Render Markdown file into HTML
	frontmatter(string) *FrontMatter  | Read front matter from file
	now() time.Time                   | Current time

File is defined as:

	// File holds data about a page endpoint.
	type File struct {
	    FrontMatter FrontMatter
	    Filename    string
	}

If File is not a Markdown file, then FrontMatter.Title is set to the file name and FrontMatter.Date is set to the modification
time. The array is sorted by reverse date (most recent items first).

Note that FrontMatter.OriginalFile is very useful because, for image templates, it will hold the name of the image file. You probably
want to use it in the template.

# Image Templates

Folders named "photos", "images", "pictures", "cartoons", "toons", "sketches", "artwork", or "drawings" use a special handler that can
serve images using an HTML template called "image".

# Non-Goals

It's not a goal to make templates reusable. I expect templates need editing for new sites.

It's not a goal to automate creation of the menu.

It's not a goal to be a fully-featured server. I run https://caddyserver.com/ in front of it.

# More Detail

For more information take a look at the virtual package, which implements a virtual filesystem that handles rendering markdown into HTML.
*/
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
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
		fTemplateReload    = flag.Duration("templatereload", 10*time.Minute, "How often to reload templates.")
		fExpires           = flag.Duration("expires", 0, "Default cache-control max-age header.")
		fStaticExpires     = flag.Duration("staticexpires", 0, "Default cache-control max-age header for static content.")
		fWaitForFiles      = flag.Bool("wait", false, "Wait for files to appear in root folder before starting up.")
	)
	flag.Parse()
	flagenv.Parse()

	// If requested, wait for files to show up in root folder, up to 60 seconds
	if *fWaitForFiles {
		err := waitForFiles(*fRoot)
		if err != nil {
			slog.Error("Unable to wait for files", "error", err)
			os.Exit(1)
		}
	}

	// Setup groupcache (in this example with no peers)
	groupcache.RegisterPeerPicker(func() groupcache.PeerPicker { return groupcache.NoPeers{} })

	// Open root
	root, err := os.OpenRoot(*fRoot)
	if err != nil {
		slog.Error("Unable to open root folder", "error", err)
		os.Exit(2)
	}
	defer root.Close()

	// Create the virtual file system
	// virtualFileSystem, err := virtual.New(os.DirFS(*fRoot))
	virtualFileSystem, err := virtual.New(root.FS())
	if err != nil {
		slog.Error("Unable to create virtual file system", "error", err)
		os.Exit(3)
	}
	defer virtualFileSystem.Close()
	virtualFileSystem.ReloadTemplates(*fTemplateReload)

	// get the config
	cfg, err := virtualFileSystem.Config()
	if err != nil {
		slog.Error("Cannot load config", "error", err)
		os.Exit(4)
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
	slog.Info("Expirations", "normal", cfg.Expires, "static", cfg.StaticExpires)
	slog.Info("Cache", "size", fmt.Sprintf("%dMB", cfg.CacheSize), "duration", cfg.CacheDuration.String())

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
			slog.Error("HTTP server Shutdown", "error", err)
		}
		close(monc)
	}()

	// Listen for requests
	slog.Info("Listening for requests")
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		slog.Error("HTTP server", "error", err)
	} else {
		slog.Info("Goodbye.")
	}
}

type logStats groupcache.CacheStats

func (s logStats) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int64("items", s.Items),
		slog.Int64("bytes", s.Bytes),
		slog.Int64("gets", s.Gets),
		slog.Int64("hits", s.Hits),
		slog.Int64("evictions", s.Evictions),
	)
}

func stats(groupName string) chan<- bool {
	c := make(chan bool)
	g := groupcache.GetGroup(groupName)
	if g != nil {
		go func() {
			t := time.NewTicker(5 * time.Second)
			for {
				select {
				case _, ok := <-c:
					if !ok {
						return
					}
				case <-t.C:
					sh := g.CacheStats(groupcache.HotCache)
					sm := g.CacheStats(groupcache.MainCache)
					slog.Info("Cache Stats", "hot", logStats(sh), "main", logStats(sm))
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
			slog.Warn("os.Dir", "error", err)
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
						slog.Warn("cpln-error.txt", "cpln-error", errData, "error", err)
					}
				}
			}
			slog.Info("Found", "files", dir)
			if !hasError {
				foundFiles = true
				break
			}
		}
		if i%10 == 0 {
			slog.Info("Waiting for files...")
		}
		time.Sleep(time.Second)
	}
	if !foundFiles {
		return fmt.Errorf("no files found in %q", pathname)
	}
	return nil
}
