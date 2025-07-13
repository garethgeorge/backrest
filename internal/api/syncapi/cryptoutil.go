package syncapi

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
)

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
