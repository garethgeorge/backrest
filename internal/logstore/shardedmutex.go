package logstore

import (
	"hash/fnv"
	"sync"
)

type shardedRWMutex struct {
	mu []sync.RWMutex
}

func newShardedRWMutex(n int) shardedRWMutex {
	mu := make([]sync.RWMutex, n)
	return shardedRWMutex{
		mu: mu,
	}
}

func (sm *shardedRWMutex) Lock(key string) {
	idx := hash(key) % uint32(len(sm.mu))
	sm.mu[idx].Lock()
}

func (sm *shardedRWMutex) Unlock(key string) {
	idx := hash(key) % uint32(len(sm.mu))
	sm.mu[idx].Unlock()
}

func (sm *shardedRWMutex) RLock(key string) {
	idx := hash(key) % uint32(len(sm.mu))
	sm.mu[idx].RLock()
}

func (sm *shardedRWMutex) RUnlock(key string) {
	idx := hash(key) % uint32(len(sm.mu))
	sm.mu[idx].RUnlock()
}

func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}
