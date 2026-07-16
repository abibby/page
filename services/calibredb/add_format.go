package calibredb

import (
	"context"
	"strconv"

	"github.com/abibby/page/services/calibredb/flags"
)

// Add the e-book in ebook_file to the available formats for the logical book
// identified by id. You can get id by using the search command. If the format
// already exists, it is replaced, unless the do not replace option is
// specified.
//
// Whenever you pass arguments to calibredb that have spaces in them, enclose
// the arguments in quotation marks. For example: “/some path/with spaces”
type AddFormatFlags struct {
	// Add the file as an extra data file to the book, not an ebook format
	AsExtraDataFile bool `flag:"--as-extra-data-file"`

	// Do not replace the format if it already exists
	DontReplace bool `flag:"--dont-replace"`
}

func (o *AddFormatFlags) appendArgs(args []string) []string {
	return flags.Append(args, o)
}

func (c *Client) AddFormat(ctx context.Context, id int, ebookFile string, options *AddFormatFlags) error {
	_, err := c.exec(ctx, true, options, "add_format", strconv.Itoa(id), ebookFile)
	return err
}
