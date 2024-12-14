package syncapi

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"slices"
	"testing"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1/v1connect"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/logstore"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/oplog/memstore"
	"github.com/garethgeorge/backrest/internal/orchestrator"
	"github.com/garethgeorge/backrest/internal/resticinstaller"
	"github.com/garethgeorge/backrest/internal/testutil"
	"github.com/google/go-cmp/cmp"
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
	basicHostOperationTempl = &v1.Operation{
		InstanceId:      defaultHostID,
		RepoId:          defaultRepoID,
		PlanId:          defaultPlanID,
		UnixTimeStartMs: 1234,
		UnixTimeEndMs:   5678,
		Status:          v1.OperationStatus_STATUS_SUCCESS,
		Op:              &v1.Operation_OperationBackup{},
	}

	basicClientOperationTempl = &v1.Operation{
		InstanceId:      defaultClientID,
		RepoId:          defaultRepoID,
		PlanId:          defaultPlanID,
		UnixTimeStartMs: 1234,
		UnixTimeEndMs:   5678,
		Status:          v1.OperationStatus_STATUS_SUCCESS,
		Op:              &v1.Operation_OperationBackup{},
	}
)

func TestConnectionSucceeds(t *testing.T) {
	testutil.InstallZapLogger(t)
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	peerHostAddr := allocBindAddrForTest(t)
	peerClientAddr := allocBindAddrForTest(t)

	peerHostConfig := &v1.Config{
		Instance: defaultHostID,
		Repos:    []*v1.Repo{},
		Multihost: &v1.Multihost{
			AuthorizedClients: []*v1.Multihost_Peer{
				{
					InstanceId: defaultClientID,
				},
			},
		},
	}

	peerClientConfig := &v1.Config{
		Instance: defaultClientID,
		Repos:    []*v1.Repo{},
		Multihost: &v1.Multihost{
			KnownHosts: []*v1.Multihost_Peer{
				{
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

	tryConnect(t, ctx, peerClient, defaultHostID)
}

func TestSyncConfigChange(t *testing.T) {
	testutil.InstallZapLogger(t)
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	peerHostAddr := allocBindAddrForTest(t)
	peerClientAddr := allocBindAddrForTest(t)

	peerHostConfig := &v1.Config{
		Instance: defaultHostID,
		Repos: []*v1.Repo{
			{
				Id:                     defaultRepoID,
				AllowedPeerInstanceIds: []string{defaultClientID},
			},
			{
				Id:                     "do-not-sync",
				AllowedPeerInstanceIds: []string{"some-other-client"},
			},
		},
		Multihost: &v1.Multihost{
			AuthorizedClients: []*v1.Multihost_Peer{
				{
					InstanceId: defaultClientID,
				},
			},
		},
	}

	peerClientConfig := &v1.Config{
		Instance: defaultClientID,
		Repos: []*v1.Repo{
			{
				Id:  defaultRepoID,
				Uri: "backrest:" + defaultHostID,
			},
		},
		Multihost: &v1.Multihost{
			KnownHosts: []*v1.Multihost_Peer{
				{
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

	tryConnect(t, ctx, peerClient, defaultHostID)

	// wait for the initial config to propagate
	tryExpectConfig(t, ctx, peerClient, defaultHostID, &v1.RemoteConfig{
		Repos: []*v1.RemoteRepo{
			{
				Id: defaultRepoID,
			},
		},
	})
	hostConfigChanged := proto.Clone(peerHostConfig).(*v1.Config)
	hostConfigChanged.Repos[0].Env = []string{"SOME_ENV=VALUE"}
	peerHost.configMgr.Update(hostConfigChanged)

	tryExpectConfig(t, ctx, peerClient, defaultHostID, &v1.RemoteConfig{
		Repos: []*v1.RemoteRepo{
			{
				Id:  defaultRepoID,
				Env: []string{"SOME_ENV=VALUE"},
			},
		},
	})
}

func TestSimpleOperationSync(t *testing.T) {
	testutil.InstallZapLogger(t)
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	peerHostAddr := allocBindAddrForTest(t)
	peerClientAddr := allocBindAddrForTest(t)

	peerHostConfig := &v1.Config{
		Instance: defaultHostID,
		Repos: []*v1.Repo{
			{
				Id:                     defaultRepoID,
				AllowedPeerInstanceIds: []string{defaultClientID},
			},
		},
		Multihost: &v1.Multihost{
			AuthorizedClients: []*v1.Multihost_Peer{
				{
					InstanceId: defaultClientID,
				},
			},
		},
	}

	peerClientConfig := &v1.Config{
		Instance: defaultClientID,
		Repos: []*v1.Repo{
			{
				Id:  defaultRepoID,
				Uri: "backrest://" + defaultHostID, // TODO: get rid of the :// requirement
			},
		},
		Multihost: &v1.Multihost{
			KnownHosts: []*v1.Multihost_Peer{
				{
					InstanceId:  defaultHostID,
					InstanceUrl: fmt.Sprintf("http://%s", peerHostAddr),
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
	peerHost.oplog.Add(testutil.OperationsWithDefaults(basicClientOperationTempl, []*v1.Operation{
		{
			DisplayMessage: "clientop-missing",
			OriginalId:     1234, // must be an ID that doesn't exist remotely
		},
	})...)

	peerClient.oplog.Add(testutil.OperationsWithDefaults(basicClientOperationTempl, []*v1.Operation{
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
	})...)

	startRunningSyncAPI(t, peerHost, peerHostAddr)
	startRunningSyncAPI(t, peerClient, peerClientAddr)

	tryConnect(t, ctx, peerClient, defaultHostID)

	tryExpectOperationsSynced(t, ctx, peerHost, peerClient, oplog.Query{RepoID: defaultRepoID, InstanceID: defaultClientID})
	tryExpectExactOperations(t, ctx, peerHost, oplog.Query{RepoID: defaultRepoID, InstanceID: defaultClientID},
		testutil.OperationsWithDefaults(basicClientOperationTempl, []*v1.Operation{
			{
				Id:             3, // b/c of the already inserted host ops the sync'd ops start at 3
				FlowId:         3,
				OriginalId:     1,
				OriginalFlowId: 1,
				DisplayMessage: "clientop1",
			},
			{
				Id:             4,
				FlowId:         3,
				OriginalId:     2,
				OriginalFlowId: 1,
				DisplayMessage: "clientop2",
			},
			{
				Id:             5,
				FlowId:         5,
				OriginalId:     3,
				OriginalFlowId: 2,
				DisplayMessage: "clientop3",
			},
		}))
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

func tryExpectExactOperations(t *testing.T, ctx context.Context, peer *peerUnderTest, query oplog.Query, wantOps []*v1.Operation) {
	err := testutil.Retry(t, ctx, func() error {
		ops := getOperations(t, peer.oplog, query)
		if diff := cmp.Diff(ops, wantOps, protocmp.Transform()); diff != "" {
			return fmt.Errorf("unexpected diff: %v", diff)
		}
		return nil
	})
	if err != nil {
		opsJson, _ := protojson.MarshalOptions{Indent: "  "}.Marshal(&v1.OperationList{Operations: getOperations(t, peer.oplog, query)})
		t.Logf("found operations: %v", string(opsJson))
		t.Fatalf("timeout without finding wanted operations: %v", err)
	}
}

func tryExpectOperationsSynced(t *testing.T, ctx context.Context, peer1 *peerUnderTest, peer2 *peerUnderTest, query oplog.Query) {
	err := testutil.Retry(t, ctx, func() error {
		peer1Ops := getOperations(t, peer1.oplog, query)
		peer2Ops := getOperations(t, peer2.oplog, query)
		// clear fields that we expect will be re-mapped
		for _, op := range peer1Ops {
			op.Id = 0
			op.FlowId = 0
			op.OriginalId = 0
			op.OriginalFlowId = 0
		}
		for _, op := range peer2Ops {
			op.Id = 0
			op.FlowId = 0
			op.OriginalId = 0
			op.OriginalFlowId = 0
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
			return fmt.Errorf("unexpected diff: %v", diff)
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

func tryExpectConfig(t *testing.T, ctx context.Context, peer *peerUnderTest, instanceID string, wantCfg *v1.RemoteConfig) {
	testutil.Try(t, ctx, func() error {
		cfg, err := peer.manager.remoteConfigStore.Get(instanceID)
		if err != nil {
			return err
		}
		if diff := cmp.Diff(cfg, wantCfg, protocmp.Transform()); diff != "" {
			return fmt.Errorf("unexpected diff: %v", diff)
		}
		return nil
	})
}

func tryConnect(t *testing.T, ctx context.Context, peer *peerUnderTest, instanceID string) {
	testutil.Try(t, ctx, func() error {
		allClients := peer.manager.GetSyncClients()
		client, ok := allClients[instanceID]
		if !ok {
			return fmt.Errorf("client not found, got %v", allClients)
		}
		state, _ := client.GetConnectionState()
		if state != v1.SyncConnectionState_CONNECTION_STATE_CONNECTED {
			return fmt.Errorf("expected connection state to be CONNECTED, got %v", v1.SyncConnectionState.String(state))
		}
		return nil
	})
}

func allocBindAddrForTest(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()

	// Get the port number from the listener
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("failed to split host and port: %v", err)
	}

	return "localhost:" + port
}

func startRunningSyncAPI(t *testing.T, peer *peerUnderTest, bindAddr string) {
	t.Helper()

	mux := http.NewServeMux()
	syncHandler := NewBackrestSyncHandler(peer.manager)
	mux.Handle(v1connect.NewBackrestSyncServiceHandler(syncHandler))

	server := &http.Server{
		Addr:    bindAddr,
		Handler: h2c.NewHandler(mux, &http2.Server{}), // h2c is HTTP/2 without TLS for grpc-connect support.
	}

	t.Cleanup(func() { server.Shutdown(context.Background()) })

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		peer.manager.RunSync(ctx)
	}()
	t.Cleanup(cancel)
}

type peerUnderTest struct {
	manager   *SyncManager
	oplog     *oplog.OpLog
	opstore   *memstore.MemStore
	configMgr *config.ConfigManager
}

func newPeerUnderTest(t *testing.T, initialConfig *v1.Config) *peerUnderTest {
	t.Helper()

	configMgr := &config.ConfigManager{Store: &config.MemoryStore{Config: initialConfig}}
	opstore := memstore.NewMemStore()
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
	if err != nil {
		t.Fatalf("failed to create log store: %v", err)
	}

	orchestrator, err := orchestrator.NewOrchestrator(resticbin, initialConfig, oplog, logStore)
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}

	ch := configMgr.Watch()

	go func() {
		for range ch {
			cfg, _ := configMgr.Get()
			orchestrator.ApplyConfig(cfg)
		}
	}()

	t.Cleanup(func() {
		configMgr.StopWatching(ch)
	})

	return &peerUnderTest{
		manager:   NewSyncManager(configMgr, oplog, orchestrator, filepath.Join(tempDir, "sync")),
		oplog:     oplog,
		opstore:   opstore,
		configMgr: configMgr,
	}
}
