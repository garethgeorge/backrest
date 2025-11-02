package syncapi

import (
	"testing"
	"time"

	"github.com/garethgeorge/backrest/gen/go/v1sync"
	"github.com/garethgeorge/backrest/internal/kvstore"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

func PeerStateManagersForTest(t testing.TB) map[string]PeerStateManager {
	dbpool := kvstore.NewInMemorySqliteDbForKvStore(t)
	t.Cleanup(func() {
		dbpool.Close()
	})
	sqlitepsm, err := NewSqlitePeerStateManager(dbpool)
	if err != nil {
		t.Fatalf("error creating sqlite peer state manager: %s", err)
	}
	return map[string]PeerStateManager{
		"memory": NewInMemoryPeerStateManager(),
		"sqlite": sqlitepsm,
	}
}

func TestPeerStateManager_GetSet(t *testing.T) {
	t.Parallel()
	for name, psm := range PeerStateManagersForTest(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			keyID := "testKey"
			state := &PeerState{
				InstanceID:             "testInstance",
				KeyID:                  keyID,
				LastHeartbeat:          time.Now().Round(time.Millisecond),
				ConnectionState:        v1sync.ConnectionState_CONNECTION_STATE_CONNECTED,
				ConnectionStateMessage: "hello world!",
				KnownRepos: map[string]*v1sync.RepoMetadata{
					"repo1": {
						Id:   "repo1",
						Guid: "guid1",
					},
					"repo2": {
						Id:   "repo2",
						Guid: "guid2",
					},
				},
				KnownPlans: map[string]*v1sync.PlanMetadata{
					"plan1": {
						Id: "plan1",
					},
				},
			}
			psm.SetPeerState(keyID, state)
			gotState := psm.GetPeerState(keyID)
			if diff := cmp.Diff(state, gotState, cmp.AllowUnexported(PeerState{}), protocmp.Transform()); diff != "" {
				t.Errorf("unexpected diff: %v", diff)
			}
		})
	}
}

func TestPeerStateManager_GetMissing(t *testing.T) {
	t.Parallel()
	for name, psm := range PeerStateManagersForTest(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			gotState := psm.GetPeerState("missingKey")
			if gotState != nil {
				t.Errorf("expected nil for missing key, got %v", gotState)
			}
		})
	}
}

func TestPeerStateManager_OnStateChanged(t *testing.T) {
	t.Skip("skipping syncapi tests")
	t.Parallel()
	for name, psm := range PeerStateManagersForTest(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			keyID := "testKey"
			state := &PeerState{
				LastHeartbeat: time.Now(),
			}

			ch := psm.OnStateChanged().Subscribe()
			t.Cleanup(func() {
				psm.OnStateChanged().Unsubscribe(ch)
			})

			psm.SetPeerState(keyID, state)

			select {
			case changedState := <-ch:
				if diff := cmp.Diff(state, changedState, cmp.AllowUnexported(PeerState{})); diff != "" {
					t.Errorf("unexpected diff: %v", diff)
				}
			case <-time.After(1 * time.Second):
				panic("timeout waiting for state change event")
			}
		})
	}
}
