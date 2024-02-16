//go:build linux || darwin
// +build linux darwin

//go:generate sh -c "rm -rf ./dist && UI_OS=unix npm run build && gzip ./dist/*"

package webui

import (
	"embed"
)

//go:embed dist
var content embed.FS
var contentPrefix = "dist"
