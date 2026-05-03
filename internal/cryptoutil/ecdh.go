package cryptoutil

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

// ECDHKeyPair holds an ephemeral ECDH key pair for key exchange.
type ECDHKeyPair struct {
	Private *ecdh.PrivateKey
	Public  *ecdh.PublicKey
}

// GenerateECDHKeyPair generates an ephemeral ECDH P-256 key pair.
func GenerateECDHKeyPair() (*ECDHKeyPair, error) {
	privKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ECDH key: %w", err)
	}
	return &ECDHKeyPair{
		Private: privKey,
		Public:  privKey.PublicKey(),
	}, nil
}

// DeriveSessionKey performs ECDH with the peer's public key and derives an
// AES-256-GCM AEAD using HKDF-SHA256. Both ephemeral public keys are included
// as HKDF salt to bind the derived key to this specific exchange. Authentication
// of the peers is provided by the handshake layer (signature verification) which
// runs over the encrypted channel.
func DeriveSessionKey(localPrivate *ecdh.PrivateKey, peerPublic *ecdh.PublicKey) (cipher.AEAD, error) {
	sharedSecret, err := localPrivate.ECDH(peerPublic)
	if err != nil {
		return nil, fmt.Errorf("ECDH key agreement: %w", err)
	}

	// Sort public keys so both sides produce the same salt regardless of role
	pubA, pubB := localPrivate.PublicKey().Bytes(), peerPublic.Bytes()
	if bytes.Compare(pubA, pubB) > 0 {
		pubA, pubB = pubB, pubA
	}
	salt := append(pubA, pubB...)
	info := []byte("backrest-sync-v2")

	hkdfReader := hkdf.New(sha256.New, sharedSecret, salt, info)
	key := make([]byte, 32)
	if _, err := io.ReadFull(hkdfReader, key); err != nil {
		return nil, fmt.Errorf("HKDF key derivation: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	return gcm, nil
}

// ParseECDHPublicKey parses raw ECDH P-256 public key bytes.
func ParseECDHPublicKey(raw []byte) (*ecdh.PublicKey, error) {
	pub, err := ecdh.P256().NewPublicKey(raw)
	if err != nil {
		return nil, fmt.Errorf("parse ECDH public key: %w", err)
	}
	return pub, nil
}
