package syncapi

import (
	"maps"
	"sync"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/eventemitter"
	"google.golang.org/protobuf/proto"
)

type PeerStateManager struct {
	mu             sync.Mutex
	peerStates     map[string]*PeerState // keyID -> PeerState
	onStateChanged eventemitter.BlockingEventEmitter[*PeerState]
}

func newPeerStateManager() *PeerStateManager {
	return &PeerStateManager{
		peerStates: make(map[string]*PeerState),
		onStateChanged: eventemitter.BlockingEventEmitter[*PeerState]{
			EventEmitter: eventemitter.EventEmitter[*PeerState]{
				DefaultCapacity: 10, // Default capacity for the event emitter
			},
		},
	}
}

func (m *PeerStateManager) GetPeerState(keyID string) *PeerState {
	m.mu.Lock()
	defer m.mu.Unlock()
	if state, exists := m.peerStates[keyID]; exists {
		return state
	}
	return nil
}

func (m *PeerStateManager) SetPeerState(keyID string, state *PeerState) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existingState, exists := m.peerStates[keyID]; exists {
		*existingState = *state // Update existing state
	} else {
		m.peerStates[keyID] = state.Clone() // Add new state
	}
	m.onStateChanged.Emit(m.peerStates[keyID])
}

func (m *PeerStateManager) ResetStates() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.peerStates = make(map[string]*PeerState)
}

func (m *PeerStateManager) GetAllPeerStates() []*PeerState {
	m.mu.Lock()
	defer m.mu.Unlock()
	states := make([]*PeerState, 0, len(m.peerStates))
	for _, state := range m.peerStates {
		states = append(states, state)
	}
	return states
}

type PeerState struct {
	InstanceID string
	KeyID      string

	LastHeartbeat time.Time

	ConnectionState        v1.SyncConnectionState
	ConnectionStateMessage string

	// Plans and repos available on this peer
	KnownRepos map[string]struct{}
	KnownPlans map[string]struct{}

	// Partial configuration available for this peer
	Config *v1.RemoteConfig

	manager *PeerStateManager
}

func newPeerState(instanceID, keyID string) *PeerState {
	return &PeerState{
		InstanceID:             instanceID,
		KeyID:                  keyID,
		LastHeartbeat:          time.Now(),
		ConnectionState:        v1.SyncConnectionState_CONNECTION_STATE_DISCONNECTED,
		ConnectionStateMessage: "disconnected",
		KnownRepos:             make(map[string]struct{}),
		KnownPlans:             make(map[string]struct{}),
		Config:                 nil, // Will be set when the config is received
	}
}

func (ps *PeerState) Clone() *PeerState {
	clone := &PeerState{}
	*clone = *ps                                 // Shallow copy of the PeerState
	clone.KnownRepos = maps.Clone(ps.KnownRepos) // Clone maps to ensure deep copy
	clone.KnownPlans = maps.Clone(ps.KnownPlans)
	clone.Config = proto.Clone(ps.Config).(*v1.RemoteConfig) // Clone the protobuf Config
	return clone
}
