package config

import (
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

func TestCleanupOrphanedRemoteReposAndPlans(t *testing.T) {
	tests := []struct {
		name          string
		config        *v1.Config
		wantRepoIDs   []string
		wantPlanIDs   []string
	}{
		{
			name: "no remote repos, nothing removed",
			config: &v1.Config{
				Repos: []*v1.Repo{
					{Id: "local-repo", Uri: "file:///tmp/repo", Guid: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
				},
				Plans: []*v1.Plan{
					{Id: "plan1", Repo: "local-repo", Paths: []string{"/data"}},
				},
			},
			wantRepoIDs: []string{"local-repo"},
			wantPlanIDs: []string{"plan1"},
		},
		{
			name: "remote repo with valid peer is kept",
			config: &v1.Config{
				Repos: []*v1.Repo{
					{Id: "local-repo", Uri: "file:///tmp/repo", Guid: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
					{Id: "remote-repo", Uri: "file:///tmp/remote", Guid: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", OriginInstanceId: "server-a"},
				},
				Plans: []*v1.Plan{
					{Id: "plan1", Repo: "local-repo", Paths: []string{"/data"}},
					{Id: "plan2", Repo: "remote-repo", Paths: []string{"/data"}},
				},
				Multihost: &v1.Multihost{
					KnownHosts: []*v1.Multihost_Peer{
						{InstanceId: "server-a", Keyid: "key-a", InstanceUrl: "http://server-a:9898"},
					},
				},
			},
			wantRepoIDs: []string{"local-repo", "remote-repo"},
			wantPlanIDs: []string{"plan1", "plan2"},
		},
		{
			name: "remote repo orphaned when peer removed",
			config: &v1.Config{
				Repos: []*v1.Repo{
					{Id: "local-repo", Uri: "file:///tmp/repo", Guid: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
					{Id: "remote-repo", Uri: "file:///tmp/remote", Guid: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", OriginInstanceId: "server-a"},
				},
				Plans: []*v1.Plan{
					{Id: "plan1", Repo: "local-repo", Paths: []string{"/data"}},
					{Id: "plan2", Repo: "remote-repo", Paths: []string{"/data"}},
				},
				Multihost: &v1.Multihost{},
			},
			wantRepoIDs: []string{"local-repo"},
			wantPlanIDs: []string{"plan1"},
		},
		{
			name: "authorized client peer keeps remote repo",
			config: &v1.Config{
				Repos: []*v1.Repo{
					{Id: "remote-repo", Uri: "file:///tmp/remote", Guid: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", OriginInstanceId: "client-b"},
				},
				Plans: []*v1.Plan{
					{Id: "plan1", Repo: "remote-repo", Paths: []string{"/data"}},
				},
				Multihost: &v1.Multihost{
					AuthorizedClients: []*v1.Multihost_Peer{
						{InstanceId: "client-b", Keyid: "key-b"},
					},
				},
			},
			wantRepoIDs: []string{"remote-repo"},
			wantPlanIDs: []string{"plan1"},
		},
		{
			name: "multiple orphaned repos and plans cleaned up",
			config: &v1.Config{
				Repos: []*v1.Repo{
					{Id: "local", Uri: "file:///tmp/repo", Guid: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
					{Id: "remote-a", Uri: "file:///tmp/a", Guid: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", OriginInstanceId: "gone-server"},
					{Id: "remote-b", Uri: "file:///tmp/b", Guid: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", OriginInstanceId: "gone-server"},
				},
				Plans: []*v1.Plan{
					{Id: "local-plan", Repo: "local", Paths: []string{"/data"}},
					{Id: "plan-a", Repo: "remote-a", Paths: []string{"/data"}},
					{Id: "plan-b", Repo: "remote-b", Paths: []string{"/data"}},
				},
				Multihost: &v1.Multihost{},
			},
			wantRepoIDs: []string{"local"},
			wantPlanIDs: []string{"local-plan"},
		},
		{
			name: "plan referencing local repo not removed even if remote repos cleaned",
			config: &v1.Config{
				Repos: []*v1.Repo{
					{Id: "local", Uri: "file:///tmp/repo", Guid: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
					{Id: "remote", Uri: "file:///tmp/remote", Guid: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", OriginInstanceId: "gone"},
				},
				Plans: []*v1.Plan{
					{Id: "kept-plan", Repo: "local", Paths: []string{"/data"}},
					{Id: "removed-plan", Repo: "remote", Paths: []string{"/data"}},
				},
				Multihost: &v1.Multihost{},
			},
			wantRepoIDs: []string{"local"},
			wantPlanIDs: []string{"kept-plan"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanupOrphanedRemoteReposAndPlans(tc.config)

			gotRepoIDs := make([]string, len(tc.config.Repos))
			for i, r := range tc.config.Repos {
				gotRepoIDs[i] = r.Id
			}

			gotPlanIDs := make([]string, len(tc.config.Plans))
			for i, p := range tc.config.Plans {
				gotPlanIDs[i] = p.Id
			}

			if !sliceEqual(gotRepoIDs, tc.wantRepoIDs) {
				t.Errorf("repos = %v, want %v", gotRepoIDs, tc.wantRepoIDs)
			}
			if !sliceEqual(gotPlanIDs, tc.wantPlanIDs) {
				t.Errorf("plans = %v, want %v", gotPlanIDs, tc.wantPlanIDs)
			}
		})
	}
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
