package migrations

import (
	"context"

	"gosalusa.com/database"
	"gosalusa.com/database/migrate"
	"gosalusa.com/database/schema"
)

func init() {
	migrations.Add(&migrate.Migration{
		Name: "20261012_070000-LibraryStorage",
		Up: schema.Run(func(ctx context.Context, db database.DB) error {
			for _, query := range []string{
				`CREATE TABLE IF NOT EXISTS library_authors (name TEXT PRIMARY KEY COLLATE NOCASE, path TEXT NOT NULL)`,
				`CREATE TABLE IF NOT EXISTS library_imports (source TEXT PRIMARY KEY)`,
				`CREATE UNIQUE INDEX IF NOT EXISTS library_books_path ON library_books (path)`,
				`CREATE INDEX IF NOT EXISTS library_books_hardcover ON library_books (json_extract(identifiers, '$."hardcover-id"'))`,
			} {
				if _, err := database.Exec(ctx, db, query, nil); err != nil {
					return err
				}
			}
			return nil
		}),
		Down: schema.Run(func(ctx context.Context, db database.DB) error {
			for _, query := range []string{
				`DROP INDEX IF EXISTS library_books_hardcover`,
				`DROP INDEX IF EXISTS library_books_path`,
				`DROP TABLE IF EXISTS library_imports`,
				`DROP TABLE IF EXISTS library_authors`,
			} {
				if _, err := database.Exec(ctx, db, query, nil); err != nil {
					return err
				}
			}
			return nil
		}),
	})
}
