package calibredb

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
)

// Add the specified files as books to the database. You can also specify
// folders, see the folder related options below.
//
// Whenever you pass arguments to calibredb that have spaces in them, enclose
// the arguments in quotation marks. For example: “/some path/with spaces”
type AddFlags struct {
	// Set the authors of the added book(s)
	Authors []string

	// If books with similar titles and authors are found, merge the incoming formats (files) automatically into existing book records. A value of "ignore" means duplicate formats are discarded. A value of "overwrite" means duplicate formats in the library are overwritten with the newly added files. A value of "new_record" means duplicate formats are placed into a new book record.
	AutoMerge bool

	// Path to the cover to use for the added book
	Cover string

	// Add books to database even if they already exist. Comparison is done based on book titles and authors. Note that the --automerge option takes precedence.
	Duplicates bool

	// Add an empty book (a book with no formats)
	Empty bool

	// Set the identifiers for this book, e.g. -I asin:XXX -I isbn:YYY
	Identifier map[string]string

	// Set the ISBN of the added book(s)
	ISBN string

	// A comma separated list of languages (best to use ISO639 language codes, though some language names may also be recognized)
	Languages []string

	// Set the series of the added book(s)
	Series string

	// Set the series number of the added book(s)
	SeriesIndex float64

	// Set the tags of the added book(s)
	Tags []string

	// Set the title of the added book(s)
	Title string
}

func (f *AddFlags) appendArgs(args []string) []string {
	if len(f.Authors) > 0 {
		args = append(args, "--authors", f.AuthorsString())
	}

	if f.AutoMerge {
		args = append(args, "--automerge")
	}
	if f.Cover != "" {
		args = append(args, "--cover", f.Cover)
	}
	if f.Duplicates {
		args = append(args, "--duplicates")
	}
	if f.Empty {
		args = append(args, "--empty")
	}
	if f.Identifier != nil {
		for k, v := range f.Identifier {
			args = append(args, "--identifier", fmt.Sprintf("%s:%s", k, v))
		}
	}
	if f.ISBN != "" {
		args = append(args, "--isbn", f.ISBN)
	}
	if len(f.Languages) > 0 {
		args = append(args, "--languages", strings.Join(f.Languages, ", "))
	}
	if f.Series != "" {
		args = append(args, "--series", f.Series)
	}
	if f.SeriesIndex != 0 {
		args = append(args, "--series-index", strconv.FormatFloat(f.SeriesIndex, 'f', -1, 64))
	}
	if len(f.Tags) > 0 {
		args = append(args, "--tags", strings.Join(f.Tags, ", "))
	}
	if f.Title != "" {
		args = append(args, "--title", f.Title)
	}
	return args
}

func (f *AddFlags) AuthorsString() string {
	if len(f.Authors) == 0 {
		return ""
	}
	return strings.Join(f.Authors, " & ")
}

func (i *Client) Add(ctx context.Context, file string, options *AddFlags) (int, error) {
	b, err := i.exec(ctx, true, options, "add", file)
	if err != nil {
		return 0, fmt.Errorf("calibredb.Client.Add: %w", err)
	}

	idBytes, ok := bytes.CutPrefix(b, []byte("Added book ids: "))
	if !ok {
		return 0, fmt.Errorf("calibredb.Client.Add: invalid prefix: %s", b)
	}
	commaIndex := bytes.Index(idBytes, []byte(","))
	if commaIndex > 0 {
		idBytes = idBytes[:commaIndex]
	}
	id, err := strconv.Atoi(string(bytes.Trim(idBytes, "\n \t")))
	if err != nil {
		return 0, fmt.Errorf("calibredb.Client.Add: %w", err)
	}
	return id, nil
}
