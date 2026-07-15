// Package calibre imports book files into a Calibre library via calibredb.
package calibre

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/abibby/page/config"
	"github.com/abibby/page/services/cache"
	"github.com/abibby/page/services/calibredb"
	"github.com/abibby/page/services/hardcover"
)

// Importer adds files to a Calibre library using the calibredb CLI.
type Importer struct {
	DryRun      bool
	libraryPath string

	client *calibredb.Client

	idCache *cache.Cache[int, *calibredb.Book]
}

func NewClient(cfg *config.Config, client *calibredb.Client) *Importer {
	return &Importer{
		DryRun:      cfg.DryRun,
		libraryPath: cfg.CalibreLibrary,
		client:      client,
		idCache:     cache.New[int, *calibredb.Book](),
	}
}

// AddBook imports file into the Calibre library, applying metadata from book (which
// may be nil if no Hardcover match was found).
func (i *Importer) AddBook(ctx context.Context, file string, isAudiobook bool, book *hardcover.Book) error {
	if isAudiobook {
		return i.addAudiobook(ctx, file, book)
	}
	return i.addEbook(ctx, file, book)
}

func hardcoverBookToAddFlags(ctx context.Context, book *hardcover.Book) (*calibredb.AddFlags, func()) {
	coverPath := ""
	if book.CoverURL != "" {
		if p, err := downloadCover(ctx, book.CoverURL); err == nil {
			coverPath = p
		}
	}
	return &calibredb.AddFlags{
			Title:   strings.TrimSpace(book.Title),
			Authors: book.Authors,
			ISBN:    firstNonEmpty(book.ISBN13, book.ISBN10),
			Identifier: map[string]string{
				"hardcover-id": strconv.Itoa(book.HardcoverID),
			},
			Series:      book.Series,
			SeriesIndex: book.SeriesIndex,
			Cover:       coverPath,
		}, func() {
			if coverPath != "" {
				os.Remove(coverPath)
			}
		}
}

func (i *Importer) addEbook(ctx context.Context, file string, book *hardcover.Book) error {
	if existingBook, ok := i.getExistingID(ctx, book); ok {
		return i.client.AddFormat(ctx, existingBook.ID, file, nil)
	}
	flags, cleanup := hardcoverBookToAddFlags(ctx, book)
	defer cleanup()
	_, err := i.client.Add(ctx, file, flags)
	return err
}

func (i *Importer) addAudiobook(ctx context.Context, file string, book *hardcover.Book) error {
	if book == nil {
		return fmt.Errorf("calibre.Importer.Add: hardcover book must not be null")
	}

	existingBook, ok := i.getExistingID(ctx, book)
	if !ok {

		flags, cleanup := hardcoverBookToAddFlags(ctx, book)
		defer cleanup()
		flags.Empty = true
		id, err := i.client.Add(ctx, "", flags)
		if err != nil {
			return fmt.Errorf("failed to create book: %w", err)
		}

		existingBook = &calibredb.Book{
			ID:      id,
			Title:   flags.Title,
			Authors: flags.AuthorsString(),
		}
	}

	newFile := path.Join(path.Dir(existingBook.Cover), path.Base(file))

	if i.DryRun {
		fmt.Printf("[dry-run] ln '%s' '%s'\n", file, newFile)
		return nil
	}

	err := linkOrCopy(file, newFile)
	if err != nil {
		return fmt.Errorf("hard link failed: %w", err)
	}
	return nil
}

func linkOrCopy(oldname, newname string) error {
	err := os.Link(oldname, newname)
	if err == nil {
		return nil
	}

	source, err := os.Open(oldname)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer source.Close()

	destination, err := os.Create(newname)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	if err != nil {
		return fmt.Errorf("failed to copy contents: %w", err)
	}

	return destination.Sync()
}

func (i *Importer) getExistingID(ctx context.Context, b *hardcover.Book) (*calibredb.Book, bool) {
	cb := i.idCache.Get(b.HardcoverID, time.Minute, func() *calibredb.Book {
		books, err := i.client.List(ctx, &calibredb.ListFlags{
			Fields: []calibredb.Field{calibredb.FieldAll},
			Search: fmt.Sprintf("identifiers:hardcover-id:%d", b.HardcoverID),
		})
		if err != nil {
			log.Printf("failed to fetch book list: %v", err)
			return nil
		}
		if len(books) == 0 {
			return nil
		}
		return &books[0]
	})

	return cb, cb != nil
}

func (i *Importer) ClearCache() {
	i.idCache = cache.New[int, *calibredb.Book]()
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

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
