package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/ancientlore/whisper/cachefs"
	"github.com/ancientlore/whisper/virtual"
	"github.com/ancientlore/whisper/web"
	"github.com/golang/groupcache"
)

func main() {
	folder := flag.String("folder", "../../example", "Base folder")
	addr := flag.String("addr", ":9000", "Server address")

	flag.Parse()

	// Setup groupcache (in this example with no peers)
	groupcache.RegisterPeerPicker(func() groupcache.PeerPicker { return groupcache.NoPeers{} })

	// Create the virtual file system
	fs, err := virtual.New(os.DirFS(*folder))
	if err != nil {
		log.Fatal(err)
	}

	// Create the cached file system with group name "groupName", a 10MB cache, and a ten second expiration
	cachedFileSystem := cachefs.New(fs, &cachefs.Config{GroupName: "simple", SizeInBytes: 10 * 1024 * 1024, Duration: 10 * time.Second})

	// get the config
	cfg, err := fs.Config()
	if err != nil {
		log.Fatal(err)
	}

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

	// run the server
	log.Fatal(http.ListenAndServe(*addr, handler))
}
