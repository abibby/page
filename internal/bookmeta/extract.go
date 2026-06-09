package bookmeta

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Supported reports whether a file extension is one this package can read.
func Supported(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".epub", ".m4b", ".m4a":
		return true
	default:
		return false
	}
}

// Extract reads metadata from an EPUB or M4B/M4A file on disk.
func Extract(path string) (Meta, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".epub":
		return extractEPUB(path)
	case ".m4b", ".m4a":
		return extractMP4(path)
	default:
		return Meta{}, fmt.Errorf("unsupported file type: %s", path)
	}
}
