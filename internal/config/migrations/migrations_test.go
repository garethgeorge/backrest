package migrations

import (
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

func TestApplyMigrations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  *v1.Config
		wantErr bool
	}{
		{
			name: "too old to migrate",
			config: &v1.Config{
				Version: 1,
			},
			wantErr: true, // too old to migrate
		},
		{
			name:    "empty config",
			config:  &v1.Config{},
			wantErr: false,
		},
		{
			name: "latest version",
			config: &v1.Config{
				Version: CurrentVersion,
			},
		},
		{
			name: "apply relative scheduling migration",
			config: &v1.Config{
				Version: 2, // higest version that still needs the migration
				Repos: []*v1.Repo{
					{
						Id: "repo-relative",
						CheckPolicy: &v1.CheckPolicy{
							Schedule: &v1.Schedule{
								Schedule: &v1.Schedule_MaxFrequencyDays{MaxFrequencyDays: 1},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := ApplyMigrations(tc.config)
			if (err != nil) != tc.wantErr {
				t.Errorf("ApplyMigrations() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
