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
	"gosalusa.com/database/builder"
	"gosalusa.com/database/model"
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
		return models.BookQuery(ctx).Where("id", "=", id).Load(tx, b)
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
	query := models.BookQuery(ctx)
	search := strings.TrimSpace(options.Search)
	if search != "" {
		field, value, prefixed := strings.Cut(search, ":")
		switch {
		case prefixed && field == "id":
			query.Where("id", "=", value)
		case prefixed && field == "identifiers":
			key, value, ok := strings.Cut(value, ":")
			if !ok {
				return nil, errors.New("identifier search requires identifiers:type:value")
			}
			query.WhereExists(builder.NewBuilder().SelectRaw("1").From("json_each").
				WhereColumn("json", "=", "library_books.identifiers").
				Where("key", "=", key).Where("value", "=", value))
		case prefixed && (field == "title" || field == "authors" || field == "series" || field == "isbn" || field == "tags"):
			query.WhereRaw(field+" LIKE ? ESCAPE '\\'", like(value))
		default:
			query.And(func(q *builder.Conditions) {
				for _, column := range []string{"title", "authors", "series", "isbn", "tags"} {
					q.OrWhereRaw(column+" LIKE ? ESCAPE '\\'", like(search))
				}
			})
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
	if options.Ascending {
		query.OrderBy(order)
	} else {
		query.OrderByDesc(order)
	}
	query.OrderBy("id")
	if options.Limit > 0 {
		query.Limit(options.Limit)
	}
	var records []*models.Book
	err := s.read(ctx, func(tx *sqlx.Tx) error {
		var err error
		records, err = query.Get(tx)
		return err
	})
	if err != nil {
		return nil, err
	}
	books := make([]models.Book, len(records))
	for i, book := range records {
		if err := s.expand(book); err != nil {
			return nil, err
		}
		books[i] = *book
	}
	return books, nil
}

func like(s string) string {
	return "%" + strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(strings.Trim(s, `"`)) + "%"
}

func existingBook(ctx context.Context, tx *sqlx.Tx, hardcoverID int) (*models.Book, error) {
	if hardcoverID <= 0 {
		return nil, sql.ErrNoRows
	}
	b := &models.Book{}
	err := models.BookQuery(ctx).
		WhereRaw(`json_extract(identifiers, '$."hardcover-id"') = ?`, fmt.Sprint(hardcoverID)).
		OrderBy("id").Limit(1).Load(tx, b)
	return b, err
}

// rememberAuthor keeps the first path registered for an author. Callers hold
// the store's update transaction across the lookup and insert.
func rememberAuthor(ctx context.Context, tx *sqlx.Tx, name, path string) error {
	count, err := models.LibraryAuthorQuery(ctx).Where("name", "=", name).Count(tx)
	if err != nil || count != 0 {
		return err
	}
	return model.SaveContext(ctx, tx, &models.LibraryAuthor{Name: name, Path: path})
}
