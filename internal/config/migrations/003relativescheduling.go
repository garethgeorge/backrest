package migrations

import (
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

func convertToRelativeSchedule(sched *v1.Schedule) {
	switch s := sched.GetSchedule().(type) {
	case *v1.Schedule_MaxFrequencyDays:
		sched.Schedule = &v1.Schedule_MinDaysSinceLastRun{
			MinDaysSinceLastRun: s.MaxFrequencyDays,
		}
	case *v1.Schedule_MaxFrequencyHours:
		sched.Schedule = &v1.Schedule_MinHoursSinceLastRun{
			MinHoursSinceLastRun: s.MaxFrequencyHours,
		}
	case *v1.Schedule_Cron:
		sched.Schedule = &v1.Schedule_CronSinceLastRun{
			CronSinceLastRun: s.Cron,
		}
	default:
		// do nothing
	}
}

func migration003RelativeScheduling(config *v1.Config) {
	// loop over plans and examine prune policy's
	for _, repo := range config.Repos {
		prunePolicy := repo.GetPrunePolicy()
		if prunePolicy == nil {
			continue
		}

		if schedule := repo.GetPrunePolicy().GetSchedule(); schedule != nil {
			convertToRelativeSchedule(schedule)
		}

		if schedule := repo.GetCheckPolicy().GetSchedule(); schedule != nil {
			convertToRelativeSchedule(schedule)
		}
	}
}
