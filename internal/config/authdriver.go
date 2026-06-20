package config

import v1 "github.com/garethgeorge/backrest/gen/go/v1"

// Auth driver identifiers stored in Auth.auth_driver.
const (
	AuthDriverDisabled = "disabled"
	AuthDriverLocal    = "local"
	AuthDriverOIDC     = "oidc"
)

// AuthDriverOf returns the effective auth driver for the given config.
//
// Precedence:
//   - nil auth                  -> disabled
//   - deprecated disabled==true -> disabled (takes precedence over auth_driver)
//   - empty auth_driver         -> local (preserves pre-existing enabled installs)
//   - otherwise                 -> the configured auth_driver value
func AuthDriverOf(auth *v1.Auth) string {
	if auth == nil {
		return AuthDriverDisabled
	}
	if auth.GetDisabled() {
		return AuthDriverDisabled
	}
	if auth.GetAuthDriver() == "" {
		return AuthDriverLocal
	}
	return auth.GetAuthDriver()
}

// AuthDisabled reports whether authentication is turned off for the given config.
func AuthDisabled(auth *v1.Auth) bool {
	return AuthDriverOf(auth) == AuthDriverDisabled
}
