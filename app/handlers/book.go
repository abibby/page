package handlers

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/abibby/page/config"
	"github.com/abibby/page/services/calibre"
	"github.com/abibby/page/services/calibredb"
	"github.com/abibby/page/services/hardcover"
	"github.com/abibby/page/services/importer"
	"github.com/abibby/salusa/request"
)

type FullBook struct {
	calibredb.Book

	Description string   `json:"description"`
	Files       []string `json:"files"`
}

type BookListRequest struct {
	Search    string          `query:"search"`
	Limit     int             `query:"limit"`
	SortBy    calibredb.Field `query:"order"`
	Ascending bool            `query:"ascending"`

	Ctx context.Context   `inject:""`
	DB  *calibredb.Client `inject:""`
	Cfg *config.Config    `inject:""`
}

var BookList = request.Handler(func(r *BookListRequest) ([]calibredb.Book, error) {
	limit := 20
	if r.Limit > 0 && r.Limit < 100 {
		limit = r.Limit
	}
	books, err := r.DB.List(r.Ctx, &calibredb.ListFlags{
		Fields:    []calibredb.Field{calibredb.FieldAll},
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

	Ctx context.Context   `inject:""`
	DB  *calibredb.Client `inject:""`
	Cfg *config.Config    `inject:""`
}

var BookView = request.Handler(func(r *BookViewRequest) (*FullBook, error) {
	books, err := r.DB.List(r.Ctx, &calibredb.ListFlags{
		Fields: []calibredb.Field{calibredb.FieldAll},
		Search: fmt.Sprintf("id:%d", r.ID),
	})
	if err != nil {
		return nil, err
	}
	if len(books) == 0 {
		return nil, request.ErrStatusNotFound
	}

	b := books[0]

	meta, err := r.DB.ShowMetadata(r.Ctx, b.ID)
	if err != nil {
		return nil, err
	}

	files, err := os.ReadDir(path.Dir(b.Cover))
	if err != nil {
		return nil, err
	}
	fileNames := make([]string, len(files))
	for i, f := range files {
		fileNames[i] = f.Name()
	}
	cleanBook(r.Cfg, &b)

	return &FullBook{
		Book:        b,
		Description: meta.Description,
		Files:       fileNames,
	}, nil
})

type BookImportRequest struct {
	BookID int     `json:"book_id"`
	File   fs.File `json:"file"`

	Ctx context.Context    `inject:""`
	App *importer.Importer `inject:""`
	DB  *calibredb.Client  `inject:""`
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

	f, err := os.CreateTemp("", "book-*-"+stat.Name())
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck

	defer os.Remove(f.Name()) //nolint:errcheck

	_, err = io.Copy(f, r.File)
	if err != nil {
		return nil, err
	}
	if r.BookID != 0 {
		err = r.DB.AddFormat(r.Ctx, r.BookID, f.Name(), &calibredb.AddFormatFlags{
			DontReplace: true,
		})
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
	Calibre   *calibre.Importer `inject:""`
}

type BookAddResponse struct {
	BookID int `json:"book_id"`
}

var BookAdd = request.Handler(func(r *BookAddRequest) (*BookAddResponse, error) {
	book, err := r.Hardcover.GetBook(r.Ctx, r.HardcoverID)
	if err != nil {
		return nil, err
	}

	id, err := r.Calibre.AddEmptyBook(r.Ctx, book)
	if err != nil {
		return nil, err
	}

	return &BookAddResponse{
		BookID: id,
	}, nil

})

func cleanBook(cfg *config.Config, book *calibredb.Book) {
	for i, f := range book.Formats {
		book.Formats[i] = cleanPath(cfg, f)
	}
	book.Cover = cleanPath(cfg, book.Cover)
}

func cleanPath(cfg *config.Config, p string) string {
	return "/api/d" + strings.TrimPrefix(p, cfg.CalibreLibrary)
}
