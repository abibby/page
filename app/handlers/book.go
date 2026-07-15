package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/abibby/page/services/calibredb"
	"github.com/abibby/page/services/importer"
	"github.com/abibby/salusa/request"
)

type FullBook struct {
	calibredb.Book

	Description string `json:"description"`
}

type BookListRequest struct {
	Search    string          `query:"search"`
	Limit     int             `query:"limit"`
	SortBy    calibredb.Field `query:"order"`
	Ascending bool            `query:"ascending"`

	Ctx context.Context   `inject:""`
	DB  *calibredb.Client `inject:""`
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
		cleanBook(&books[i])
	}
	return books, nil
})

type BookViewRequest struct {
	ID int `path:"id"`

	Ctx context.Context   `inject:""`
	DB  *calibredb.Client `inject:""`
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

	cleanBook(&b)

	return &FullBook{
		Book:        b,
		Description: meta.Description,
	}, nil
})

type BookImportRequest struct {
	Request *http.Request      `inject:""`
	Ctx     context.Context    `inject:""`
	App     *importer.Importer `inject:""`
}

var BookImport = request.Handler(func(r *BookImportRequest) (any, error) {
	rf, hed, err := r.Request.FormFile("file")
	if err != nil {
		return nil, err
	}
	defer rf.Close()
	// defer r.File.Close()

	f, err := os.CreateTemp("", "book-*-"+hed.Filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	defer os.Remove(f.Name())

	_, err = io.Copy(f, rf)
	if err != nil {
		return nil, err
	}

	err = r.App.ImportFile(r.Ctx, f.Name())
	if err != nil {
		return nil, err
	}

	return nil, nil
})

func cleanBook(book *calibredb.Book) {
	for i, f := range book.Formats {
		book.Formats[i] = cleanPath(f)
	}
	book.Cover = cleanPath(book.Cover)
}

func cleanPath(p string) string {
	return "/d" + strings.TrimPrefix(p, "/mnt/volume1/Public/Calibre_Library")
}
