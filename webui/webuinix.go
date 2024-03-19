//go:build linux || darwin || freebsd
// +build linux darwin freebsd

//go:generate npm install
//go:generate npm run clean
//go:generate npm run build
//go:generate gzip -r dist

package webui

import (
	"embed"
)

//go:embed dist
var content embed.FS
var contentPrefix = "dist"
