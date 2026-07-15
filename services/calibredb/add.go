package calibredb

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/abibby/page/services/calibredb/flags"
)

// Add the specified files as books to the database. You can also specify
// folders, see the folder related options below.
//
// Whenever you pass arguments to calibredb that have spaces in them, enclose
// the arguments in quotation marks. For example: “/some path/with spaces”
type AddFlags struct {
	// Set the authors of the added book(s)
	Authors []string `flag:"--authors|join: & "`

	// If books with similar titles and authors are found, merge the incoming formats (files) automatically into existing book records. A value of "ignore" means duplicate formats are discarded. A value of "overwrite" means duplicate formats in the library are overwritten with the newly added files. A value of "new_record" means duplicate formats are placed into a new book record.
	AutoMerge bool `flag:"--automerge"`

	// Path to the cover to use for the added book
	Cover string `flag:"--cover"`

	// Add books to database even if they already exist. Comparison is done based on book titles and authors. Note that the --automerge option takes precedence.
	Duplicates bool `flag:"--duplicates"`

	// Add an empty book (a book with no formats)
	Empty bool `flag:"--empty"`

	// Set the identifiers for this book, e.g. -I asin:XXX -I isbn:YYY
	Identifier map[string]string

	// Set the ISBN of the added book(s)
	ISBN string `flag:"--isbn"`

	// A comma separated list of languages (best to use ISO639 language codes, though some language names may also be recognized)
	Languages []string `flag:"--languages|join:,"`

	// Set the series of the added book(s)
	Series string `flag:"--series"`

	// Set the series number of the added book(s)
	SeriesIndex float64 `flag:"--series-index"`

	// Set the tags of the added book(s)
	Tags []string `flag:"--tags|join:,"`

	// Set the title of the added book(s)
	Title string `flag:"--title"`
}

func (f *AddFlags) appendArgs(args []string) []string {
	args = flags.Append(args, f)
	if f.Identifier != nil {
		for k, v := range f.Identifier {
			args = append(args, "--identifier", fmt.Sprintf("%s:%s", k, v))
		}
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
