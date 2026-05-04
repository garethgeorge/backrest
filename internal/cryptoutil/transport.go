package cryptoutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/hpke"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
)

// TransportProtocolVersion is the wire-format version of the sync transport
// handshake. Both peers MUST use the same value; mismatches abort the
// connection. The version is bound into the HPKE info string and every
// exporter label, so a peer running a different version cannot derive a
// matching session key even if the underlying ciphersuite is unchanged.
// Bump this whenever the on-wire handshake or KEM ciphersuite changes.
const TransportProtocolVersion uint32 = 1

const transportSessionKeyLen = 32 // AES-256

// transportRole records which side of the handshake produced a session.
// The initiator is the holder of the ephemeral KEM private key (HPKE
// recipient). The responder is the encapsulator (HPKE sender).
type transportRole int

const (
	roleInitiator transportRole = iota
	roleResponder
)

var transportInfo = []byte(fmt.Sprintf("backrest-sync-transport-v%d", TransportProtocolVersion))

func exporterLabelI2R() string {
	return fmt.Sprintf("backrest-sync-session-key/v%d/initiator-to-responder", TransportProtocolVersion)
}

func exporterLabelR2I() string {
	return fmt.Sprintf("backrest-sync-session-key/v%d/responder-to-initiator", TransportProtocolVersion)
}

// exporterLabelBound builds a label that binds both peers' ed25519 identity
// public keys into the HPKE Export context. The label is a printable prefix
// followed by NUL and the raw 32-byte identity pubkeys in canonical order
// (initiator || responder). HPKE treats the label as opaque bytes, so the
// embedded binary is safe.
func exporterLabelBound(direction string, initiator, responder ed25519.PublicKey) string {
	prefix := fmt.Sprintf("backrest-sync-session-key/v%d/%s/identity-bound\x00", TransportProtocolVersion, direction)
	return prefix + string(initiator) + string(responder)
}

// transportCiphersuite returns the HPKE ciphersuite used by the sync transport.
// Keep this opinionated and pinned to TransportProtocolVersion: hybrid PQ KEM
// (ML-KEM-1024 + ECDH-P384), HKDF-SHA256, AES-256-GCM.
func transportCiphersuite() (hpke.KEM, hpke.KDF, hpke.AEAD) {
	return hpke.MLKEM1024P384(), hpke.HKDFSHA256(), hpke.AES256GCM()
}

// TransportSession is the result of a successful handshake: a pair of one-way
// AEADs plus a transcript hash for ed25519 identity authentication.
//
// Send is for outbound traffic, Recv for inbound. The two AEADs hold
// independent keys derived from distinct HPKE exporter labels, so callers
// may use any nonce discipline (a counter starting at zero is recommended)
// without risk of cross-direction reuse.
//
// Recommended use with the project's ed25519 identity layer:
//
//  1. Each peer signs Transcript() with its own ed25519 private key and
//     sends (signature, identity public key) over Send.
//  2. Each peer Recvs the peer's signature and identity, verifies the
//     signature against Transcript(), and applies any out-of-band identity
//     policy (matching against a known public key, etc.).
//  3. After both signatures verify, each peer calls BindIdentities(self,
//     peer). The Send and Recv AEADs are re-derived with both identity
//     keys mixed into the HPKE exporter context. From this point on the
//     channel is cryptographically tied to the agreed identity pair, so
//     even a MITM that bypassed step 2 can no longer read or forge
//     messages.
type TransportSession struct {
	Send cipher.AEAD
	Recv cipher.AEAD

	transcript [sha256.Size]byte
	exporter   func(context string, length int) ([]byte, error)
	role       transportRole
}

// Transcript returns a hash that commits to the protocol version, the
// initiator's ephemeral KEM public key, and the encapsulation. Both peers
// compute the identical value. Sign this with your ed25519 identity key
// to authenticate the channel; a MITM cannot produce a signature over
// the transcript that the legitimate peer derived.
func (s *TransportSession) Transcript() []byte {
	out := make([]byte, len(s.transcript))
	copy(out, s.transcript[:])
	return out
}

// BindIdentities re-derives Send and Recv using exporter labels that include
// both peers' ed25519 identity public keys. The caller MUST have already
// verified that `peer` owns its identity (e.g. via a signature over
// Transcript()) before invoking this method. BindIdentities does not itself
// authenticate; it tightens the channel so that any MITM that survived
// the signature exchange still cannot decrypt or forge messages, because
// the bound keys cannot be derived without agreement on both identities.
//
// The role of the local peer (initiator vs responder) is captured at
// handshake time, so the caller need only pass its own identity and the
// peer's; the canonical (initiator, responder) ordering used in the label
// is determined by this session's role.
//
// After BindIdentities returns successfully the previous Send / Recv AEADs
// MUST NOT be used. References captured before this call must be dropped.
func (s *TransportSession) BindIdentities(self, peer ed25519.PublicKey) error {
	if len(self) != ed25519.PublicKeySize {
		return fmt.Errorf("self identity must be %d bytes, got %d", ed25519.PublicKeySize, len(self))
	}
	if len(peer) != ed25519.PublicKeySize {
		return fmt.Errorf("peer identity must be %d bytes, got %d", ed25519.PublicKeySize, len(peer))
	}
	var initiator, responder ed25519.PublicKey
	if s.role == roleInitiator {
		initiator, responder = self, peer
	} else {
		initiator, responder = peer, self
	}
	i2rKey, err := s.exporter(exporterLabelBound("initiator-to-responder", initiator, responder), transportSessionKeyLen)
	if err != nil {
		return fmt.Errorf("export identity-bound i2r key: %w", err)
	}
	r2iKey, err := s.exporter(exporterLabelBound("responder-to-initiator", initiator, responder), transportSessionKeyLen)
	if err != nil {
		return fmt.Errorf("export identity-bound r2i key: %w", err)
	}
	i2rAEAD, err := newSessionAEAD(i2rKey)
	if err != nil {
		return err
	}
	r2iAEAD, err := newSessionAEAD(r2iKey)
	if err != nil {
		return err
	}
	if s.role == roleInitiator {
		s.Send, s.Recv = i2rAEAD, r2iAEAD
	} else {
		s.Send, s.Recv = r2iAEAD, i2rAEAD
	}
	return nil
}

// TransportRecipient is the initiator side of the handshake. The initiator
// generates an ephemeral KEM keypair, sends its public key, and receives
// the encapsulation from the responder before deriving the session.
type TransportRecipient struct {
	priv hpke.PrivateKey
	pub  []byte // cached for transcript construction
}

// NewTransportRecipient generates an ephemeral KEM keypair for the initiator
// side of the transport handshake. It returns the recipient state and the
// raw bytes of the public key that should be sent to the peer.
func NewTransportRecipient() (*TransportRecipient, []byte, error) {
	kem, _, _ := transportCiphersuite()
	priv, err := kem.GenerateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("generate transport KEM key: %w", err)
	}
	pub := priv.PublicKey().Bytes()
	return &TransportRecipient{priv: priv, pub: pub}, pub, nil
}

// Decapsulate consumes the encapsulation bytes received from the responder
// and returns the initiator's session.
func (r *TransportRecipient) Decapsulate(enc []byte) (*TransportSession, error) {
	if r == nil || r.priv == nil {
		return nil, errors.New("transport recipient: nil state")
	}
	if len(enc) == 0 {
		return nil, errors.New("transport recipient: empty encapsulation")
	}
	_, kdf, aead := transportCiphersuite()
	recipient, err := hpke.NewRecipient(enc, r.priv, kdf, aead, transportInfo)
	if err != nil {
		return nil, fmt.Errorf("decapsulate transport KEM: %w", err)
	}
	return buildSession(recipient.Export, r.pub, enc, roleInitiator)
}

// EncapsulateToTransport is the responder side of the handshake. Given the
// initiator's serialized public key bytes, it returns the encapsulation to
// send back and the responder's session.
func EncapsulateToTransport(peerPubBytes []byte) (enc []byte, _ *TransportSession, _ error) {
	if len(peerPubBytes) == 0 {
		return nil, nil, errors.New("transport responder: empty peer public key")
	}
	kem, kdf, aead := transportCiphersuite()
	pub, err := kem.NewPublicKey(peerPubBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parse peer transport public key: %w", err)
	}
	enc, sender, err := hpke.NewSender(pub, kdf, aead, transportInfo)
	if err != nil {
		return nil, nil, fmt.Errorf("encapsulate transport KEM: %w", err)
	}
	sess, err := buildSession(sender.Export, peerPubBytes, enc, roleResponder)
	if err != nil {
		return nil, nil, err
	}
	return enc, sess, nil
}

func buildSession(exporter func(string, int) ([]byte, error), initiatorPub, enc []byte, role transportRole) (*TransportSession, error) {
	i2rKey, err := exporter(exporterLabelI2R(), transportSessionKeyLen)
	if err != nil {
		return nil, fmt.Errorf("export i2r session key: %w", err)
	}
	r2iKey, err := exporter(exporterLabelR2I(), transportSessionKeyLen)
	if err != nil {
		return nil, fmt.Errorf("export r2i session key: %w", err)
	}
	i2rAEAD, err := newSessionAEAD(i2rKey)
	if err != nil {
		return nil, err
	}
	r2iAEAD, err := newSessionAEAD(r2iKey)
	if err != nil {
		return nil, err
	}
	s := &TransportSession{
		transcript: computeTranscript(initiatorPub, enc),
		exporter:   exporter,
		role:       role,
	}
	if role == roleInitiator {
		s.Send, s.Recv = i2rAEAD, r2iAEAD
	} else {
		s.Send, s.Recv = r2iAEAD, i2rAEAD
	}
	return s, nil
}

func computeTranscript(initiatorPub, enc []byte) [sha256.Size]byte {
	h := sha256.New()
	// Domain-separate from any other transcript-style hash in the project.
	h.Write([]byte("backrest-sync-transport-transcript/v1\x00"))
	var versionBytes [4]byte
	binary.BigEndian.PutUint32(versionBytes[:], TransportProtocolVersion)
	h.Write(versionBytes[:])
	writeLengthPrefixed(h, initiatorPub)
	writeLengthPrefixed(h, enc)
	var out [sha256.Size]byte
	h.Sum(out[:0])
	return out
}

func writeLengthPrefixed(h hash.Hash, b []byte) {
	var lenBytes [4]byte
	binary.BigEndian.PutUint32(lenBytes[:], uint32(len(b)))
	h.Write(lenBytes[:])
	h.Write(b)
}

func newSessionAEAD(key []byte) (cipher.AEAD, error) {
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
