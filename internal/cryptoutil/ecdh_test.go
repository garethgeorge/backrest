package cryptoutil

import (
	"testing"
)

func TestGenerateECDHKeyPair(t *testing.T) {
	kp, err := GenerateECDHKeyPair()
	if err != nil {
		t.Fatalf("GenerateECDHKeyPair: %v", err)
	}
	if kp.Private == nil || kp.Public == nil {
		t.Fatal("key pair has nil fields")
	}
	if len(kp.Public.Bytes()) == 0 {
		t.Fatal("public key bytes are empty")
	}
}

func TestDeriveSessionKey_Symmetric(t *testing.T) {
	alice, err := GenerateECDHKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	bob, err := GenerateECDHKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	gcmAlice, err := DeriveSessionKey(alice.Private, bob.Public)
	if err != nil {
		t.Fatalf("DeriveSessionKey (alice): %v", err)
	}
	gcmBob, err := DeriveSessionKey(bob.Private, alice.Public)
	if err != nil {
		t.Fatalf("DeriveSessionKey (bob): %v", err)
	}

	// Both sides should produce the same key: encrypt with alice, decrypt with bob
	plaintext := []byte("hello backrest")
	nonce := make([]byte, gcmAlice.NonceSize())
	ciphertext := gcmAlice.Seal(nil, nonce, plaintext, nil)

	decrypted, err := gcmBob.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		t.Fatalf("bob failed to decrypt alice's message: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Fatalf("decrypted %q, want %q", decrypted, plaintext)
	}
}

func TestDeriveSessionKey_DifferentPairs(t *testing.T) {
	a, _ := GenerateECDHKeyPair()
	b, _ := GenerateECDHKeyPair()
	c, _ := GenerateECDHKeyPair()

	gcmAB, _ := DeriveSessionKey(a.Private, b.Public)
	gcmAC, _ := DeriveSessionKey(a.Private, c.Public)

	plaintext := []byte("test")
	nonce := make([]byte, gcmAB.NonceSize())
	ciphertext := gcmAB.Seal(nil, nonce, plaintext, nil)

	// AC key should NOT be able to decrypt AB ciphertext
	if _, err := gcmAC.Open(nil, nonce, ciphertext, nil); err == nil {
		t.Fatal("different key pair should not decrypt")
	}
}

func TestParseECDHPublicKey_RoundTrip(t *testing.T) {
	kp, err := GenerateECDHKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	raw := kp.Public.Bytes()
	parsed, err := ParseECDHPublicKey(raw)
	if err != nil {
		t.Fatalf("ParseECDHPublicKey: %v", err)
	}
	if string(parsed.Bytes()) != string(raw) {
		t.Fatal("round-trip failed")
	}
}

func TestParseECDHPublicKey_Invalid(t *testing.T) {
	if _, err := ParseECDHPublicKey([]byte("not a key")); err == nil {
		t.Fatal("expected error for invalid key")
	}
}
