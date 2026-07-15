package migrations

import (
	"github.com/abibby/salusa/database/migrate"
	"github.com/abibby/salusa/database/schema"
)

func init() {
	migrations.Add(&migrate.Migration{
		Name: "20240321_051037-User",
		Up: schema.Create("users", func(table *schema.Blueprint) {
			table.Blob("id").Primary()
			table.String("email").Unique()
			table.Blob("password")
			table.Bool("validated")
			table.String("lookup_token")
		}),
		Down: schema.DropIfExists("users"),
	})
}
