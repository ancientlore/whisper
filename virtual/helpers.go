package virtual

import (
	"strings"
)

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

// hasImageFolderPrefix checks if the entry is in an image folder.
func hasImageFolderPrefix(s string) bool {
	imageFolders := []string{"photos", "images", "pictures", "cartoons", "toons", `sketches`, `artwork`, `drawings`}
	for _, f := range imageFolders {
		if strings.HasPrefix(s, f) {
			return true
		}
	}
	return false
}

// hasImageExtension checks if the path ends in an image type.
func hasImageExtension(s string) bool {
	imageTypes := []string{".png", ".jpg", ".gif", ".webp", ".jpeg"}
	for _, ext := range imageTypes {
		if strings.HasSuffix(s, ext) {
			return true
		}
	}
	return false
}
