package syncapi

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
)

var maxSignatureAge = 5 * time.Minute

func runSync(
	ctx context.Context,
	localInstanceID string,
	localKey *cryptoutil.PrivateKey,
	commandStream *bidiSyncCommandStream,
	handler syncSessionHandler,
	knownPeers []*v1.Multihost_Peer, // could be known hosts or authorized clients, doesn't matter. This is used to verify the handshake packet, authorization comes later.
) error {
	// send the initial handshake packet to the peer to establish the connection.
	go func() {
		handshakePacket, err := createHandshakePacket(localInstanceID, localKey)
		if err != nil {
			commandStream.SendErrorAndTerminate(fmt.Errorf("creating handshake packet: %w", err))
			return
		}
		commandStream.Send(handshakePacket)
	}()

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
	} else {
		return NewSyncErrorAuth(fmt.Errorf("peer public key ID %s (instance ID %s) not found in known peers", handshake.GetHandshake().GetPublicKey().GetKeyid(), string(handshake.GetHandshake().GetInstanceId().GetPayload())))
	}

	if err := authorizeHandshakeAsPeer(handshake, peer); err != nil {
		return NewSyncErrorAuth(fmt.Errorf("authorizing handshake as peer: %w", err))
	}

	if err := handler.OnConnectionEstablished(ctx, commandStream, peer); err != nil {
		return err
	}

	for item := range commandStream.ReadChannel() {
		switch item.GetAction().(type) {
		case *v1.SyncStreamItem_Heartbeat:
			if err := handler.HandleHeartbeat(ctx, commandStream, item.GetHeartbeat()); err != nil {
				return fmt.Errorf("handling heartbeat: %w", err)
			}
		case *v1.SyncStreamItem_DiffOperations:
			if err := handler.HandleDiffOperations(ctx, commandStream, item.GetDiffOperations()); err != nil {
				return fmt.Errorf("handling diff operations: %w", err)
			}
		case *v1.SyncStreamItem_SendOperations:
			if err := handler.HandleSendOperations(ctx, commandStream, item.GetSendOperations()); err != nil {
				return fmt.Errorf("handling send operations: %w", err)
			}
		case *v1.SyncStreamItem_SendConfig:
			if err := handler.HandleSendConfig(ctx, commandStream, item.GetSendConfig()); err != nil {
				return fmt.Errorf("handling send config: %w", err)
			}
		case *v1.SyncStreamItem_SetConfig:
			if err := handler.HandleSetConfig(ctx, commandStream, item.GetSetConfig()); err != nil {
				return fmt.Errorf("handling set config: %w", err)
			}
		case *v1.SyncStreamItem_ListResources:
			if err := handler.HandleListResources(ctx, commandStream, item.GetListResources()); err != nil {
				return fmt.Errorf("handling list resources: %w", err)
			}
		case *v1.SyncStreamItem_Throttle:
			if err := handler.HandleThrottle(ctx, commandStream, item.GetThrottle()); err != nil {
				return fmt.Errorf("handling throttle: %w", err)
			}
		case *v1.SyncStreamItem_GetLog:
			if err := handler.HandleGetLog(ctx, commandStream, item.GetGetLog()); err != nil {
				return fmt.Errorf("handling get log: %w", err)
			}
		case *v1.SyncStreamItem_SendLogData:
			if err := handler.HandleSendLogData(ctx, commandStream, item.GetSendLogData()); err != nil {
				return fmt.Errorf("handling send log data: %w", err)
			}
		default:
			return NewSyncErrorProtocol(fmt.Errorf("unknown action type %T in sync stream item", item.GetAction()))
		}
	}
	return nil
}

func tryReceiveWithinDuration(ctx context.Context, receiveChan chan *v1.SyncStreamItem, receiveErrChan chan error, timeout time.Duration) (*v1.SyncStreamItem, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	select {
	case item := <-receiveChan:
		return item, nil
	case err := <-receiveErrChan:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func createSignedMessage(payload []byte, identity *cryptoutil.PrivateKey) (*v1.SignedMessage, error) {
	if len(payload) == 0 {
		return nil, errors.New("payload must not be empty")
	}

	timestampMillis := time.Now().UnixMilli()

	payloadWithTimestamp := make([]byte, 0, len(payload)+8)
	binary.BigEndian.AppendUint64(payloadWithTimestamp, uint64(timestampMillis))
	payloadWithTimestamp = append(payloadWithTimestamp, payload...)

	signature, err := identity.Sign(payloadWithTimestamp)
	if err != nil {
		return nil, fmt.Errorf("signing payload: %w", err)
	}

	return &v1.SignedMessage{
		Payload:         payload,
		Signature:       signature,
		Keyid:           identity.KeyID(),
		TimestampMillis: timestampMillis,
	}, nil
}

func verifySignedMessage(msg *v1.SignedMessage, publicKey *cryptoutil.PublicKey) error {
	if msg == nil {
		return errors.New("signed message must not be nil")
	}
	if len(msg.GetPayload()) == 0 {
		return errors.New("signed message payload must not be empty")
	}
	if len(msg.GetSignature()) == 0 {
		return errors.New("signed message signature must not be empty")
	}
	if len(msg.GetKeyid()) == 0 {
		return errors.New("signed message key ID must not be empty")
	}

	if publicKey.KeyID() != msg.GetKeyid() {
		return fmt.Errorf("public key ID mismatch: expected %s, got %s", publicKey.KeyID(), msg.GetKeyid())
	}

	payloadWithTimestamp := make([]byte, 0, len(msg.GetPayload())+8)
	binary.BigEndian.AppendUint64(payloadWithTimestamp, uint64(msg.GetTimestampMillis()))
	payloadWithTimestamp = append(payloadWithTimestamp, msg.GetPayload()...)

	if err := publicKey.Verify(payloadWithTimestamp, msg.GetSignature()); err != nil {
		return fmt.Errorf("verifying signed message: %w", err)
	}

	if time.Since(time.UnixMilli(msg.GetTimestampMillis())) > maxSignatureAge {
		return fmt.Errorf("signature is too old, max age is %s. Is the clock out of sync?", maxSignatureAge)
	}

	return nil
}

func createHandshakePacket(instanceID string, identity *cryptoutil.PrivateKey) (*v1.SyncStreamItem, error) {
	signedMessage, err := createSignedMessage([]byte(instanceID), identity)
	if err != nil {
		return nil, fmt.Errorf("signing instance ID: %w", err)
	}

	return &v1.SyncStreamItem{
		Action: &v1.SyncStreamItem_Handshake{
			Handshake: &v1.SyncStreamItem_SyncActionHandshake{
				ProtocolVersion: SyncProtocolVersion,
				InstanceId:      signedMessage,
				PublicKey:       identity.PublicKeyProto(),
			},
		},
	}, nil
}

// verifyHandshakePacket verifies that
//   - the signature on the instance ID is valid against the public key provided in the handshake
//   - that the public key's ID is as attested in the handshake packet e.g. matches handshake.PublicKey.Keyid
//
// To authenticate, the caller must then check that the public key is trusted by checking the key ID against a local list.
func verifyHandshakePacket(item *v1.SyncStreamItem) (*cryptoutil.PublicKey, error) {
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
func authorizeHandshakeAsPeer(item *v1.SyncStreamItem, peer *v1.Multihost_Peer) error {
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
			stream.Send(&v1.SyncStreamItem{
				Action: &v1.SyncStreamItem_Heartbeat{
					Heartbeat: &v1.SyncStreamItem_SyncActionHeartbeat{},
				},
			})
		case <-ctx.Done():
			return
		}
	}
}

// syncSessionHandler is a stateful handler for the messages within the context of a sync stream session.
// the handler does not need to be thread safe as it is guaranteed to be called from a single thread.
type syncSessionHandler interface {
	OnConnectionEstablished(ctx context.Context, stream *bidiSyncCommandStream, peer *v1.Multihost_Peer) error
	HandleHeartbeat(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionHeartbeat) error
	HandleDiffOperations(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionDiffOperations) error
	HandleSendOperations(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionSendOperations) error
	HandleSendConfig(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionSendConfig) error
	HandleSetConfig(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionSetConfig) error
	HandleListResources(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionListResources) error
	HandleThrottle(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionThrottle) error
	HandleGetLog(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionGetLog) error
	HandleSendLogData(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionSendLogData) error
}

type unimplementedSyncSessionHandler struct{}

func (h *unimplementedSyncSessionHandler) OnConnectionEstablished(ctx context.Context, stream *bidiSyncCommandStream, peer *v1.Multihost_Peer) error {
	return NewSyncErrorProtocol(fmt.Errorf("OnConnectionEstablished not implemented"))
}

func (h *unimplementedSyncSessionHandler) HandleHeartbeat(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionHeartbeat) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleHeartbeat not implemented"))
}

func (h *unimplementedSyncSessionHandler) HandleDiffOperations(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionDiffOperations) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleDiffOperations not implemented"))
}

func (h *unimplementedSyncSessionHandler) HandleSendOperations(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionSendOperations) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleSendOperations not implemented"))
}

func (h *unimplementedSyncSessionHandler) HandleSendConfig(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionSendConfig) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleSendConfig not implemented"))
}

func (h *unimplementedSyncSessionHandler) HandleSetConfig(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionSetConfig) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleSetConfig not implemented"))
}

func (h *unimplementedSyncSessionHandler) HandleListResources(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionListResources) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleListResources not implemented"))
}

func (h *unimplementedSyncSessionHandler) HandleThrottle(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionThrottle) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleThrottle not implemented"))
}

func (h *unimplementedSyncSessionHandler) HandleGetLog(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionGetLog) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleGetLog not implemented"))
}

func (h *unimplementedSyncSessionHandler) HandleSendLogData(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionSendLogData) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleSendLogData not implemented"))
}
