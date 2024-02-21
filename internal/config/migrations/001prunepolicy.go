package migrations

import (
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

func migration001PrunePolicy(config *v1.Config) {
	// loop over plans and examine prune policy's
	for _, plan := range config.Plans {
		retention := plan.GetRetention()
		if retention == nil {
			continue
		}

		if retention.Policy != nil {
			continue // already migrated
		}

		if retention.KeepLastN != 0 {
			plan.Retention = &v1.RetentionPolicy{
				Policy: &v1.RetentionPolicy_PolicyKeepLastN{
					PolicyKeepLastN: retention.KeepLastN,
				},
			}
		} else if retention.KeepDaily != 0 || retention.KeepHourly != 0 || retention.KeepMonthly != 0 || retention.KeepWeekly != 0 || retention.KeepYearly != 0 {
			plan.Retention = &v1.RetentionPolicy{
				Policy: &v1.RetentionPolicy_PolicyTimeBucketed{
					PolicyTimeBucketed: &v1.RetentionPolicy_TimeBucketedCounts{
						Hourly:  retention.KeepHourly,
						Daily:   retention.KeepDaily,
						Weekly:  retention.KeepWeekly,
						Monthly: retention.KeepMonthly,
						Yearly:  retention.KeepYearly,
					},
				},
			}
		} else {
			plan.Retention = &v1.RetentionPolicy{
				Policy: &v1.RetentionPolicy_PolicyKeepAll{
					PolicyKeepAll: true,
				},
			}
		}
	}
}
