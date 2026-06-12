// Package ui embeds the built single-page frontend so the `easy-infra serve`
// command can ship the UI inside the binary. The bundle lives in dist/, which
// is produced by the Vite build (`make ui`); until it is built the directory
// holds only a placeholder, so callers must tolerate a missing index.html.
package ui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Dist returns the built SPA as a filesystem rooted at the dist directory.
// It is effectively empty until the frontend is built with `make ui`.
func Dist() (fs.FS, error) {
	return fs.Sub(dist, "dist")
}
