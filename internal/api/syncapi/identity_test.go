package syncapi

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestCreateAndLoad(t *testing.T) {
	dir := t.TempDir()

	// Create a new identity
	ident, err := NewIdentity("test-instance", filepath.Join(dir, "myidentity.pem"))
	if err != nil {
		t.Fatalf("failed to create identity: %v", err)
	}

	// Load the identity
	loaded, err := NewIdentity("test-instance", filepath.Join(dir, "myidentity.pem"))
	if err != nil {
		t.Fatalf("failed to load identity: %v", err)
	}

	// Verify the identity
	if !ident.privateKey.Equal(loaded.privateKey) {
		t.Fatalf("identities do not match")
	}
}

func TestSignatures(t *testing.T) {
	dir := t.TempDir()

	// Create a new identity
	ident, err := NewIdentity("test-instance", filepath.Join(dir, "myidentity.pem"))
	if err != nil {
		t.Fatalf("failed to create identity: %v", err)
	}

	// Sign a message
	signature, err := ident.SignMessage([]byte("hello world!"))
	if err != nil {
		t.Fatalf("failed to sign message: %v", err)
	}
	fmt.Printf("signed message: %x\n", signature)

	// verify the signature
	if err := ident.VerifySignature([]byte("hello world!"), signature); err != nil {
		t.Fatalf("failed to verify signature: %v", err)
	}
}
