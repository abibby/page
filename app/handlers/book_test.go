package handlers

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/abibby/page/config"
	"github.com/abibby/page/migrations"
	"github.com/abibby/page/services/hardcover"
	"github.com/abibby/page/services/library"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUploadAndViewBookWithoutCover(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	cfg := &config.Config{LibraryPath: filepath.Join(root, "library")}
	dbCfg := config.NewSQLiteConfig(filepath.Join(root, "page.sqlite"))
	db, err := sqlx.Open(dbCfg.DriverName(), dbCfg.DataSourceName())
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck
	require.NoError(t, migrations.Use().Up(ctx, db))
	store, err := library.New(ctx, cfg, db)
	require.NoError(t, err)
	id, err := store.AddEmptyBook(ctx, &hardcover.Book{HardcoverID: 1, Title: "Title #1", Authors: []string{"Author"}})
	require.NoError(t, err)
	source := filepath.Join(root, "01.mp3")
	require.NoError(t, os.WriteFile(source, []byte("audio"), 0644))
	file, err := os.Open(source)
	require.NoError(t, err)
	response, err := BookImport.Run(&BookImportRequest{Ctx: ctx, DB: store, BookID: id, File: file})
	require.NoError(t, err)
	require.Equal(t, id, response.BookID)
	view, err := BookView.Run(&BookViewRequest{Ctx: ctx, DB: store, Cfg: cfg, ID: id})
	require.NoError(t, err)
	require.Equal(t, []string{"01.mp3"}, view.Files)
	require.Empty(t, view.Cover)
	require.Equal(t, "/api/d/Author/Title%20%231%20%281%29/01.mp3", view.Formats[0])
}
