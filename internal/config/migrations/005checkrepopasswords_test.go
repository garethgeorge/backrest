package migrations

import (
	"strings"
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

func TestMigration005CheckRepoPasswords(t *testing.T) {
	// Cannot use t.Parallel() because subtests use t.Setenv.

	tests := []struct {
		name        string
		config      func() *v1.Config   // factory so each run gets a fresh config
		envVars     map[string]string    // env vars to set for the test
		ackEnvVar   bool                 // whether to set ISSUE_1139_FIX_PASSWORDS=1
		wantErr     bool
		errContains string              // substring that must appear in the error
		checkConfig func(*testing.T, *v1.Config) // optional post-migration assertions
	}{
		{
			name: "no repos with passwords - no conflict",
			config: func() *v1.Config {
				return &v1.Config{Repos: []*v1.Repo{{Id: "r", Uri: "/tmp/r"}}}
			},
			envVars: map[string]string{"RESTIC_PASSWORD": "somevalue"},
			wantErr: false,
		},
		{
			name: "repos with passwords but no conflicting env vars - no conflict",
			config: func() *v1.Config {
				return &v1.Config{Repos: []*v1.Repo{{Id: "r", Uri: "/tmp/r", Password: "secret"}}}
			},
			envVars: map[string]string{},
			wantErr: false,
		},
		{
			name: "RESTIC_PASSWORD conflict blocks startup without ack",
			config: func() *v1.Config {
				return &v1.Config{Repos: []*v1.Repo{{Id: "my-repo", Uri: "/tmp/r", Password: "wrong"}}}
			},
			envVars:     map[string]string{"RESTIC_PASSWORD": "envpassword"},
			wantErr:     true,
			errContains: "RESTIC_PASSWORD",
		},
		{
			name: "RESTIC_PASSWORD_FILE conflict blocks startup without ack",
			config: func() *v1.Config {
				return &v1.Config{Repos: []*v1.Repo{{Id: "r", Uri: "/tmp/r", Password: "wrong"}}}
			},
			envVars:     map[string]string{"RESTIC_PASSWORD_FILE": "/run/secrets/restic"},
			wantErr:     true,
			errContains: "RESTIC_PASSWORD_FILE",
		},
		{
			name: "RESTIC_PASSWORD_COMMAND conflict blocks startup without ack",
			config: func() *v1.Config {
				return &v1.Config{Repos: []*v1.Repo{{Id: "r", Uri: "/tmp/r", Password: "wrong"}}}
			},
			envVars:     map[string]string{"RESTIC_PASSWORD_COMMAND": "cat /secrets/pw"},
			wantErr:     true,
			errContains: "RESTIC_PASSWORD_COMMAND",
		},
		{
			name: "error message names the affected repo",
			config: func() *v1.Config {
				return &v1.Config{Repos: []*v1.Repo{{Id: "my-repo", Uri: "/tmp/r", Password: "wrong"}}}
			},
			envVars:     map[string]string{"RESTIC_PASSWORD": "envpassword"},
			wantErr:     true,
			errContains: "my-repo",
		},
		{
			name: "error message references ISSUE_1139_FIX_PASSWORDS",
			config: func() *v1.Config {
				return &v1.Config{Repos: []*v1.Repo{{Id: "r", Uri: "/tmp/r", Password: "wrong"}}}
			},
			envVars:     map[string]string{"RESTIC_PASSWORD": "envpassword"},
			wantErr:     true,
			errContains: "ISSUE_1139_FIX_PASSWORDS",
		},
		{
			name: "RESTIC_PASSWORD: ack inlines value into repo.Password",
			config: func() *v1.Config {
				return &v1.Config{Repos: []*v1.Repo{{Id: "r", Uri: "/tmp/r", Password: "wrong"}}}
			},
			envVars:   map[string]string{"RESTIC_PASSWORD": "real-password"},
			ackEnvVar: true,
			wantErr:   false,
			checkConfig: func(t *testing.T, cfg *v1.Config) {
				t.Helper()
				if got := cfg.Repos[0].Password; got != "real-password" {
					t.Errorf("expected repo.Password = %q, got %q", "real-password", got)
				}
			},
		},
		{
			name: "RESTIC_PASSWORD_FILE: ack moves to repo.Env and clears Password",
			config: func() *v1.Config {
				return &v1.Config{Repos: []*v1.Repo{{Id: "r", Uri: "/tmp/r", Password: "wrong"}}}
			},
			envVars:   map[string]string{"RESTIC_PASSWORD_FILE": "/run/secrets/pw"},
			ackEnvVar: true,
			wantErr:   false,
			checkConfig: func(t *testing.T, cfg *v1.Config) {
				t.Helper()
				repo := cfg.Repos[0]
				if repo.Password != "" {
					t.Errorf("expected repo.Password to be cleared, got %q", repo.Password)
				}
				wantEnv := "RESTIC_PASSWORD_FILE=/run/secrets/pw"
				for _, e := range repo.Env {
					if e == wantEnv {
						return
					}
				}
				t.Errorf("expected repo.Env to contain %q, got %v", wantEnv, repo.Env)
			},
		},
		{
			name: "RESTIC_PASSWORD_COMMAND: ack moves to repo.Env and clears Password",
			config: func() *v1.Config {
				return &v1.Config{Repos: []*v1.Repo{{Id: "r", Uri: "/tmp/r", Password: "wrong"}}}
			},
			envVars:   map[string]string{"RESTIC_PASSWORD_COMMAND": "cat /secrets/pw"},
			ackEnvVar: true,
			wantErr:   false,
			checkConfig: func(t *testing.T, cfg *v1.Config) {
				t.Helper()
				repo := cfg.Repos[0]
				if repo.Password != "" {
					t.Errorf("expected repo.Password to be cleared, got %q", repo.Password)
				}
				wantEnv := "RESTIC_PASSWORD_COMMAND=cat /secrets/pw"
				for _, e := range repo.Env {
					if e == wantEnv {
						return
					}
				}
				t.Errorf("expected repo.Env to contain %q, got %v", wantEnv, repo.Env)
			},
		},
		{
			name: "only repos with passwords are considered affected",
			config: func() *v1.Config {
				return &v1.Config{Repos: []*v1.Repo{
					{Id: "no-pass", Uri: "/tmp/r1"},
					{Id: "with-pass", Uri: "/tmp/r2", Password: "wrong"},
				}}
			},
			envVars:     map[string]string{"RESTIC_PASSWORD": "envpassword"},
			wantErr:     true,
			errContains: "with-pass",
		},
		{
			name: "no repos at all - no conflict",
			config: func() *v1.Config {
				return &v1.Config{}
			},
			envVars: map[string]string{"RESTIC_PASSWORD": "envpassword"},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}
			// Ensure conflicting env vars not in tc.envVars are unset.
			for _, k := range conflictingEnvVars {
				if _, ok := tc.envVars[k]; !ok {
					t.Setenv(k, "")
				}
			}
			if tc.ackEnvVar {
				t.Setenv("ISSUE_1139_FIX_PASSWORDS", "1")
			} else {
				t.Setenv("ISSUE_1139_FIX_PASSWORDS", "")
			}

			cfg := tc.config()
			err := migration005CheckRepoPasswords(cfg)
			if (err != nil) != tc.wantErr {
				t.Errorf("migration005CheckRepoPasswords() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.errContains != "" && err != nil {
				if !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("expected error to contain %q, got: %s", tc.errContains, err.Error())
				}
			}
			if tc.checkConfig != nil {
				tc.checkConfig(t, cfg)
			}
		})
	}
}
