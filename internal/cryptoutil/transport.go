package cryptoutil

import (
	"crypto/aes"
	"crypto/cipher"
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

var transportInfo = []byte(fmt.Sprintf("backrest-sync-transport-v%d", TransportProtocolVersion))

func exporterLabelI2R() string {
	return fmt.Sprintf("backrest-sync-session-key/v%d/initiator-to-responder", TransportProtocolVersion)
}

func exporterLabelR2I() string {
	return fmt.Sprintf("backrest-sync-session-key/v%d/responder-to-initiator", TransportProtocolVersion)
}

// transportCiphersuite returns the HPKE ciphersuite used by the sync transport.
// Keep this opinionated and pinned to TransportProtocolVersion: hybrid PQ KEM
// (ML-KEM-1024 + ECDH-P384), HKDF-SHA256, AES-256-GCM.
func transportCiphersuite() (hpke.KEM, hpke.KDF, hpke.AEAD) {
	return hpke.MLKEM1024P384(), hpke.HKDFSHA256(), hpke.AES256GCM()
}

// TransportSession is the result of a successful handshake: a pair of one-way
// AEADs plus a transcript hash for higher-layer identity authentication.
//
// Send is for outbound traffic, Recv for inbound. The two AEADs hold
// independent keys derived from distinct HPKE exporter labels, so callers
// may use any nonce discipline (a counter starting at zero is recommended)
// without risk of cross-direction reuse.
//
// Identity authentication is the responsibility of the caller. A higher
// layer that performs ed25519 (or any other) identity verification should
// have each peer sign Transcript() under its long-term key and exchange the
// signatures over the encrypted channel; verifying that signature is what
// defeats a MITM that completes a separate KEM with each side, since the
// two legs of the MITM produce different transcripts and the legitimate
// peer's signature only commits to its own transcript.
type TransportSession struct {
	Send cipher.AEAD
	Recv cipher.AEAD

	transcript [sha256.Size]byte
}

// Transcript returns a hash that commits to the protocol version, the
// initiator's ephemeral KEM public key, and the encapsulation. Both peers
// compute the identical value. Sign this with your identity key and send
// the signature to the peer to authenticate the channel.
func (s *TransportSession) Transcript() []byte {
	out := make([]byte, len(s.transcript))
	copy(out, s.transcript[:])
	return out
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
	i2r, r2i, err := deriveDirectionalAEADs(recipient.Export)
	if err != nil {
		return nil, err
	}
	// Initiator: Send is initiator-to-responder, Recv is responder-to-initiator.
	return &TransportSession{
		Send:       i2r,
		Recv:       r2i,
		transcript: computeTranscript(r.pub, enc),
	}, nil
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
	i2r, r2i, err := deriveDirectionalAEADs(sender.Export)
	if err != nil {
		return nil, nil, err
	}
	// Responder: Send is responder-to-initiator, Recv is initiator-to-responder.
	return enc, &TransportSession{
		Send:       r2i,
		Recv:       i2r,
		transcript: computeTranscript(peerPubBytes, enc),
	}, nil
}

// deriveDirectionalAEADs exports two independent AES-256-GCM keys from the
// HPKE context — one for each traffic direction — and wraps them in AEADs.
// The caller assigns them to Send/Recv based on its role.
func deriveDirectionalAEADs(exporter func(string, int) ([]byte, error)) (i2r, r2i cipher.AEAD, _ error) {
	i2rKey, err := exporter(exporterLabelI2R(), transportSessionKeyLen)
	if err != nil {
		return nil, nil, fmt.Errorf("export i2r session key: %w", err)
	}
	r2iKey, err := exporter(exporterLabelR2I(), transportSessionKeyLen)
	if err != nil {
		return nil, nil, fmt.Errorf("export r2i session key: %w", err)
	}
	i2rAEAD, err := newSessionAEAD(i2rKey)
	if err != nil {
		return nil, nil, err
	}
	r2iAEAD, err := newSessionAEAD(r2iKey)
	if err != nil {
		return nil, nil, err
	}
	return i2rAEAD, r2iAEAD, nil
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
