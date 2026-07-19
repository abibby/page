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
	"github.com/abibby/salusa/clog"
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
func (i *Importer) AddBook(ctx context.Context, file string, isAudiobook bool, book *hardcover.Book) (int, error) {
	if isAudiobook {
		return i.addAudiobook(ctx, file, book)
	}
	return i.addEbook(ctx, file, book)
}

func (i *Importer) AddEmptyBook(ctx context.Context, book *hardcover.Book) (int, error) {
	flags, cleanup := hardcoverBookToAddFlags(ctx, book)
	defer cleanup()
	flags.Empty = true
	id, err := i.client.Add(ctx, "", flags)
	if err != nil {
		return 0, fmt.Errorf("failed to create book: %w", err)
	}
	return id, nil
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
				err := os.Remove(coverPath)
				if err != nil {
					clog.Use(ctx).Warn("failed to remove temp cover file", "error", err, "file", coverPath)
				}
			}
		}
}

func (i *Importer) addEbook(ctx context.Context, file string, book *hardcover.Book) (int, error) {
	if existingBook, ok := i.getExistingID(ctx, book); ok {
		return existingBook.ID, i.client.AddFormat(ctx, existingBook.ID, file, &calibredb.AddFormatFlags{
			DontReplace: true,
		})
	}
	flags, cleanup := hardcoverBookToAddFlags(ctx, book)
	defer cleanup()
	return i.client.Add(ctx, file, flags)
}

func (i *Importer) addAudiobook(ctx context.Context, file string, book *hardcover.Book) (int, error) {
	if book == nil {
		return 0, fmt.Errorf("calibre.Importer.Add: hardcover book must not be null")
	}

	existingBook, ok := i.getExistingID(ctx, book)
	if !ok {
		id, err := i.AddEmptyBook(ctx, book)
		if err != nil {
			return 0, err
		}

		existingBook = &calibredb.Book{
			ID:      id,
			Title:   book.Title,
			Authors: strings.Join(book.Authors, " & "),
		}
	}

	newFile := path.Join(path.Dir(existingBook.Cover), path.Base(file))

	if i.DryRun {
		fmt.Printf("[dry-run] ln '%s' '%s'\n", file, newFile)
		return 0, nil
	}

	err := linkOrCopy(file, newFile)
	if err != nil {
		return 0, fmt.Errorf("hard link failed: %w", err)
	}
	return existingBook.ID, nil
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
	defer source.Close() //nolint:errcheck

	destination, err := os.Create(newname)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destination.Close() //nolint:errcheck

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
	defer resp.Body.Close() //nolint:errcheck
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
	defer f.Close() //nolint:errcheck
	if _, err := io.Copy(f, io.LimitReader(resp.Body, 25<<20)); err != nil {
		removeErr := os.Remove(f.Name())
		if removeErr != nil {
			clog.Use(ctx).Warn("failed to remove file", "error", removeErr, "file", f.Name())
		}
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
