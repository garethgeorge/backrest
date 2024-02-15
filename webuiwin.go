//go:build windows
// +build windows

//go:generate sh -c "rm -rf ./webui/dist-windows && npm --prefix webui run build-windows && gzip ./webui/dist-windows/*"

package main

import rice "github.com/GeertJohan/go.rice"

func WebUIBox() (*rice.Box, error) {
	return rice.FindBox("webui/dist-windows")
}
