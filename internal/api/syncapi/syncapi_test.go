package syncapi

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1sync"
	"github.com/garethgeorge/backrest/gen/go/v1sync/v1syncconnect"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/config/migrations"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
	"github.com/garethgeorge/backrest/internal/logstore"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/oplog/sqlitestore"
	"github.com/garethgeorge/backrest/internal/orchestrator"
	"github.com/garethgeorge/backrest/internal/resticinstaller"
	"github.com/garethgeorge/backrest/internal/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

const (
	defaultClientID = "test-client"
	defaultHostID   = "test-host"
	defaultRepoID   = "test-repo"
	defaultPlanID   = "test-plan"
)

var (
	defaultRepoGUID = cryptoutil.MustRandomID(cryptoutil.DefaultIDBits)

	identity1, _ = cryptoutil.GeneratePrivateKey()
	identity2, _ = cryptoutil.GeneratePrivateKey()
)

var (
	basicHostOperationTempl = &v1.Operation{
		InstanceId:      defaultHostID,
		RepoId:          defaultRepoID,
		RepoGuid:        defaultRepoGUID,
		PlanId:          defaultPlanID,
		UnixTimeStartMs: time.Now().UnixMilli() - 1000, // 1 second ago
		UnixTimeEndMs:   time.Now().UnixMilli(),
		Status:          v1.OperationStatus_STATUS_SUCCESS,
		Op:              &v1.Operation_OperationBackup{},
	}

	basicClientOperationTempl = &v1.Operation{
		InstanceId:      defaultClientID,
		RepoId:          defaultRepoID,
		RepoGuid:        defaultRepoGUID,
		PlanId:          defaultPlanID,
		UnixTimeStartMs: time.Now().UnixMilli() - 1000, // 1 second ago
		UnixTimeEndMs:   time.Now().UnixMilli(),
		Status:          v1.OperationStatus_STATUS_SUCCESS,
		Op:              &v1.Operation_OperationBackup{},
	}
)

func TestConnectionSucceeds(t *testing.T) {
	testutil.InstallZapLogger(t)
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	peerHostAddr := testutil.AllocOpenBindAddr(t)
	peerClientAddr := testutil.AllocOpenBindAddr(t)

	peerHostConfig := &v1.Config{
		Version:  migrations.CurrentVersion,
		Instance: defaultHostID,
		Repos:    []*v1.Repo{},
		Multihost: &v1.Multihost{
			Identity: identity1,
			AuthorizedClients: []*v1.Multihost_Peer{
				{
					Keyid:         identity2.Keyid,
					KeyidVerified: true,
					InstanceId:    defaultClientID,
				},
			},
		},
	}

	peerClientConfig := &v1.Config{
		Version:  migrations.CurrentVersion,
		Instance: defaultClientID,
		Repos:    []*v1.Repo{},
		Multihost: &v1.Multihost{
			Identity: identity2,
			KnownHosts: []*v1.Multihost_Peer{
				{
					Keyid:       identity1.Keyid,
					InstanceId:  defaultHostID,
					InstanceUrl: fmt.Sprintf("http://%s", peerHostAddr),
				},
			},
		},
	}

	peerHost := newPeerUnderTest(t, peerHostConfig)
	peerClient := newPeerUnderTest(t, peerClientConfig)

	startRunningSyncAPI(t, peerHost, peerHostAddr)
	startRunningSyncAPI(t, peerClient, peerClientAddr)

	tryConnect(t, ctx, peerClient, peerClientConfig.Multihost.KnownHosts[0])
}

func TestConnectionBadKeyRejected(t *testing.T) {
	testutil.InstallZapLogger(t)
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	peerHostAddr := testutil.AllocOpenBindAddr(t)
	peerClientAddr := testutil.AllocOpenBindAddr(t)

	// Host has identity1, and authorizes no one.
	peerHostConfig := &v1.Config{
		Version:  migrations.CurrentVersion,
		Instance: defaultHostID,
		Repos:    []*v1.Repo{},
		Multihost: &v1.Multihost{
			Identity:          identity1,
			AuthorizedClients: []*v1.Multihost_Peer{}, // No authorized clients
		},
	}

	// Client has identity2 and tries to connect to host.
	peerClientConfig := &v1.Config{
		Version:  migrations.CurrentVersion,
		Instance: defaultClientID,
		Repos:    []*v1.Repo{},
		Multihost: &v1.Multihost{
			Identity: identity2,
			KnownHosts: []*v1.Multihost_Peer{
				{
					Keyid:       identity1.Keyid,
					InstanceId:  defaultHostID,
					InstanceUrl: fmt.Sprintf("http://%s", peerHostAddr),
				},
			},
		},
	}

	peerHost := newPeerUnderTest(t, peerHostConfig)
	peerClient := newPeerUnderTest(t, peerClientConfig)

	startRunningSyncAPI(t, peerHost, peerHostAddr)
	startRunningSyncAPI(t, peerClient, peerClientAddr)

	waitForConnectionState(t, ctx, peerClient, peerClientConfig.Multihost.KnownHosts[0], v1sync.ConnectionState_CONNECTION_STATE_ERROR_AUTH)
}

func TestSyncConfigChange(t *testing.T) {
	testutil.InstallZapLogger(t)
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	peerHostAddr := testutil.AllocOpenBindAddr(t)
	peerClientAddr := testutil.AllocOpenBindAddr(t)

	peerHostConfig := &v1.Config{
		Version:  migrations.CurrentVersion,
		Modno:    0,
		Instance: defaultHostID,
		Repos: []*v1.Repo{
			{
				Uri:  "test-uri-should-sync",
				Id:   defaultRepoID,
				Guid: defaultRepoGUID,
			},
			{
				Uri:  "test-uri-do-not-sync",
				Id:   "do-not-sync",
				Guid: cryptoutil.MustRandomID(cryptoutil.DefaultIDBits),
			},
		},
		Multihost: &v1.Multihost{
			Identity: identity1,
			AuthorizedClients: []*v1.Multihost_Peer{
				{
					Keyid:         identity2.Keyid,
					KeyidVerified: true,
					InstanceId:    defaultClientID,
					Permissions: []*v1.Multihost_Permission{
						{
							Type: v1.Multihost_Permission_PERMISSION_READ_CONFIG,
							Scopes: []string{
								"repo:" + defaultRepoID,
							},
						},
					},
				},
			},
		},
	}

	peerClientConfig := &v1.Config{
		Version:  migrations.CurrentVersion,
		Instance: defaultClientID,
		Repos: []*v1.Repo{
			{
				Id:   defaultRepoID,
				Guid: defaultRepoGUID,
				Uri:  "backrest://" + defaultHostID, // TODO: get rid of the :// requirement
			},
		},
		Multihost: &v1.Multihost{
			Identity: identity2,
			KnownHosts: []*v1.Multihost_Peer{
				{
					Keyid:       identity1.Keyid,
					InstanceId:  defaultHostID,
					InstanceUrl: fmt.Sprintf("http://%s", peerHostAddr),
				},
			},
		},
	}

	peerHost := newPeerUnderTest(t, peerHostConfig)
	peerClient := newPeerUnderTest(t, peerClientConfig)

	startRunningSyncAPI(t, peerHost, peerHostAddr)
	startRunningSyncAPI(t, peerClient, peerClientAddr)

	tryConnect(t, ctx, peerClient, peerClientConfig.Multihost.KnownHosts[0])

	// wait for the initial config to propagate
	tryExpectConfigFromHost(t, ctx, peerClient, peerClientConfig.Multihost.KnownHosts[0], &v1sync.RemoteConfig{
		Version: migrations.CurrentVersion,
		Repos: []*v1.Repo{
			{
				Id:   defaultRepoID,
				Guid: defaultRepoGUID,
				Uri:  "test-uri-should-sync",
			},
		},
	})
	hostConfigChanged := proto.Clone(peerHostConfig).(*v1.Config)
	hostConfigChanged.Repos[1].Env = []string{"SOME_ENV=VALUE"}
	hostConfigChanged.Modno += 1
	zap.S().Infof("updating host config to: %s", protojson.Format(hostConfigChanged))
	peerHost.configMgr.Update(hostConfigChanged)

	tryConnect(t, ctx, peerClient, peerClientConfig.Multihost.KnownHosts[0])

	tryExpectConfigFromHost(t, ctx, peerClient, peerClientConfig.Multihost.KnownHosts[0], &v1sync.RemoteConfig{
		Version: migrations.CurrentVersion,
		Modno:   1,
		Repos: []*v1.Repo{
			{
				Id:   defaultRepoID,
				Guid: defaultRepoGUID,
				Uri:  "test-uri-should-sync",
				Env:  []string{"SOME_ENV=VALUE"},
			},
		},
	})
}

func TestSimpleOperationSync(t *testing.T) {
	testutil.InstallZapLogger(t)
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	peerHostAddr := testutil.AllocOpenBindAddr(t)
	peerClientAddr := testutil.AllocOpenBindAddr(t)

	peerHostConfig := &v1.Config{
		Version:  migrations.CurrentVersion,
		Instance: defaultHostID,
		Repos: []*v1.Repo{
			{
				Id:   defaultRepoID,
				Guid: defaultRepoGUID,
				Uri:  "test-uri",
			},
		},
		Multihost: &v1.Multihost{
			Identity: identity1,
			AuthorizedClients: []*v1.Multihost_Peer{
				{
					Keyid:         identity2.Keyid,
					KeyidVerified: true,
					InstanceId:    defaultClientID,
				},
			},
		},
	}

	peerClientConfig := &v1.Config{
		Version:  migrations.CurrentVersion,
		Instance: defaultClientID,
		Repos: []*v1.Repo{
			{
				Id:   defaultRepoID,
				Guid: defaultRepoGUID,
				Uri:  "backrest://" + defaultHostID, // TODO: get rid of the :// requirement
			},
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
							Type: v1.Multihost_Permission_PERMISSION_READ_OPERATIONS,
							Scopes: []string{
								"repo:" + defaultRepoID,
							},
						},
					},
				},
			},
		},
	}

	peerHost := newPeerUnderTest(t, peerHostConfig)
	peerClient := newPeerUnderTest(t, peerClientConfig)

	peerHost.oplog.Add(testutil.OperationsWithDefaults(basicHostOperationTempl, []*v1.Operation{
		{
			DisplayMessage: "hostop1",
		},
	})...)
	// peerHost.oplog.Add(testutil.OperationsWithDefaults(basicClientOperationTempl, []*v1.Operation{
	// 	{
	// 		DisplayMessage: "clientop-deleted",
	// 		OriginalId:     1234, // must be an ID that doesn't exist remotely
	// 	},
	// })...)

	if err := peerClient.oplog.Add(testutil.OperationsWithDefaults(basicClientOperationTempl, []*v1.Operation{
		{
			DisplayMessage: "clientop1",
			FlowId:         1,
		},
		{
			DisplayMessage: "clientop2",
			FlowId:         1,
		},
		{
			DisplayMessage: "clientop3",
			FlowId:         2, // in a different flow from the other two
		},
	})...); err != nil {
		t.Fatalf("failed to add operations: %v", err)
	}

	startRunningSyncAPI(t, peerHost, peerHostAddr)
	startRunningSyncAPI(t, peerClient, peerClientAddr)

	tryConnect(t, ctx, peerClient, peerClientConfig.Multihost.KnownHosts[0])

	tryExpectOperationsSynced(t, ctx, peerHost, peerClient, oplog.Query{}.SetInstanceID(defaultClientID).SetRepoGUID(defaultRepoGUID), "host and client should be synced")
	tryExpectExactOperations(t, ctx, peerHost, oplog.Query{}.SetInstanceID(defaultClientID).SetRepoGUID(defaultRepoGUID),
		testutil.OperationsWithDefaults(basicClientOperationTempl, []*v1.Operation{
			{
				Id:                    2, // b/c of the already inserted host ops the sync'd ops start at 3
				FlowId:                2,
				OriginalId:            1,
				OriginalFlowId:        1,
				OriginalInstanceKeyid: identity2.Keyid,
				DisplayMessage:        "clientop1",
			},
			{
				Id:                    3,
				FlowId:                2,
				OriginalId:            2,
				OriginalFlowId:        1,
				OriginalInstanceKeyid: identity2.Keyid,
				DisplayMessage:        "clientop2",
			},
			{
				Id:                    4,
				FlowId:                4,
				OriginalId:            3,
				OriginalFlowId:        2,
				OriginalInstanceKeyid: identity2.Keyid,
				DisplayMessage:        "clientop3",
			},
		}), "host and client should be synced")
}

func TestSyncMutations(t *testing.T) {
	testutil.InstallZapLogger(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	peerHostAddr := testutil.AllocOpenBindAddr(t)
	peerClientAddr := testutil.AllocOpenBindAddr(t)

	peerHostConfig := &v1.Config{
		Version:  migrations.CurrentVersion,
		Instance: defaultHostID,
		Repos: []*v1.Repo{
			{
				Id:   defaultRepoID,
				Guid: defaultRepoGUID,
				Uri:  "test-uri",
			},
		},
		Multihost: &v1.Multihost{
			Identity: identity1,
			AuthorizedClients: []*v1.Multihost_Peer{
				{
					Keyid:         identity2.Keyid,
					KeyidVerified: true,
					InstanceId:    defaultClientID,
				},
			},
		},
	}

	peerClientConfig := &v1.Config{
		Version:  migrations.CurrentVersion,
		Instance: defaultClientID,
		Repos: []*v1.Repo{
			{
				Id:   defaultRepoID,
				Guid: defaultRepoGUID,
				Uri:  "backrest://" + defaultHostID, // TODO: get rid of the :// requirement
			},
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
							Type: v1.Multihost_Permission_PERMISSION_READ_OPERATIONS,
							Scopes: []string{
								"repo:" + defaultRepoID,
							},
						},
					},
				},
			},
		},
	}

	peerHost := newPeerUnderTest(t, peerHostConfig)
	peerClient := newPeerUnderTest(t, peerClientConfig)

	op := testutil.OperationsWithDefaults(basicClientOperationTempl, []*v1.Operation{
		{
			DisplayMessage: "clientop1",
		},
	})[0]

	if err := peerClient.oplog.Add(op); err != nil {
		t.Fatalf("failed to add operations: %v", err)
	}

	syncCtx, cancelSyncCtx := context.WithCancel(ctx)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		runSyncAPIWithCtx(syncCtx, peerHost, peerHostAddr)
	}()
	go func() {
		defer wg.Done()
		runSyncAPIWithCtx(syncCtx, peerClient, peerClientAddr)
	}()
	tryConnect(t, ctx, peerClient, peerClientConfig.Multihost.KnownHosts[0])

	tryExpectOperationsSynced(t, ctx, peerClient, peerHost, oplog.Query{}.SetRepoGUID(defaultRepoGUID), "host and client should sync initially")

	op.DisplayMessage = "clientop1-mod-while-online"
	if err := peerClient.oplog.Update(op); err != nil {
		t.Fatalf("failed to update operation: %v", err)
	}

	tryExpectExactOperations(t, ctx, peerHost, oplog.Query{}.SetRepoGUID(defaultRepoGUID),
		testutil.OperationsWithDefaults(basicClientOperationTempl, []*v1.Operation{
			{
				Id:                    1,
				DisplayMessage:        "clientop1-mod-while-online",
				OriginalFlowId:        1,
				OriginalId:            1,
				FlowId:                1,
				OriginalInstanceKeyid: identity2.Keyid,
			},
		}), "host and client should sync online edits")

	// Wait for shutdown
	cancelSyncCtx()
	wg.Wait()

	// Now make an offline edit
	op.DisplayMessage = "clientop1-mod-while-offline"
	if err := peerClient.oplog.Update(op); err != nil {
		t.Fatalf("failed to add operations: %v", err)
	}

	// Now restart sync and check that the offline edit is applied
	syncCtx, cancelSyncCtx = context.WithCancel(ctx)
	wg.Add(2)
	go func() {
		defer wg.Done()
		runSyncAPIWithCtx(syncCtx, peerHost, peerHostAddr)
	}()

	go func() {
		defer wg.Done()
		runSyncAPIWithCtx(syncCtx, peerClient, peerClientAddr)
	}()
	tryConnect(t, ctx, peerClient, peerClientConfig.Multihost.KnownHosts[0])

	// Verify all operations are synced after reconnection
	tryExpectExactOperations(t, ctx, peerHost, oplog.Query{}.SetRepoGUID(defaultRepoGUID),
		testutil.OperationsWithDefaults(basicClientOperationTempl, []*v1.Operation{
			{
				Id:                    1,
				DisplayMessage:        "clientop1-mod-while-offline",
				OriginalFlowId:        1,
				OriginalId:            1,
				FlowId:                1,
				OriginalInstanceKeyid: identity2.Keyid,
			},
		}), "host and client should sync offline edits")

	// Clean up
	cancelSyncCtx()
	wg.Wait()
}

func getOperations(t *testing.T, oplog *oplog.OpLog, query oplog.Query) []*v1.Operation {
	ops := []*v1.Operation{}
	if err := oplog.Query(query, func(op *v1.Operation) error {
		ops = append(ops, op)
		return nil
	}); err != nil {
		t.Fatalf("failed to get operations: %v", err)
	}
	return ops
}

func tryExpectExactOperations(t *testing.T, ctx context.Context, peer *peerUnderTest, query oplog.Query, wantOps []*v1.Operation, message string) {
	err := testutil.Retry(t, ctx, func() error {
		ops := getOperations(t, peer.oplog, query)
		for _, op := range ops {
			op.Modno = 0
		}
		if diff := cmp.Diff(ops, wantOps, protocmp.Transform()); diff != "" {
			return fmt.Errorf("unexpected diff: %v", diff)
		}
		return nil
	})
	if err != nil {
		opsJson, _ := protojson.MarshalOptions{Indent: "  "}.Marshal(&v1.OperationList{Operations: getOperations(t, peer.oplog, query)})
		t.Logf("found operations: %v", string(opsJson))
		t.Fatalf("%v: timeout without finding wanted operations: %v", message, err)
	}
}

func tryExpectOperationsSynced(t *testing.T, ctx context.Context, peer1 *peerUnderTest, peer2 *peerUnderTest, query oplog.Query, message string) {
	err := testutil.Retry(t, ctx, func() error {
		peer1Ops := getOperations(t, peer1.oplog, query)
		peer2Ops := getOperations(t, peer2.oplog, query)
		// clear fields that we expect will be re-mapped
		for _, op := range peer1Ops {
			op.Id = 0
			op.FlowId = 0
			op.OriginalId = 0
			op.OriginalFlowId = 0
			op.OriginalInstanceKeyid = ""
			op.Modno = 0
		}
		for _, op := range peer2Ops {
			op.Id = 0
			op.FlowId = 0
			op.OriginalId = 0
			op.OriginalFlowId = 0
			op.OriginalInstanceKeyid = ""
			op.Modno = 0
		}

		sortFn := func(a, b *v1.Operation) int {
			if a.DisplayMessage < b.DisplayMessage {
				return -1
			}
			return 1
		}

		slices.SortFunc(peer1Ops, sortFn)
		slices.SortFunc(peer2Ops, sortFn)

		if len(peer1Ops) == 0 {
			return errors.New("no operations found in peer1")
		}
		if len(peer2Ops) == 0 {
			return errors.New("no operations found in peer2")
		}
		if diff := cmp.Diff(peer1Ops, peer2Ops, protocmp.Transform()); diff != "" {
			return fmt.Errorf("%s: unexpected diff: %v", message, diff)
		}

		return nil
	})
	if err != nil {
		ops1Json, _ := protojson.MarshalOptions{Indent: "  "}.Marshal(&v1.OperationList{Operations: getOperations(t, peer1.oplog, query)})
		ops2Json, _ := protojson.MarshalOptions{Indent: "  "}.Marshal(&v1.OperationList{Operations: getOperations(t, peer2.oplog, query)})
		t.Logf("peer1 operations: %v", string(ops1Json))
		t.Logf("peer2 operations: %v", string(ops2Json))
		t.Fatalf("timeout without syncing operations: %v", err)
	}
}

func tryExpectConfigFromHost(t *testing.T, ctx context.Context, peer *peerUnderTest, hostPeer *v1.Multihost_Peer, wantCfg *v1sync.RemoteConfig) {
	testutil.Try(t, ctx, func() error {
		state := peer.manager.peerStateManager.GetPeerState(hostPeer.Keyid)
		if state == nil {
			return fmt.Errorf("no state found for host peer %s", hostPeer.InstanceId)
		}
		if diff := cmp.Diff(state.Config, wantCfg, protocmp.Transform()); diff != "" {
			return fmt.Errorf("unexpected diff: %v", diff)
		}
		return nil
	})
}

func waitForConnectionState(t *testing.T, ctx context.Context, peer *peerUnderTest, hostPeer *v1.Multihost_Peer, wantState v1sync.ConnectionState) {
	ctx, cancel := testutil.WithDeadlineFromTest(t, ctx)
	defer cancel()

	// Important that we subscribe to the state change before we try the initial check to avoid races.
	onStateChanged := peer.manager.peerStateManager.OnStateChanged().Subscribe()
	defer peer.manager.peerStateManager.OnStateChanged().Unsubscribe(onStateChanged)

	// First check if the peer is already connected.
	state := peer.manager.peerStateManager.GetPeerState(hostPeer.Keyid)
	if state != nil && state.ConnectionState == v1sync.ConnectionState_CONNECTION_STATE_CONNECTED {
		return // Already connected, nothing to do
	}

	// If not connected, wait for a connection event
	var lastState *PeerState
	stop := false
	for !stop {
		select {
		case state, ok := <-onStateChanged:
			if !ok {
				stop = true
				continue
			}
			if state.KeyID == hostPeer.Keyid && state.InstanceID == hostPeer.InstanceId {
				lastState = state
				if state.ConnectionState == v1sync.ConnectionState_CONNECTION_STATE_CONNECTED {
					stop = true
					continue
				}
			}
		case <-ctx.Done():
			stop = true
			continue
		}
	}
	if lastState == nil {
		t.Fatalf("timeout waiting for connection to host peer %s", hostPeer.InstanceId)
	} else if lastState.ConnectionState != v1sync.ConnectionState_CONNECTION_STATE_CONNECTED {
		t.Fatalf("expected connection state to be CONNECTED, got %v (reason: %q)", lastState.ConnectionState, lastState.ConnectionStateMessage)
	}
}

func tryConnect(t *testing.T, ctx context.Context, peer *peerUnderTest, hostPeer *v1.Multihost_Peer) {
	waitForConnectionState(t, ctx, peer, hostPeer, v1sync.ConnectionState_CONNECTION_STATE_CONNECTED)
}

func runSyncAPIWithCtx(ctx context.Context, peer *peerUnderTest, bindAddr string) {
	mux := http.NewServeMux()
	syncHandler := NewBackrestSyncHandler(peer.manager)
	mux.Handle(v1syncconnect.NewBackrestSyncServiceHandler(syncHandler))

	server := &http.Server{
		Addr:    bindAddr,
		Handler: h2c.NewHandler(mux, &http2.Server{}), // h2c is HTTP/2 without TLS for grpc-connect support.
	}

	var wg sync.WaitGroup

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	wg.Add(1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		peer.manager.RunSync(ctx)
		wg.Done()
	}()

	wg.Wait()
}

func startRunningSyncAPI(t *testing.T, peer *peerUnderTest, bindAddr string) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go runSyncAPIWithCtx(ctx, peer, bindAddr)
}

type peerUnderTest struct {
	manager   *SyncManager
	oplog     *oplog.OpLog
	opstore   oplog.OpStore
	configMgr *config.ConfigManager
}

func newPeerUnderTest(t *testing.T, initialConfig *v1.Config) *peerUnderTest {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())

	configMgr := &config.ConfigManager{Store: &config.MemoryStore{Config: initialConfig}}
	opstore, err := sqlitestore.NewMemorySqliteStore(t)
	t.Cleanup(func() { opstore.Close() })
	if err != nil {
		t.Fatalf("failed to create opstore: %v", err)
	}
	oplog, err := oplog.NewOpLog(opstore)
	if err != nil {
		t.Fatalf("failed to create oplog: %v", err)
	}

	resticbin, err := resticinstaller.FindOrInstallResticBinary()
	if err != nil {
		t.Fatalf("failed to find or install restic binary: %v", err)
	}

	tempDir := t.TempDir()
	logStore, err := logstore.NewLogStore(filepath.Join(tempDir, "tasklogs"))
	t.Cleanup(func() { logStore.Close() })
	if err != nil {
		t.Fatalf("failed to create log store: %v", err)
	}

	var wg sync.WaitGroup
	orchestrator, err := orchestrator.NewOrchestrator(resticbin, configMgr, oplog, logStore)
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}
	wg.Add(1)
	go func() {
		orchestrator.Run(ctx)
		wg.Done()
	}()

	dbpool, err := sql.Open("sqlite3", memdb.TestDB(t))
	if err != nil {
		t.Fatalf("failed to open sqlite pool: %v", err)
	}

	t.Cleanup(func() {
		cancel()
		wg.Wait()
		dbpool.Close()
	})

	peerStateManager, err := NewSqlitePeerStateManager(dbpool)
	if err != nil {
		t.Fatalf("failed to create peer state manager: %v", err)
	}

	manager := NewSyncManager(configMgr, oplog, orchestrator, peerStateManager)
	manager.syncClientRetryDelay = 250 * time.Millisecond

	return &peerUnderTest{
		manager:   manager,
		oplog:     oplog,
		opstore:   opstore,
		configMgr: configMgr,
	}
}
