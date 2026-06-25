package calibre

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type Book struct {
	ID      int      `json:"id"`
	Title   string   `json:"title"`
	Authors string   `json:"authors"`
	Formats []string `json:"formats"`
}

func (i *Importer) list(ctx context.Context, q string) ([]Book, error) {
	b, err := i.exec(ctx, "list",
		q,
		"--with-library", i.Library,
		"--for-machine",
		"--fields", "title,authors,isbn,formats",
		"--search", q,
	)
	if err != nil {
		return nil, err
	}

	books := []Book{}

	err = json.Unmarshal(b, &books)
	if err != nil {
		return nil, err
	}
	return books, nil
}

func (i *Importer) exec(ctx context.Context, args ...string) ([]byte, error) {
	b, err := exec.CommandContext(ctx, i.Bin, args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("calibredb command failed: %s %s: %w", i.Bin, strings.Join(args, " "), err)
	}
	return b, nil
}
