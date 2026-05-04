package syncapi

import (
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/garethgeorge/backrest/gen/go/v1sync"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
	"google.golang.org/protobuf/proto"
)

// encryptedStream wraps a syncCommandStreamTrait with AES-256-GCM encryption
// using the per-direction AEADs derived during the transport handshake.
//
// Each direction has an independent key (initiator-to-responder vs
// responder-to-initiator), so a counter-based nonce starting at zero is
// sufficient: there is no shared (key, nonce) space to collide in.
type encryptedStream struct {
	inner syncCommandStreamTrait
	send  cipher.AEAD
	recv  cipher.AEAD

	sendMu      sync.Mutex
	sendCounter uint64

	recvMu      sync.Mutex
	recvCounter uint64
}

func newEncryptedStream(inner syncCommandStreamTrait, sess *cryptoutil.TransportSession) *encryptedStream {
	return &encryptedStream{
		inner: inner,
		send:  sess.Send,
		recv:  sess.Recv,
	}
}

func (s *encryptedStream) Send(item *v1sync.SyncStreamItem) error {
	plaintext, err := proto.Marshal(item)
	if err != nil {
		return fmt.Errorf("marshal for encryption: %w", err)
	}

	s.sendMu.Lock()
	nonce := makeNonce(s.send.NonceSize(), s.sendCounter)
	s.sendCounter++
	s.sendMu.Unlock()

	ciphertext := s.send.Seal(nil, nonce, plaintext, nil)

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
	expectedNonce := makeNonce(s.recv.NonceSize(), s.recvCounter)
	s.recvCounter++
	s.recvMu.Unlock()

	if len(encrypted.Nonce) != s.recv.NonceSize() {
		return nil, fmt.Errorf("invalid nonce size: got %d, want %d", len(encrypted.Nonce), s.recv.NonceSize())
	}

	// Verify nonce matches expected counter to prevent replay/reorder attacks.
	for i := range expectedNonce {
		if expectedNonce[i] != encrypted.Nonce[i] {
			return nil, fmt.Errorf("nonce mismatch: possible replay or reorder attack")
		}
	}

	plaintext, err := s.recv.Open(nil, encrypted.Nonce, encrypted.Ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt message: %w", err)
	}

	var inner v1sync.SyncStreamItem
	if err := proto.Unmarshal(plaintext, &inner); err != nil {
		return nil, fmt.Errorf("unmarshal decrypted message: %w", err)
	}

	return &inner, nil
}

// makeNonce builds an N-byte AES-GCM nonce by big-endian-encoding the counter
// in the trailing 8 bytes; the leading bytes are zero. The counter never
// repeats within a single session direction, so the nonce never repeats.
func makeNonce(size int, counter uint64) []byte {
	nonce := make([]byte, size)
	binary.BigEndian.PutUint64(nonce[size-8:], counter)
	return nonce
}
