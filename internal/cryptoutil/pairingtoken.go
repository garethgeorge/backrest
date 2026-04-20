package cryptoutil

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

// PairingToken format: "<keyid>:<secret>#<instanceid>"
// The secret is a random hex string that the client sends during the sync handshake
// to prove it holds a valid pairing token.

const pairingSecretBytes = 32

// GeneratePairingSecret generates a cryptographically random secret for use in a pairing token.
func GeneratePairingSecret() (string, error) {
	b := make([]byte, pairingSecretBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate pairing secret: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// FormatPairingToken formats the components into a pairing token string.
func FormatPairingToken(keyID, secret, instanceID string) string {
	return fmt.Sprintf("%s:%s#%s", keyID, secret, instanceID)
}

// ParsedPairingToken holds the parsed components of a pairing token.
type ParsedPairingToken struct {
	KeyID      string
	Secret     string
	InstanceID string
}

// ParsePairingToken parses a pairing token string into its components.
func ParsePairingToken(token string) (*ParsedPairingToken, error) {
	// Split on '#' to get the instance ID suffix
	hashIdx := strings.LastIndex(token, "#")
	if hashIdx == -1 {
		return nil, fmt.Errorf("invalid pairing token: missing '#' separator")
	}
	instanceID := token[hashIdx+1:]
	remainder := token[:hashIdx]

	// Split remainder on ':' to get key ID and secret
	colonIdx := strings.Index(remainder, ":")
	if colonIdx == -1 {
		return nil, fmt.Errorf("invalid pairing token: missing ':' separator")
	}
	keyID := remainder[:colonIdx]
	secret := remainder[colonIdx+1:]

	if keyID == "" {
		return nil, fmt.Errorf("invalid pairing token: empty key ID")
	}
	if secret == "" {
		return nil, fmt.Errorf("invalid pairing token: empty secret")
	}
	if instanceID == "" {
		return nil, fmt.Errorf("invalid pairing token: empty instance ID")
	}

	return &ParsedPairingToken{
		KeyID:      keyID,
		Secret:     secret,
		InstanceID: instanceID,
	}, nil
}
