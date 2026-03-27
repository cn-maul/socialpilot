package main

import (
	"embed"
	"io/fs"
)

//go:embed webui/dist
var webUIFS embed.FS

// GetWebUIFS returns the embedded web UI filesystem
func GetWebUIFS() (fs.FS, error) {
	return fs.Sub(webUIFS, "webui/dist")
}
