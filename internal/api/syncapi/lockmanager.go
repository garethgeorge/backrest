package syncapi

import (
	"sync"
	"time"
)

const lockExpiry = 30 * time.Second

type lockEntry struct {
	holderID  string
	expiresAt time.Time
}

// LockManager provides an in-memory best-effort lock store for coordinating
// repo access between sync peers. Locks expire after 30 seconds if not refreshed.
type LockManager struct {
	mu    sync.Mutex
	locks map[string]*lockEntry
}

func NewLockManager() *LockManager {
	return &LockManager{
		locks: make(map[string]*lockEntry),
	}
}

// Acquire attempts to acquire a lock for the given key. Returns true if the
// lock was acquired (key was free, expired, or already held by the same holder).
func (lm *LockManager) Acquire(key, holderID string) bool {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	entry, exists := lm.locks[key]
	if !exists || time.Now().After(entry.expiresAt) || entry.holderID == holderID {
		lm.locks[key] = &lockEntry{
			holderID:  holderID,
			expiresAt: time.Now().Add(lockExpiry),
		}
		return true
	}
	return false
}

// Release releases a lock if the holder matches.
func (lm *LockManager) Release(key, holderID string) bool {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	entry, exists := lm.locks[key]
	if exists && entry.holderID == holderID {
		delete(lm.locks, key)
		return true
	}
	return false
}

// Refresh extends the expiry of a lock if the holder matches.
func (lm *LockManager) Refresh(key, holderID string) bool {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	entry, exists := lm.locks[key]
	if exists && entry.holderID == holderID {
		entry.expiresAt = time.Now().Add(lockExpiry)
		return true
	}
	return false
}
