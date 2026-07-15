//go:build dev

package resources

import (
	"io/fs"
	"os"
)

var Content fs.FS = os.DirFS("resources")
