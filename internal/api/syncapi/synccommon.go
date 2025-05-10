package syncapi

import (
	"context"
	"errors"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
)

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

func createHandshakePacket(instanceID string, identity *cryptoutil.PrivateKey) (*v1.SyncStreamItem, error) {
	instanceIDBytes := []byte(instanceID)
	instanceIDBytesSignature, err := identity.Sign(instanceIDBytes)
	if err != nil {
		return nil, fmt.Errorf("signing instance ID: %w", err)
	}

	return &v1.SyncStreamItem{
		Action: &v1.SyncStreamItem_Handshake{
			Handshake: &v1.SyncStreamItem_SyncActionHandshake{
				ProtocolVersion: SyncProtocolVersion,
				InstanceId: &v1.SignedMessage{
					Payload:   instanceIDBytes,
					Signature: instanceIDBytesSignature,
					Keyid:     identity.KeyID(),
				},
				PublicKey: identity.PublicKeyProto(),
			},
		},
	}, nil
}

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

	if err := peerKey.Verify(handshake.InstanceId.GetPayload(), handshake.InstanceId.GetSignature()); err != nil {
		return nil, fmt.Errorf("verifying instance ID: %w", err)
	}

	return peerKey, nil
}
