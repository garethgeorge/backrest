package syncapi

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
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
	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

const (
	defaultClientID = "test-client"
	defaultHostID   = "test-host"
	defaultRepoID   = "test-repo"
)

func TestConnectionSucceeds(t *testing.T) {
	installZapLogger(t)
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
	installZapLogger(t)
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

func TestSyncOperations(t *testing.T) {
	// TODO: other tests that exhaustively cover permissions e.g. trying to sync to invalid locations.

	installZapLogger(t)
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

func tryExpectConfig(t *testing.T, ctx context.Context, peer *peerUnderTest, instanceID string, wantCfg *v1.RemoteConfig) {
	try(t, ctx, func() error {
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
	try(t, ctx, func() error {
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

// try is a helper that spins until the condition becomes true OR the context is done.
func try(t *testing.T, ctx context.Context, f func() error) {
	t.Helper()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	var err error

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("try timeout before OK: %v", err)
		case <-ticker.C:
			err = f()
			if err == nil {
				return
			}
		}
	}
}

func installZapLogger(t *testing.T) {
	t.Helper()
	logger := zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.AddSync(&testLogger{t: t}),
		zapcore.DebugLevel,
	))
	zap.ReplaceGlobals(logger)
}

type testLogger struct {
	t *testing.T
}

func (l *testLogger) Write(p []byte) (n int, err error) {
	l.t.Log("global log: " + strings.Trim(string(p), "\n"))
	return len(p), nil
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
