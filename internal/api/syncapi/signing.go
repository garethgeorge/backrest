package syncapi

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
)

const maxSignatureAge = 5 * time.Minute

// handshakeBindLabel is the domain-separation prefix for the bytes signed in
// SyncActionHandshake.signature. Bumping it invalidates all old handshake
// signatures, so change it only alongside SyncProtocolVersion.
const handshakeBindLabel = "backrest-sync-handshake/v1\x00"

// computeHandshakeBindInput builds the byte string a peer signs (and the
// other peer recomputes locally) for the post-encryption identity exchange.
// Every field that influences peer authorization is length-prefixed so that
// no two distinct (instance, secret, transcript) tuples can ever produce
// the same input. The transport transcript ties the signature to the
// specific post-quantum KEM exchange of this connection.
func computeHandshakeBindInput(protocolVersion int64, instanceID, pairingSecret string, transcript []byte) []byte {
	h := sha256.New()
	h.Write([]byte(handshakeBindLabel))
	var versionBytes [8]byte
	binary.BigEndian.PutUint64(versionBytes[:], uint64(protocolVersion))
	h.Write(versionBytes[:])
	writeLengthPrefixedBytes(h, []byte(instanceID))
	writeLengthPrefixedBytes(h, []byte(pairingSecret))
	writeLengthPrefixedBytes(h, transcript)
	return h.Sum(nil)
}

func writeLengthPrefixedBytes(h hash.Hash, b []byte) {
	var lenBytes [4]byte
	binary.BigEndian.PutUint32(lenBytes[:], uint32(len(b)))
	h.Write(lenBytes[:])
	h.Write(b)
}

// signHandshake produces an ed25519 signature over the handshake bind input
// for the given fields, under the caller's identity key.
func signHandshake(protocolVersion int64, instanceID, pairingSecret string, transcript []byte, identity *cryptoutil.PrivateKey) ([]byte, error) {
	if len(transcript) == 0 {
		return nil, errors.New("transport transcript must not be empty")
	}
	bindInput := computeHandshakeBindInput(protocolVersion, instanceID, pairingSecret, transcript)
	sig, err := identity.Sign(bindInput)
	if err != nil {
		return nil, fmt.Errorf("signing handshake: %w", err)
	}
	return sig, nil
}

// verifyHandshakeSignature recomputes the bind input from the locally-known
// transcript and verifies the peer's signature against the peer's claimed
// public key. A mismatch can mean: tampering, a MITM whose KEM produced a
// different transcript on this leg, or the peer disagreeing on
// protocol_version / instance_id / pairing_secret.
func verifyHandshakeSignature(protocolVersion int64, instanceID, pairingSecret string, transcript, signature []byte, peerKey *cryptoutil.PublicKey) error {
	if len(transcript) == 0 {
		return errors.New("transport transcript must not be empty")
	}
	if len(signature) == 0 {
		return errors.New("handshake signature must not be empty")
	}
	bindInput := computeHandshakeBindInput(protocolVersion, instanceID, pairingSecret, transcript)
	if err := peerKey.Verify(bindInput, signature); err != nil {
		return fmt.Errorf("handshake signature: %w", err)
	}
	return nil
}

func createSignedMessage(payload []byte, identity *cryptoutil.PrivateKey) (*v1.SignedMessage, error) {
	if len(payload) == 0 {
		return nil, errors.New("payload must not be empty")
	}

	timestampMillis := time.Now().UnixMilli()

	payloadWithTimestamp := make([]byte, 0, len(payload)+8)
	payloadWithTimestamp = binary.BigEndian.AppendUint64(payloadWithTimestamp, uint64(timestampMillis))
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
	payloadWithTimestamp = binary.BigEndian.AppendUint64(payloadWithTimestamp, uint64(msg.GetTimestampMillis()))
	payloadWithTimestamp = append(payloadWithTimestamp, msg.GetPayload()...)

	if err := publicKey.Verify(payloadWithTimestamp, msg.GetSignature()); err != nil {
		return fmt.Errorf("verifying signed message: %w", err)
	}

	if time.Since(time.UnixMilli(msg.GetTimestampMillis())) > maxSignatureAge {
		return fmt.Errorf("signature is too old, max age is %s. Is the clock out of sync?", maxSignatureAge)
	}

	return nil
}
