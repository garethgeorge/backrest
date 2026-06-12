//go:build tray && windows

package main

import _ "embed"

//go:embed icon.ico
var icon []byte

//go:embed icon_ok.ico
var iconOK []byte

//go:embed icon_syncing.ico
var iconSyncing []byte

//go:embed icon_warning.ico
var iconWarning []byte

//go:embed icon_error.ico
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
