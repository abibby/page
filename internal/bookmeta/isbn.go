// Package bookmeta extracts book metadata (ISBN, title, author) from local
// EPUB and M4B files.
package bookmeta

import (
	"regexp"
	"strings"
)

// Meta is the metadata recovered from a book file. Any field may be empty.
type Meta struct {
	ISBN   string // normalised, digits only (ISBN-13 preferred)
	Title  string
	Author string
}

var (
	isbn13Re = regexp.MustCompile(`97[89][0-9]{10}`)
	isbn10Re = regexp.MustCompile(`\b[0-9]{9}[0-9Xx]\b`)
)

// findISBN scans free text for the first plausible ISBN-13, then ISBN-10,
// after stripping hyphens and spaces. Returns "" if none found.
func findISBN(texts ...string) string {
	for _, raw := range texts {
		// Collapse common ISBN separators so "978-0-..." matches.
		cleaned := strings.NewReplacer("-", "", " ", "", "‐", "").Replace(raw)
		if m := isbn13Re.FindString(cleaned); m != "" {
			return m
		}
	}
	for _, raw := range texts {
		cleaned := strings.NewReplacer("-", "", " ", "").Replace(raw)
		if m := isbn10Re.FindString(cleaned); m != "" {
			return strings.ToUpper(m)
		}
	}
	return ""
}
