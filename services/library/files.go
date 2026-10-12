package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/abibby/page/app/models"
	"github.com/abibby/page/services/hardcover"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"gosalusa.com/database/builder"
	"gosalusa.com/database/model"
)

func component(name string) string {
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, strings.TrimSpace(name))
	name = strings.Trim(name, ". ")
	if name == "" {
		return "Unknown"
	}
	// Leave room for the book ID and extension on common filesystems.
	for len(name) > 180 {
		name = string([]rune(name)[:len([]rune(name))-1])
	}
	return name
}

var authorIDSuffix = regexp.MustCompile(` \([0-9]+\)$`)

func (s *Store) authorDirectory(ctx context.Context, tx *sqlx.Tx, author string) (string, error) {
	var directory string
	err := models.LibraryAuthorQuery(ctx).Select("path").Where("name", "=", author).Load(tx, &directory)
	if err == nil {
		return directory, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	directory = component(author)
	entries, err := os.ReadDir(s.root)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	matches := []string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strings.EqualFold(entry.Name(), directory) {
			matches = []string{entry.Name()}
			break
		}
		if strings.EqualFold(authorIDSuffix.ReplaceAllString(entry.Name(), ""), directory) {
			matches = append(matches, entry.Name())
		}
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("multiple existing folders for author %q", author)
	}
	if len(matches) == 1 {
		directory = matches[0]
	}
	err = model.SaveContext(ctx, tx, &models.LibraryAuthor{Name: author, Path: directory})
	return directory, err
}

func (s *Store) AddEmptyBook(ctx context.Context, meta *hardcover.Book) (int, error) {
	return s.AddBook(ctx, "", false, meta)
}

func (s *Store) AddBook(ctx context.Context, file string, audiobook bool, meta *hardcover.Book) (int, error) {
	if meta == nil {
		return 0, errors.New("book metadata is required")
	}
	if s.dryRun {
		return 0, nil
	}
	var id int
	var createdDir string
	var createdFiles []string
	err := s.update(ctx, func(tx *sqlx.Tx) error {
		b, err := existingBook(ctx, tx, meta.HardcoverID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if errors.Is(err, sql.ErrNoRows) {
			authors := []string{}
			for _, author := range meta.Authors {
				if author = strings.TrimSpace(author); author != "" {
					authors = append(authors, author)
				}
			}
			if len(authors) == 0 {
				authors = []string{"Unknown"}
			}
			directory, err := s.authorDirectory(ctx, tx, authors[0])
			if err != nil {
				return err
			}
			now := time.Now().UTC()
			b = &models.Book{
				Title: strings.TrimSpace(meta.Title), Authors: strings.Join(authors, " & "), AuthorSort: strings.Join(authors, " & "),
				Identifiers: map[string]string{}, Formats: []string{}, Tags: []string{}, Languages: []string{},
				Series: meta.Series, SeriesIndex: meta.SeriesIndex, ISBN: meta.ISBN13,
				LastModified: now, Timestamp: now.Format(time.RFC3339Nano), UUID: uuid.NewString(),
			}
			if b.Title == "" {
				b.Title = "Unknown"
			}
			if b.ISBN == "" {
				b.ISBN = meta.ISBN10
			}
			if b.ISBN != "" {
				b.Identifiers["isbn"] = b.ISBN
			}
			if meta.HardcoverID > 0 {
				b.Identifiers["hardcover-id"] = strconv.Itoa(meta.HardcoverID)
			}
			if meta.ReleaseYear > 0 {
				b.Pubdate = time.Date(meta.ReleaseYear, 1, 1, 0, 0, 0, 0, time.UTC)
			}
			b.Directory = b.UUID // Temporary unique path, updated before the transaction commits.
			if err := model.SaveContext(ctx, tx, b); err != nil {
				return err
			}
			b.Directory = filepath.Join(directory, fmt.Sprintf("%s (%d)", component(b.Title), b.ID))
			full, err := s.fullPath(b.Directory)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
				return err
			}
			// Never adopt or overwrite an unrelated, unindexed book directory.
			if err := os.Mkdir(full, 0755); err != nil {
				return err
			}
			createdDir = full
			if meta.CoverURL != "" {
				cover := filepath.Join(full, "cover.jpg")
				if err := downloadCover(ctx, meta.CoverURL, cover); err == nil {
					b.Cover = filepath.Join(b.Directory, "cover.jpg")
				}
			}
			if err := models.BookQuery(ctx).Where("id", "=", b.ID).Update(tx, builder.Updates{
				"path": b.Directory, "cover": b.Cover,
			}); err != nil {
				return err
			}
		}
		id = b.ID
		if file != "" {
			created, err := s.addFile(ctx, tx, b, file, audiobook)
			if created != "" {
				createdFiles = append(createdFiles, created)
			}
			return err
		}
		return nil
	})
	if err != nil {
		for _, f := range createdFiles {
			_ = os.Remove(f)
		}
		if createdDir != "" {
			_ = os.RemoveAll(createdDir)
		}
		return 0, err
	}
	return id, nil
}

// AddFormat preserves existing ebook formats and stores audio tracks by filename.
func (s *Store) AddFormat(ctx context.Context, id int, file string) error {
	if s.dryRun {
		return nil
	}
	var created string
	err := s.update(ctx, func(tx *sqlx.Tx) error {
		b := &models.Book{}
		if err := models.BookQuery(ctx).Where("id", "=", id).Load(tx, b); err != nil {
			return err
		}
		var err error
		created, err = s.addFile(ctx, tx, b, file, isAudio(file))
		return err
	})
	if err != nil && created != "" {
		_ = os.Remove(created)
	}
	return err
}

func isAudio(file string) bool {
	switch strings.ToLower(filepath.Ext(file)) {
	case ".mp3", ".m4b", ".m4a", ".flac", ".ogg", ".wav":
		return true
	}
	return false
}

func (s *Store) addFile(ctx context.Context, tx *sqlx.Tx, b *models.Book, file string, audiobook bool) (string, error) {
	ext := strings.ToLower(filepath.Ext(file))
	if ext == "" {
		return "", errors.New("book file must have an extension")
	}
	if !audiobook {
		for _, f := range b.Formats {
			if strings.EqualFold(filepath.Ext(f), ext) {
				return "", nil
			}
		}
	}
	name := component(strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))) + ext
	if !audiobook {
		name = component(b.Title) + ext
	}
	relative := filepath.Join(b.Directory, name)
	destination, err := s.fullPath(relative)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return "", err
	}
	created := ""
	if err := linkOrCopy(file, destination); err == nil {
		created = destination
	} else if !errors.Is(err, os.ErrExist) {
		return "", err
	}
	found := false
	for _, f := range b.Formats {
		if f == relative {
			found = true
			break
		}
	}
	if !found {
		info, err := os.Stat(destination)
		if err != nil {
			return created, err
		}
		if !info.Mode().IsRegular() {
			return created, fmt.Errorf("not a regular book file: %s", destination)
		}
		b.Formats = append(b.Formats, relative)
		b.Size += int(info.Size())
	}
	b.LastModified = time.Now().UTC()
	err = models.BookQuery(ctx).Where("id", "=", b.ID).Update(tx, builder.Updates{
		"formats": b.Formats, "size": b.Size, "last_modified": b.LastModified,
	})
	return created, err
}

// Never truncate an existing file when a hard link fails (including EEXIST).
func linkOrCopy(source, destination string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular file: %s", source)
	}
	if err := os.Link(source, destination); err == nil {
		return nil
	} else if errors.Is(err, os.ErrExist) {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close() //nolint:errcheck
	out, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	syncErr := out.Sync()
	closeErr := out.Close()
	if err := errors.Join(copyErr, syncErr, closeErr); err != nil {
		_ = os.Remove(destination)
		return err
	}
	return nil
}

func downloadCover(ctx context.Context, url, destination string) error {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cover download: status %d", resp.StatusCode)
	}
	f, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	n, copyErr := io.Copy(f, io.LimitReader(resp.Body, (25<<20)+1))
	err = errors.Join(copyErr, f.Close())
	if n > 25<<20 {
		err = errors.New("cover exceeds 25 MiB")
	}
	if err != nil {
		_ = os.Remove(destination)
	}
	return err
}
