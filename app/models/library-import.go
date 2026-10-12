package models

import (
	"context"

	"gosalusa.com/database/builder"
	"gosalusa.com/database/model"
)

//go:generate spice generate:migration
type LibraryImport struct {
	model.BaseModel

	Source string `json:"source" db:"source,primary"`
}

func (*LibraryImport) Table() string {
	return "library_imports"
}

func LibraryImportQuery(ctx context.Context) *builder.ModelBuilder[*LibraryImport] {
	return builder.From[*LibraryImport]().WithContext(ctx)
}
