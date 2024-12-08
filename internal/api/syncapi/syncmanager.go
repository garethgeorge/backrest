package syncapi

import (
	"context"
	"errors"
	"path/filepath"
	"sync"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/orchestrator"
	"go.uber.org/zap"
)

type SyncManager struct {
	configMgr         *config.ConfigManager
	orchestrator      *orchestrator.Orchestrator
	oplog             *oplog.OpLog
	remoteConfigStore RemoteConfigStore

	// mutable properties
	mu          sync.Mutex
	syncClients map[string]*SyncClient
}

func NewSyncManager(configMgr *config.ConfigManager, oplog *oplog.OpLog, orchestrator *orchestrator.Orchestrator, syncStateDir string) *SyncManager {
	remoteConfigStore := NewJSONDirRemoteConfigStore(filepath.Join(syncStateDir, "remote_configs"))

	return &SyncManager{
		configMgr:         configMgr,
		orchestrator:      orchestrator,
		oplog:             oplog,
		remoteConfigStore: remoteConfigStore,
	}
}

// Note: top level function will be called holding the lock, must kick off goroutines and then return.
func (m *SyncManager) RunSync(ctx context.Context) {
	var cancelLastSync context.CancelFunc

	configWatchCh := m.configMgr.Watch()
	defer m.configMgr.StopWatching(configWatchCh)

	runSyncWithNewConfig := func() {
		m.mu.Lock()
		defer m.mu.Unlock()

		// TODO: rather than cancel the top level context, something clever e.g. diffing the set of peers could be done here.
		if cancelLastSync != nil {
			cancelLastSync()
		}
		syncCtx, cancel := context.WithCancel(ctx)
		cancelLastSync = cancel

		config, err := m.configMgr.Get()
		if err != nil {
			zap.S().Errorf("syncmanager failed to refresh config with latest changes so sync is stopped: %v", err)
			return
		}

		if len(config.Multihost.GetKnownHosts()) == 0 {
			zap.L().Debug("syncmanager no known host peers declared, sync is disabled")
			return
		}

		zap.S().Info("syncmanager applying new config, starting sync goroutines for %d known peers", len(config.Multihost.GetKnownHosts()))
		for _, knownHostPeer := range config.Multihost.KnownHosts {
			go func(knownHostPeer *v1.Multihost_Peer) {
				zap.S().Debugf("syncmanager starting sync goroutine with peer %q", knownHostPeer.InstanceId)
				err := m.runSyncWithPeerInternal(syncCtx, config, knownHostPeer)
				if err != nil {
					zap.S().Errorf("syncmanager error running sync with peer %q: %v", knownHostPeer.InstanceId, err)
				}
			}(knownHostPeer)
		}
	}

	runSyncWithNewConfig()

	for {
		select {
		case <-ctx.Done():
			return
		case <-configWatchCh:
			runSyncWithNewConfig()
		}
	}
}

// runSyncWithPeerInternal starts the sync process with a single peer. It is expected to spawn a goroutine that will
// return when the context is canceled. Errors can only be returned upfront.
func (m *SyncManager) runSyncWithPeerInternal(ctx context.Context, config *v1.Config, knownHostPeer *v1.Multihost_Peer) error {
	if config.Instance == "" {
		return errors.New("local instance must set instance name before peersync can be enabled")
	}

	client, ok := m.syncClients[knownHostPeer.InstanceId]
	if !ok {
		newClient, err := NewSyncClient(m, config.Instance, knownHostPeer, m.oplog)
		if err != nil {
			zap.S().Errorf("syncmanager failed to create sync client for peer %q", knownHostPeer.InstanceId)
		}
		m.syncClients[knownHostPeer.InstanceId] = client
		client = newClient
	}

	go client.RunSync(ctx)
	return nil
}
