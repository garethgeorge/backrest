package syncapi

import (
	"crypto/rand"
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
)

func newTestIdentity(t *testing.T) *cryptoutil.PrivateKey {
	t.Helper()
	proto, err := cryptoutil.GeneratePrivateKey()
	if err != nil {
		t.Fatalf("generate identity: %v", err)
	}
	priv, err := cryptoutil.NewPrivateKey(proto)
	if err != nil {
		t.Fatalf("load identity: %v", err)
	}
	return priv
}

func freshTranscript(t *testing.T) []byte {
	t.Helper()
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return buf
}

func TestHandshake_RoundTrip(t *testing.T) {
	identity := newTestIdentity(t)
	transcript := freshTranscript(t)

	packet, err := createHandshakePacket("alice", identity, "", transcript)
	if err != nil {
		t.Fatalf("createHandshakePacket: %v", err)
	}
	peerKey, err := verifyHandshakePacket(packet, transcript)
	if err != nil {
		t.Fatalf("verifyHandshakePacket: %v", err)
	}
	if peerKey.KeyID() != identity.KeyID() {
		t.Fatalf("verified key ID mismatch: %s vs %s", peerKey.KeyID(), identity.KeyID())
	}
}

func TestHandshake_TranscriptMismatchFails(t *testing.T) {
	identity := newTestIdentity(t)
	transcriptA := freshTranscript(t)
	transcriptB := freshTranscript(t)

	packet, err := createHandshakePacket("alice", identity, "", transcriptA)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifyHandshakePacket(packet, transcriptB); err == nil {
		t.Fatal("expected handshake to reject mismatched transcript (MITM scenario)")
	}
}

func TestHandshake_TamperedSignatureFails(t *testing.T) {
	identity := newTestIdentity(t)
	transcript := freshTranscript(t)

	packet, err := createHandshakePacket("alice", identity, "", transcript)
	if err != nil {
		t.Fatal(err)
	}
	packet.GetHandshake().Signature[0] ^= 0xff
	if _, err := verifyHandshakePacket(packet, transcript); err == nil {
		t.Fatal("expected tampered signature to fail verification")
	}
}

func TestHandshake_TamperedInstanceIdFails(t *testing.T) {
	identity := newTestIdentity(t)
	transcript := freshTranscript(t)

	packet, err := createHandshakePacket("alice", identity, "", transcript)
	if err != nil {
		t.Fatal(err)
	}
	packet.GetHandshake().InstanceId = "mallory"
	if _, err := verifyHandshakePacket(packet, transcript); err == nil {
		t.Fatal("expected tampered instance ID to fail verification")
	}
}

func TestHandshake_TamperedPairingSecretFails(t *testing.T) {
	identity := newTestIdentity(t)
	transcript := freshTranscript(t)

	packet, err := createHandshakePacket("alice", identity, "secret-1", transcript)
	if err != nil {
		t.Fatal(err)
	}
	packet.GetHandshake().PairingSecret = "secret-2"
	if _, err := verifyHandshakePacket(packet, transcript); err == nil {
		t.Fatal("expected tampered pairing secret to fail verification")
	}
}

func TestHandshake_WrongKeyFails(t *testing.T) {
	signer := newTestIdentity(t)
	imposter := newTestIdentity(t)
	transcript := freshTranscript(t)

	packet, err := createHandshakePacket("alice", signer, "", transcript)
	if err != nil {
		t.Fatal(err)
	}
	// Substitute the imposter's public key (and matching keyid) — the
	// signature was made under the real signer's key, so verification must
	// fail.
	packet.GetHandshake().PublicKey = imposter.PublicKey.PublicKeyProto()
	if _, err := verifyHandshakePacket(packet, transcript); err == nil {
		t.Fatal("expected wrong public key to fail verification")
	}
}

func TestHandshake_AuthorizationByKeyID(t *testing.T) {
	identity := newTestIdentity(t)
	transcript := freshTranscript(t)
	packet, err := createHandshakePacket("alice", identity, "", transcript)
	if err != nil {
		t.Fatal(err)
	}
	peer := &v1.Multihost_Peer{
		InstanceId: "alice",
		Keyid:      identity.KeyID(),
	}
	if err := authorizeHandshakeAsPeer(packet, peer); err != nil {
		t.Fatalf("authorize matching peer: %v", err)
	}
	mismatchedInstance := &v1.Multihost_Peer{InstanceId: "bob", Keyid: identity.KeyID()}
	if err := authorizeHandshakeAsPeer(packet, mismatchedInstance); err == nil {
		t.Fatal("expected instance-ID mismatch to fail authorization")
	}
	mismatchedKey := &v1.Multihost_Peer{InstanceId: "alice", Keyid: "ed25519.bogus"}
	if err := authorizeHandshakeAsPeer(packet, mismatchedKey); err == nil {
		t.Fatal("expected key-ID mismatch to fail authorization")
	}
}
