package sftputil

import (
	"os"
	"strings"
	"testing"
)

func TestGenerateKey(t *testing.T) {
	tmpDir := t.TempDir()
	host := "example.com"

	privPEM, pubBytes, keyPath, err := GenerateKey(host, tmpDir)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	if len(privPEM) == 0 {
		t.Error("Private key is empty")
	}
	if len(pubBytes) == 0 {
		t.Error("Public key is empty")
	}
	if !strings.Contains(keyPath, "id_ed25519_example.com_") {
		t.Errorf("Unexpected key path: %s", keyPath)
	}

	// Verify files exist
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Error("Private key file not created")
	}
	if _, err := os.Stat(keyPath + ".pub"); os.IsNotExist(err) {
		t.Error("Public key file not created")
	}

	// Verify content match
	readPriv, _ := os.ReadFile(keyPath)
	if string(readPriv) != string(privPEM) {
		t.Error("Saved private key content mismatch")
	}
}
