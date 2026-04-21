package syncapi

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/config/migrations"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/testutil"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/google/go-cmp/cmp"
)

// TestFuzzOperationSync exercises the sync protocol with randomized operation
// mutations (add, update, delete) interleaved with connection drops and
// reconnections. After every round the test asserts that the host's view of
// the client's operations exactly matches the client's local state.
func TestFuzzOperationSync(t *testing.T) {
	testutil.InstallZapLogger(t)

	const (
		numRounds      = 10  // rounds of mutations
		opsPerRound    = 20  // mutations per round
		reconnectEvery = 3   // force reconnect every N rounds
		testTimeout    = 60 * time.Second
	)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	peerHostAddr := testutil.AllocOpenBindAddr(t)
	peerClientAddr := testutil.AllocOpenBindAddr(t)

	repoGUID := cryptoutil.MustRandomID(cryptoutil.DefaultIDBits)

	peerHostConfig := &v1.Config{
		Version:  migrations.CurrentVersion,
		Instance: defaultHostID,
		Repos: []*v1.Repo{
			{Id: defaultRepoID, Guid: repoGUID, Uri: "test-uri"},
		},
		Multihost: &v1.Multihost{
			Identity: identity1,
			AuthorizedClients: []*v1.Multihost_Peer{
				{Keyid: identity2.Keyid, InstanceId: defaultClientID},
			},
		},
	}

	peerClientConfig := &v1.Config{
		Version:  migrations.CurrentVersion,
		Instance: defaultClientID,
		Repos: []*v1.Repo{
			{Id: defaultRepoID, Guid: repoGUID, Uri: "backrest://" + defaultHostID},
		},
		Multihost: &v1.Multihost{
			Identity: identity2,
			KnownHosts: []*v1.Multihost_Peer{
				{
					Keyid:       identity1.Keyid,
					InstanceId:  defaultHostID,
					InstanceUrl: fmt.Sprintf("http://%s", peerHostAddr),
					Permissions: []*v1.Multihost_Permission{
						{
							Type:   v1.Multihost_Permission_PERMISSION_READ_OPERATIONS,
							Scopes: []string{"repo:" + defaultRepoID},
						},
					},
				},
			},
		},
	}

	peerHost := newPeerUnderTest(t, peerHostConfig)
	peerClient := newPeerUnderTest(t, peerClientConfig)

	opTempl := &v1.Operation{
		InstanceId:      defaultClientID,
		RepoId:          defaultRepoID,
		RepoGuid:        repoGUID,
		PlanId:          defaultPlanID,
		UnixTimeStartMs: time.Now().UnixMilli() - 1000,
		UnixTimeEndMs:   time.Now().UnixMilli(),
		Status:          v1.OperationStatus_STATUS_SUCCESS,
		Op:              &v1.Operation_OperationBackup{},
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	query := oplog.Query{}.SetInstanceID(defaultClientID).SetRepoGUID(repoGUID)

	// Track live operations on the client by their ID.
	liveOps := map[int64]*v1.Operation{}

	// Start sync infrastructure.
	syncCtx, cancelSync := context.WithCancel(ctx)
	var syncWg sync.WaitGroup
	startSync := func() {
		syncCtx, cancelSync = context.WithCancel(ctx)
		syncWg.Add(2)
		go func() { defer syncWg.Done(); runSyncAPIWithCtx(syncCtx, peerHost, peerHostAddr) }()
		go func() { defer syncWg.Done(); runSyncAPIWithCtx(syncCtx, peerClient, peerClientAddr) }()
		tryConnect(t, ctx, peerClient, peerClientConfig.Multihost.KnownHosts[0])
	}
	stopSync := func() {
		cancelSync()
		syncWg.Wait()
	}

	startSync()

	for round := 0; round < numRounds; round++ {
		t.Logf("=== Round %d: %d live ops ===", round, len(liveOps))

		// Reconnect periodically to exercise the RequestOperations catch-up path.
		if round > 0 && round%reconnectEvery == 0 {
			t.Logf("--- reconnecting ---")
			stopSync()
			startSync()
		}

		for i := 0; i < opsPerRound; i++ {
			action := rng.Intn(10)
			switch {
			case action < 5: // 50%: add a new operation
				op := proto.Clone(opTempl).(*v1.Operation)
				op.DisplayMessage = fmt.Sprintf("r%d-op%d", round, i)
				op.UnixTimeStartMs = time.Now().UnixMilli() - int64(rng.Intn(10000))
				op.UnixTimeEndMs = op.UnixTimeStartMs + int64(rng.Intn(5000))
				statuses := []v1.OperationStatus{
					v1.OperationStatus_STATUS_PENDING,
					v1.OperationStatus_STATUS_INPROGRESS,
					v1.OperationStatus_STATUS_SUCCESS,
					v1.OperationStatus_STATUS_ERROR,
				}
				op.Status = statuses[rng.Intn(len(statuses))]
				if err := peerClient.oplog.Add(op); err != nil {
					t.Fatalf("round %d: add: %v", round, err)
				}
				liveOps[op.Id] = op

			case action < 8 && len(liveOps) > 0: // 30%: update a random op
				op := pickRandom(rng, liveOps)
				op = proto.Clone(op).(*v1.Operation)
				op.DisplayMessage = fmt.Sprintf("r%d-op%d-updated", round, i)
				op.Status = v1.OperationStatus_STATUS_SUCCESS
				if err := peerClient.oplog.Update(op); err != nil {
					t.Fatalf("round %d: update: %v", round, err)
				}
				liveOps[op.Id] = op

			case len(liveOps) > 0: // 20%: delete a random op
				op := pickRandom(rng, liveOps)
				if err := peerClient.oplog.Delete(op.Id); err != nil {
					t.Fatalf("round %d: delete: %v", round, err)
				}
				delete(liveOps, op.Id)
			}
		}

		// Wait for the host to converge with the client.
		assertOpsConverge(t, ctx, peerClient, peerHost, query,
			fmt.Sprintf("round %d: ops should converge", round))
	}

	// Final reconnect to exercise one more catch-up after all mutations.
	t.Logf("=== Final reconnect ===")
	stopSync()
	startSync()
	assertOpsConverge(t, ctx, peerClient, peerHost, query, "final: ops should converge after reconnect")

	// Assert no duplicates on the host.
	assertNoDuplicateOriginalIDs(t, peerHost, query)

	stopSync()
}

// TestFuzzOperationSyncOfflineMutations creates operations, syncs, disconnects,
// mutates heavily offline, then reconnects and verifies convergence.
func TestFuzzOperationSyncOfflineMutations(t *testing.T) {
	testutil.InstallZapLogger(t)

	const (
		initialOps  = 20
		offlineOps  = 40
		testTimeout = 15 * time.Second
	)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	peerHostAddr := testutil.AllocOpenBindAddr(t)
	peerClientAddr := testutil.AllocOpenBindAddr(t)

	repoGUID := cryptoutil.MustRandomID(cryptoutil.DefaultIDBits)

	peerHostConfig := &v1.Config{
		Version:  migrations.CurrentVersion,
		Instance: defaultHostID,
		Repos:    []*v1.Repo{{Id: defaultRepoID, Guid: repoGUID, Uri: "test-uri"}},
		Multihost: &v1.Multihost{
			Identity:          identity1,
			AuthorizedClients: []*v1.Multihost_Peer{{Keyid: identity2.Keyid, InstanceId: defaultClientID}},
		},
	}

	peerClientConfig := &v1.Config{
		Version:  migrations.CurrentVersion,
		Instance: defaultClientID,
		Repos:    []*v1.Repo{{Id: defaultRepoID, Guid: repoGUID, Uri: "backrest://" + defaultHostID}},
		Multihost: &v1.Multihost{
			Identity: identity2,
			KnownHosts: []*v1.Multihost_Peer{{
				Keyid:       identity1.Keyid,
				InstanceId:  defaultHostID,
				InstanceUrl: fmt.Sprintf("http://%s", peerHostAddr),
				Permissions: []*v1.Multihost_Permission{{
					Type:   v1.Multihost_Permission_PERMISSION_READ_OPERATIONS,
					Scopes: []string{"repo:" + defaultRepoID},
				}},
			}},
		},
	}

	peerHost := newPeerUnderTest(t, peerHostConfig)
	peerClient := newPeerUnderTest(t, peerClientConfig)

	opTempl := &v1.Operation{
		InstanceId:      defaultClientID,
		RepoId:          defaultRepoID,
		RepoGuid:        repoGUID,
		PlanId:          defaultPlanID,
		UnixTimeStartMs: time.Now().UnixMilli(),
		UnixTimeEndMs:   time.Now().UnixMilli(),
		Status:          v1.OperationStatus_STATUS_SUCCESS,
		Op:              &v1.Operation_OperationBackup{},
	}

	rng := rand.New(rand.NewSource(42)) // deterministic seed
	query := oplog.Query{}.SetInstanceID(defaultClientID).SetRepoGUID(repoGUID)
	liveOps := map[int64]*v1.Operation{}

	// Phase 1: add initial operations while connected
	syncCtx, cancelSync := context.WithCancel(ctx)
	var syncWg sync.WaitGroup
	syncWg.Add(2)
	go func() { defer syncWg.Done(); runSyncAPIWithCtx(syncCtx, peerHost, peerHostAddr) }()
	go func() { defer syncWg.Done(); runSyncAPIWithCtx(syncCtx, peerClient, peerClientAddr) }()
	tryConnect(t, ctx, peerClient, peerClientConfig.Multihost.KnownHosts[0])

	for i := 0; i < initialOps; i++ {
		op := proto.Clone(opTempl).(*v1.Operation)
		op.DisplayMessage = fmt.Sprintf("init-%d", i)
		if err := peerClient.oplog.Add(op); err != nil {
			t.Fatalf("init add: %v", err)
		}
		liveOps[op.Id] = op
	}

	assertOpsConverge(t, ctx, peerClient, peerHost, query, "initial sync")

	// Phase 2: disconnect and mutate heavily
	cancelSync()
	syncWg.Wait()

	for i := 0; i < offlineOps; i++ {
		action := rng.Intn(10)
		switch {
		case action < 5: // 50%: add
			op := proto.Clone(opTempl).(*v1.Operation)
			op.DisplayMessage = fmt.Sprintf("offline-add-%d", i)
			if err := peerClient.oplog.Add(op); err != nil {
				t.Fatalf("offline add: %v", err)
			}
			liveOps[op.Id] = op
		case action < 8 && len(liveOps) > 0: // 30%: update
			op := pickRandom(rng, liveOps)
			op = proto.Clone(op).(*v1.Operation)
			op.DisplayMessage = fmt.Sprintf("offline-upd-%d", i)
			if err := peerClient.oplog.Update(op); err != nil {
				t.Fatalf("offline update: %v", err)
			}
			liveOps[op.Id] = op
		case len(liveOps) > 0: // 20%: delete
			op := pickRandom(rng, liveOps)
			if err := peerClient.oplog.Delete(op.Id); err != nil {
				t.Fatalf("offline delete: %v", err)
			}
			delete(liveOps, op.Id)
		}
	}

	// Phase 3: reconnect and verify convergence
	syncCtx, cancelSync = context.WithCancel(ctx)
	syncWg.Add(2)
	go func() { defer syncWg.Done(); runSyncAPIWithCtx(syncCtx, peerHost, peerHostAddr) }()
	go func() { defer syncWg.Done(); runSyncAPIWithCtx(syncCtx, peerClient, peerClientAddr) }()
	tryConnect(t, ctx, peerClient, peerClientConfig.Multihost.KnownHosts[0])

	assertOpsConverge(t, ctx, peerClient, peerHost, query, "after offline mutations")
	assertNoDuplicateOriginalIDs(t, peerHost, query)

	cancelSync()
	syncWg.Wait()
}

// --- helpers ---

func pickRandom(rng *rand.Rand, m map[int64]*v1.Operation) *v1.Operation {
	keys := make([]int64, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return m[keys[rng.Intn(len(keys))]]
}

func assertOpsConverge(t *testing.T, ctx context.Context, client, host *peerUnderTest, query oplog.Query, msg string) {
	t.Helper()
	err := testutil.Retry(t, ctx, func() error {
		clientOps := getOperations(t, client.oplog, query)
		hostOps := getOperations(t, host.oplog, query)

		// Normalize: clear locally-assigned fields that differ between peers.
		normalize := func(ops []*v1.Operation) []*v1.Operation {
			out := make([]*v1.Operation, len(ops))
			for i, op := range ops {
				c := proto.Clone(op).(*v1.Operation)
				c.Id = 0
				c.FlowId = 0
				c.OriginalId = 0
				c.OriginalFlowId = 0
				c.OriginalInstanceKeyid = ""
				c.Modno = 0
				out[i] = c
			}
			return out
		}

		cn := normalize(clientOps)
		hn := normalize(hostOps)

		sortByMessage := func(a, b *v1.Operation) int {
			if a.DisplayMessage < b.DisplayMessage {
				return -1
			}
			if a.DisplayMessage > b.DisplayMessage {
				return 1
			}
			return 0
		}

		sortByMessageStable(cn, sortByMessage)
		sortByMessageStable(hn, sortByMessage)

		if len(cn) == 0 && len(hn) == 0 {
			return nil // both empty is fine
		}
		if diff := cmp.Diff(cn, hn, protocmp.Transform()); diff != "" {
			return fmt.Errorf("not converged (client has %d, host has %d): %s", len(clientOps), len(hostOps), diff)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

func sortByMessageStable(ops []*v1.Operation, cmp func(a, b *v1.Operation) int) {
	for i := 1; i < len(ops); i++ {
		for j := i; j > 0 && cmp(ops[j-1], ops[j]) > 0; j-- {
			ops[j-1], ops[j] = ops[j], ops[j-1]
		}
	}
}

func assertNoDuplicateOriginalIDs(t *testing.T, peer *peerUnderTest, query oplog.Query) {
	t.Helper()
	ops := getOperations(t, peer.oplog, query)
	seen := map[int64]bool{}
	for _, op := range ops {
		origID := op.OriginalId
		if origID == 0 {
			continue
		}
		if seen[origID] {
			t.Errorf("duplicate original_id %d found on host", origID)
		}
		seen[origID] = true
	}
}
