package syncapi

import (
	"maps"
	"sync"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/eventemitter"
	"google.golang.org/protobuf/proto"
)

type PeerStateManager struct {
	mu              sync.Mutex
	peerStates      map[string]*PeerState
	onChangeHandles map[string]*eventemitter.EventEmitter[struct{}]
}

func NewPeerStateManager() *PeerStateManager {
	return &PeerStateManager{
		peerStates:      make(map[string]*PeerState),
		onChangeHandles: make(map[string]*eventemitter.EventEmitter[struct{}]),
	}
}

func (m *PeerStateManager) GetPeerState(instanceID string) *PeerState {
	m.mu.Lock()
	defer m.mu.Unlock()
	if state, exists := m.peerStates[instanceID]; exists {
		return state
	}
	return nil
}

func (m *PeerStateManager) SetPeerState(instanceID string, state *PeerState) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existingState, exists := m.peerStates[instanceID]; exists {
		*existingState = *state // Update existing state
	} else {
		m.peerStates[instanceID] = state.Clone() // Add new state
	}

	if handle, exists := m.onChangeHandles[instanceID]; exists {
		handle.Emit(struct{}{}) // Notify subscribers of the change
	} else {
		m.onChangeHandles[instanceID] = &eventemitter.EventEmitter[struct{}]{}
	}
}

func (m *PeerStateManager) GetOnChangeForPeer(instanceID string) *eventemitter.EventEmitter[struct{}] {
	m.mu.Lock()
	defer m.mu.Unlock()

	if handle, exists := m.onChangeHandles[instanceID]; exists {
		return handle
	}

	handle := &eventemitter.EventEmitter[struct{}]{}
	m.onChangeHandles[instanceID] = handle
	return handle
}

type PeerState struct {
	InstanceID string
	KeyID      string

	LastHeartbeat int64 // Timestamp of the last heartbeat received from this peer

	ConnectionState        v1.SyncConnectionState
	ConnectionStateMessage string

	// Plans and repos available on this peer
	KnownRepos map[string]struct{}
	KnownPlans map[string]struct{}

	// Partial configuration available for this peer
	Config *v1.Config
}

func (ps *PeerState) Clone() *PeerState {
	clone := &PeerState{}
	*clone = *ps                                 // Shallow copy of the PeerState
	clone.KnownRepos = maps.Clone(ps.KnownRepos) // Clone maps to ensure deep copy
	clone.KnownPlans = maps.Clone(ps.KnownPlans)
	clone.Config = proto.Clone(ps.Config).(*v1.Config) // Clone the protobuf Config
	return clone
}
