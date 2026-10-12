package migrations

import (
	"gosalusa.com/database/migrate"
	"gosalusa.com/database/schema"
)

func init() {
	migrations.Add(&migrate.Migration{
		Name: "20261012_065659-LibraryImport",
		Up: schema.Create("library_imports", func(table *schema.Blueprint) {
			table.String("source").Primary()
		}),
		Down: schema.DropIfExists("library_imports"),
	})
}
