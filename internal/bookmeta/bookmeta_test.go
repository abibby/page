package bookmeta

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestFindISBN(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want string
	}{
		{"isbn13 hyphenated", []string{"ISBN: 978-0-13-468599-1"}, "9780134685991"},
		{"isbn13 plain", []string{"urn:isbn:9780306406157"}, "9780306406157"},
		{"isbn10", []string{"0306406152"}, "0306406152"},
		{"isbn13 preferred over 10", []string{"0306406152", "9780306406157"}, "9780306406157"},
		{"none", []string{"not a number"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findISBN(tt.in...); got != tt.want {
				t.Errorf("findISBN(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestExtractEPUB(t *testing.T) {
	container := `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles><rootfile full-path="content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`
	opf := `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
 <metadata xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf">
  <dc:title>Test Book</dc:title>
  <dc:creator opf:role="aut">Jane Author</dc:creator>
  <dc:identifier opf:scheme="ISBN">978-0-13-468599-1</dc:identifier>
 </metadata>
</package>`

	path := filepath.Join(t.TempDir(), "book.epub")
	writeZip(t, path, map[string]string{
		"META-INF/container.xml": container,
		"content.opf":            opf,
	})

	meta, err := Extract(path)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if meta.Title != "Test Book" {
		t.Errorf("Title = %q, want %q", meta.Title, "Test Book")
	}
	if meta.Author != "Jane Author" {
		t.Errorf("Author = %q, want %q", meta.Author, "Jane Author")
	}
	if meta.ISBN != "9780134685991" {
		t.Errorf("ISBN = %q, want %q", meta.ISBN, "9780134685991")
	}
}

func TestExtractMP4(t *testing.T) {
	data := box("data", append([]byte{0, 0, 0, 1, 0, 0, 0, 0}, []byte("Audio Title")...))
	titleItem := box("\xa9nam", data)

	authData := box("data", append([]byte{0, 0, 0, 1, 0, 0, 0, 0}, []byte("Spoken Author")...))
	authItem := box("\xa9ART", authData)

	// Comment carrying the ISBN.
	cmtData := box("data", append([]byte{0, 0, 0, 1, 0, 0, 0, 0}, []byte("ISBN 978-0-306-40615-7")...))
	cmtItem := box("\xa9cmt", cmtData)

	ilst := box("ilst", concat(titleItem, authItem, cmtItem))
	// meta is a FullBox: 4 zero bytes of version/flags, then children.
	meta := box("meta", append([]byte{0, 0, 0, 0}, ilst...))
	udta := box("udta", meta)
	moov := box("moov", udta)

	path := filepath.Join(t.TempDir(), "book.m4b")
	if err := os.WriteFile(path, moov, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Extract(path)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if got.Title != "Audio Title" {
		t.Errorf("Title = %q, want %q", got.Title, "Audio Title")
	}
	if got.Author != "Spoken Author" {
		t.Errorf("Author = %q, want %q", got.Author, "Spoken Author")
	}
	if got.ISBN != "9780306406157" {
		t.Errorf("ISBN = %q, want %q", got.ISBN, "9780306406157")
	}
}

func TestExtractMP4Freeform(t *testing.T) {
	// A "----" freeform atom: mean + name + data.
	mean := box("mean", append([]byte{0, 0, 0, 0}, []byte("com.apple.iTunes")...))
	name := box("name", append([]byte{0, 0, 0, 0}, []byte("ISBN")...))
	data := box("data", append([]byte{0, 0, 0, 1, 0, 0, 0, 0}, []byte("9780306406157")...))
	freeform := box("----", concat(mean, name, data))

	ilst := box("ilst", freeform)
	meta := box("meta", append([]byte{0, 0, 0, 0}, ilst...))
	moov := box("moov", box("udta", meta))

	path := filepath.Join(t.TempDir(), "ff.m4b")
	if err := os.WriteFile(path, moov, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Extract(path)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if got.ISBN != "9780306406157" {
		t.Errorf("freeform ISBN = %q, want %q", got.ISBN, "9780306406157")
	}
}

// box builds an MP4 box from a 4-char type and payload.
func box(typ string, payload []byte) []byte {
	if len(typ) != 4 {
		panic("box type must be 4 bytes")
	}
	out := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint32(out[:4], uint32(8+len(payload)))
	copy(out[4:8], typ)
	copy(out[8:], payload)
	return out
}

func concat(parts ...[]byte) []byte {
	var buf bytes.Buffer
	for _, p := range parts {
		buf.Write(p)
	}
	return buf.Bytes()
}

func writeZip(t *testing.T, path string, files map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}
