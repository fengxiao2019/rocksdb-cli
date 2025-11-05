//go:build minimal
// +build minimal

package webui

import (
	"errors"
	"io/fs"
)

// GetDistFS returns an error indicating Web UI is not available in minimal build
func GetDistFS() (fs.FS, error) {
	return nil, errors.New("Web UI is not available in minimal build. Please use the full version or rebuild without -tags=minimal")
}
