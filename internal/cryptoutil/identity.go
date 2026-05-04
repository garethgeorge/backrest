package cryptoutil

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"google.golang.org/protobuf/proto"
)

const keyIDPrefix = "ed25519."

type PublicKey struct {
	proto           *v1.PublicKey
	publicCryptoKey ed25519.PublicKey
}

func NewPublicKey(pubkey *v1.PublicKey) (*PublicKey, error) {
	pubBytes, err := base64.RawStdEncoding.DecodeString(pubkey.Ed25519Pub)
	if err != nil {
		return nil, fmt.Errorf("decode public key: %w", err)
	}
	if len(pubBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid ed25519 public key size: got %d, want %d", len(pubBytes), ed25519.PublicKeySize)
	}

	edPubKey := ed25519.PublicKey(pubBytes)
	if derived := deriveKeyId(edPubKey); derived != pubkey.Keyid {
		return nil, fmt.Errorf("public key_id provided does not match the derived key: %s != %s", derived, pubkey.Keyid)
	}

	return &PublicKey{
		proto:           pubkey,
		publicCryptoKey: edPubKey,
	}, nil
}

func (pk *PublicKey) KeyID() string {
	return pk.proto.Keyid
}

func (pk *PublicKey) PublicKeyProto() *v1.PublicKey {
	return proto.Clone(pk.proto).(*v1.PublicKey)
}

func (pk *PublicKey) Verify(message, sig []byte) error {
	if !ed25519.Verify(pk.publicCryptoKey, message, sig) {
		return errors.New("signature verification failed")
	}
	return nil
}

type PrivateKey struct {
	*PublicKey
	proto            *v1.PrivateKey
	privateCryptoKey ed25519.PrivateKey
}

func NewPrivateKey(privkey *v1.PrivateKey) (*PrivateKey, error) {
	seed, err := base64.RawStdEncoding.DecodeString(privkey.Ed25519Priv)
	if err != nil {
		return nil, fmt.Errorf("decode private key: %w", err)
	}
	if len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("invalid ed25519 private key seed size: got %d, want %d", len(seed), ed25519.SeedSize)
	}
	edPrivKey := ed25519.NewKeyFromSeed(seed)

	pubKey, err := NewPublicKey(&v1.PublicKey{
		Keyid:      privkey.Keyid,
		Ed25519Pub: privkey.Ed25519Pub,
	})
	if err != nil {
		return nil, err
	}

	derivedPub := edPrivKey.Public().(ed25519.PublicKey)
	if !bytes.Equal(derivedPub, pubKey.publicCryptoKey) {
		return nil, errors.New("private key does not match public key")
	}

	return &PrivateKey{
		PublicKey:        pubKey,
		proto:            privkey,
		privateCryptoKey: edPrivKey,
	}, nil
}

func GeneratePrivateKey() (*v1.PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ed25519 key: %w", err)
	}

	return &v1.PrivateKey{
		Keyid:       deriveKeyId(pub),
		Ed25519Priv: base64.RawStdEncoding.EncodeToString(priv.Seed()),
		Ed25519Pub:  base64.RawStdEncoding.EncodeToString(pub),
	}, nil
}

func (pk *PrivateKey) PrivateKeyProto() *v1.PrivateKey {
	return proto.Clone(pk.proto).(*v1.PrivateKey)
}

func (pk *PrivateKey) Sign(message []byte) ([]byte, error) {
	return ed25519.Sign(pk.privateCryptoKey, message), nil
}

func deriveKeyId(key ed25519.PublicKey) string {
	shasum := sha256.Sum256(key)
	return keyIDPrefix + base64.RawURLEncoding.EncodeToString(shasum[:])
}
