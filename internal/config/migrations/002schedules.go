package migrations

import (
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

func migration002Schedules(config *v1.Config) {
	// loop over plans and examine prune policy's
	for _, repo := range config.Repos {
		prunePolicy := repo.GetPrunePolicy()
		if prunePolicy == nil {
			continue
		}

		if prunePolicy.MaxFrequencyDays != 0 {
			prunePolicy.Schedule = &v1.Schedule{
				Schedule: &v1.Schedule_MaxFrequencyDays{
					MaxFrequencyDays: prunePolicy.MaxFrequencyDays,
				},
			}
			prunePolicy.MaxFrequencyDays = 0
		}
	}

	// loop over plans and convert 'cron' and 'disabled' fields to schedule
	for _, plan := range config.Plans {
		if plan.Disabled {
			plan.Schedule = &v1.Schedule{
				Schedule: &v1.Schedule_Disabled{
					Disabled: true,
				},
			}
		} else if plan.Cron != "" {
			plan.Schedule = &v1.Schedule{
				Schedule: &v1.Schedule_Cron{
					Cron: plan.Cron,
				},
			}
			plan.Cron = ""
		}
	}
}
