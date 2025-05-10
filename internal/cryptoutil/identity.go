package cryptoutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

var (
	curve = elliptic.P256() // ed25519

)

type PublicKey struct {
	publicCryptoKey ecdsa.PublicKey
}

func NewPublicKey(pubkey *v1.PublicKey) (*PublicKey, error) {
	pubKeyBlock, _ := pem.Decode([]byte(pubkey.Ed25519Pub))
	if pubKeyBlock == nil {
		return nil, errors.New("no public key found in pem")
	}

	pkixPubKey, err := x509.ParsePKIXPublicKey(pubKeyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	ecdsaPubKey, ok := pkixPubKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("not an ECDSA public key")
	}

	if derived := deriveKeyId(ecdsaPubKey); derived != pubkey.Keyid {
		return nil, fmt.Errorf("public key id does not match: %s != %s", derived, pubkey.Keyid)
	}

	return &PublicKey{
		publicCryptoKey: *ecdsaPubKey,
	}, nil
}

// VerifySignature verifies the signature of a message
func (pk *PublicKey) Verify(message, sig []byte) error {
	hash := sha256.Sum256(message)
	if !ecdsa.VerifyASN1(&pk.publicCryptoKey, hash[:], sig) {
		return errors.New("signature verification failed")
	}
	return nil
}

type PrivateKey struct {
	*PublicKey
	privateCryptoKey *ecdsa.PrivateKey
}

func NewPrivateKey(privkey *v1.PrivateKey) (*PrivateKey, error) {
	privKeyBlock, _ := pem.Decode([]byte(privkey.Ed25519Priv))
	if privKeyBlock == nil {
		return nil, errors.New("no private key found in pem")
	}

	ecdsaPrivKey, err := x509.ParseECPrivateKey(privKeyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	pubKey, err := NewPublicKey(&v1.PublicKey{
		Keyid:      privkey.Keyid,
		Ed25519Pub: privkey.Ed25519Pub,
	})
	if err != nil {
		return nil, err
	}

	if ecdsaPrivKey.PublicKey.X.Cmp(pubKey.publicCryptoKey.X) != 0 ||
		ecdsaPrivKey.PublicKey.Y.Cmp(pubKey.publicCryptoKey.Y) != 0 {
		return nil, errors.New("private key does not match public key")
	}

	return &PrivateKey{
		privateCryptoKey: ecdsaPrivKey,
		PublicKey:        pubKey,
	}, nil
}

func GeneratePrivateKey() (*v1.PrivateKey, error) {
	privKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, err
	}

	privateKeyBytes, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("marshal private key: %w", err)
	}
	pemPrivateKeyBytes := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE", Bytes: privateKeyBytes})
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("marshal public key: %w", err)
	}
	pemPublicKeyBytes := pem.EncodeToMemory(&pem.Block{Type: "EC PUBLIC", Bytes: publicKeyBytes})

	return &v1.PrivateKey{
		Keyid:       deriveKeyId(&privKey.PublicKey),
		Ed25519Priv: string(pemPrivateKeyBytes),
		Ed25519Pub:  string(pemPublicKeyBytes),
	}, nil
}

// SignMessage signs a message using the private key
func (pk *PrivateKey) Sign(message []byte) ([]byte, error) {
	hash := sha256.Sum256(message)
	sig, err := ecdsa.SignASN1(rand.Reader, pk.privateCryptoKey, hash[:])
	if err != nil {
		return nil, fmt.Errorf("sign message: %w", err)
	}
	return sig, nil
}

func deriveKeyId(key *ecdsa.PublicKey) string {
	shasum := sha256.New()
	shasum.Write(key.X.Bytes())
	shasum.Write(key.Y.Bytes())
	return "ecdsa" + hex.EncodeToString(shasum.Sum(nil))
}

func PrivateKeyToPublicKey(privkey *v1.PrivateKey) *v1.PublicKey {
	return &v1.PublicKey{
		Keyid:      privkey.Keyid,
		Ed25519Pub: privkey.Ed25519Pub,
	}
}
