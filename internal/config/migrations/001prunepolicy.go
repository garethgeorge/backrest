package migrations

import v1 "github.com/garethgeorge/backrest/gen/go/v1"

func migration001PrunePolicy(config *v1.Config) {
	// loop over plans and examine prune policy's
	for _, plan := range config.Plans {
		policy := plan.GetRetention()
		if policy == nil {
			continue
		}

		if policy.Policy != nil {
			continue // already migrated
		}

		if policy.KeepLastN != 0 {
			plan.Retention = &v1.RetentionPolicy{
				Policy: &v1.RetentionPolicy_PolicyKeepLastN{
					PolicyKeepLastN: policy.KeepLastN,
				},
			}
		} else if policy.KeepDaily != 0 || policy.KeepHourly != 0 || policy.KeepMonthly != 0 || policy.KeepWeekly != 0 || policy.KeepYearly != 0 {
			plan.Retention = &v1.RetentionPolicy{
				Policy: &v1.RetentionPolicy_PolicyTimeBucketed{
					PolicyTimeBucketed: &v1.RetentionPolicy_TimeBucketedCounts{
						Hourly:  policy.KeepHourly,
						Daily:   policy.KeepDaily,
						Weekly:  policy.KeepWeekly,
						Monthly: policy.KeepMonthly,
						Yearly:  policy.KeepYearly,
					},
				},
			}
		} else {
			policy.Policy = &v1.RetentionPolicy_PolicyKeepAll{
				PolicyKeepAll: true,
			}
		}
	}
}
