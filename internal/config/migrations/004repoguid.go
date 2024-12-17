package migrations

import (
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
)

func migration004RepoGuid(config *v1.Config) {
	for _, repo := range config.Repos {
		if repo.Guid != "" {
			continue
		}

		repo.Guid = cryptoutil.MustRandomID(cryptoutil.DefaultIDBits)
	}
}
