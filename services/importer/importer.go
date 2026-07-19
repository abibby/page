package importer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/abibby/page/config"
	"github.com/abibby/page/services/bookmeta"
	"github.com/abibby/page/services/calibre"
	"github.com/abibby/page/services/hardcover"
	"github.com/autobrr/go-qbittorrent"
)

var ErrNoBook = errors.New("no book")

type Importer struct {
	cfg     *config.Config
	hc      *hardcover.Client
	calibre *calibre.Importer
	qbt     *qbittorrent.Client
	log     *slog.Logger

	hcCache map[string]*hardcover.Book
}

func New(cfg *config.Config, hc *hardcover.Client, importer *calibre.Importer, qbt *qbittorrent.Client, logger *slog.Logger) *Importer {
	return &Importer{
		cfg:     cfg,
		hc:      hc,
		calibre: importer,
		qbt:     qbt,
		log:     logger,
		hcCache: map[string]*hardcover.Book{},
	}
}

// runPass logs in afresh (the SID may have expired between polls), lists
// completed tagged torrents, and imports any not yet processed.
func (a *Importer) RunPass(ctx context.Context) error {
	defer a.clearCache()

	torrents, err := a.qbt.GetTorrentsCtx(ctx, qbittorrent.TorrentFilterOptions{
		Tag:    a.cfg.QbitTag,
		Filter: "completed",
	})
	if err != nil {
		return err
	}

	for _, t := range torrents {
		if slices.Contains(strings.Split(t.Tags, ", "), a.cfg.QbitDoneTag) {
			continue
		}
		// The "completed" filter can include still-moving torrents; require 100%.
		if t.Progress < 1.0 {
			continue
		}
		if err := a.processTorrent(ctx, t); err != nil {
			a.log.Error("torrent failed to process", "torrent", t.Name, "error", err)
			continue // leave unmarked so we retry next pass
		}
		if err := a.qbt.AddTagsCtx(ctx, []string{t.Hash}, a.cfg.QbitDoneTag); err != nil {
			a.log.Error("torrent failed to add done tag", "torrent", t.Name, "error", err)
		}
	}
	return nil
}

func (a *Importer) processTorrent(ctx context.Context, t qbittorrent.Torrent) error {
	files, err := a.qbt.GetFilesInformationCtx(ctx, t.Hash)
	if err != nil {
		return err
	}

	imported := 0
	hasError := false
	for _, f := range *files {
		if !bookmeta.Supported(f.Name) || f.Progress < 1.0 {
			continue
		}
		hostPath := a.cfg.RemapPath(path.Join(t.SavePath, f.Name))
		if _, err := os.Stat(hostPath); err != nil {
			a.log.Warn("file not readable on host, skipping", "file", filepath.Base(hostPath), "error", err)
			hasError = true
			continue
		}
		if _, err := a.ImportFile(ctx, hostPath); err != nil {
			a.log.Warn("failed to import file", "file", filepath.Base(hostPath), "error", err)
			hasError = true
			continue
		}
		imported++
	}

	if imported == 0 {
		a.log.Info("no book files imported", "torrent", t.Name)
	}

	if hasError {
		if err := a.qbt.AddTagsCtx(ctx, []string{t.Hash}, a.cfg.QbitErrorTag); err != nil {
			a.log.Error("torrent failed to add error tag", "torrent", t.Name, "error", err)
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

func (a *Importer) ImportFile(ctx context.Context, path string) (int, error) {
	book, meta, err := a.findBook(ctx, path)
	if err != nil {
		return 0, fmt.Errorf("metadata extract failed: %w", err)
	}

	label := meta.Title
	if book != nil && book.Title != "" {
		label = book.Title
	}
	if label == "" {
		label = filepath.Base(path)
	}
	id, err := a.calibre.AddBook(ctx, path, meta.IsAudiobook, book)
	if err != nil {
		return 0, err
	}
	a.log.Info("imported book", "title", label, "isbn", meta.ISBN)
	return id, nil
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

	a.calibre.ClearCache()
}
