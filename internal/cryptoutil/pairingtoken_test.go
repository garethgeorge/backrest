package cryptoutil

import (
	"testing"
)

func TestFormatAndParsePairingToken(t *testing.T) {
	keyID := "ecdsa.abc123"
	secret := "deadbeef1234567890abcdef"
	instanceID := "my-server"

	token := FormatPairingToken(keyID, secret, instanceID)
	want := "ecdsa.abc123:deadbeef1234567890abcdef#my-server"
	if token != want {
		t.Fatalf("FormatPairingToken() = %q, want %q", token, want)
	}

	parsed, err := ParsePairingToken(token)
	if err != nil {
		t.Fatalf("ParsePairingToken() error: %v", err)
	}
	if parsed.KeyID != keyID {
		t.Errorf("KeyID = %q, want %q", parsed.KeyID, keyID)
	}
	if parsed.Secret != secret {
		t.Errorf("Secret = %q, want %q", parsed.Secret, secret)
	}
	if parsed.InstanceID != instanceID {
		t.Errorf("InstanceID = %q, want %q", parsed.InstanceID, instanceID)
	}
}

func TestParsePairingTokenWithColonsInKeyID(t *testing.T) {
	// Key IDs contain base64url which shouldn't have colons, but test robustness
	token := "ecdsa.key:secretvalue#server-1"
	parsed, err := ParsePairingToken(token)
	if err != nil {
		t.Fatalf("ParsePairingToken() error: %v", err)
	}
	if parsed.KeyID != "ecdsa.key" {
		t.Errorf("KeyID = %q, want %q", parsed.KeyID, "ecdsa.key")
	}
	if parsed.Secret != "secretvalue" {
		t.Errorf("Secret = %q, want %q", parsed.Secret, "secretvalue")
	}
	if parsed.InstanceID != "server-1" {
		t.Errorf("InstanceID = %q, want %q", parsed.InstanceID, "server-1")
	}
}

func TestParsePairingTokenWithHashInInstanceID(t *testing.T) {
	// Instance ID with a '#' — we use LastIndex so the last '#' is the delimiter
	token := "ecdsa.key:secret#inst#2"
	parsed, err := ParsePairingToken(token)
	if err != nil {
		t.Fatalf("ParsePairingToken() error: %v", err)
	}
	if parsed.InstanceID != "2" {
		t.Errorf("InstanceID = %q, want %q", parsed.InstanceID, "2")
	}
}

func TestParsePairingTokenErrors(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{"missing hash", "ecdsa.key:secret"},
		{"missing colon", "ecdsa.keysecret#server"},
		{"empty key ID", ":secret#server"},
		{"empty secret", "ecdsa.key:#server"},
		{"empty instance ID", "ecdsa.key:secret#"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParsePairingToken(tc.token)
			if err == nil {
				t.Errorf("ParsePairingToken(%q) should have returned error", tc.token)
			}
		})
	}
}

func TestGeneratePairingSecret(t *testing.T) {
	secret, err := GeneratePairingSecret()
	if err != nil {
		t.Fatalf("GeneratePairingSecret() error: %v", err)
	}
	if len(secret) != pairingSecretBytes*2 { // hex encoding doubles length
		t.Errorf("secret length = %d, want %d", len(secret), pairingSecretBytes*2)
	}

	// Ensure two secrets are different (probabilistic but extremely reliable)
	secret2, _ := GeneratePairingSecret()
	if secret == secret2 {
		t.Error("two generated secrets should not be equal")
	}
}

func TestRoundTrip(t *testing.T) {
	secret, err := GeneratePairingSecret()
	if err != nil {
		t.Fatalf("GeneratePairingSecret() error: %v", err)
	}

	keyID := "ecdsa.test-key-id-1234"
	instanceID := "my-backrest-server"

	token := FormatPairingToken(keyID, secret, instanceID)
	parsed, err := ParsePairingToken(token)
	if err != nil {
		t.Fatalf("ParsePairingToken() error: %v", err)
	}
	if parsed.KeyID != keyID || parsed.Secret != secret || parsed.InstanceID != instanceID {
		t.Errorf("round trip failed: got %+v", parsed)
	}
}
