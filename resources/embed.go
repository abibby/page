//go:build !dev

package resources

import (
	"embed"
)

//go:embed dist/*
var Content embed.FS
