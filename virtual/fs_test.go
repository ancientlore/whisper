package virtual

import (
	"io/fs"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestFS(t *testing.T) {
	const count = 1
	fileSys := New(os.DirFS("../example"))
	var wg sync.WaitGroup
	wg.Add(count)
	for i := 0; i < count; i++ {
		go func() {
			defer wg.Done()
			numEntries := 0
			fs.WalkDir(fileSys, ".", func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					t.Error(err)
				}
				if path == "" {
					t.Error("Path is empty")
				}
				if d.Name() == ".git" {
					return fs.SkipDir
				}
				t.Log(path, d, err)
				numEntries++
				if !d.IsDir() {
					b, err := fs.ReadFile(fileSys, path)
					if err != nil {
						t.Errorf("Cannot read %q: %v", path, err)
					}
					if len(b) == 0 {
						t.Errorf("File %q has no data", path)
					}
				} else {
					_, err := fs.ReadDir(fileSys, path)
					if err != nil {
						t.Errorf("Cannot readdir %q: %v", path, err)
					}
				}
				fi, err := fs.Stat(fileSys, path)
				if err != nil {
					t.Errorf("Cannot stat %q: %v", path, err)
				}
				if !strings.HasSuffix(path, fi.Name()) {
					t.Errorf("%q should be part of %q", fi.Name(), path)
				}
				if !fi.IsDir() {
					if fi.Size() == 0 {
						t.Errorf("Expected %q to have non-zero size", path)
					}
				}
				if fi.ModTime().IsZero() {
					t.Errorf("Expected %q to have non-zero mod time", path)
				}
				t.Log(fi)
				return nil
			})
			t.Logf("saw %d entries", numEntries)
		}()
	}
	wg.Wait()
}