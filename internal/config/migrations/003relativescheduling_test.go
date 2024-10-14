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
				Id: "prune",
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
		},
	}

	want := proto.Clone(config).(*v1.Config)
	want.Repos[0].PrunePolicy.Schedule.Clock = v1.Schedule_CLOCK_LAST_RUN_TIME
	want.Repos[0].CheckPolicy.Schedule.Clock = v1.Schedule_CLOCK_LAST_RUN_TIME

	migration003RelativeScheduling(config)

	if !proto.Equal(config, want) {
		t.Errorf("got %v, want %v", config, want)
	}
}
