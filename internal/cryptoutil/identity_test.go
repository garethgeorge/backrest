package cryptoutil

import (
	"testing"
)

func TestGenerateKeypair(t *testing.T) {
	privateKey, err := GeneratePrivateKey()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	if len(privateKey.Ed25519Priv) == 0 {
		t.Fatalf("must populate private key")
	}

	if len(privateKey.Ed25519Pub) == 0 {
		t.Fatalf("must populate public key")
	}
}

func TestLoadKey(t *testing.T) {
	privateKey, err := GeneratePrivateKey()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	_, err = NewPrivateKey(privateKey)
	if err != nil {
		t.Fatalf("failed to load key: %v", err)
	}
}

func TestSignAndVerify(t *testing.T) {
	privateKey, err := GeneratePrivateKey()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	key, err := NewPrivateKey(privateKey)
	if err != nil {
		t.Fatalf("failed to load key: %v", err)
	}

	message := "hello world!"
	signature, err := key.Sign([]byte(message))
	if err != nil {
		t.Fatalf("failed to sign message: %v", err)
	}

	if err := key.Verify([]byte(message), signature); err != nil {
		t.Fatalf("failed to verify message: %v", err)
	}
}
