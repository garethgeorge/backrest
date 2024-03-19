package migrations

import v1 "github.com/garethgeorge/backrest/gen/go/v1"

var migrations = []func(*v1.Config){
	migration001PrunePolicy,
}

var CurrentVersion = int32(len(migrations))

func ApplyMigrations(config *v1.Config) error {
	startMigration := int(config.Version - 1)
	if startMigration < 0 {
		startMigration = 0
	}
	for idx := startMigration; idx < len(migrations); idx += 1 {
		migrations[idx](config)
	}
	config.Version = CurrentVersion
	return nil
}
