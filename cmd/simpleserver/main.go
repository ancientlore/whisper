package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ancientlore/whisper/cachefs"
	"github.com/ancientlore/whisper/virtual"
	"github.com/golang/groupcache"
)

func main() {
	folder := flag.String("folder", "../../example", "Base folder")

	flag.Parse()

	// Setup groupcache (in this example with no peers)
	groupcache.RegisterPeerPicker(func() groupcache.PeerPicker { return groupcache.NoPeers{} })

	// Create the virtual file system
	fs, err := virtual.New(os.DirFS(*folder))
	if err != nil {
		log.Fatal(err)
	}

	// Create the cached file system with group name "groupName", a 10MB cache, and a ten second expiration
	cachedFileSystem := cachefs.New(fs, "simple", 10*1024*1024, 10*time.Second)

	http.ListenAndServe(":9000", http.FileServer(http.FS(cachedFileSystem)))
}
