package syncapi

import (
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/garethgeorge/backrest/gen/go/v1sync"
	"google.golang.org/protobuf/proto"
)

// encryptedStream wraps a syncCommandStreamTrait with AES-256-GCM encryption.
// Outgoing SyncStreamItems are serialized, encrypted, and sent as SyncActionEncrypted.
// Incoming SyncActionEncrypted messages are decrypted and deserialized back to SyncStreamItems.
//
// To avoid nonce reuse (since both sides share the same key), each direction
// uses a different nonce prefix byte: the side with the lexicographically smaller
// ECDH public key uses prefix 0x00 for sending and expects 0x01 for receiving,
// and vice versa.
type encryptedStream struct {
	inner syncCommandStreamTrait
	gcm   cipher.AEAD

	sendPrefix byte
	recvPrefix byte

	sendMu      sync.Mutex
	sendCounter uint64

	recvMu      sync.Mutex
	recvCounter uint64
}

func newEncryptedStream(inner syncCommandStreamTrait, gcm cipher.AEAD, localIsSmaller bool) *encryptedStream {
	var sendPrefix, recvPrefix byte
	if localIsSmaller {
		sendPrefix, recvPrefix = 0x00, 0x01
	} else {
		sendPrefix, recvPrefix = 0x01, 0x00
	}
	return &encryptedStream{
		inner:      inner,
		gcm:        gcm,
		sendPrefix: sendPrefix,
		recvPrefix: recvPrefix,
	}
}

func (s *encryptedStream) Send(item *v1sync.SyncStreamItem) error {
	plaintext, err := proto.Marshal(item)
	if err != nil {
		return fmt.Errorf("marshal for encryption: %w", err)
	}

	s.sendMu.Lock()
	nonce := s.makeNonce(s.sendPrefix, s.sendCounter)
	s.sendCounter++
	s.sendMu.Unlock()

	ciphertext := s.gcm.Seal(nil, nonce, plaintext, nil)

	return s.inner.Send(&v1sync.SyncStreamItem{
		Action: &v1sync.SyncStreamItem_Encrypted{
			Encrypted: &v1sync.SyncStreamItem_SyncActionEncrypted{
				Nonce:      nonce,
				Ciphertext: ciphertext,
			},
		},
	})
}

func (s *encryptedStream) Receive() (*v1sync.SyncStreamItem, error) {
	envelope, err := s.inner.Receive()
	if err != nil {
		return nil, err
	}

	encrypted := envelope.GetEncrypted()
	if encrypted == nil {
		return nil, fmt.Errorf("expected encrypted message, got %T", envelope.GetAction())
	}

	s.recvMu.Lock()
	expectedNonce := s.makeNonce(s.recvPrefix, s.recvCounter)
	s.recvCounter++
	s.recvMu.Unlock()

	if len(encrypted.Nonce) != s.gcm.NonceSize() {
		return nil, fmt.Errorf("invalid nonce size: got %d, want %d", len(encrypted.Nonce), s.gcm.NonceSize())
	}

	// Verify nonce matches expected counter to prevent replay/reorder attacks
	for i := range expectedNonce {
		if expectedNonce[i] != encrypted.Nonce[i] {
			return nil, fmt.Errorf("nonce mismatch: possible replay or reorder attack")
		}
	}

	plaintext, err := s.gcm.Open(nil, encrypted.Nonce, encrypted.Ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt message: %w", err)
	}

	var inner v1sync.SyncStreamItem
	if err := proto.Unmarshal(plaintext, &inner); err != nil {
		return nil, fmt.Errorf("unmarshal decrypted message: %w", err)
	}

	return &inner, nil
}

// makeNonce creates a 12-byte GCM nonce. The first byte is the direction prefix
// (0x00 or 0x01), bytes 1-3 are zero, and bytes 4-11 are the counter in big-endian.
func (s *encryptedStream) makeNonce(prefix byte, counter uint64) []byte {
	nonce := make([]byte, s.gcm.NonceSize()) // 12 bytes for GCM
	nonce[0] = prefix
	binary.BigEndian.PutUint64(nonce[4:], counter)
	return nonce
}
