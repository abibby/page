package config

import "testing"

func TestRemapPath(t *testing.T) {
	maps, err := parsePathMap("/downloads=/mnt/user/downloads,/data/books=/mnt/user/books")
	if err != nil {
		t.Fatal(err)
	}
	c := &Config{pathMaps: maps}

	tests := []struct {
		in, want string
	}{
		{"/downloads/book.epub", "/mnt/user/downloads/book.epub"},
		{"/downloads", "/mnt/user/downloads"},
		{"/data/books/x/y.m4b", "/mnt/user/books/x/y.m4b"},
		{"/unmapped/path", "/unmapped/path"},
		// must not partial-match a path segment
		{"/downloadsX/book.epub", "/downloadsX/book.epub"},
	}
	for _, tt := range tests {
		if got := c.RemapPath(tt.in); got != tt.want {
			t.Errorf("RemapPath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParsePathMapInvalid(t *testing.T) {
	if _, err := parsePathMap("/no-equals-sign"); err == nil {
		t.Error("expected error for entry without '='")
	}
}
