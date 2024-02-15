//go:build linux || darwin
// +build linux darwin

//go:generate sh -c "rm -rf ./webui/dist && UI_OS=unix npm --prefix webui run build && gzip ./webui/dist/*"

package main

import rice "github.com/GeertJohan/go.rice"

func WebUIBox() (*rice.Box, error) {
	return rice.FindBox("webui/dist")
}
