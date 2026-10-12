package library

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/abibby/page/app/models"
	"github.com/abibby/page/services/bookmeta"
	"github.com/abibby/page/services/opf"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"gosalusa.com/database/model"
)

// Scan indexes missing author/book directories in place. Each directory is one
// book, including any audio tracks in subdirectories. Existing records are left
// untouched; repeated scans can safely run alongside normal imports.
func (s *Store) Scan(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	authors, err := os.ReadDir(s.root)
	if err != nil {
		return err
	}
	var scanErrors []error
	for _, author := range authors {
		if !author.IsDir() {
			continue
		}
		books, err := os.ReadDir(filepath.Join(s.root, author.Name()))
		if err != nil {
			scanErrors = append(scanErrors, fmt.Errorf("scan author %q: %w", author.Name(), err))
			continue
		}
		for _, book := range books {
			if err := ctx.Err(); err != nil {
				return errors.Join(append(scanErrors, err)...)
			}
			if !book.IsDir() {
				continue
			}
			directory := filepath.Join(author.Name(), book.Name())
			if err := s.scanBook(ctx, directory); err != nil {
				scanErrors = append(scanErrors, fmt.Errorf("scan book %q: %w", directory, err))
			}
		}
	}
	return errors.Join(scanErrors...)
}

func (s *Store) scanBook(ctx context.Context, directory string) error {
	// Keep the existence check and insert in the same locked transaction so an
	// overlapping scan or import cannot register a partially written book.
	return s.update(ctx, func(tx *sqlx.Tx) error {
		count, err := models.BookQuery(ctx).Where("path", "=", directory).Count(tx)
		if err != nil {
			return err
		}
		if count != 0 {
			return nil
		}
		full, err := s.fullPath(directory)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		author := authorIDSuffix.ReplaceAllString(filepath.Base(filepath.Dir(directory)), "")
		b := &models.Book{
			Title:   authorIDSuffix.ReplaceAllString(filepath.Base(directory), ""),
			Authors: author, AuthorSort: author, Directory: directory,
			Formats: []string{}, Identifiers: map[string]string{}, Tags: []string{}, Languages: []string{},
			LastModified: now, Timestamp: now.Format(time.RFC3339Nano), UUID: uuid.NewString(),
		}
		err = filepath.WalkDir(full, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			// WalkDir never follows directory symlinks. Ignore file symlinks too.
			if !entry.Type().IsRegular() {
				return nil
			}
			relative, err := filepath.Rel(s.root, path)
			if err != nil {
				return err
			}
			if path == filepath.Join(full, "cover.jpg") {
				b.Cover = relative
			}
			if !scanFormat(path) {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			b.Formats = append(b.Formats, relative)
			b.Size += int(info.Size())
			return nil
		})
		if err != nil || len(b.Formats) == 0 {
			return err
		}
		if err := s.scanMetadata(b, full); err != nil {
			return err
		}
		if s.dryRun {
			return nil
		}
		if err := model.SaveContext(ctx, tx, b); err != nil {
			return err
		}
		return rememberAuthor(ctx, tx, strings.Split(b.Authors, " & ")[0], filepath.Dir(directory))
	})
}

func scanFormat(path string) bool {
	if isAudio(path) {
		return true
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".epub", ".pdf", ".mobi", ".azw", ".azw3", ".kfx", ".djvu", ".cbz", ".cbr":
		return true
	}
	return false
}

func (s *Store) scanMetadata(b *models.Book, directory string) error {
	metadataPath := filepath.Join(directory, "metadata.opf")
	info, err := os.Lstat(metadataPath)
	if err == nil && info.Mode().IsRegular() {
		data, err := os.ReadFile(metadataPath)
		if err != nil {
			return err
		}
		var pkg opf.Package
		if err := xml.Unmarshal(data, &pkg); err != nil {
			return fmt.Errorf("read metadata.opf: %w", err)
		}
		authors := []string{}
		for _, creator := range pkg.Metadata.Creators {
			if name := strings.TrimSpace(creator.Value); name != "" {
				authors = append(authors, name)
			}
		}
		setScanNames(b, pkg.Metadata.Title, strings.Join(authors, " & "))
		b.Description = pkg.Metadata.Description
		if language := strings.TrimSpace(pkg.Metadata.Language); language != "" {
			b.Languages = append(b.Languages, language)
		}
		b.Tags = append(b.Tags, pkg.Metadata.Subjects...)
		for _, id := range pkg.Metadata.Identifiers {
			if scheme := strings.ToLower(strings.TrimSpace(id.Scheme)); scheme != "" {
				b.Identifiers[scheme] = strings.TrimSpace(id.Value)
			}
		}
		b.ISBN = b.Identifiers["isbn"]
		return nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	// EPUB metadata is local. Audio extraction currently calls external AI, so
	// audio-only folders use their names when no OPF metadata is available.
	for _, format := range b.Formats {
		if strings.EqualFold(filepath.Ext(format), ".epub") {
			meta, err := bookmeta.Extract(filepath.Join(s.root, format))
			if err != nil {
				return err
			}
			setScanNames(b, meta.Title, meta.Author)
			b.ISBN = meta.ISBN
			if meta.ISBN != "" {
				b.Identifiers["isbn"] = meta.ISBN
			}
			break
		}
	}
	return nil
}

func setScanNames(b *models.Book, title, author string) {
	if title = strings.TrimSpace(title); title != "" {
		b.Title = title
	}
	if author = strings.TrimSpace(author); author != "" {
		b.Authors = author
		b.AuthorSort = author
	}
}
