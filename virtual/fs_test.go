package virtual

import (
	"errors"
	"io"
	"io/fs"
	"net/http"
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
						if path != "articles/badFrontMatter.html" {
							t.Errorf("Cannot read %q: %v", path, err)
						}
						return nil
					}
					if len(b) == 0 && path != "err.html" {
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
					if path != "articles/badFrontMatter.html" {
						t.Errorf("Cannot stat %q: %v", path, err)
					}
					return nil
				}
				if !strings.HasSuffix(path, fi.Name()) {
					t.Errorf("%q should be part of %q", fi.Name(), path)
				}
				if !fi.IsDir() {
					if fi.Size() == 0 {
						if fi.Name() != "err.html" {
							t.Errorf("Expected %q to have non-zero size", path)
						}
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
	b, err := fs.ReadFile(fileSys, "index.html")
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
	f, err := fileSys.Open("index.html")
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

func TestHttpReadDir(t *testing.T) {
	fileSys, err := New(os.DirFS("../example"))
	if err != nil {
		t.Error(err)
		return
	}
	hfs := http.FS(fileSys)

	f, err := hfs.Open("/")
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {
		err := f.Close()
		if err != nil {
			t.Error(err)
		}
	}()

	entries, err := f.Readdir(0)
	if err != nil {
		t.Error(err)
		return
	} else {
		for _, entry := range entries {
			if err != nil {
				t.Error(err)
			} else {
				t.Logf("%s %10d  %s  %s", entry.Mode(), entry.Size(), entry.ModTime().Format(time.UnixDate), entry.Name())
			}
		}
	}
}

func TestHttpRead(t *testing.T) {
	fileSys, err := New(os.DirFS("../example"))
	if err != nil {
		t.Error(err)
		return
	}
	hfs := http.FS(fileSys)

	f, err := hfs.Open("/index.html")
	if err != nil {
		t.Error(err)
		return
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

func TestFileSize(t *testing.T) {
	fileSys, err := New(os.DirFS("../example"))
	if err != nil {
		t.Error(err)
		return
	}

	entries, err := fs.ReadDir(fileSys, ".")
	if err != nil {
		t.Error(err)
		return
	}
	var fi1 fs.FileInfo
	for _, entry := range entries {
		if entry.Name() == "index.html" {
			fi1, err = entry.Info()
			if err != nil {
				t.Error(err)
				return
			}
		}
	}
	fi2, err := fs.Stat(fileSys, "index.html")
	if err != nil {
		t.Error(err)
		return
	}

	t.Logf("Using ReadDir size is %d, using Stat size is %d", fi1.Size(), fi2.Size())

	if fi1.Size() != fi2.Size() {
		// TODO: Reenable test
		//t.Errorf("Sizes don't match: %d vs %d", fi1.Size(), fi2.Size())
	}
}

func TestReadDirLoop(t *testing.T) {
	const count = 10
	rootFS := os.DirFS(".")
	fileSys, err := New(rootFS)
	if err != nil {
		t.Error(err)
		return
	}

	f, err := fileSys.Open(".")
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {
		err := f.Close()
		if err != nil {
			t.Error(err)
		}
	}()

	rdf, ok := f.(fs.ReadDirFile)
	if !ok {
		t.Error("Root is not a ReadDirFile")
		return
	}

	var dirs []fs.DirEntry
	for {
		dirs, err = rdf.ReadDir(2)
		if errors.Is(err, io.EOF) {
			if len(dirs) != 0 {
				t.Errorf("Expected empty directory at EOF")
			}
			break
		}
		if err != nil {
			t.Error(err)
			break
		}
		if len(dirs) == 0 {
			t.Errorf("Should not return empty directory if not EOF")
			break
		}
		if len(dirs) > 2 {
			t.Errorf("Returned more than 2 entries: %d", len(dirs))
		}
		t.Log(dirs)
	}
}
