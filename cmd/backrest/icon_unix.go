//go:build tray && (linux || darwin)

package main

import _ "embed"

//go:embed icon.png
var icon []byte

//go:embed icon_ok.png
var iconOK []byte

//go:embed icon_syncing.png
var iconSyncing []byte

//go:embed icon_warning.png
var iconWarning []byte

//go:embed icon_error.png
var iconError []byte

// statusIcon returns the tray icon for the given backup state.
func statusIcon(s trayState) []byte {
	switch s {
	case stateRunning:
		return iconSyncing
	case stateOK:
		return iconOK
	case stateWarning:
		return iconWarning
	case stateError:
		return iconError
	default:
		return icon
	}
}
