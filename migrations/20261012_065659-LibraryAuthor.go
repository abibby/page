package migrations

import (
	"gosalusa.com/database/migrate"
	"gosalusa.com/database/schema"
)

func init() {
	migrations.Add(&migrate.Migration{
		Name: "20261012_065659-LibraryAuthor",
		Up: schema.Create("library_authors", func(table *schema.Blueprint) {
			table.String("name").Primary()
			table.String("path")
		}),
		Down: schema.DropIfExists("library_authors"),
	})
}
