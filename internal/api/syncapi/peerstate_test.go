package syncapi

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	_ "github.com/ncruces/go-sqlite3/driver"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
)

func PeerStateManagersForTest(t testing.TB) map[string]PeerStateManager {
	dbpool := newDbForTest(t)
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
				InstanceID:    "testInstance",
				KeyID:         keyID,
				LastHeartbeat: time.Now().Round(time.Millisecond),
				KnownRepos:    map[string]struct{}{"repo1": {}, "repo2": {}},
				KnownPlans:    map[string]struct{}{"plan1": {}, "plan2": {}},
			}
			psm.SetPeerState(keyID, state)
			gotState := psm.GetPeerState(keyID)
			if diff := cmp.Diff(state, gotState, cmp.AllowUnexported(PeerState{})); diff != "" {
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

func newDbForTest(t testing.TB) *sql.DB {
	t.Helper()
	dbpool, err := sql.Open("sqlite3", memdb.TestDB(t))
	if err != nil {
		t.Fatalf("error creating sqlite pool: %s", err)
	}
	return dbpool
}
