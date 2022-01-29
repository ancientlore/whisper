package cache_test

import (
	"io/fs"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ancientlore/whisper/cache"
)

func TestFS(t *testing.T) {
	const count = 4
	fileSys := cache.New(os.DirFS(".."), "folder", 1024*1024, time.Second)
	var wg sync.WaitGroup
	wg.Add(count)
	for i := 0; i < count; i++ {
		go func() {
			defer wg.Done()
			fs.WalkDir(fileSys, ".", func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					t.Error(err)
				}
				if path == "" {
					t.Error("Path is empty")
				}
				t.Log(path, d, err)
				return nil
			})
		}()
	}
	wg.Wait()
}
