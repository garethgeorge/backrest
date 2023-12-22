package config

import (
	"strings"
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"google.golang.org/protobuf/proto"
)

func TestConfig(t *testing.T) {
	dir := t.TempDir()

	testRepo := &v1.Repo{
		Id:       "test-repo",
		Uri:      "/tmp/test",
		Password: "test",
	}

	testPlan := &v1.Plan{
		Id:    "test-plan",
		Repo:  "test-repo",
		Paths: []string{"/tmp/foo"},
		Cron:  "* * * * *",
	}

	tests := []struct {
		name            string
		config          *v1.Config
		wantErr         bool
		wantErrContains string
		store           ConfigStore
	}{
		{
			name:   "default config",
			config: NewDefaultConfig(),
			store:  &CachingValidatingStore{ConfigStore: &JsonFileStore{Path: dir + "/default-config.json"}},
		},
		{
			name: "simple valid config",
			config: &v1.Config{
				Repos: []*v1.Repo{testRepo},
				Plans: []*v1.Plan{testPlan},
			},
			store: &CachingValidatingStore{ConfigStore: &JsonFileStore{Path: dir + "/valid-config.json"}},
		},
		{
			name: "plan references non-existent repo",
			config: &v1.Config{
				Plans: []*v1.Plan{testPlan},
			},
			store:           &CachingValidatingStore{ConfigStore: &JsonFileStore{Path: dir + "/invalid-config.json"}},
			wantErr:         true,
			wantErrContains: "repo \"test-repo\" not found",
		},
		{
			name: "repo with duplicate id",
			config: &v1.Config{
				Repos: []*v1.Repo{
					testRepo,
					testRepo,
				},
			},
			store:           &CachingValidatingStore{ConfigStore: &JsonFileStore{Path: dir + "/invalid-config2.json"}},
			wantErr:         true,
			wantErrContains: "repo test-repo: duplicate id",
		},
		{
			name: "plan with bad cron",
			config: &v1.Config{
				Repos: []*v1.Repo{
					testRepo,
				},
				Plans: []*v1.Plan{
					{
						Id:    "test-plan",
						Repo:  "test-repo",
						Paths: []string{"/tmp/foo"},
						Cron:  "bad cron",
					},
				},
			},
			store:           &CachingValidatingStore{ConfigStore: &JsonFileStore{Path: dir + "/invalid-config3.json"}},
			wantErr:         true,
			wantErrContains: "invalid cron \"bad cron\"",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.store.Update(tc.config)
			if (err != nil) != tc.wantErr {
				t.Errorf("Config.Update() error = %v, wantErr %v", err, tc.wantErr)
			}

			if tc.wantErrContains != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErrContains)) {
				t.Errorf("Config.Update() error = %v, wantErrContains %v", err, tc.wantErrContains)
			}

			if err == nil {
				config, err := tc.store.Get()
				if err != nil {
					t.Errorf("Config.Get() error = %v, wantErr nil", err)
				}

				if !proto.Equal(config, tc.config) {
					t.Errorf("Config.Get() = %v, want %v", config, tc.config)
				}
			}
		})
	}
}
