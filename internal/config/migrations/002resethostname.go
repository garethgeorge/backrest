package migrations

import (
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"go.uber.org/zap"
)

// migration002ResetHostname resets the hostname to an empty string.
// This is done because the hostname is converted to an immutable property.
// Setting to empty string will allow the user to set it again if needed.
func migration002ResetHostname(config *v1.Config) {
	zap.S().Warn("Hostname is now an immutable property. Resetting to empty string to allow configuring after this update, please open the UI and set a permanent hostname for this instance.")
	config.Host = ""
}
