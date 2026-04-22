package types

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/orchestrator/tasks"
	"go.uber.org/zap"
)

// SyncLockClient is an interface for acquiring and releasing locks on remote peers.
// This is implemented by syncapi.SyncClient.
type SyncLockClient interface {
	AcquireLock(ctx context.Context, lockKey string) (bool, error)
	ReleaseLock(lockKey string)
	RefreshLock(lockKey string)
}

// SyncLockClientProvider provides SyncLockClients by target instance ID.
type SyncLockClientProvider interface {
	GetSyncLockClient(instanceID string) SyncLockClient
}

var (
	syncLockProviderMu sync.Mutex
	syncLockProvider   SyncLockClientProvider
)

// SetSyncLockClientProvider registers the provider used by the synclock hook handler.
func SetSyncLockClientProvider(provider SyncLockClientProvider) {
	syncLockProviderMu.Lock()
	defer syncLockProviderMu.Unlock()
	syncLockProvider = provider
}

func getSyncLockClientProvider() SyncLockClientProvider {
	syncLockProviderMu.Lock()
	defer syncLockProviderMu.Unlock()
	return syncLockProvider
}

const (
	lockRefreshInterval   = 10 * time.Second
	lockMaxRetryDelay     = 60 * time.Second
	lockInitialRetryDelay = 1 * time.Second
	lockMaxRetries        = 7 // 1s, 2s, 4s, 8s, 16s, 32s, 60s
)

type syncLockHandler struct{}

func (syncLockHandler) Name() string {
	return "synclock"
}

func (syncLockHandler) ActionType() reflect.Type {
	return reflect.TypeOf(&v1.Hook_ActionSyncLock{})
}

func (h syncLockHandler) Execute(ctx context.Context, hook *v1.Hook, vars interface{}, runner tasks.TaskRunner, event v1.Hook_Condition) error {
	lockConfig := hook.GetActionSyncLock()
	if lockConfig == nil {
		return fmt.Errorf("synclock hook missing action config")
	}

	provider := getSyncLockClientProvider()
	if provider == nil {
		zap.L().Warn("synclock: no provider registered, skipping lock operation")
		return nil
	}

	client := provider.GetSyncLockClient(lockConfig.GetTargetInstanceId())
	if client == nil {
		zap.L().Warn("synclock: no client for target instance, skipping lock operation",
			zap.String("targetInstance", lockConfig.GetTargetInstanceId()))
		return nil
	}

	switch event {
	case v1.Hook_CONDITION_ANY_START:
		return h.acquireLock(ctx, client, lockConfig)
	case v1.Hook_CONDITION_ANY_END:
		h.releaseLock(client, lockConfig)
		return nil
	default:
		return nil
	}
}

func (h syncLockHandler) acquireLock(ctx context.Context, client SyncLockClient, config *v1.Hook_SyncLock) error {
	lockKey := config.GetLockKey()
	delay := lockInitialRetryDelay

	for attempt := 0; attempt <= lockMaxRetries; attempt++ {
		acquired, err := client.AcquireLock(ctx, lockKey)
		if err != nil {
			zap.L().Warn("synclock: error acquiring lock, proceeding without lock (best-effort)",
				zap.String("lockKey", lockKey), zap.Error(err))
			return nil // best-effort: proceed without lock
		}
		if acquired {
			zap.L().Info("synclock: acquired lock", zap.String("lockKey", lockKey))
			// Start refresh goroutine
			go h.refreshLoop(ctx, client, lockKey)
			return nil
		}

		if attempt == lockMaxRetries {
			break
		}

		zap.L().Info("synclock: lock not acquired, retrying",
			zap.String("lockKey", lockKey),
			zap.Int("attempt", attempt+1),
			zap.Duration("delay", delay))

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			zap.L().Warn("synclock: context cancelled while waiting for lock, proceeding without lock",
				zap.String("lockKey", lockKey))
			return nil
		}

		delay *= 2
		if delay > lockMaxRetryDelay {
			delay = lockMaxRetryDelay
		}
	}

	zap.L().Warn("synclock: could not acquire lock after retries, proceeding without lock (best-effort)",
		zap.String("lockKey", lockKey))
	return nil // best-effort: default open
}

func (h syncLockHandler) releaseLock(client SyncLockClient, config *v1.Hook_SyncLock) {
	client.ReleaseLock(config.GetLockKey())
	zap.L().Info("synclock: released lock", zap.String("lockKey", config.GetLockKey()))
}

func (h syncLockHandler) refreshLoop(ctx context.Context, client SyncLockClient, lockKey string) {
	ticker := time.NewTicker(lockRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			client.RefreshLock(lockKey)
		case <-ctx.Done():
			return
		}
	}
}

func init() {
	DefaultRegistry().RegisterHandler(&syncLockHandler{})
}
