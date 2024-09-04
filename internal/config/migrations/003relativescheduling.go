package migrations

import (
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

func migration003RelativeScheduling(config *v1.Config) {
	// loop over plans and examine prune policy's
	for _, repo := range config.Repos {
		prunePolicy := repo.GetPrunePolicy()
		if prunePolicy == nil {
			continue
		}

		if schedule := repo.GetPrunePolicy().GetSchedule(); schedule != nil {
			schedule.Clock = v1.Schedule_CLOCK_LAST_RUN_TIME
		}

		if schedule := repo.GetCheckPolicy().GetSchedule(); schedule != nil {
			schedule.Clock = v1.Schedule_CLOCK_LAST_RUN_TIME
		}
	}
}
