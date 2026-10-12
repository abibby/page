package models

import (
	"context"

	"gosalusa.com/database/builder"
	"gosalusa.com/database/model"
)

//go:generate spice generate:migration
type LibraryAuthor struct {
	model.BaseModel

	Name string `json:"name" db:"name,primary"`
	Path string `json:"path" db:"path"`
}

func (*LibraryAuthor) Table() string {
	return "library_authors"
}

func LibraryAuthorQuery(ctx context.Context) *builder.ModelBuilder[*LibraryAuthor] {
	return builder.From[*LibraryAuthor]().WithContext(ctx)
}
