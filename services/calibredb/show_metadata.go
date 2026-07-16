package calibredb

import (
	"context"
	"encoding/xml"
	"fmt"
	"strconv"

	"github.com/abibby/page/services/opf"
)

func (c *Client) ShowMetadata(ctx context.Context, id int) (*opf.Metadata, error) {
	b, err := c.exec(ctx, false, nil, "show_metadata", "--as-opf", strconv.Itoa(id))
	if err != nil {
		return nil, fmt.Errorf("calibredb.Client.ShowMetadata: %w", err)
	}

	pkg := &opf.Package{}

	err = xml.Unmarshal(b, pkg)
	if err != nil {
		return nil, fmt.Errorf("calibredb.Client.ShowMetadata: xml unmarshal %w", err)
	}
	return &pkg.Metadata, nil
}
