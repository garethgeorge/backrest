package migrations

import v1 "github.com/garethgeorge/backrest/gen/go/v1"

func ApplyMigrations(config *v1.Config) {
	if config.Version <= 1 {
		migration001PrunePolicy(config)
	}
	config.Version = 1
}
