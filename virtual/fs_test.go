package virtual

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestFS(t *testing.T) {
	const count = 10
	fileSys, err := New(os.DirFS("../example"))
	if err != nil {
		t.Error(err)
		return
	}
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
					return nil
				}
				if d.Name() == ".git" {
					return fs.SkipDir
				}
				t.Log(path, d, err)
				numEntries++
				if !d.IsDir() {
					b, err := fs.ReadFile(fileSys, path)
					if err != nil {
						if path != "articles/badFrontMatter" {
							t.Errorf("Cannot read %q: %v", path, err)
						}
						return nil
					}
					if len(b) == 0 && path != "err" {
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
					if path != "articles/badFrontMatter" {
						t.Errorf("Cannot stat %q: %v", path, err)
					}
					return nil
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
				t.Log("\t", fi)
				return nil
			})
			t.Logf("saw %d entries", numEntries)
		}()
	}
	wg.Wait()
}

func TestReadFile(t *testing.T) {
	fileSys, err := New(os.DirFS("../example"))
	if err != nil {
		t.Error(err)
		return
	}
	b, err := fs.ReadFile(fileSys, "index")
	if err != nil {
		t.Error(err)
	}
	t.Log(string(b))
}

func TestReadDir(t *testing.T) {
	fileSys, err := New(os.DirFS("../example"))
	if err != nil {
		t.Error(err)
		return
	}
	entries, err := fs.ReadDir(fileSys, ".")
	if err != nil {
		t.Error(err)
	}
	for _, entry := range entries {
		inf, err := entry.Info()
		if err != nil {
			t.Error(err)
		} else {
			t.Logf("%s %10d  %s  %s", inf.Mode(), inf.Size(), inf.ModTime().Format(time.UnixDate), inf.Name())
		}
	}
}

func TestOpenRead(t *testing.T) {
	fileSys, err := New(os.DirFS("../example"))
	if err != nil {
		t.Error(err)
		return
	}
	f, err := fileSys.Open("index")
	if err != nil {
		t.Error(err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			t.Error(err)
		}
	}()

	b := make([]byte, 1024)
	for {
		n, err := f.Read(b)
		if errors.Is(err, io.EOF) {
			break
		}
		t.Log(string(b[:n]))
	}
}

func TestOpenReadDir(t *testing.T) {
	fileSys, err := New(os.DirFS("../example"))
	if err != nil {
		t.Error(err)
		return
	}
	f, err := fileSys.Open("photos")
	if err != nil {
		t.Error(err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			t.Error(err)
		}
	}()

	rdf, ok := f.(fs.ReadDirFile)
	if !ok {
		t.Errorf("Not a directory")
		return
	}

	entries, err := rdf.ReadDir(0)
	if err != nil {
		t.Error(err)
	} else {
		for _, entry := range entries {
			inf, err := entry.Info()
			if err != nil {
				t.Error(err)
			} else {
				t.Logf("%s %10d  %s  %s", inf.Mode(), inf.Size(), inf.ModTime().Format(time.UnixDate), inf.Name())
			}
		}
	}
}
