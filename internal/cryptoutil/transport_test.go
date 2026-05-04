package cryptoutil

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func TestTransportHandshake_RoundTrip(t *testing.T) {
	recipient, pubBytes, err := NewTransportRecipient()
	if err != nil {
		t.Fatalf("NewTransportRecipient: %v", err)
	}
	if len(pubBytes) == 0 {
		t.Fatal("public key bytes are empty")
	}

	enc, respSess, err := EncapsulateToTransport(pubBytes)
	if err != nil {
		t.Fatalf("EncapsulateToTransport: %v", err)
	}
	if len(enc) == 0 {
		t.Fatal("encapsulation bytes are empty")
	}

	initSess, err := recipient.Decapsulate(enc)
	if err != nil {
		t.Fatalf("Decapsulate: %v", err)
	}

	// Both peers must derive the same transcript.
	if !bytes.Equal(initSess.Transcript(), respSess.Transcript()) {
		t.Fatal("transcripts differ between initiator and responder")
	}

	plaintext := []byte("hello backrest pq")
	nonce := make([]byte, respSess.Send.NonceSize())

	// responder -> initiator
	ct1 := respSess.Send.Seal(nil, nonce, plaintext, nil)
	got, err := initSess.Recv.Open(nil, nonce, ct1, nil)
	if err != nil {
		t.Fatalf("initiator Recv failed to open responder Send: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("plaintext round-trip mismatch: got %q want %q", got, plaintext)
	}

	// initiator -> responder, reusing the same nonce: must succeed because
	// the two directions hold independent keys.
	ct2 := initSess.Send.Seal(nil, nonce, plaintext, nil)
	got2, err := respSess.Recv.Open(nil, nonce, ct2, nil)
	if err != nil {
		t.Fatalf("responder Recv failed to open initiator Send: %v", err)
	}
	if !bytes.Equal(got2, plaintext) {
		t.Fatalf("reverse plaintext round-trip mismatch: got %q want %q", got2, plaintext)
	}
}

func TestTransportHandshake_PerDirectionKeysAreDistinct(t *testing.T) {
	recipient, pubBytes, err := NewTransportRecipient()
	if err != nil {
		t.Fatal(err)
	}
	enc, respSess, err := EncapsulateToTransport(pubBytes)
	if err != nil {
		t.Fatal(err)
	}
	initSess, err := recipient.Decapsulate(enc)
	if err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("direction-isolation")
	nonce := make([]byte, respSess.Send.NonceSize())

	// A ciphertext from the responder->initiator direction must NOT be
	// openable by the responder's own Recv (which is the initiator->responder
	// key). If keys weren't direction-split this would succeed and silently
	// reuse a (key, nonce) pair.
	ct := respSess.Send.Seal(nil, nonce, plaintext, nil)
	if _, err := respSess.Recv.Open(nil, nonce, ct, nil); err == nil {
		t.Fatal("responder Recv must not open responder Send ciphertext")
	}
	if _, err := initSess.Send.Open(nil, nonce, ct, nil); err == nil {
		t.Fatal("initiator Send must not open responder Send ciphertext")
	}
}

func TestTransportHandshake_DistinctSessions(t *testing.T) {
	r1, pub1, err := NewTransportRecipient()
	if err != nil {
		t.Fatal(err)
	}
	r2, pub2, err := NewTransportRecipient()
	if err != nil {
		t.Fatal(err)
	}

	enc1, sess1, err := EncapsulateToTransport(pub1)
	if err != nil {
		t.Fatal(err)
	}
	enc2, sess2, err := EncapsulateToTransport(pub2)
	if err != nil {
		t.Fatal(err)
	}

	// Each recipient pairs only with its own encapsulation. ML-KEM uses
	// implicit rejection, so Decapsulate may succeed against a foreign enc;
	// the mismatch must be caught by AEAD authentication.
	sessR1, err := r1.Decapsulate(enc1)
	if err != nil {
		t.Fatalf("r1 decapsulate own enc: %v", err)
	}
	sessR2, err := r2.Decapsulate(enc2)
	if err != nil {
		t.Fatalf("r2 decapsulate own enc: %v", err)
	}

	plaintext := []byte("isolation check")
	nonce := make([]byte, sess1.Send.NonceSize())

	ct1 := sess1.Send.Seal(nil, nonce, plaintext, nil)
	if got, err := sessR1.Recv.Open(nil, nonce, ct1, nil); err != nil {
		t.Fatalf("paired session 1 should decrypt: %v", err)
	} else if !bytes.Equal(got, plaintext) {
		t.Fatalf("paired session 1 plaintext mismatch")
	}
	if _, err := sessR2.Recv.Open(nil, nonce, ct1, nil); err == nil {
		t.Fatal("session 2 must not decrypt session 1 ciphertext")
	}

	ct2 := sess2.Send.Seal(nil, nonce, plaintext, nil)
	if got, err := sessR2.Recv.Open(nil, nonce, ct2, nil); err != nil {
		t.Fatalf("paired session 2 should decrypt: %v", err)
	} else if !bytes.Equal(got, plaintext) {
		t.Fatalf("paired session 2 plaintext mismatch")
	}
	if _, err := sessR1.Recv.Open(nil, nonce, ct2, nil); err == nil {
		t.Fatal("session 1 must not decrypt session 2 ciphertext")
	}
}

func TestTransportHandshake_BindIdentities(t *testing.T) {
	initPub, initPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	respPub, respPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	_ = initPriv
	_ = respPriv

	recipient, pubBytes, err := NewTransportRecipient()
	if err != nil {
		t.Fatal(err)
	}
	enc, respSess, err := EncapsulateToTransport(pubBytes)
	if err != nil {
		t.Fatal(err)
	}
	initSess, err := recipient.Decapsulate(enc)
	if err != nil {
		t.Fatal(err)
	}

	if err := initSess.BindIdentities(initPub, respPub); err != nil {
		t.Fatalf("initiator BindIdentities: %v", err)
	}
	if err := respSess.BindIdentities(respPub, initPub); err != nil {
		t.Fatalf("responder BindIdentities: %v", err)
	}

	plaintext := []byte("bound channel")
	nonce := make([]byte, initSess.Send.NonceSize())

	ct := initSess.Send.Seal(nil, nonce, plaintext, nil)
	got, err := respSess.Recv.Open(nil, nonce, ct, nil)
	if err != nil {
		t.Fatalf("bound channel initiator->responder failed: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatal("bound channel plaintext mismatch")
	}

	ct2 := respSess.Send.Seal(nil, nonce, plaintext, nil)
	got2, err := initSess.Recv.Open(nil, nonce, ct2, nil)
	if err != nil {
		t.Fatalf("bound channel responder->initiator failed: %v", err)
	}
	if !bytes.Equal(got2, plaintext) {
		t.Fatal("bound channel reverse plaintext mismatch")
	}
}

func TestTransportHandshake_BindIdentities_MismatchFailsToDecrypt(t *testing.T) {
	initPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	respPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	wrongPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	recipient, pubBytes, err := NewTransportRecipient()
	if err != nil {
		t.Fatal(err)
	}
	enc, respSess, err := EncapsulateToTransport(pubBytes)
	if err != nil {
		t.Fatal(err)
	}
	initSess, err := recipient.Decapsulate(enc)
	if err != nil {
		t.Fatal(err)
	}

	// Initiator binds with the correct pair; responder binds against a
	// different "initiator" identity (e.g. a MITM impersonating). The bound
	// keys must diverge so traffic fails to authenticate.
	if err := initSess.BindIdentities(initPub, respPub); err != nil {
		t.Fatal(err)
	}
	if err := respSess.BindIdentities(respPub, wrongPub); err != nil {
		t.Fatal(err)
	}

	nonce := make([]byte, initSess.Send.NonceSize())
	ct := initSess.Send.Seal(nil, nonce, []byte("should not decrypt"), nil)
	if _, err := respSess.Recv.Open(nil, nonce, ct, nil); err == nil {
		t.Fatal("identity-bound channels with mismatched bindings must not decrypt each other")
	}
}

func TestTransportHandshake_TranscriptCommitsToHandshake(t *testing.T) {
	r1, pub1, err := NewTransportRecipient()
	if err != nil {
		t.Fatal(err)
	}
	enc1, sess1, err := EncapsulateToTransport(pub1)
	if err != nil {
		t.Fatal(err)
	}
	sessR1, err := r1.Decapsulate(enc1)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sess1.Transcript(), sessR1.Transcript()) {
		t.Fatal("paired sessions must agree on transcript")
	}

	// A second handshake must produce a different transcript even though
	// the protocol version and ciphersuite are unchanged.
	_, pub2, err := NewTransportRecipient()
	if err != nil {
		t.Fatal(err)
	}
	_, sess2, err := EncapsulateToTransport(pub2)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(sess1.Transcript(), sess2.Transcript()) {
		t.Fatal("distinct handshakes must produce distinct transcripts")
	}
}

func TestEncapsulateToTransport_RejectsBadInput(t *testing.T) {
	if _, _, err := EncapsulateToTransport(nil); err == nil {
		t.Fatal("expected error for empty peer public key")
	}
	if _, _, err := EncapsulateToTransport([]byte{0x01, 0x02, 0x03}); err == nil {
		t.Fatal("expected error for malformed peer public key")
	}
}

func TestTransportRecipient_Decapsulate_BadInput(t *testing.T) {
	recipient, _, err := NewTransportRecipient()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := recipient.Decapsulate(nil); err == nil {
		t.Fatal("expected error for empty encapsulation")
	}
	if _, err := recipient.Decapsulate([]byte{0xde, 0xad}); err == nil {
		t.Fatal("expected error for malformed encapsulation")
	}
}

func TestTransportSession_BindIdentities_RejectsBadKeyLengths(t *testing.T) {
	recipient, pubBytes, err := NewTransportRecipient()
	if err != nil {
		t.Fatal(err)
	}
	enc, _, err := EncapsulateToTransport(pubBytes)
	if err != nil {
		t.Fatal(err)
	}
	sess, err := recipient.Decapsulate(enc)
	if err != nil {
		t.Fatal(err)
	}
	good, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := sess.BindIdentities(make([]byte, 16), good); err == nil {
		t.Fatal("expected error for short self key")
	}
	if err := sess.BindIdentities(good, make([]byte, 16)); err == nil {
		t.Fatal("expected error for short peer key")
	}
}
