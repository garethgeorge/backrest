package syncapi

import (
	"database/sql"
	"testing"
	"time"

<<<<<<< HEAD
	"github.com/google/go-cmp/cmp"
	_ "github.com/ncruces/go-sqlite3/driver"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
=======
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
>>>>>>> 9041d3c (improve sync api security by using 'Authorization' headers for initial key exchange)
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
				KnownRepos: map[string]*v1.SyncRepoMetadata{
					"repo1": {Id: "repo1"},
					"repo2": {Id: "repo2"},
				},
				KnownPlans: map[string]*v1.SyncPlanMetadata{
					"plan1": {Id: "plan1"},
					"plan2": {Id: "plan2"},
				},
			}
			psm.SetPeerState(keyID, state)
			gotState := psm.GetPeerState(keyID)
			if diff := cmp.Diff(state, gotState, protocmp.Transform()); diff != "" {
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
				if diff := cmp.Diff(state, changedState, protocmp.Transform()); diff != "" {
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
