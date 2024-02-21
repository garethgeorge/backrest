package migrations

import v1 "github.com/garethgeorge/backrest/gen/go/v1"

var CurrentVersion = int32(1)

func ApplyMigrations(config *v1.Config) error {
	if config.Version <= 1 {
		migration001PrunePolicy(config)
	}
	config.Version = CurrentVersion
	return nil
}
