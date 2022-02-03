package cachefs_test

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"testing"

	"github.com/ancientlore/whisper/cachefs"
)

func TestReadDir(t *testing.T) {
	const count = 10
	rootFS := os.DirFS(".")
	fileSys := cachefs.New(rootFS, nil)

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

	dirs, err := rdf.ReadDir(0)
	if err != nil {
		t.Error(err)
		return
	}

	rf, err := rootFS.Open(".")
	if err != nil {
		t.Error(err)
		return
	}
	defer rf.Close()

	rootDirs, err := rf.(fs.ReadDirFile).ReadDir(0)

	if len(rootDirs) != len(dirs) {
		t.Errorf("rootDirs has length %d but dirs has length %d", len(rootDirs), len(dirs))
		return
	}

	for i := range dirs {
		if dirs[i].Name() != rootDirs[i].Name() {
			t.Errorf("Entry %d of %q does not match %q", i, dirs[i].Name(), rootDirs[i].Name())
		}
	}
}

func TestReadDirLoop(t *testing.T) {
	const count = 10
	rootFS := os.DirFS(".")
	fileSys := cachefs.New(rootFS, nil)

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
