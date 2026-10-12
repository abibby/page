package library

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/abibby/page/config"
	"github.com/abibby/page/migrations"
	"github.com/abibby/page/services/hardcover"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"gosalusa.com/database/builder"
	"gosalusa.com/database/dialects"
	"gosalusa.com/database/schema"
)

func newDatabase(t *testing.T) *sqlx.DB {
	t.Helper()
	cfg := config.NewSQLiteConfig(filepath.Join(t.TempDir(), "page.db"))
	db, err := sqlx.Open(cfg.DriverName(), cfg.DataSourceName())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, migrations.Use().Up(context.Background(), db))
	return db
}

func writeFile(t *testing.T, name, content string) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(name), 0755))
	require.NoError(t, os.WriteFile(name, []byte(content), 0644))
	return name
}

func readFile(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(name)
	require.NoError(t, err)
	return data
}

func TestImportAndReuseAuthorFolders(t *testing.T) {
	ctx := context.Background()
	for _, existing := range []string{"", "Ada Author", "Ada Author (123)"} {
		t.Run("existing="+existing, func(t *testing.T) {
			root := t.TempDir()
			if existing != "" {
				writeFile(t, filepath.Join(root, existing, "keep.txt"), "untouched")
			}
			db := newDatabase(t)
			cfg := &config.Config{LibraryPath: root}
			store, err := New(ctx, cfg, db)
			require.NoError(t, err)
			meta := &hardcover.Book{HardcoverID: 100, Title: "Example", Authors: []string{"Ada Author"}, ISBN13: "1234"}
			source := writeFile(t, filepath.Join(t.TempDir(), "original.epub"), "original content")
			id, err := store.AddBook(ctx, source, false, meta)
			require.NoError(t, err)
			book, err := store.Get(ctx, id)
			require.NoError(t, err)
			wantAuthor := existing
			if wantAuthor == "" {
				wantAuthor = "Ada Author"
			}
			require.Equal(t, filepath.Join(root, wantAuthor), filepath.Dir(book.Directory))
			require.Len(t, book.Formats, 1)
			require.Equal(t, "original content", string(readFile(t, book.Formats[0])))
			replacement := writeFile(t, filepath.Join(t.TempDir(), "replacement.epub"), "do not overwrite")
			sameID, err := store.AddBook(ctx, replacement, false, meta)
			require.NoError(t, err)
			require.Equal(t, id, sameID)
			require.Equal(t, "original content", string(readFile(t, book.Formats[0])))
			emptyID, err := store.AddEmptyBook(ctx, meta)
			require.NoError(t, err)
			require.Equal(t, id, emptyID)
			// Both cached author paths and identifiers must survive a new store instance.
			store, err = New(ctx, cfg, db)
			require.NoError(t, err)
			meta.HardcoverID++
			meta.Title = "Another book"
			second, err := store.AddEmptyBook(ctx, meta)
			require.NoError(t, err)
			another, err := store.Get(ctx, second)
			require.NoError(t, err)
			require.Equal(t, filepath.Dir(book.Directory), filepath.Dir(another.Directory))
			if existing != "" {
				require.Equal(t, "untouched", string(readFile(t, filepath.Join(root, existing, "keep.txt"))))
			}
		})
	}
}

func TestAudioTracksAndCoverlessBook(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, &config.Config{LibraryPath: t.TempDir()}, newDatabase(t))
	require.NoError(t, err)
	meta := &hardcover.Book{HardcoverID: 1, Title: "Audio", Authors: []string{"Author"}}
	one := writeFile(t, filepath.Join(t.TempDir(), "01.mp3"), "one")
	two := writeFile(t, filepath.Join(t.TempDir(), "02.mp3"), "two")
	id, err := store.AddBook(ctx, one, true, meta)
	require.NoError(t, err)
	require.NoError(t, store.AddFormat(ctx, id, two))
	require.NoError(t, store.AddFormat(ctx, id, two))
	b, err := store.Get(ctx, id)
	require.NoError(t, err)
	require.Empty(t, b.Cover)
	require.Len(t, b.Formats, 2)
	require.Equal(t, 6, b.Size)
	require.Equal(t, filepath.Join(b.Directory, "01.mp3"), b.Formats[0])
	require.Equal(t, "one", string(readFile(t, b.Formats[0])))
	require.Equal(t, "two", string(readFile(t, b.Formats[1])))
}

func TestListSearchSortAndLimit(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, &config.Config{LibraryPath: t.TempDir()}, newDatabase(t))
	require.NoError(t, err)
	for i, title := range []string{"Zebra", "100% real", "Alpha"} {
		_, err := store.AddEmptyBook(ctx, &hardcover.Book{HardcoverID: i + 1, Title: title, Authors: []string{"Author"}, Series: "Series"})
		require.NoError(t, err)
	}
	books, err := store.List(ctx, &ListOptions{SortBy: "title", Ascending: true, Limit: 2})
	require.NoError(t, err)
	require.Len(t, books, 2)
	require.Equal(t, "100% real", books[0].Title)
	require.Equal(t, "Alpha", books[1].Title)
	for _, search := range []string{"100%", "title:100%", "identifiers:hardcover-id:2", "id:2"} {
		books, err := store.List(ctx, &ListOptions{Search: search})
		require.NoError(t, err)
		require.Len(t, books, 1, search)
		require.Equal(t, "100% real", books[0].Title)
	}
	books, err = store.List(ctx, &ListOptions{Search: `'; DROP TABLE library_books; --`})
	require.NoError(t, err)
	require.Empty(t, books)
	_, err = store.List(ctx, &ListOptions{SortBy: "title; DROP TABLE library_books"})
	require.Error(t, err)
	books, err = store.List(ctx, nil)
	require.NoError(t, err)
	require.Len(t, books, 3)
}

func TestFailedImportRollsBackAndDryRunDoesNotWrite(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	db := newDatabase(t)
	store, err := New(ctx, &config.Config{LibraryPath: root}, db)
	require.NoError(t, err)
	meta := &hardcover.Book{HardcoverID: 1, Title: "Book", Authors: []string{"Author"}}
	_, err = store.AddBook(ctx, filepath.Join(root, "missing.epub"), false, meta)
	require.Error(t, err)
	books, err := store.List(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, books)
	require.NoDirExists(t, filepath.Join(root, "Author", "Book (1)"))
	store.dryRun = true
	_, err = store.AddEmptyBook(ctx, meta)
	require.NoError(t, err)
	books, err = store.List(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, books)
	require.NoDirExists(t, filepath.Join(root, "Author", "Book (1)"))
	store.dryRun = false
	_, err = store.AddEmptyBook(ctx, nil)
	require.Error(t, err)
}

func TestMissingBookReturnsNoRows(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, &config.Config{LibraryPath: t.TempDir()}, newDatabase(t))
	require.NoError(t, err)
	_, err = store.Get(ctx, 999)
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.ErrorIs(t, store.AddFormat(ctx, 999, "missing.epub"), sql.ErrNoRows)
}

func TestConcurrentImportsDoNotDuplicate(t *testing.T) {
	ctx := context.Background()
	store, err := New(ctx, &config.Config{LibraryPath: t.TempDir()}, newDatabase(t))
	require.NoError(t, err)
	meta := &hardcover.Book{HardcoverID: 7, Title: "Concurrent", Authors: []string{"Author"}}
	var wg sync.WaitGroup
	results := make(chan error, 8)
	for range 8 {
		wg.Go(func() { _, err := store.AddEmptyBook(ctx, meta); results <- err })
	}
	wg.Wait()
	close(results)
	for err := range results {
		require.NoError(t, err)
	}
	books, err := store.List(ctx, nil)
	require.NoError(t, err)
	require.Len(t, books, 1)
}

// Fixture uses the standard Calibre tables, without Calibre-specific views or
// custom SQLite functions, to exercise a completely CLI-free migration.
func legacyFixture(t *testing.T, root string) string {
	t.Helper()
	source := filepath.Join(root, "metadata.db")
	db, err := sqlx.Open("sqlite", source)
	require.NoError(t, err)
	for _, table := range []*schema.CreateTableBuilder{
		schema.Create("books", func(table *schema.Blueprint) {
			table.Int("id").Primary()
			for _, column := range []string{"title", "author_sort", "path", "uuid"} {
				table.Text(column).Nullable()
			}
			for _, column := range []string{"timestamp", "pubdate", "last_modified"} {
				table.DateTime(column).Nullable()
			}
			table.Float64("series_index").Nullable()
		}),
		schema.Create("authors", func(table *schema.Blueprint) {
			table.Int("id").Primary()
			table.Text("name").Nullable()
		}),
		schema.Create("books_authors_link", func(table *schema.Blueprint) {
			table.Int("id").Primary()
			table.Int("book").Nullable()
			table.Int("author").Nullable()
		}),
		schema.Create("identifiers", func(table *schema.Blueprint) {
			table.Int("book").Nullable()
			table.Text("type").Nullable()
			table.Text("val").Nullable()
		}),
		schema.Create("comments", func(table *schema.Blueprint) {
			table.Int("book").Nullable()
			table.Text("text").Nullable()
		}),
		schema.Create("series", func(table *schema.Blueprint) {
			table.Int("id").Primary()
			table.Text("name").Nullable()
		}),
		schema.Create("books_series_link", func(table *schema.Blueprint) {
			table.Int("book").Nullable()
			table.Int("series").Nullable()
		}),
		schema.Create("tags", func(table *schema.Blueprint) {
			table.Int("id").Primary()
			table.Text("name").Nullable()
		}),
		schema.Create("books_tags_link", func(table *schema.Blueprint) {
			table.Int("id").Primary()
			table.Int("book").Nullable()
			table.Int("tag").Nullable()
		}),
		schema.Create("languages", func(table *schema.Blueprint) {
			table.Int("id").Primary()
			table.Text("lang_code").Nullable()
		}),
		schema.Create("books_languages_link", func(table *schema.Blueprint) {
			table.Int("book").Nullable()
			table.Int("lang_code").Nullable()
			table.Int("item_order").Nullable()
		}),
		schema.Create("data", func(table *schema.Blueprint) {
			table.Int("id").Primary()
			table.Int("book").Nullable()
			table.Text("name").Nullable()
			table.Text("format").Nullable()
			table.Int("uncompressed_size").Nullable()
		}),
	} {
		require.NoError(t, table.Run(context.Background(), db))
	}
	insertLegacyRows(t, db, "books", legacyBookRow(42, "Old Author (9)/Old Book (42)"))
	insertLegacyRows(t, db, "authors", map[string]any{"id": 9, "name": "Old Author"})
	insertLegacyRows(t, db, "books_authors_link", map[string]any{"id": 1, "book": 42, "author": 9})
	insertLegacyRows(t, db, "identifiers",
		map[string]any{"book": 42, "type": "hardcover-id", "val": "12"},
		map[string]any{"book": 42, "type": "isbn", "val": "9781234567890"})
	insertLegacyRows(t, db, "comments", map[string]any{"book": 42, "text": "<p>Existing description</p>"})
	insertLegacyRows(t, db, "series", map[string]any{"id": 1, "name": "Existing series"})
	insertLegacyRows(t, db, "books_series_link", map[string]any{"book": 42, "series": 1})
	insertLegacyRows(t, db, "tags", map[string]any{"id": 1, "name": "Fiction"})
	insertLegacyRows(t, db, "books_tags_link", map[string]any{"id": 1, "book": 42, "tag": 1})
	insertLegacyRows(t, db, "languages", map[string]any{"id": 1, "lang_code": "eng"})
	insertLegacyRows(t, db, "books_languages_link", map[string]any{"book": 42, "lang_code": 1, "item_order": 0})
	insertLegacyRows(t, db, "data", map[string]any{"id": 1, "book": 42, "name": "Original - Old Author", "format": "EPUB", "uncompressed_size": 8})
	require.NoError(t, db.Close())
	directory := filepath.Join(root, "Old Author (9)", "Old Book (42)")
	writeFile(t, filepath.Join(directory, "Original - Old Author.epub"), "original")
	writeFile(t, filepath.Join(directory, "cover.jpg"), "cover")
	writeFile(t, filepath.Join(directory, "01.mp3"), "audio")
	writeFile(t, filepath.Join(directory, "metadata.opf"), "original metadata")
	return source
}

// Calibre tables have no Page models, so fixture inserts use Salusa's dialect
// query representation directly.
func insertLegacyRows(t *testing.T, db *sqlx.DB, table string, rows ...map[string]any) {
	t.Helper()
	dialect, err := dialects.New(db.DriverName())
	require.NoError(t, err)
	query, err := dialect.EncodeInsertQuery(&dialects.InsertQuery{Table: table, Values: rows})
	require.NoError(t, err)
	_, err = db.ExecContext(context.Background(), query.SQL, query.Bindings...)
	require.NoError(t, err)
}

func legacyBookRow(id int, path string) map[string]any {
	return map[string]any{
		"id": id, "title": "Old Book", "author_sort": "Author, Old", "path": path,
		"uuid": "original-uuid", "timestamp": "2020-01-02 03:04:05+00:00",
		"pubdate": "2019-01-01 00:00:00+00:00", "last_modified": "2021-01-02 03:04:05+00:00", "series_index": 2.5,
	}
}

func TestMigrateCalibrePreservesIDsPathsAndMetadata(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	source := legacyFixture(t, root)
	originalHash := sha256.Sum256(readFile(t, source))
	db := newDatabase(t)
	cfg := &config.Config{LibraryPath: root}
	store, err := New(ctx, cfg, db)
	require.NoError(t, err)
	b, err := store.Get(ctx, 42)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, "Old Author (9)", "Old Book (42)"), b.Directory)
	require.Equal(t, "Old Author", b.Authors)
	require.Equal(t, "<p>Existing description</p>", b.Description)
	require.Equal(t, "9781234567890", b.ISBN)
	require.Equal(t, "original-uuid", b.UUID)
	require.Equal(t, "Existing series", b.Series)
	require.Equal(t, 2.5, b.SeriesIndex)
	require.Equal(t, []string{"Fiction"}, []string(b.Tags))
	require.Equal(t, []string{"eng"}, []string(b.Languages))
	require.Equal(t, 2019, b.Pubdate.Year())
	require.Equal(t, 2021, b.LastModified.Year())
	require.Len(t, b.Formats, 2)
	require.Equal(t, "original", string(readFile(t, b.Formats[0])))
	require.Equal(t, "cover", string(readFile(t, b.Cover)))
	newFile := writeFile(t, filepath.Join(t.TempDir(), "02.mp3"), "new audio")
	id, err := store.AddBook(ctx, newFile, true, &hardcover.Book{HardcoverID: 12})
	require.NoError(t, err)
	require.Equal(t, 42, id)
	require.FileExists(t, filepath.Join(b.Directory, "02.mp3"))
	id, err = store.AddEmptyBook(ctx, &hardcover.Book{HardcoverID: 13, Title: "New Book", Authors: []string{"Old Author"}})
	require.NoError(t, err)
	require.Greater(t, id, 42)
	added, err := store.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, filepath.Dir(b.Directory), filepath.Dir(added.Directory))
	// A restart must neither re-import nor overwrite newer Page changes.
	store, err = New(ctx, cfg, db)
	require.NoError(t, err)
	books, err := store.List(ctx, nil)
	require.NoError(t, err)
	require.Len(t, books, 2)
	b, err = store.Get(ctx, 42)
	require.NoError(t, err)
	require.Len(t, b.Formats, 3)
	require.Equal(t, originalHash, sha256.Sum256(readFile(t, source)))
	require.Equal(t, "original metadata", string(readFile(t, filepath.Join(b.Directory, "metadata.opf"))))
}

func TestLegacyImportFailureIsAtomic(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	source := legacyFixture(t, root)
	old, err := sqlx.Open("sqlite", source)
	require.NoError(t, err)
	insertLegacyRows(t, old, "books", legacyBookRow(43, "../outside"))
	require.NoError(t, old.Close())
	db := newDatabase(t)
	_, err = New(ctx, &config.Config{LibraryPath: root}, db)
	require.Error(t, err)
	for _, table := range []string{"library_books", "library_authors", "library_imports"} {
		count, err := builder.NewBuilder().WithContext(ctx).From(table).Count(db)
		require.NoError(t, err)
		require.Zero(t, count, table)
	}
}

func TestMigrateCalibreOptionalMetadata(t *testing.T) {
	for _, missing := range []bool{false, true} {
		name := "null values"
		if missing {
			name = "missing rows"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			root := t.TempDir()
			source := legacyFixture(t, root)
			old, err := sqlx.Open("sqlite", source)
			require.NoError(t, err)
			require.NoError(t, builder.NewBuilder().WithContext(ctx).From("books").
				Where("id", "=", 42).Update(old, builder.Updates{
				"author_sort": nil, "uuid": nil, "timestamp": nil, "pubdate": nil,
			}))
			for table, column := range map[string]string{"comments": "text", "series": "name"} {
				query := builder.NewBuilder().WithContext(ctx).From(table)
				if missing {
					require.NoError(t, query.Delete(old))
				} else {
					require.NoError(t, query.Update(old, builder.Updates{column: nil}))
				}
			}
			require.NoError(t, old.Close())
			store, err := New(ctx, &config.Config{LibraryPath: root}, newDatabase(t))
			require.NoError(t, err)
			book, err := store.Get(ctx, 42)
			require.NoError(t, err)
			require.Empty(t, book.Description)
			require.Empty(t, book.Series)
			require.Empty(t, book.AuthorSort)
			require.Empty(t, book.UUID)
			require.Empty(t, book.Timestamp)
			require.True(t, book.Pubdate.IsZero())
		})
	}
}
