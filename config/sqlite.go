package config

import (
	"gosalusa.com/database/dialects/sqlite"
	_ "modernc.org/sqlite"
)

// SQLiteConfig uses Salusa's SQLite dialect with the pure-Go driver, including
// in the CGO-disabled Docker build.
type SQLiteConfig struct{ *sqlite.Config }

func NewSQLiteConfig(path string) *SQLiteConfig {
	return &SQLiteConfig{sqlite.NewConfig(path)}
}

func (*SQLiteConfig) DriverName() string { return "sqlite" }
