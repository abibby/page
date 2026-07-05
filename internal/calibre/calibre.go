// Package calibre imports book files into a Calibre library via calibredb.
package calibre

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/abibby/page/internal/bookmeta"
	"github.com/abibby/page/internal/config"
	"github.com/abibby/page/internal/hardcover"
)

// Importer adds files to a Calibre library using the calibredb CLI.
type Importer struct {
	Bin    string // path to the calibredb binary
	DryRun bool   // log the command instead of running it

	cfg *config.Config

	idCache map[int]*Book
}

func NewClient(cfg *config.Config) *Importer {
	return &Importer{
		Bin:     cfg.CalibredbBin,
		DryRun:  cfg.DryRun,
		cfg:     cfg,
		idCache: map[int]*Book{},
	}
}

// Add imports file into the Calibre library, applying metadata from book (which
// may be nil if no Hardcover match was found).
func (i *Importer) Add(ctx context.Context, file string, meta *bookmeta.Meta, book *hardcover.Book) error {
	if meta.IsAudiobook {
		return i.addAudiobook(ctx, file, book)
	}
	return i.addEbook(ctx, file, book)
}

func (i *Importer) addEbook(ctx context.Context, file string, book *hardcover.Book) error {
	args := []string{"add"}

	var coverPath string
	if book != nil {
		if existingBook, ok := i.getExistingID(ctx, book); ok {
			args = []string{"add_format", fmt.Sprint(existingBook.ID)}
		} else {
			if title := strings.TrimSpace(book.Title); title != "" {
				args = append(args, "--title", title)
			}
			if len(book.Authors) > 0 {
				args = append(args, "--authors", strings.Join(book.Authors, " & "))
			}
			if isbn := firstNonEmpty(book.ISBN13, book.ISBN10); isbn != "" {
				args = append(args, "--isbn", isbn)
			}
			if book.HardcoverID != 0 {
				args = append(args, "--identifier", fmt.Sprintf("hardcover-id:%d", book.HardcoverID))
			}
			if book.Series != "" {
				args = append(args, "--series", book.Series)
				if idx := book.SeriesIndexString(); idx != "" {
					args = append(args, "--series-index", idx)
				}
			}
			if book.CoverURL != "" {
				if p, err := downloadCover(ctx, book.CoverURL); err == nil {
					coverPath = p
					args = append(args, "--cover", p)
				}
			}
		}
	}

	if file == "" {
		args = append(args, "--empty")
	} else {
		args = append(args, file)
	}

	if coverPath != "" {
		defer os.Remove(coverPath)
	}

	if i.DryRun {
		fmt.Printf("[dry-run] %s %s\n", i.Bin, strings.Join(quoteArgs(args), " "))
		return nil
	}

	out, err := i.exec(ctx, args...)
	if err != nil {
		return fmt.Errorf("calibredb add failed: %w: %s", err, strings.TrimSpace(string(out)))
	}

	return nil
}

func (i *Importer) addAudiobook(ctx context.Context, file string, book *hardcover.Book) error {
	if book == nil {
		return fmt.Errorf("calibre.Importer.Add: hardcover book must not be null")
	}

	existingBook, ok := i.getExistingID(ctx, book)
	if !ok {

		err := i.addEbook(ctx, "", book)
		if err != nil {
			return err
		}
		existingBook, ok = i.getExistingID(ctx, book)
		if !ok {
			return fmt.Errorf("calibre.Importer.Add: could not find created book")
		}
	}

	newFile := path.Join(path.Join(i.cfg.CalibreLibrary, book.Authors[0], fmt.Sprintf("%s (%d)", book.Title, existingBook.ID)), path.Base(file))

	if i.DryRun {
		fmt.Printf("[dry-run] ln '%s' '%s'\n", file, newFile)
		return nil
	}

	err := os.Link(file, newFile)
	if err != nil {
		return fmt.Errorf("hard link failed: %w", err)
	}
	return nil
}

func (i *Importer) getExistingID(ctx context.Context, b *hardcover.Book) (*Book, bool) {
	id, ok := i.idCache[b.HardcoverID]
	if ok {
		return id, true
	}
	books, err := i.list(ctx, fmt.Sprintf("identifiers:hardcover-id:%d", b.HardcoverID))
	if err != nil {
		return nil, false
	}
	if len(books) == 0 {
		return nil, false
	}
	cb := books[0]
	i.idCache[b.HardcoverID] = &cb
	return &cb, true
}

func (i *Importer) ClearCache() {
	i.idCache = map[int]*Book{}
}

func downloadCover(ctx context.Context, url string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cover download: status %d", resp.StatusCode)
	}
	ext := filepath.Ext(url)
	if ext == "" || len(ext) > 5 {
		ext = ".jpg"
	}
	f, err := os.CreateTemp("", "cover-*"+ext)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, io.LimitReader(resp.Body, 25<<20)); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func quoteArgs(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		if strings.ContainsAny(a, " \t") {
			out[i] = fmt.Sprintf("%q", a)
		} else {
			out[i] = a
		}
	}
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
