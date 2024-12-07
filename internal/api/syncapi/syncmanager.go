package syncapi

import (
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/orchestrator"
)

type SyncManager struct {
	configMgr      *config.ConfigManager
	orchestrator   *orchestrator.Orchestrator
	oplog          *oplog.OpLog
	configCacheDir string
}

func NewSyncManager(configMgr *config.ConfigManager, oplog *oplog.OpLog, orchestrator *orchestrator.Orchestrator, configCacheDir string) *SyncManager {
	return &SyncManager{
		configMgr:      configMgr,
		orchestrator:   orchestrator,
		oplog:          oplog,
		configCacheDir: configCacheDir,
	}
}

func (m *SyncManager) loadCachedConfig(instanceID string) (*config.Config, error) {
	return m.configMgr.Store.Get(instanceID)
}
