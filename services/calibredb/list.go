package calibredb

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/abibby/page/services/calibredb/flags"
	"github.com/abibby/salusa/clog"
)

type Field string

const (
	FieldAuthorSort   = "author_sort"
	FieldAuthors      = "authors"
	FieldComments     = "comments"
	FieldCover        = "cover"
	FieldFormats      = "formats"
	FieldIdentifiers  = "identifiers"
	FieldISBN         = "isbn"
	FieldLanguages    = "languages"
	FieldLastModified = "last_modified"
	FieldPubdate      = "pubdate"
	FieldPublisher    = "publisher"
	FieldRating       = "rating"
	FieldSeries       = "series"
	FieldSeriesIndex  = "series_index"
	FieldSize         = "size"
	FieldTags         = "tags"
	FieldTemplate     = "template"
	FieldTimestamp    = "timestamp"
	FieldTitle        = "title"
	FieldUUID         = "uuid"
	FieldAll          = "all"
)

type ListFlags struct {
	// Sort results in ascending order
	Ascending bool `flag:"--ascending"`

	// The fields to display when listing books in the database. Should be a comma separated list of fields. Available fields: author_sort, authors, comments, cover, formats, identifiers, isbn, languages, last_modified, pubdate, publisher, rating, series, series_index, size, tags, template, timestamp, title, uuid Default: title,authors. The special field "all" can be used to select all fields. In addition to the builtin fields above, custom fields are also available as *field_name, for example, for a custom field #rating, use the name: *rating
	Fields []Field `flag:"--fields|join:,"`

	// The maximum number of results to display. Default: all
	Limit int `flag:"--limit"`

	// The prefix for all file paths. Default is the absolute path to the library folder.
	Prefix string `flag:"--prefix"`

	// Filter the results by the search query. For the format of the search query, please see the search related documentation in the User Manual. Default is to do no filtering.
	Search string `flag:"--search"`

	// The field by which to sort the results. You can specify multiple fields by separating them with commas. Available fields: author_sort, authors, comments, cover, formats, identifiers, isbn, languages, last_modified, pubdate, publisher, rating, series, series_index, size, tags, template, timestamp, title, uuid Default: id. In addition to the builtin fields above, custom fields are also available as *field_name, for example, for a custom field #rating, use the name: *rating
	SortBy Field `flag:"--sort-by"`

	// // The template to run if "template" is in the field list. Note that templates are ignored while connecting to a calibre server. Default: None
	// Template string

	// // Path to a file containing the template to run if "template" is in the field list. Default: None
	// TemplateFile string

	// // Heading for the template column. Default: template. This option is ignored if the option --for-machine is set
	// TemplateHeading string

}

func (o *ListFlags) appendArgs(args []string) []string {
	args = flags.Append(args, o)
	args = append(args, "--for-machine")
	return args
}

type Book struct {
	ID           int               `json:"id"`
	Title        string            `json:"title"`
	Authors      string            `json:"authors"`
	AuthorSort   string            `json:"author_sort"`
	Formats      []string          `json:"formats"`
	Identifiers  map[string]string `json:"identifiers"`
	Cover        string            `json:"cover"`
	ISBN         string            `json:"isbn"`
	Languages    []string          `json:"languages"`
	LastModified time.Time         `json:"last_modified"`
	Pubdate      time.Time         `json:"pubdate"`
	Series       string            `json:"series"`
	SeriesIndex  float64           `json:"series_index"`
	Size         int               `json:"size"`
	Tags         []string          `json:"tags"`
	Timestamp    string            `json:"timestamp"`
	UUID         string            `json:"uuid"`
}

func (c *Client) List(ctx context.Context, options *ListFlags) ([]Book, error) {
	b, err := c.exec(ctx, false, options, "list")
	if err != nil {
		return nil, err
	}

	books := []Book{}

	start := bytes.Index(b, []byte("["))
	if start > 0 {
		clog.Use(ctx).Warn("calibredb list warning", "warnings", strings.TrimSpace(string(b[:start])))
	}
	err = json.Unmarshal(b[start:], &books)
	if err != nil {
		return nil, err
	}
	return books, nil
}
