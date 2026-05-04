package migrations

import (
	"strings"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"go.uber.org/zap"
)

// migration006Ed25519Identity drops any existing multihost identity that does
// not use the ed25519 key scheme. The previous ECDSA-based identity used
// PEM-encoded keys with an "ecdsa." keyid prefix; those keys are no longer
// readable by the cryptoutil package. Clearing the identity here lets
// PopulateRequiredFields generate a fresh ed25519 identity on next load.
var migration006Ed25519Identity = func(config *v1.Config) error {
	multihost := config.GetMultihost()
	if multihost == nil {
		return nil
	}
	identity := multihost.GetIdentity()
	if identity == nil {
		return nil
	}
	if strings.HasPrefix(identity.GetKeyid(), "ed25519.") && identity.GetEd25519Priv() != "" && identity.GetEd25519Pub() != "" {
		return nil
	}
	zap.S().Warnf("dropping legacy multihost identity %q; a new ed25519 identity will be generated", identity.GetKeyid())
	multihost.Identity = nil
	return nil
}
