package migrations

import (
	"gosalusa.com/database/migrate"
	"gosalusa.com/database/schema"
)

func init() {
	migrations.Add(&migrate.Migration{
		Name: "20261012_060923-Book",
		Up: schema.Create("library_books", func(table *schema.Blueprint) {
			table.Int("id").Primary().AutoIncrement()
			table.String("title")
			table.String("authors")
			table.String("author_sort")
			table.JSON("formats")
			table.JSON("identifiers")
			table.String("cover")
			table.String("isbn")
			table.JSON("languages")
			table.DateTime("last_modified")
			table.DateTime("pubdate")
			table.String("series")
			table.Float64("series_index")
			table.Int("size")
			table.JSON("tags")
			table.String("timestamp")
			table.String("uuid")
			table.String("path").Unique()
			table.String("description")
		}),
		Down: schema.DropIfExists("library_books"),
	})
}
