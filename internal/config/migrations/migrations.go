package migrations

import (
	"fmt"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"go.uber.org/zap"
)

const migrationOffset = 2

var migrations = []func(*v1.Config){
	migration003RelativeScheduling, // version 3
}

var CurrentVersion = migrationOffset + int32(len(migrations))

func ApplyMigrations(config *v1.Config) error {
	curVersion := int(config.Version - 1)
	if curVersion < 0 {
		curVersion = 0
	} else {
		if curVersion < migrationOffset {
			return fmt.Errorf("config version %d is too old to migrate. Please update to 1.4.0 first which is the newest version that _may_ be able to migrate from your version", config.Version)
		}
		curVersion -= migrationOffset
	}

	for idx := curVersion; idx < len(migrations); idx += 1 {
		zap.S().Infof("applying config migration %d", idx+1)
		migrations[idx](config)
	}
	config.Version = CurrentVersion
	return nil
}
