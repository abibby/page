// Package library stores the book catalogue in SQLite and manages library files.
package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/abibby/page/app/models"
	"github.com/abibby/page/config"
	"github.com/jmoiron/sqlx"
	"gosalusa.com/database"
)

type ListOptions struct {
	Search    string
	Limit     int
	SortBy    string
	Ascending bool
}

type Store struct {
	db     *sqlx.DB
	root   string
	dryRun bool
	mu     sync.Mutex
}

func New(ctx context.Context, cfg *config.Config, db *sqlx.DB) (*Store, error) {
	if cfg.LibraryPath == "" {
		return nil, errors.New("LIBRARY_PATH must be configured (CALIBRE_LIBRARY is also supported)")
	}
	root, err := filepath.Abs(cfg.LibraryPath)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db, root: root, dryRun: cfg.DryRun}
	if err := s.importCalibre(ctx); err != nil {
		return nil, fmt.Errorf("import existing library: %w", err)
	}
	return s, nil
}

func (s *Store) read(ctx context.Context, f func(*sqlx.Tx) error) error {
	return database.NewRead(ctx, &s.mu, s.db)(f)
}
func (s *Store) update(ctx context.Context, f func(*sqlx.Tx) error) error {
	return database.NewUpdate(ctx, &s.mu, s.db)(f)
}

// Paths are relative in the database, so moving the library root needs no rewrite.
func (s *Store) fullPath(p string) (string, error) {
	if !filepath.IsLocal(p) {
		return "", fmt.Errorf("invalid library path %q", p)
	}
	return filepath.Join(s.root, p), nil
}

func (s *Store) expand(b *models.Book) error {
	var err error
	b.Directory, err = s.fullPath(b.Directory)
	if err != nil {
		return err
	}
	if b.Cover != "" {
		b.Cover, err = s.fullPath(b.Cover)
		if err != nil {
			return err
		}
	}
	for i, f := range b.Formats {
		b.Formats[i], err = s.fullPath(f)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Get(ctx context.Context, id int) (*models.Book, error) {
	b := &models.Book{}
	err := s.read(ctx, func(tx *sqlx.Tx) error {
		return tx.GetContext(ctx, b, "SELECT * FROM library_books WHERE id = ?", id)
	})
	if err != nil {
		return nil, err
	}
	if err := s.expand(b); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *Store) List(ctx context.Context, options *ListOptions) ([]models.Book, error) {
	if options == nil {
		options = &ListOptions{}
	}
	query := "SELECT * FROM library_books"
	args := []any{}
	search := strings.TrimSpace(options.Search)
	if search != "" {
		field, value, prefixed := strings.Cut(search, ":")
		switch {
		case prefixed && field == "id":
			query += " WHERE id = ?"
			args = append(args, value)
		case prefixed && field == "identifiers":
			key, value, ok := strings.Cut(value, ":")
			if !ok {
				return nil, errors.New("identifier search requires identifiers:type:value")
			}
			query += " WHERE EXISTS (SELECT 1 FROM json_each(identifiers) WHERE key = ? AND value = ?)"
			args = append(args, key, value)
		case prefixed && (field == "title" || field == "authors" || field == "series" || field == "isbn" || field == "tags"):
			query += " WHERE " + field + " LIKE ? ESCAPE '\\'"
			args = append(args, like(value))
		default:
			query += " WHERE (title LIKE ? ESCAPE '\\' OR authors LIKE ? ESCAPE '\\' OR series LIKE ? ESCAPE '\\' OR isbn LIKE ? ESCAPE '\\' OR tags LIKE ? ESCAPE '\\')"
			for range 5 {
				args = append(args, like(search))
			}
		}
	}
	order := options.SortBy
	if order == "" {
		order = "id"
	}
	switch order {
	case "id", "title", "authors", "author_sort", "series", "series_index", "isbn", "timestamp", "last_modified", "pubdate", "size", "uuid", "cover", "formats", "identifiers", "languages", "tags":
	default:
		return nil, fmt.Errorf("unsupported book sort field %q", order)
	}
	query += " ORDER BY " + order
	if options.Ascending {
		query += " ASC"
	} else {
		query += " DESC"
	}
	query += ", id ASC"
	if options.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, options.Limit)
	}
	books := []models.Book{}
	err := s.read(ctx, func(tx *sqlx.Tx) error { return tx.SelectContext(ctx, &books, query, args...) })
	if err != nil {
		return nil, err
	}
	for i := range books {
		if err := s.expand(&books[i]); err != nil {
			return nil, err
		}
	}
	return books, nil
}

func like(s string) string {
	return "%" + strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(strings.Trim(s, `"`)) + "%"
}

const bookColumns = `title, authors, author_sort, path, description, formats, identifiers, cover, isbn, languages, last_modified, pubdate, series, series_index, size, tags, timestamp, uuid`
const bookValues = `:title, :authors, :author_sort, :path, :description, :formats, :identifiers, :cover, :isbn, :languages, :last_modified, :pubdate, :series, :series_index, :size, :tags, :timestamp, :uuid`

func insertBook(ctx context.Context, tx *sqlx.Tx, b *models.Book) error {
	query := "INSERT INTO library_books (" + bookColumns + ") VALUES (" + bookValues + ")"
	if b.ID != 0 {
		query = "INSERT INTO library_books (id, " + bookColumns + ") VALUES (:id, " + bookValues + ")"
	}
	result, err := tx.NamedExecContext(ctx, query, b)
	if err != nil {
		return err
	}
	if b.ID == 0 {
		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		b.ID = int(id)
	}
	return nil
}

func existingBook(ctx context.Context, tx *sqlx.Tx, hardcoverID int) (*models.Book, error) {
	if hardcoverID <= 0 {
		return nil, sql.ErrNoRows
	}
	b := &models.Book{}
	err := tx.GetContext(ctx, b, `SELECT * FROM library_books WHERE json_extract(identifiers, '$."hardcover-id"') = ? ORDER BY id LIMIT 1`, fmt.Sprint(hardcoverID))
	return b, err
}
