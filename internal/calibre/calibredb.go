package calibre

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type Book struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	Authors string `json:"authors"`
	// Formats     []string          `json:"formats"`
	Identifiers map[string]string `json:"identifiers"`
}

func (i *Importer) list(ctx context.Context, q string) ([]Book, error) {
	b, err := i.exec(ctx, "list",
		"--for-machine",
		"--fields", "title,authors,isbn,formats,identifiers",
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
	if i.cfg.CalibreServer == "" || i.cfg.CalibreUsername == "" || i.cfg.CalibrePassword == "" {
		return nil, fmt.Errorf("CALIBRE_SERVER, CALIBRE_USERNAME, and CALIBRE_PASSWORD must all be set")
	}
	staticArgs := []string{
		"--with-library", i.cfg.CalibreServer,
		"--username", i.cfg.CalibreUsername,
		"--password", i.cfg.CalibrePassword,
	}
	// fmt.Printf("%s %s\n", i.Bin, strings.Join(append(staticArgs, args...), " "))
	b, err := exec.CommandContext(ctx, i.Bin, append(staticArgs, args...)...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("calibredb command failed: %s %s: %w", i.Bin, strings.Join(args, " "), err)
	}
	// fmt.Printf("%s\n", b)
	return b, nil
}
