package calibredb

import (
	"context"
	"encoding/xml"
	"strconv"

	"github.com/abibby/page/services/opf"
)

func (c *Client) ShowMetadata(ctx context.Context, id int) (*opf.Metadata, error) {
	b, err := c.exec(ctx, false, nil, "show_metadata", "--as-opf", strconv.Itoa(id))
	if err != nil {
		return nil, err
	}

	pkg := &opf.Package{}

	err = xml.Unmarshal(b, pkg)
	if err != nil {
		return nil, err
	}
	return &pkg.Metadata, nil
}
