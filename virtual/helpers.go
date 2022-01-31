package virtual

import "strings"

var hiddenFiles = []string{
	"template",
	"whisper.cfg",
}

// isHiddenFile returns true if the given file is considered
// hidden from outside view.
func isHiddenFile(name string) bool {
	for _, s := range hiddenFiles {
		if name == s {
			return true
		}
	}
	return false
}

// containsSpecialFile reports whether name contains a path element starting with a period
// or is another kind of special file. The name is assumed to be a delimited by forward
// slashes, as guaranteed by the fs.FS interface.
func containsSpecialFile(name string) bool {
	parts := strings.Split(name, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ".") {
			return true
		}
	}
	return false
}
