package migrations

import (
	"fmt"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"google.golang.org/protobuf/proto"
)

var migrations = []*func(*v1.Config){
	&noop, // migration001PrunePolicy
	&noop, // migration002Schedules is deprecated
	&migration003RelativeScheduling,
}

var CurrentVersion = int32(len(migrations))

func ApplyMigrations(config *v1.Config) error {
	if config.Version == 0 {
		if proto.Equal(config, &v1.Config{}) {
			config.Version = CurrentVersion
			return nil
		}
		return fmt.Errorf("config version 0 is invalid")
	}

	startMigration := int(config.Version)
	if startMigration < 0 {
		startMigration = 0
	}

	for idx := startMigration; idx < len(migrations); idx += 1 {
		m := migrations[idx]
		if m == &noop {
			return fmt.Errorf("config version %d is too old to migrate, please try first upgrading to backrest 1.4.0 which is the last version that may be compatible with your config", config.Version)
		}
		(*m)(config)
	}
	config.Version = CurrentVersion
	return nil
}

var noop = func(config *v1.Config) {
	// do nothing
}
