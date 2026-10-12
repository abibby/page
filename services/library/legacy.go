package library

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/abibby/page/app/models"
	"github.com/abibby/page/config"
	"github.com/jmoiron/sqlx"
	"gosalusa.com/database"
)

// importCalibre copies the catalogue once, in a transaction. metadata.db is
// opened read-only; the original database and every existing path stay intact.
func (s *Store) importCalibre(ctx context.Context) error {
	var imported int
	if err := s.read(ctx, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, &imported, "SELECT count(*) FROM library_imports WHERE source = 'calibre'")
	}); err != nil {
		return err
	}
	if imported != 0 {
		return nil
	}
	source := filepath.Join(s.root, "metadata.db")
	if _, err := os.Stat(source); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	uri := (&url.URL{Scheme: "file", Path: source, RawQuery: "mode=ro"}).String()
	cfg := config.NewSQLiteConfig(uri)
	legacy, err := sqlx.Open(cfg.DriverName(), cfg.DataSourceName())
	if err != nil {
		return err
	}
	defer legacy.Close() //nolint:errcheck
	return s.update(ctx, func(tx *sqlx.Tx) error {
		var count int
		if err := tx.GetContext(ctx, &count, "SELECT count(*) FROM library_books"); err != nil {
			return err
		}
		if count != 0 {
			return errors.New("cannot import metadata.db into a nonempty Page catalogue; use a fresh DATABASE_PATH")
		}
		err := database.NewRead(ctx, nil, legacy)(func(old *sqlx.Tx) error {
			var records []struct {
				ID           int     `db:"id"`
				Title        string  `db:"title"`
				AuthorSort   string  `db:"author_sort"`
				Path         string  `db:"path"`
				UUID         string  `db:"uuid"`
				Timestamp    string  `db:"timestamp"`
				Pubdate      string  `db:"pubdate"`
				LastModified string  `db:"last_modified"`
				SeriesIndex  float64 `db:"series_index"`
			}
			err := old.SelectContext(ctx, &records, `SELECT id, title, coalesce(author_sort, '') AS author_sort, path,
    coalesce(uuid, '') AS uuid, coalesce(CAST(timestamp AS TEXT), '') AS timestamp,
    coalesce(CAST(pubdate AS TEXT), '') AS pubdate, CAST(last_modified AS TEXT) AS last_modified,
    series_index FROM books ORDER BY id`)
			if err != nil {
				return err
			}
			for _, record := range records {
				if _, err := s.fullPath(record.Path); err != nil {
					return err
				}
				b := &models.Book{ID: record.ID, Title: record.Title, AuthorSort: record.AuthorSort,
					Directory: record.Path, UUID: record.UUID, Timestamp: record.Timestamp,
					Pubdate: legacyTime(record.Pubdate), LastModified: legacyTime(record.LastModified), SeriesIndex: record.SeriesIndex,
					Formats: []string{}, Identifiers: map[string]string{}, Tags: []string{}, Languages: []string{},
				}
				authors := []string{}
				if err := old.SelectContext(ctx, &authors, `SELECT a.name FROM authors a JOIN books_authors_link l ON l.author = a.id WHERE l.book = ? ORDER BY l.id`, b.ID); err != nil {
					return err
				}
				// Calibre escapes commas in author names as pipes internally.
				for i := range authors {
					authors[i] = strings.ReplaceAll(authors[i], "|", ",")
				}
				b.Authors = strings.Join(authors, " & ")
				if len(authors) > 0 {
					_, err := tx.ExecContext(ctx, `INSERT INTO library_authors (name, path) VALUES (?, ?) ON CONFLICT(name) DO NOTHING`, authors[0], filepath.Dir(b.Directory))
					if err != nil {
						return err
					}
				}
				var identifiers []struct {
					Type string
					Val  string
				}
				if err := old.SelectContext(ctx, &identifiers, "SELECT type, val FROM identifiers WHERE book = ?", b.ID); err != nil {
					return err
				}
				for _, ident := range identifiers {
					b.Identifiers[ident.Type] = ident.Val
				}
				b.ISBN = b.Identifiers["isbn"]
				if err := old.GetContext(ctx, &b.Description, "SELECT coalesce((SELECT text FROM comments WHERE book = ?), '')", b.ID); err != nil {
					return err
				}
				if err := old.GetContext(ctx, &b.Series, `SELECT coalesce((SELECT s.name FROM series s JOIN books_series_link l ON l.series = s.id WHERE l.book = ?), '')`, b.ID); err != nil {
					return err
				}
				if err := old.SelectContext(ctx, &b.Tags, `SELECT t.name FROM tags t JOIN books_tags_link l ON l.tag = t.id WHERE l.book = ? ORDER BY l.id`, b.ID); err != nil {
					return err
				}
				if err := old.SelectContext(ctx, &b.Languages, `SELECT a.lang_code FROM languages a JOIN books_languages_link l ON l.lang_code = a.id WHERE l.book = ? ORDER BY l.item_order`, b.ID); err != nil {
					return err
				}
				var formats []struct {
					Name   string
					Format string
					Size   int
				}
				if err := old.SelectContext(ctx, &formats, `SELECT name, format, uncompressed_size AS size FROM data WHERE book = ? ORDER BY id`, b.ID); err != nil {
					return err
				}
				known := map[string]bool{}
				for _, f := range formats {
					relative := filepath.Join(b.Directory, f.Name+"."+strings.ToLower(f.Format))
					if _, err := s.fullPath(relative); err != nil {
						return err
					}
					b.Formats = append(b.Formats, relative)
					b.Size += f.Size
					known[relative] = true
				}
				// The previous importer stored audio tracks alongside the book without
				// registering them in Calibre's data table. Keep them in the catalogue too.
				directory, err := s.fullPath(b.Directory)
				if err != nil {
					return err
				}
				entries, err := os.ReadDir(directory)
				if err != nil && !errors.Is(err, os.ErrNotExist) {
					return err
				}
				for _, entry := range entries {
					if entry.IsDir() {
						continue
					}
					relative := filepath.Join(b.Directory, entry.Name())
					if entry.Name() == "cover.jpg" {
						b.Cover = relative
					}
					if isAudio(entry.Name()) && !known[relative] {
						info, err := entry.Info()
						if err != nil {
							return err
						}
						b.Formats = append(b.Formats, relative)
						b.Size += int(info.Size())
					}
				}
				if err := insertBook(ctx, tx, b); err != nil {
					return fmt.Errorf("book %d: %w", b.ID, err)
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "INSERT INTO library_imports (source) VALUES ('calibre')")
		return err
	})
}

func legacyTime(value string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999Z07:00", "2006-01-02 15:04:05.999999999", "2006-01-02"} {
		if t, err := time.Parse(layout, value); err == nil {
			return t
		}
	}
	return time.Time{}
}
