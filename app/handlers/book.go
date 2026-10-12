package handlers

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/abibby/page/app/models"
	"github.com/abibby/page/config"
	"github.com/abibby/page/services/hardcover"
	"github.com/abibby/page/services/importer"
	"github.com/abibby/page/services/library"
	"gosalusa.com/request"
)

type FullBook struct {
	models.Book

	Description string   `json:"description"`
	Files       []string `json:"files"`
}

type BookListRequest struct {
	Search    string `query:"search"`
	Limit     int    `query:"limit"`
	SortBy    string `query:"order"`
	Ascending bool   `query:"ascending"`

	Ctx context.Context `inject:""`
	DB  *library.Store  `inject:""`
	Cfg *config.Config  `inject:""`
}

var BookList = request.Handler(func(r *BookListRequest) ([]models.Book, error) {
	limit := 20
	if r.Limit > 0 && r.Limit < 100 {
		limit = r.Limit
	}
	books, err := r.DB.List(r.Ctx, &library.ListOptions{
		Search:    r.Search,
		Limit:     limit,
		SortBy:    r.SortBy,
		Ascending: r.Ascending,
	})
	if err != nil {
		return nil, err
	}
	for i := range books {
		cleanBook(r.Cfg, &books[i])
	}
	return books, nil
})

type BookViewRequest struct {
	ID int `path:"id"`

	Ctx context.Context `inject:""`
	DB  *library.Store  `inject:""`
	Cfg *config.Config  `inject:""`
}

var BookView = request.Handler(func(r *BookViewRequest) (*FullBook, error) {
	b, err := r.DB.Get(r.Ctx, r.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, request.ErrStatusNotFound
	}
	if err != nil {
		return nil, err
	}
	files, err := os.ReadDir(b.Directory)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	fileNames := make([]string, len(files))
	for i, f := range files {
		fileNames[i] = f.Name()
	}
	cleanBook(r.Cfg, b)

	return &FullBook{
		Book:        *b,
		Description: b.Description,
		Files:       fileNames,
	}, nil
})

type BookImportRequest struct {
	BookID int     `json:"book_id"`
	File   fs.File `json:"file"`

	Ctx context.Context    `inject:""`
	App *importer.Importer `inject:""`
	DB  *library.Store     `inject:""`
}
type BookImportResponse struct {
	BookID int `json:"book_id"`
}

var BookImport = request.Handler(func(r *BookImportRequest) (*BookImportResponse, error) {
	defer r.File.Close() //nolint:errcheck

	stat, err := r.File.Stat()
	if err != nil {
		return nil, err
	}

	tempDir, err := os.MkdirTemp("", "book-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck
	f, err := os.Create(filepath.Join(tempDir, filepath.Base(stat.Name())))
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck

	_, err = io.Copy(f, r.File)
	if err != nil {
		return nil, err
	}
	if r.BookID != 0 {
		err = r.DB.AddFormat(r.Ctx, r.BookID, f.Name())
		if errors.Is(err, sql.ErrNoRows) {
			return nil, request.ErrStatusNotFound
		}
		if err != nil {
			return nil, err
		}
		return &BookImportResponse{
			BookID: r.BookID,
		}, nil
	}

	id, err := r.App.ImportFile(r.Ctx, f.Name())
	if err != nil {
		return nil, err
	}

	return &BookImportResponse{
		BookID: id,
	}, nil
})

type BookAddRequest struct {
	HardcoverID int `json:"hardcover_id"`

	Ctx       context.Context   `inject:""`
	Hardcover *hardcover.Client `inject:""`
	Library   *library.Store    `inject:""`
}

type BookAddResponse struct {
	BookID int `json:"book_id"`
}

var BookAdd = request.Handler(func(r *BookAddRequest) (*BookAddResponse, error) {
	book, err := r.Hardcover.GetBook(r.Ctx, r.HardcoverID)
	if err != nil {
		return nil, err
	}

	id, err := r.Library.AddEmptyBook(r.Ctx, book)
	if err != nil {
		return nil, err
	}

	return &BookAddResponse{
		BookID: id,
	}, nil

})

func cleanBook(cfg *config.Config, book *models.Book) {
	for i, f := range book.Formats {
		book.Formats[i] = cleanPath(cfg, f)
	}
	book.Cover = cleanPath(cfg, book.Cover)
}

func cleanPath(cfg *config.Config, p string) string {
	if p == "" {
		return ""
	}
	root, err := filepath.Abs(cfg.LibraryPath)
	if err != nil {
		return ""
	}
	relative, err := filepath.Rel(root, p)
	if err != nil || !filepath.IsLocal(relative) {
		return ""
	}
	parts := strings.Split(filepath.ToSlash(relative), "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return "/api/d/" + strings.Join(parts, "/")
}
