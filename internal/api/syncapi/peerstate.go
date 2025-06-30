package syncapi

import (
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/eventemitter"
	"github.com/garethgeorge/backrest/internal/kvstore"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"zombiezen.com/go/sqlite/sqlitex"
)

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
	if ps == nil {
		return nil
	}

	clone := &PeerState{}
	*clone = *ps                                 // Shallow copy of the PeerState
	clone.KnownRepos = maps.Clone(ps.KnownRepos) // Clone maps to ensure deep copy
	clone.KnownPlans = maps.Clone(ps.KnownPlans)
	clone.Config = proto.Clone(ps.Config).(*v1.RemoteConfig) // Clone the protobuf Config
	return clone
}

func peerStateToProto(state *PeerState) *v1.PeerState {
	if state == nil {
		return &v1.PeerState{}
	}
	return &v1.PeerState{
		PeerInstanceId:      state.InstanceID,
		PeerKeyid:           state.KeyID,
		LastHeartbeatMillis: state.LastHeartbeat.UnixMilli(),
		State:               state.ConnectionState,
		StatusMessage:       state.ConnectionStateMessage,
		KnownRepos:          slices.Collect(maps.Keys(state.KnownRepos)),
		KnownPlans:          slices.Collect(maps.Keys(state.KnownPlans)),
		RemoteConfig:        state.Config,
	}
}

func peerStateFromProto(state *v1.PeerState) *PeerState {
	if state.PeerInstanceId == "" || state.PeerKeyid == "" {
		return nil
	}
	knownRepos := make(map[string]struct{}, len(state.KnownRepos))
	for _, repo := range state.KnownRepos {
		knownRepos[repo] = struct{}{}
	}
	knownPlans := make(map[string]struct{}, len(state.KnownPlans))
	for _, plan := range state.KnownPlans {
		knownPlans[plan] = struct{}{}
	}

	return &PeerState{
		InstanceID:             state.PeerInstanceId,
		KeyID:                  state.PeerKeyid,
		LastHeartbeat:          time.UnixMilli(state.LastHeartbeatMillis),
		ConnectionState:        state.State,
		ConnectionStateMessage: state.StatusMessage,
		KnownRepos:             knownRepos,
		KnownPlans:             knownPlans,
		Config:                 state.RemoteConfig,
	}
}

type PeerStateManager interface {
	GetPeerState(keyID string) *PeerState
	GetAll() []*PeerState
	SetPeerState(keyID string, state *PeerState)
	OnStateChanged() eventemitter.Receiver[*PeerState]
	Close() error
}

type InMemoryPeerStateManager struct {
	mu             sync.Mutex
	peerStates     map[string]*PeerState // keyID -> PeerState
	onStateChanged eventemitter.BlockingEventEmitter[*PeerState]
}

var _ PeerStateManager = (*InMemoryPeerStateManager)(nil) // Ensure InMemoryPeerStateManager implements PeerStateManager interface

func NewInMemoryPeerStateManager() *InMemoryPeerStateManager {
	return &InMemoryPeerStateManager{
		peerStates: make(map[string]*PeerState),
		onStateChanged: eventemitter.BlockingEventEmitter[*PeerState]{
			EventEmitter: eventemitter.EventEmitter[*PeerState]{
				DefaultCapacity: 10, // Default capacity for the event emitter
			},
		},
	}
}

func (m *InMemoryPeerStateManager) OnStateChanged() eventemitter.Receiver[*PeerState] {
	return &m.onStateChanged
}

func (m *InMemoryPeerStateManager) GetPeerState(keyID string) *PeerState {
	m.mu.Lock()
	defer m.mu.Unlock()
	if state, exists := m.peerStates[keyID]; exists {
		return state.Clone()
	}
	return nil
}

func (m *InMemoryPeerStateManager) GetAll() []*PeerState {
	m.mu.Lock()
	defer m.mu.Unlock()
	states := make([]*PeerState, 0, len(m.peerStates))
	for _, state := range m.peerStates {
		states = append(states, state.Clone())
	}
	return states
}

func (m *InMemoryPeerStateManager) SetPeerState(keyID string, state *PeerState) {
	m.mu.Lock()
	defer m.mu.Unlock()

	copy := state.Clone()
	m.peerStates[keyID] = copy
	m.onStateChanged.Emit(copy)
}

func (m *InMemoryPeerStateManager) Close() error {
	return nil
}

type SqlitePeerStateManager struct {
	mu             sync.Mutex
	onStateChanged eventemitter.BlockingEventEmitter[*PeerState]
	kvstore        kvstore.KvStore
}

func NewSqlitePeerStateManager(dbpool *sqlitex.Pool) (*SqlitePeerStateManager, error) {
	kv, err := kvstore.NewSqliteKVStore(dbpool, "peer_states")
	if err != nil {
		return nil, fmt.Errorf("failed to create kvstore: %v", err)
	}
	m := &SqlitePeerStateManager{
		onStateChanged: eventemitter.BlockingEventEmitter[*PeerState]{
			EventEmitter: eventemitter.EventEmitter[*PeerState]{
				DefaultCapacity: 10, // Default capacity for the event emitter
			},
		},
		kvstore: kv,
	}
	return m, nil
}

func (m *SqlitePeerStateManager) OnStateChanged() eventemitter.Receiver[*PeerState] {
	return &m.onStateChanged
}

func (m *SqlitePeerStateManager) GetPeerState(keyID string) *PeerState {
	m.mu.Lock()
	defer m.mu.Unlock()

	stateBytes, err := m.kvstore.Get(keyID)
	if err != nil {
		zap.S().Warnf("error getting peer state for key %s: %v", keyID, err)
		return nil
	} else if stateBytes == nil {
		return nil
	}

	var stateProto v1.PeerState
	if err := proto.Unmarshal(stateBytes, &stateProto); err != nil {
		zap.S().Warnf("error unmarshalling peer state for key %s: %v", keyID, err)
		return nil
	}

	return peerStateFromProto(&stateProto)
}

func (m *SqlitePeerStateManager) GetAll() []*PeerState {
	m.mu.Lock()
	defer m.mu.Unlock()

	states := make([]*PeerState, 0)
	m.kvstore.ForEach("", func(key string, value []byte) error {
		var stateProto v1.PeerState
		if err := proto.Unmarshal(value, &stateProto); err != nil {
			zap.S().Warnf("error unmarshalling peer state for key %s: %v", key, err)
			return nil // Skip this entry
		}
		state := peerStateFromProto(&stateProto)
		if state != nil {
			states = append(states, state)
		}
		return nil
	})
	return states
}

func (m *SqlitePeerStateManager) SetPeerState(keyID string, state *PeerState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	stateProto := peerStateToProto(state)
	stateBytes, err := proto.Marshal(stateProto)
	if err != nil {
		zap.S().Warnf("error marshalling peer state for key %s: %v", keyID, err)
		return
	}

	if err := m.kvstore.Set(keyID, stateBytes); err != nil {
		zap.S().Warnf("error setting peer state for key %s: %v", keyID, err)
		return
	}
	m.onStateChanged.Emit(state.Clone())
}

func (m *SqlitePeerStateManager) Close() error {
	return nil
}
