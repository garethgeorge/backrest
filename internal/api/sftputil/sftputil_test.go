package sftputil

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateKey_ReuseAndSanitization(t *testing.T) {
	// Create a temporary directory for SSH keys
	tempDir, err := os.MkdirTemp("", "sftp_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	host := "example.com"

	// First call: should generate new keys
	priv1, pub1, path1, err := GenerateKey(host, tempDir)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}
	if priv1 == nil || pub1 == nil {
		t.Fatal("GenerateKey returned nil keys")
	}

	// Verify file existence
	expectedFilename := "id_ed25519_example.com"
	if filepath.Base(path1) != expectedFilename {
		t.Errorf("expected filename %s, got %s", expectedFilename, filepath.Base(path1))
	}
	if _, err := os.Stat(path1); os.IsNotExist(err) {
		t.Errorf("private key file does not exist at %s", path1)
	}
	if _, err := os.Stat(path1 + ".pub"); os.IsNotExist(err) {
		t.Errorf("public key file does not exist at %s.pub", path1)
	}

	// Second call: should reuse existing keys
	priv2, pub2, path2, err := GenerateKey(host, tempDir)
	if err != nil {
		t.Fatalf("GenerateKey (2nd call) failed: %v", err)
	}

	// Verify keys are identical
	if !bytes.Equal(priv1, priv2) {
		t.Error("GenerateKey did not reuse the private key")
	}
	if !bytes.Equal(pub1, pub2) {
		t.Error("GenerateKey did not reuse the public key")
	}
	if path1 != path2 {
		t.Errorf("GenerateKey returned different paths: %s vs %s", path1, path2)
	}
}

func TestGenerateKey_Sanitization(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "sftp_retry_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Host with special characters
	unsafeHost := "bad/host:name!@#"
	_, _, keyPath, err := GenerateKey(unsafeHost, tempDir)
	if err != nil {
		t.Fatalf("GenerateKey failed for unsafe host: %v", err)
	}

	// Expected sanitization: bad_host_name___
	// characters allowed: a-z, A-Z, 0-9, ., -
	// '/' -> '_'
	// ':' -> '_'
	// '!' -> '_'
	// '@' -> '_'
	// '#' -> '_'
	expectedFilename := "id_ed25519_bad_host_name___"
	if filepath.Base(keyPath) != expectedFilename {
		t.Errorf("Sanitization failed. Expected %s, got %s", expectedFilename, filepath.Base(keyPath))
	}
}
