// Package calibre imports book files into a Calibre library via calibredb.
package calibre

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/abibby/page/internal/hardcover"
)

// Importer adds files to a Calibre library using the calibredb CLI.
type Importer struct {
	Bin           string // path to the calibredb binary
	Library       string // --with-library value (local path)
	Tag           string // tag applied to every imported book
	AddDuplicates bool   // pass --duplicates to force-add
	DryRun        bool   // log the command instead of running it
}

// Add imports file into the Calibre library, applying metadata from book (which
// may be nil if no Hardcover match was found).
func (i *Importer) Add(ctx context.Context, file string, book *hardcover.Book) error {
	args := []string{"add", "--with-library", i.Library}
	if i.AddDuplicates {
		args = append(args, "--duplicates")
	}
	if i.Tag != "" {
		args = append(args, "--tags", i.Tag)
	}

	var coverPath string
	if book != nil {
		if id, ok := i.getExistingID(ctx, book); ok {
			args = []string{"add_format", "--dont-replace", id}
		} else {
			if title := strings.TrimSpace(book.Title); title != "" {
				args = append(args, "--title", title)
			}
			if len(book.Authors) > 0 {
				args = append(args, "--authors", strings.Join(book.Authors, " & "))
			}
			if isbn := firstNonEmpty(book.ISBN13, book.ISBN10); isbn != "" {
				args = append(args, "--isbn", isbn)
			}
			if book.HardcoverID != "" {
				args = append(args, "--identifier", fmt.Sprintf("hardcover:%s", book.HardcoverID))
			}
			if book.Series != "" {
				args = append(args, "--series", book.Series)
				if idx := book.SeriesIndexString(); idx != "" {
					args = append(args, "--series-index", idx)
				}
			}
			if book.CoverURL != "" {
				if p, err := downloadCover(ctx, book.CoverURL); err == nil {
					coverPath = p
					args = append(args, "--cover", p)
				}
			}
		}
	}
	args = append(args, file)

	if coverPath != "" {
		defer os.Remove(coverPath)
	}

	if i.DryRun {
		fmt.Printf("[dry-run] %s %s\n", i.Bin, strings.Join(quoteArgs(args), " "))
		os.Exit(1)
		return nil
	}

	cmd := exec.CommandContext(ctx, i.Bin, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("calibredb add failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (i *Importer) getExistingID(ctx context.Context, b *hardcover.Book) (string, bool) {
	cmd := exec.CommandContext(ctx, i.Bin, "search", "title:"+b.Title)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", false
	}
	return string(out), true
}

func downloadCover(ctx context.Context, url string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cover download: status %d", resp.StatusCode)
	}
	ext := filepath.Ext(url)
	if ext == "" || len(ext) > 5 {
		ext = ".jpg"
	}
	f, err := os.CreateTemp("", "cover-*"+ext)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, io.LimitReader(resp.Body, 25<<20)); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func quoteArgs(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		if strings.ContainsAny(a, " \t") {
			out[i] = fmt.Sprintf("%q", a)
		} else {
			out[i] = a
		}
	}
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
