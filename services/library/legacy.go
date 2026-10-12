package library

import (
	"context"
	"database/sql"
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
	"gosalusa.com/database/builder"
	"gosalusa.com/database/model"
)

// importCalibre copies the catalogue once, in a transaction. metadata.db is
// opened read-only; the original database and every existing path stay intact.
func (s *Store) importCalibre(ctx context.Context) error {
	var imported int
	if err := s.read(ctx, func(tx *sqlx.Tx) error {
		var err error
		imported, err = models.LibraryImportQuery(ctx).Where("source", "=", "calibre").Count(tx)
		return err
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
		count, err := models.BookQuery(ctx).Count(tx)
		if err != nil {
			return err
		}
		if count != 0 {
			return errors.New("cannot import metadata.db into a nonempty Page catalogue; use a fresh DATABASE_PATH")
		}
		err = database.NewRead(ctx, nil, legacy)(func(old *sqlx.Tx) error {
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
			err := builder.NewBuilder().WithContext(ctx).From("books").
				Select("id", "title", "path", "series_index").
				AddSelectRaw(
					"coalesce(author_sort, '') AS author_sort",
					"coalesce(uuid, '') AS uuid",
					"coalesce(CAST(timestamp AS TEXT), '') AS timestamp",
					"coalesce(CAST(pubdate AS TEXT), '') AS pubdate",
					"CAST(last_modified AS TEXT) AS last_modified",
				).OrderBy("id").Load(old, &records)
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
				if err := builder.NewBuilder().WithContext(ctx).From("authors").Select("authors.name").
					Join("books_authors_link", "books_authors_link.author", "=", "authors.id").
					Where("books_authors_link.book", "=", b.ID).OrderBy("books_authors_link.id").Load(old, &authors); err != nil {
					return err
				}
				// Calibre escapes commas in author names as pipes internally.
				for i := range authors {
					authors[i] = strings.ReplaceAll(authors[i], "|", ",")
				}
				b.Authors = strings.Join(authors, " & ")
				if len(authors) > 0 {
					err := rememberAuthor(ctx, tx, authors[0], filepath.Dir(b.Directory))
					if err != nil {
						return err
					}
				}
				var identifiers []struct {
					Type string
					Val  string
				}
				if err := builder.NewBuilder().WithContext(ctx).From("identifiers").Select("type", "val").
					Where("book", "=", b.ID).Load(old, &identifiers); err != nil {
					return err
				}
				for _, ident := range identifiers {
					b.Identifiers[ident.Type] = ident.Val
				}
				b.ISBN = b.Identifiers["isbn"]
				var description, series sql.NullString
				if err := builder.NewBuilder().WithContext(ctx).SelectSubquery(
					builder.NewBuilder().From("comments").Select("text").Where("book", "=", b.ID), "description",
				).Load(old, &description); err != nil {
					return err
				}
				b.Description = description.String
				if err := builder.NewBuilder().WithContext(ctx).SelectSubquery(
					builder.NewBuilder().From("series").Select("series.name").
						Join("books_series_link", "books_series_link.series", "=", "series.id").
						Where("books_series_link.book", "=", b.ID), "series",
				).Load(old, &series); err != nil {
					return err
				}
				b.Series = series.String
				if err := builder.NewBuilder().WithContext(ctx).From("tags").Select("tags.name").
					Join("books_tags_link", "books_tags_link.tag", "=", "tags.id").
					Where("books_tags_link.book", "=", b.ID).OrderBy("books_tags_link.id").Load(old, &b.Tags); err != nil {
					return err
				}
				if err := builder.NewBuilder().WithContext(ctx).From("languages").Select("languages.lang_code").
					Join("books_languages_link", "books_languages_link.lang_code", "=", "languages.id").
					Where("books_languages_link.book", "=", b.ID).OrderBy("books_languages_link.item_order").Load(old, &b.Languages); err != nil {
					return err
				}
				var formats []struct {
					Name   string
					Format string
					Size   int
				}
				if err := builder.NewBuilder().WithContext(ctx).From("data").Select("name", "format", "uncompressed_size as size").
					Where("book", "=", b.ID).OrderBy("id").Load(old, &formats); err != nil {
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
				if err := model.SaveContext(ctx, tx, b); err != nil {
					return fmt.Errorf("book %d: %w", b.ID, err)
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		return model.SaveContext(ctx, tx, &models.LibraryImport{Source: "calibre"})
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
