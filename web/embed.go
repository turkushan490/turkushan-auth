// Package web embeds the built frontend (run `npm run build` in this folder first).
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Dist returns the contents of web/dist.
func Dist() (fs.FS, error) {
	return fs.Sub(dist, "dist")
}
