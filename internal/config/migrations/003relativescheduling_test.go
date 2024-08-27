package migrations

import (
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"google.golang.org/protobuf/proto"
)

func Test003Migration(t *testing.T) {
	config := &v1.Config{
		Repos: []*v1.Repo{
			{
				Id: "prune daily",
				PrunePolicy: &v1.PrunePolicy{
					Schedule: &v1.Schedule{
						Schedule: &v1.Schedule_MaxFrequencyDays{
							MaxFrequencyDays: 1,
						},
					},
				},
				CheckPolicy: &v1.CheckPolicy{
					Schedule: &v1.Schedule{
						Schedule: &v1.Schedule_MaxFrequencyDays{
							MaxFrequencyDays: 1,
						},
					},
				},
			},
			{
				Id: "prune hourly",
				PrunePolicy: &v1.PrunePolicy{
					Schedule: &v1.Schedule{
						Schedule: &v1.Schedule_MaxFrequencyHours{
							MaxFrequencyHours: 1,
						},
					},
				},
				CheckPolicy: &v1.CheckPolicy{
					Schedule: &v1.Schedule{
						Schedule: &v1.Schedule_MaxFrequencyHours{
							MaxFrequencyHours: 1,
						},
					},
				},
			},
			{
				Id: "prune cron",
				PrunePolicy: &v1.PrunePolicy{
					Schedule: &v1.Schedule{
						Schedule: &v1.Schedule_Cron{
							Cron: "0 0 * * *",
						},
					},
				},
				CheckPolicy: &v1.CheckPolicy{
					Schedule: &v1.Schedule{
						Schedule: &v1.Schedule_Cron{
							Cron: "0 0 * * *",
						},
					},
				},
			},
		},
	}

	want := &v1.Config{
		Repos: []*v1.Repo{
			{
				Id: "prune daily",
				PrunePolicy: &v1.PrunePolicy{
					Schedule: &v1.Schedule{
						Schedule: &v1.Schedule_MinDaysSinceLastRun{
							MinDaysSinceLastRun: 1,
						},
					},
				},
				CheckPolicy: &v1.CheckPolicy{
					Schedule: &v1.Schedule{
						Schedule: &v1.Schedule_MinDaysSinceLastRun{
							MinDaysSinceLastRun: 1,
						},
					},
				},
			},
			{
				Id: "prune hourly",
				PrunePolicy: &v1.PrunePolicy{
					Schedule: &v1.Schedule{
						Schedule: &v1.Schedule_MinHoursSinceLastRun{
							MinHoursSinceLastRun: 1,
						},
					},
				},
				CheckPolicy: &v1.CheckPolicy{
					Schedule: &v1.Schedule{
						Schedule: &v1.Schedule_MinHoursSinceLastRun{
							MinHoursSinceLastRun: 1,
						},
					},
				},
			},
			{
				Id: "prune cron",
				PrunePolicy: &v1.PrunePolicy{
					Schedule: &v1.Schedule{
						Schedule: &v1.Schedule_CronSinceLastRun{
							CronSinceLastRun: "0 0 * * *",
						},
					},
				},
				CheckPolicy: &v1.CheckPolicy{
					Schedule: &v1.Schedule{
						Schedule: &v1.Schedule_CronSinceLastRun{
							CronSinceLastRun: "0 0 * * *",
						},
					},
				},
			},
		},
	}

	migration003RelativeScheduling(config)

	if !proto.Equal(config, want) {
		t.Errorf("got %v, want %v", config, want)
	}
}
