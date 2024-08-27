package migrations

import (
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func Test001Migration(t *testing.T) {
	cases := []struct {
		name   string
		config string
		want   *v1.Config
	}{
		{
			name: "time bucketed policy",
			config: `{
				"plans": [
					{
						"retention": {
							"keepHourly": 1,
							"keepDaily": 2,
							"keepWeekly": 3,
							"keepMonthly": 4,
							"keepYearly": 5
						}
					}
				]
			}`,
			want: &v1.Config{
				Plans: []*v1.Plan{
					{
						Retention: &v1.RetentionPolicy{
							Policy: &v1.RetentionPolicy_PolicyTimeBucketed{
								PolicyTimeBucketed: &v1.RetentionPolicy_TimeBucketedCounts{
									Hourly:  1,
									Daily:   2,
									Weekly:  3,
									Monthly: 4,
									Yearly:  5,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "keep all policy",
			config: `{
				"plans": [
					{
						"retention": {}
					}
				]
			}`,
			want: &v1.Config{
				Plans: []*v1.Plan{
					{
						Retention: &v1.RetentionPolicy{
							Policy: &v1.RetentionPolicy_PolicyKeepAll{
								PolicyKeepAll: true,
							},
						},
					},
				},
			},
		},
		{
			name: "keep by count",
			config: `{
				"plans": [
					{
						"retention": {
							"keepLastN": 5
						}
					}
				]
			}`,
			want: &v1.Config{
				Plans: []*v1.Plan{
					{
						Retention: &v1.RetentionPolicy{
							Policy: &v1.RetentionPolicy_PolicyKeepLastN{
								PolicyKeepLastN: 5,
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			config := v1.Config{}
			err := protojson.Unmarshal([]byte(tc.config), &config)
			if err != nil {
				t.Fatalf("failed to unmarshal config: %v", err)
			}

			migration001PrunePolicy(&config)

			if !proto.Equal(&config, tc.want) {
				t.Errorf("got: %+v, want: %+v", &config, tc.want)
			}
		})
	}
}
