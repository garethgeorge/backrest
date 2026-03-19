//go:build tray && (linux || darwin)

package main

import _ "embed"

//go:embed icon.png
var icon []byte
