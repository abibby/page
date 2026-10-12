package models

import (
	"context"
	"time"

	"gosalusa.com/database/builder"
	"gosalusa.com/database/jsoncolumn"
	"gosalusa.com/database/model"
)

//go:generate spice generate:migration
type Book struct {
	model.BaseModel

	ID           int                            `json:"id" db:"id,primary,autoincrement"`
	Title        string                         `json:"title" db:"title"`
	Authors      string                         `json:"authors" db:"authors"`
	AuthorSort   string                         `json:"author_sort" db:"author_sort"`
	Formats      jsoncolumn.Slice[string]       `json:"formats" db:"formats"`
	Identifiers  jsoncolumn.Map[string, string] `json:"identifiers" db:"identifiers"`
	Cover        string                         `json:"cover" db:"cover"`
	ISBN         string                         `json:"isbn" db:"isbn"`
	Languages    jsoncolumn.Slice[string]       `json:"languages" db:"languages"`
	LastModified time.Time                      `json:"last_modified" db:"last_modified"`
	Pubdate      time.Time                      `json:"pubdate" db:"pubdate"`
	Series       string                         `json:"series" db:"series"`
	SeriesIndex  float64                        `json:"series_index" db:"series_index"`
	Size         int                            `json:"size" db:"size"`
	Tags         jsoncolumn.Slice[string]       `json:"tags" db:"tags"`
	Timestamp    string                         `json:"timestamp" db:"timestamp"`
	UUID         string                         `json:"uuid" db:"uuid"`
	Directory    string                         `json:"-" db:"path"`
	Description  string                         `json:"-" db:"description"`
}

func (*Book) Table() string {
	return "library_books"
}

func BookQuery(ctx context.Context) *builder.ModelBuilder[*Book] {
	return builder.From[*Book]().WithContext(ctx)
}
