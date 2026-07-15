package importer

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/abibby/page/config"
	"github.com/abibby/page/services/bookmeta"
	"github.com/abibby/page/services/calibre"
	"github.com/abibby/page/services/hardcover"
	"github.com/abibby/page/services/qbittorrent"
)

var ErrNoBook = errors.New("no book")

type Importer struct {
	cfg      *config.Config
	hc       *hardcover.Client
	importer *calibre.Importer

	hcCache map[string]*hardcover.Book
}

func New(cfg *config.Config, hc *hardcover.Client, importer *calibre.Importer) *Importer {
	return &Importer{
		cfg:      cfg,
		hc:       hc,
		importer: importer,
		hcCache:  map[string]*hardcover.Book{},
	}
}

// runPass logs in afresh (the SID may have expired between polls), lists
// completed tagged torrents, and imports any not yet processed.
func (a *Importer) RunPass(ctx context.Context) error {
	defer a.clearCache()

	qb, err := qbittorrent.New(a.cfg.QbitURL, a.cfg.QbitUsername, a.cfg.QbitPassword)
	if err != nil {
		return err
	}

	torrents, err := qb.TorrentsByTag(a.cfg.QbitTag, "completed")
	if err != nil {
		return err
	}

	for _, t := range torrents {
		tags := strings.Split(t.Tags, ", ")
		if slices.Contains(tags, a.cfg.QbitDoneTag) {
			continue
		}
		// The "completed" filter can include still-moving torrents; require 100%.
		if t.Progress < 1.0 {
			continue
		}
		if err := a.processTorrent(ctx, qb, t); err != nil {
			log.Printf("torrent %q: %v", t.Name, err)
			continue // leave unmarked so we retry next pass
		}
		if err := qb.AddTag(&t, a.cfg.QbitDoneTag); err != nil {
			log.Printf("state mark %q: %v", t.Name, err)
		}
	}
	return nil
}

func (a *Importer) processTorrent(ctx context.Context, qb *qbittorrent.Client, t qbittorrent.Torrent) error {
	files, err := qb.Files(t.Hash)
	if err != nil {
		return err
	}

	imported := 0
	hasError := false
	for _, f := range files {
		if !bookmeta.Supported(f.Name) || f.Progress < 1.0 {
			continue
		}
		hostPath := a.cfg.RemapPath(t.AbsPath(f))
		if _, err := os.Stat(hostPath); err != nil {
			log.Printf("  skip %s: not readable on host (%v)", filepath.Base(hostPath), err)
			hasError = true
			continue
		}
		if err := a.ImportFile(ctx, hostPath); err != nil {
			log.Printf("  %s: %v", filepath.Base(hostPath), err)
			hasError = true
			continue
		}
		imported++
	}

	if imported == 0 {
		log.Printf("torrent %q: no book files imported", t.Name)
	}

	if hasError {
		if err := qb.AddTag(&t, a.cfg.QbitErrorTag); err != nil {
			log.Printf("state mark %q: %v", t.Name, err)
		}
	}
	return nil
}

func (a *Importer) findBook(ctx context.Context, path string) (*hardcover.Book, *bookmeta.Meta, error) {
	meta, err := bookmeta.Extract(path)
	if err != nil {
		return nil, nil, err
	}

	b, ok := a.lookup(ctx, meta)
	if !ok {
		return nil, nil, ErrNoBook
	}
	return b, &meta, nil
}

func (a *Importer) ImportFile(ctx context.Context, path string) error {
	book, meta, err := a.findBook(ctx, path)
	if err != nil {
		log.Printf("  %s: metadata extract failed (%v); importing without enrichment", filepath.Base(path), err)
		return nil
	}

	label := meta.Title
	if book != nil && book.Title != "" {
		label = book.Title
	}
	if label == "" {
		label = filepath.Base(path)
	}

	if err := a.importer.AddBook(ctx, path, meta, book); err != nil {
		return err
	}
	log.Printf("  imported %q (isbn=%s)", label, meta.ISBN)
	return nil
}

// lookup enriches metadata via Hardcover: by ISBN first, then a title/author
// search fallback. Falls back to the file's own metadata if Hardcover has no
// match, or nil if nothing is known.
func (a *Importer) lookup(ctx context.Context, meta bookmeta.Meta) (*hardcover.Book, bool) {
	b, ok := a.hcCache[meta.CacheID()]
	if ok {
		return b, true
	}
	if meta.ISBN != "" {
		book, err := a.hc.LookupByISBN(ctx, meta.ISBN)
		if err != nil {
			log.Printf("  hardcover isbn lookup: %v", err)
		} else if book != nil {
			a.hcCache[meta.CacheID()] = book
			return book, true
		}
	}
	if meta.Title != "" {
		book, err := a.hc.SearchByTitleAuthor(ctx, meta.Title, meta.Author)
		if err != nil {
			log.Printf("  hardcover search: %v", err)
		} else if book != nil {
			a.hcCache[meta.CacheID()] = book
			return book, true
		}
	}
	// if meta.Title != "" || meta.Author != "" {
	// 	b := &hardcover.Book{Title: meta.Title, ISBN13: meta.ISBN}
	// 	if meta.Author != "" {
	// 		b.Authors = []string{meta.Author}
	// 	}
	// 	return b, true
	// }
	return nil, false
}
func (a *Importer) clearCache() {
	a.hcCache = map[string]*hardcover.Book{}

	a.importer.ClearCache()
}
