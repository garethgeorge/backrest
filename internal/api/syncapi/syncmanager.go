package syncapi

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sync"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1sync"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/orchestrator"
	"go.uber.org/zap"
)

type SyncManager struct {
	configMgr    *config.ConfigManager
	orchestrator *orchestrator.Orchestrator
	oplog        *oplog.OpLog

	// mutable properties
	mu sync.Mutex

	snapshot *syncConfigSnapshot // the current snapshot of the sync context, protected by mu

	syncClientRetryDelay time.Duration // the default retry delay for sync clients, protected by mu

	syncClients map[string]*SyncClient // current sync clients, protected by mu

	peerStateManager PeerStateManager
}

func NewSyncManager(configMgr *config.ConfigManager, oplog *oplog.OpLog, orchestrator *orchestrator.Orchestrator, peerStateManager PeerStateManager) *SyncManager {
	// Fetch the config, and mark all sync clients and known hosts as disconnected (but preserve other fields).
	config, err := configMgr.Get()
	if err == nil {
		for _, knownHostPeer := range config.GetMultihost().GetKnownHosts() {
			state := peerStateManager.GetPeerState(knownHostPeer.Keyid).Clone()
			if state == nil {
				state = newPeerState(knownHostPeer.InstanceId, knownHostPeer.Keyid)
			}
			state.ConnectionState = v1sync.ConnectionState_CONNECTION_STATE_DISCONNECTED
			state.ConnectionStateMessage = "disconnected"
			peerStateManager.SetPeerState(knownHostPeer.Keyid, state)
		}
		for _, authorizedClient := range config.GetMultihost().GetAuthorizedClients() {
			state := peerStateManager.GetPeerState(authorizedClient.Keyid).Clone()
			if state == nil {
				state = newPeerState(authorizedClient.InstanceId, authorizedClient.Keyid)
			}
			state.ConnectionState = v1sync.ConnectionState_CONNECTION_STATE_DISCONNECTED
			state.ConnectionStateMessage = "disconnected"
			peerStateManager.SetPeerState(authorizedClient.Keyid, state)
		}
	} else {
		zap.S().Errorf("syncmanager failed to get initial config: %v", err)
	}
	return &SyncManager{
		configMgr:    configMgr,
		orchestrator: orchestrator,
		oplog:        oplog,

		syncClientRetryDelay: 60 * time.Second,
		syncClients:          make(map[string]*SyncClient),

		peerStateManager: peerStateManager,
	}
}

// GetSyncClients returns a copy of the sync clients map. This makes the map safe to read from concurrently.
func (m *SyncManager) GetSyncClients() map[string]*SyncClient {
	m.mu.Lock()
	defer m.mu.Unlock()
	return maps.Clone(m.syncClients)
}

// Note: top level function will be called holding the lock, must kick off goroutines and then return.
func (m *SyncManager) RunSync(ctx context.Context) {
	var syncWg sync.WaitGroup
	var cancelLastSync context.CancelFunc

	configWatchCh := m.configMgr.OnChange.Subscribe()
	defer m.configMgr.OnChange.Unsubscribe(configWatchCh)
	defer func() {
		zap.L().Info("syncmanager exited")
	}()

	runSyncWithNewConfig := func() {
		m.mu.Lock()
		defer m.mu.Unlock()

		if cancelLastSync != nil {
			cancelLastSync()
			zap.L().Info("syncmanager applying new config, waiting for existing sync goroutines to exit")
			syncWg.Wait()
		} else {
			zap.L().Info("syncmanager applying new config, starting sync goroutines")
		}
		syncCtx, cancel := context.WithCancel(ctx)
		cancelLastSync = cancel

		config, err := m.configMgr.Get()
		if err != nil {
			zap.S().Errorf("syncmanager failed to refresh config with latest changes so sync is stopped: %v", err)
			return
		}

		if config.Multihost.GetIdentity() == nil {
			zap.S().Info("syncmanager no identity key configured, sync feature is disabled.")
			m.snapshot = nil // Clear the snapshot to indicate sync is disabled
			return
		}

		// Pull out configuration from the new config and cache it for sync handler e.g. the config and identity key.
		identityKey, err := cryptoutil.NewPrivateKey(config.Multihost.GetIdentity())
		if err != nil {
			zap.S().Warnf("syncmanager failed to load local instance identity key, synchandler will reject requests: %v", err)
			return
		}

		m.snapshot = &syncConfigSnapshot{
			config:      config,
			identityKey: identityKey,
		}

		// Past this point, determine if sync clients are configured and start threads for any.
		if len(config.Multihost.GetKnownHosts()) == 0 {
			zap.L().Info("syncmanager no known host peers declared, sync client exiting early")
			return
		}

		zap.S().Infof("sync using identity %v, spawning goroutines for %d known peers",
			config.Multihost.GetIdentity().GetKeyid(), len(config.Multihost.GetKnownHosts()))
		for _, knownHostPeer := range config.Multihost.KnownHosts {
			if knownHostPeer.InstanceId == "" {
				continue
			}

			syncWg.Add(1)
			go func(knownHostPeer *v1.Multihost_Peer) {
				defer syncWg.Done()
				zap.S().Debugf("syncmanager starting sync goroutine with peer %q", knownHostPeer.InstanceId)
				err := m.runSyncWithPeerInternal(syncCtx, config, knownHostPeer)
				if err != nil {
					zap.S().Errorf("syncmanager error starting client for peer %q: %v", knownHostPeer.InstanceId, err)
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
	} else if config.Multihost == nil {
		return errors.New("multihost config must be set before peersync can be enabled")
	}

	newClient, err := NewSyncClient(m, *m.snapshot, knownHostPeer, m.oplog)
	if err != nil {
		return fmt.Errorf("creating sync client: %w", err)
	}
	m.mu.Lock()
	m.syncClients[knownHostPeer.InstanceId] = newClient
	m.mu.Unlock()

	go func() {
		newClient.RunSync(ctx)
		m.mu.Lock()
		delete(m.syncClients, knownHostPeer.InstanceId)
		m.mu.Unlock()
	}()

	return nil
}

type syncConfigSnapshot struct {
	config      *v1.Config
	identityKey *cryptoutil.PrivateKey // the local instance's identity key, used for signing sync messages
}

func (m *SyncManager) getSyncConfigSnapshot() *syncConfigSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.snapshot == nil {
		return nil
	}

	// defensive copy
	copy := *m.snapshot
	return &copy
}
