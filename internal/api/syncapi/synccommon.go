package syncapi

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"
	"unique"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1sync"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
	"github.com/garethgeorge/backrest/internal/oplog"
	lru "github.com/hashicorp/golang-lru/v2"
)

// onUnknownPeerFunc is called when a peer is not found in the known peers list during handshake.
// It receives the handshake item and may return a peer definition to authorize the connection
// (e.g. by validating a pairing token and adding the peer to the config).
// If it returns nil, the connection is rejected.
type onUnknownPeerFunc func(handshake *v1sync.SyncStreamItem) (*v1.Multihost_Peer, error)

func runSync(
	ctx context.Context,
	localInstanceID string,
	localKey *cryptoutil.PrivateKey,
	commandStream *bidiSyncCommandStream,
	handler syncSessionHandler,
	knownPeers []*v1.Multihost_Peer, // could be known hosts or authorized clients, doesn't matter. This is used to verify the handshake packet, authorization comes later.
	pairingSecret string, // optional one-time pairing secret to send during the handshake
	onUnknownPeer onUnknownPeerFunc, // optional callback for handling unknown peers (e.g. pairing), nil to reject all unknown peers
) error {
	// Session-scoped context: cancelled when this runSync invocation returns. Any per-session
	// goroutines the handler spawns (heartbeats, watchers, etc.) should use this ctx so they
	// die with the session rather than outliving it into the next reconnect cycle.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// send the initial handshake packet to the peer to establish the connection.
	handshakePacket, err := createHandshakePacket(localInstanceID, localKey, pairingSecret)
	if err != nil {
		return NewSyncErrorAuth(fmt.Errorf("creating handshake packet: %w", err))
	}
	commandStream.Send(handshakePacket)

	// Wait for the handshake packet to be acknowledged by the peer.
	handshake := commandStream.ReceiveWithinDuration(15 * time.Second)
	if handshake == nil {
		return NewSyncErrorAuth(fmt.Errorf("no handshake packet received from peer within timeout"))
	}
	if _, err := verifyHandshakePacket(handshake); err != nil {
		return NewSyncErrorAuth(fmt.Errorf("verifying handshake packet: %w", err))
	}

	// Find the peer definition in the known peers list.
	var peer *v1.Multihost_Peer
	peerIdx := slices.IndexFunc(knownPeers, func(p *v1.Multihost_Peer) bool {
		return p.Keyid == handshake.GetHandshake().GetPublicKey().GetKeyid()
	})
	if peerIdx >= 0 {
		peer = knownPeers[peerIdx]
	} else if onUnknownPeer != nil {
		// Peer not in known list — try the onUnknownPeer callback (e.g. pairing token validation).
		peer, err = onUnknownPeer(handshake)
		if err != nil {
			return NewSyncErrorAuth(fmt.Errorf("pairing failed: %w", err))
		}
	}
	if peer == nil {
		return NewSyncErrorAuth(fmt.Errorf("peer public key ID %s (instance ID %s) not found in known peers", handshake.GetHandshake().GetPublicKey().GetKeyid(), string(handshake.GetHandshake().GetInstanceId().GetPayload())))
	}

	if err := authorizeHandshakeAsPeer(handshake, peer); err != nil {
		return NewSyncErrorAuth(fmt.Errorf("authorizing handshake as peer: %w", err))
	}

	defer handler.OnConnectionDisconnected()

	if err := handler.OnConnectionEstablished(ctx, commandStream, peer); err != nil {
		return err
	}

	for item := range commandStream.ReadChannel() {
		switch item.GetAction().(type) {
		case *v1sync.SyncStreamItem_Heartbeat:
			if err := handler.HandleHeartbeat(ctx, commandStream, item.GetHeartbeat()); err != nil {
				return fmt.Errorf("handling heartbeat: %w", err)
			}
		case *v1sync.SyncStreamItem_OperationManifest:
			if err := handler.HandleOperationManifest(ctx, commandStream, item.GetOperationManifest()); err != nil {
				return fmt.Errorf("handling operation manifest: %w", err)
			}
		case *v1sync.SyncStreamItem_RequestOperationData:
			if err := handler.HandleRequestOperationData(ctx, commandStream, item.GetRequestOperationData()); err != nil {
				return fmt.Errorf("handling request operation data: %w", err)
			}
		case *v1sync.SyncStreamItem_ReceiveOperations:
			if err := handler.HandleReceiveOperations(ctx, commandStream, item.GetReceiveOperations()); err != nil {
				return fmt.Errorf("handling receive operations: %w", err)
			}
		case *v1sync.SyncStreamItem_ReceiveConfig:
			if err := handler.HandleReceiveConfig(ctx, commandStream, item.GetReceiveConfig()); err != nil {
				return fmt.Errorf("handling receive config: %w", err)
			}
		case *v1sync.SyncStreamItem_SetConfig:
			if err := handler.HandleSetConfig(ctx, commandStream, item.GetSetConfig()); err != nil {
				return fmt.Errorf("handling set config: %w", err)
			}
		case *v1sync.SyncStreamItem_RequestResources:
			if err := handler.HandleRequestResources(ctx, commandStream, item.GetRequestResources()); err != nil {
				return fmt.Errorf("handling request resources: %w", err)
			}
		case *v1sync.SyncStreamItem_ReceiveResources:
			if err := handler.HandleReceiveResources(ctx, commandStream, item.GetReceiveResources()); err != nil {
				return fmt.Errorf("handling receive resources: %w", err)
			}
		case *v1sync.SyncStreamItem_RequestLog:
			if err := handler.HandleRequestLog(ctx, commandStream, item.GetRequestLog()); err != nil {
				return fmt.Errorf("handling request log: %w", err)
			}
		case *v1sync.SyncStreamItem_ReceiveLogData:
			if err := handler.HandleReceiveLogData(ctx, commandStream, item.GetReceiveLogData()); err != nil {
				return fmt.Errorf("handling receive log data: %w", err)
			}
		case *v1sync.SyncStreamItem_Throttle:
			if err := handler.HandleThrottle(ctx, commandStream, item.GetThrottle()); err != nil {
				return fmt.Errorf("handling throttle: %w", err)
			}
		default:
			return NewSyncErrorProtocol(fmt.Errorf("unknown action type %T in sync stream item", item.GetAction()))
		}
	}
	return nil
}

func createHandshakePacket(instanceID string, identity *cryptoutil.PrivateKey, pairingSecret string) (*v1sync.SyncStreamItem, error) {
	signedMessage, err := createSignedMessage([]byte(instanceID), identity)
	if err != nil {
		return nil, fmt.Errorf("signing instance ID: %w", err)
	}

	return &v1sync.SyncStreamItem{
		Action: &v1sync.SyncStreamItem_Handshake{
			Handshake: &v1sync.SyncStreamItem_SyncActionHandshake{
				ProtocolVersion: SyncProtocolVersion,
				InstanceId:      signedMessage,
				PublicKey:       identity.PublicKeyProto(),
				PairingSecret:   pairingSecret,
			},
		},
	}, nil
}

// verifyHandshakePacket verifies that
//   - the signature on the instance ID is valid against the public key provided in the handshake
//   - that the public key's ID is as attested in the handshake packet e.g. matches handshake.PublicKey.Keyid
//
// To authenticate, the caller must then check that the public key is trusted by checking the key ID against a local list.
func verifyHandshakePacket(item *v1sync.SyncStreamItem) (*cryptoutil.PublicKey, error) {
	handshake := item.GetHandshake()
	if handshake == nil {
		return nil, fmt.Errorf("empty or nil handshake, handshake packet must be sent first")
	}

	if handshake.ProtocolVersion != SyncProtocolVersion {
		return nil, fmt.Errorf("protocol version mismatch: expected %d, got %d", SyncProtocolVersion, handshake.ProtocolVersion)
	}

	if len(handshake.InstanceId.GetPayload()) == 0 || len(handshake.InstanceId.GetSignature()) == 0 {
		return nil, errors.New("instance ID payload and signature must not be empty")
	}

	if len(handshake.PublicKey.Keyid) == 0 {
		return nil, errors.New("public key ID must not be empty")
	}

	peerKey, err := cryptoutil.NewPublicKey(handshake.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("loading peer public key: %w", err)
	}

	if err := verifySignedMessage(handshake.InstanceId, peerKey); err != nil {
		return nil, fmt.Errorf("verifying instance ID signature: %w", err)
	}

	return peerKey, nil
}

// authorizeHandshakeAsPeer checks that the handshake packet has the expected key ID and instance ID.
// If this succeeds and the handshake is verified, then it is safe to assume the identity we are talking to.
func authorizeHandshakeAsPeer(item *v1sync.SyncStreamItem, peer *v1.Multihost_Peer) error {
	handshake := item.GetHandshake()
	if handshake == nil {
		return fmt.Errorf("empty or nil handshake, handshake packet must be sent first")
	}
	if string(handshake.GetInstanceId().GetPayload()) != peer.InstanceId {
		return fmt.Errorf("instance ID mismatch: expected %s, got %s", peer.InstanceId, string(handshake.InstanceId.GetPayload()))
	}
	if handshake.GetPublicKey().GetKeyid() != peer.Keyid {
		return fmt.Errorf("public key ID mismatch: expected %s, got %s", peer.Keyid, handshake.PublicKey.Keyid)
	}
	return nil
}

// sendHeartbeats sends a heartbeat message to the stream at regular intervals.
// This is useful for keeping the connection alive and ensuring that the peer is still responsive.
func sendHeartbeats(ctx context.Context, stream *bidiSyncCommandStream, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stream.Send(&v1sync.SyncStreamItem{
				Action: &v1sync.SyncStreamItem_Heartbeat{
					Heartbeat: &v1sync.SyncStreamItem_SyncActionHeartbeat{},
				},
			})
		case <-ctx.Done():
			return
		}
	}
}

// syncSessionHandler is a stateful handler for the messages within the context of a sync stream session.
// the handler does not need to be thread safe as it is guaranteed to be called from a single thread.
//
// The ctx passed to every method is scoped to the session: it is cancelled when runSync returns.
// Goroutines spawned by the handler should use this ctx so they don't leak across reconnect cycles.
type syncSessionHandler interface {
	OnConnectionEstablished(ctx context.Context, stream *bidiSyncCommandStream, peer *v1.Multihost_Peer) error
	OnConnectionDisconnected()
	HandleHeartbeat(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionHeartbeat) error
	HandleOperationManifest(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionOperationManifest) error
	HandleRequestOperationData(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionRequestOperationData) error
	HandleReceiveOperations(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionReceiveOperations) error
	HandleReceiveConfig(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionReceiveConfig) error
	HandleSetConfig(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionSetConfig) error
	HandleRequestResources(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionRequestResources) error
	HandleReceiveResources(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionReceiveResources) error
	HandleRequestLog(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionRequestLog) error
	HandleReceiveLogData(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionReceiveLogData) error
	HandleThrottle(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionThrottle) error
}

type unimplementedSyncSessionHandler struct{}

var _ syncSessionHandler = (*unimplementedSyncSessionHandler)(nil)

func (h *unimplementedSyncSessionHandler) OnConnectionEstablished(ctx context.Context, stream *bidiSyncCommandStream, peer *v1.Multihost_Peer) error {
	return NewSyncErrorProtocol(fmt.Errorf("OnConnectionEstablished not implemented"))
}

func (h *unimplementedSyncSessionHandler) OnConnectionDisconnected() {}

func (h *unimplementedSyncSessionHandler) HandleHeartbeat(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionHeartbeat) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleHeartbeat not implemented"))
}

func (h *unimplementedSyncSessionHandler) HandleOperationManifest(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionOperationManifest) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleOperationManifest not implemented"))
}

func (h *unimplementedSyncSessionHandler) HandleRequestOperationData(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionRequestOperationData) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleRequestOperationData not implemented"))
}

func (h *unimplementedSyncSessionHandler) HandleReceiveOperations(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionReceiveOperations) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleReceiveOperations not implemented"))
}

func (h *unimplementedSyncSessionHandler) HandleReceiveConfig(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionReceiveConfig) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleReceiveConfig not implemented"))
}

func (h *unimplementedSyncSessionHandler) HandleSetConfig(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionSetConfig) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleSetConfig not implemented"))
}

func (h *unimplementedSyncSessionHandler) HandleRequestResources(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionRequestResources) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleRequestResources not implemented"))
}

func (h *unimplementedSyncSessionHandler) HandleReceiveResources(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionReceiveResources) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleReceiveResources not implemented"))
}

func (h *unimplementedSyncSessionHandler) HandleRequestLog(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionRequestLog) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleRequestLog not implemented"))
}

func (h *unimplementedSyncSessionHandler) HandleReceiveLogData(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionReceiveLogData) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleReceiveLogData not implemented"))
}

func (h *unimplementedSyncSessionHandler) HandleThrottle(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionThrottle) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleThrottle not implemented"))
}

type remoteOpIdCacheKey struct {
	OriginalInstanceKeyid unique.Handle[string]
	ID                    int64
}

// operationIdMapper
type remoteOpIDMapper struct {
	oplog *oplog.OpLog

	opCacheMu sync.Mutex
	opIDLru   *lru.Cache[remoteOpIdCacheKey, int64]
	flowIDLru *lru.Cache[remoteOpIdCacheKey, int64]
}

func newRemoteOpIDMapper(oplog *oplog.OpLog, cacheSize int) (*remoteOpIDMapper, error) {
	opIDLru, err := lru.New[remoteOpIdCacheKey, int64](cacheSize)
	if err != nil {
		return nil, fmt.Errorf("creating opID LRU cache: %w", err)
	}
	flowIDLru, err := lru.New[remoteOpIdCacheKey, int64](cacheSize)
	if err != nil {
		return nil, fmt.Errorf("creating flowID LRU cache: %w", err)
	}
	return &remoteOpIDMapper{
		oplog:     oplog,
		opIDLru:   opIDLru,
		flowIDLru: flowIDLru,
	}, nil
}

// translateOpID translates a remote operation ID to a local one.
func (om *remoteOpIDMapper) translateOpID(originalInstanceKeyid string, originalOpId int64) (int64, error) {
	if originalOpId == 0 {
		return 0, nil
	}

	cacheKey := remoteOpIdCacheKey{
		OriginalInstanceKeyid: unique.Make(originalInstanceKeyid),
		ID:                    originalOpId,
	}

	// Check cache first
	if translatedID, ok := om.opIDLru.Get(cacheKey); ok {
		return translatedID, nil
	}

	// Cache miss - query the database. Use QueryMetadata directly to handle
	// the case where duplicates already exist (return the first match).
	var translatedID int64
	err := om.oplog.QueryMetadata(oplog.Query{
		OriginalInstanceKeyid: &originalInstanceKeyid,
		OriginalID:            &originalOpId,
	}, func(op oplog.OpMetadata) error {
		if translatedID == 0 {
			translatedID = op.ID
		}
		return oplog.ErrStopIteration
	})
	if err != nil {
		return 0, err
	}
	if translatedID == 0 {
		return 0, nil // No results means the ID is not found
	}

	// Cache the result and return
	om.opIDLru.Add(cacheKey, translatedID)
	return translatedID, nil
}

// translateFlowID translates a remote flow ID to a local one.
func (om *remoteOpIDMapper) translateFlowID(originalInstanceKeyid string, originalFlowId int64) (int64, error) {
	if originalFlowId == 0 {
		return 0, nil
	}

	cacheKey := remoteOpIdCacheKey{
		OriginalInstanceKeyid: unique.Make(originalInstanceKeyid),
		ID:                    originalFlowId,
	}

	// Check cache first
	if translatedID, ok := om.flowIDLru.Get(cacheKey); ok {
		return translatedID, nil
	}

	// Cache miss - query the database. Use QueryMetadata directly to handle
	// the case where duplicates already exist (return the first match).
	var translatedID int64
	err := om.oplog.QueryMetadata(oplog.Query{
		OriginalInstanceKeyid: &originalInstanceKeyid,
		OriginalFlowID:        &originalFlowId,
	}, func(op oplog.OpMetadata) error {
		if translatedID == 0 {
			translatedID = op.FlowID
		}
		return oplog.ErrStopIteration
	})
	if err != nil {
		return 0, err
	}
	if translatedID == 0 {
		return 0, nil // No results means the ID is not found
	}

	// Cache the result and return
	om.flowIDLru.Add(cacheKey, translatedID)
	return translatedID, nil
}

func (om *remoteOpIDMapper) TranslateOpIdAndFlowID(originalInstanceKeyid string, originalOpId int64, originalFlowId int64) (int64, int64, error) {
	om.opCacheMu.Lock()
	defer om.opCacheMu.Unlock()

	// Translate opID
	opID, err := om.translateOpID(originalInstanceKeyid, originalOpId)
	if err != nil {
		return 0, 0, err
	}

	// Translate flowID
	flowID, err := om.translateFlowID(originalInstanceKeyid, originalFlowId)
	if err != nil {
		return 0, 0, err
	}
	return opID, flowID, nil
}
