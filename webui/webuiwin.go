//go:build windows
// +build windows

//go:generate npm install
//go:generate npm run clean-windows
//go:generate npm run build-windows

package webui

import "embed"

//go:embed dist-windows/*
var content embed.FS
var contentPrefix = "dist-windows"
