//go:build !minimal
// +build !minimal

package webui

import (
	"embed"
	"io/fs"
)

//go:embed dist/*
var distFS embed.FS

// GetDistFS returns the embedded filesystem containing the built web UI
func GetDistFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
