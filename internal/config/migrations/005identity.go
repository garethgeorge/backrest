package migrations

import (
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
	"go.uber.org/zap"
)

var migration005Identity = func(config *v1.Config) {
	if config.Multihost == nil {
		config.Multihost = &v1.Multihost{}
	}

	if config.Multihost.Identity == nil {
		var err error
		config.Multihost.Identity, err = cryptoutil.GeneratePrivateKey()
		if err != nil {
			zap.S().Fatalf("failed to generate cryptographic identity: %v", err)
		}
	}
}
