//go:build linux || darwin || freebsd
// +build linux darwin freebsd

//go:generate npm run clean
//go:generate npm run build

package webui

import (
	"embed"
)

//go:embed dist
var content embed.FS
var contentPrefix = "dist"
