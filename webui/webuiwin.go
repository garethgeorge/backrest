//go:build windows
// +build windows

//go:generate sh -c "rm -rf ./dist-windows && UI_OS=unix npm run build-windows && gzip ./dist-windows/*"

package webui

import "embed"

//go:embed dist-windows/*
var content embed.FS
var contentPrefix = "dist-windows"
