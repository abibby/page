package migrations

import (
	"gosalusa.com/database/migrate"
	"gosalusa.com/database/schema"
)

func init() {
	migrations.Add(&migrate.Migration{
		Name: "20261012_070000-LibraryStorage",
		Up:   schema.Raw(`CREATE INDEX IF NOT EXISTS library_books_hardcover ON library_books (json_extract(identifiers, '$."hardcover-id"'))`),
		Down: schema.Raw(`DROP INDEX IF EXISTS library_books_hardcover`),
	})
}
