package migrations

import (
	"crypto/sha256"
	"encoding/hex"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

var migration004RepoGuid = func(config *v1.Config) {
	for _, repo := range config.Repos {
		if repo.Guid != "" {
			continue
		}
		h := sha256.New()
		h.Write([]byte(repo.Id))
		repo.Guid = hex.EncodeToString(h.Sum(nil))
	}
}
