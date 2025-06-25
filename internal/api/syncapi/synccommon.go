package syncapi

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
)

var maxSignatureAge = 5 * time.Minute

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
	if handshake.GetPublicKey().GetKeyid() != peer.Keyid {
		return fmt.Errorf("public key ID mismatch: expected %s, got %s", peer.Keyid, handshake.PublicKey.Keyid)
	}
	if string(handshake.GetInstanceId().GetPayload()) != peer.InstanceId {
		return fmt.Errorf("instance ID mismatch: expected %s, got %s", peer.InstanceId, string(handshake.InstanceId.GetPayload()))
	}
	return nil
}
