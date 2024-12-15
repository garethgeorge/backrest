package migrations

import (
	"crypto/rand"
	"encoding/hex"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

func migration004RepoGuid(config *v1.Config) {
	for _, repo := range config.Repos {
		if repo.Guid != "" {
			continue
		}

		// generate a 256 bit guid that links the repo.
		var bytes [32]byte
		if _, err := rand.Read(bytes[:]); err != nil {
			panic(err)
		}
		repo.Guid = "migration-" + hex.EncodeToString(bytes[:])
	}
}
