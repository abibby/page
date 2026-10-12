package test

import (
	"context"
	"log"
	"path/filepath"
	"testing"

	"github.com/abibby/page/app"
	"github.com/abibby/page/config"
	"github.com/abibby/page/migrations"
	"github.com/jmoiron/sqlx"
	"gosalusa.com/database/dbtest"
	"gosalusa.com/database/dialects/sqlite"
	"gosalusa.com/testing/kerneltest"
	_ "modernc.org/sqlite"
)

var runner = dbtest.NewRunner(func() (*sqlx.DB, error) {
	sqlite.UseSQLite()

	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}

	err = migrations.Use().Up(context.Background(), db)
	if err != nil {
		return nil, err
	}

	log.Print("db loaded")

	return db, nil
})

var Run = runner.Run
var RunBenchmark = runner.RunBenchmark

// Kernel isolates both the catalogue and library files for each integration test.
func Kernel(t *testing.T) *kerneltest.TestKernel[*config.Config] {
	t.Helper()
	root := t.TempDir()
	return kerneltest.NewTestKernelFactory(app.Kernel, &config.Config{
		Port:        443,
		BasePath:    "https://example.test",
		Database:    config.NewSQLiteConfig(filepath.Join(root, "page.sqlite")),
		LibraryPath: filepath.Join(root, "library"),
	})(t)
}
