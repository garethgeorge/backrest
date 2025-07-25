package tunnel

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestNewCrypt(t *testing.T) {
	secret := []byte("test-secret-key-")
	c := newCrypt(secret)

	if c == nil {
		t.Fatal("newCrypt returned nil")
	}

	if !bytes.Equal(c.secret, secret) {
		t.Errorf("expected secret %v, got %v", secret, c.secret)
	}
}

func TestPKCS5Padding(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		blockSize int
		expected  int // expected length after padding
	}{
		{
			name:      "empty data",
			data:      []byte{},
			blockSize: 16,
			expected:  16,
		},
		{
			name:      "data length equals block size",
			data:      make([]byte, 16),
			blockSize: 16,
			expected:  32,
		},
		{
			name:      "data length less than block size",
			data:      []byte("hello"),
			blockSize: 16,
			expected:  16,
		},
		{
			name:      "data length greater than block size",
			data:      make([]byte, 20),
			blockSize: 16,
			expected:  32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			padded := pkcs5Padding(tt.data, tt.blockSize)

			if len(padded) != tt.expected {
				t.Errorf("expected length %d, got %d", tt.expected, len(padded))
			}

			// Check that padding is correct
			paddingLen := tt.blockSize - (len(tt.data) % tt.blockSize)
			for i := len(tt.data); i < len(padded); i++ {
				if padded[i] != byte(paddingLen) {
					t.Errorf("invalid padding byte at position %d: expected %d, got %d", i, paddingLen, padded[i])
				}
			}
		})
	}
}

func TestPKCS5Unpadding(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		expectErr bool
		expected  []byte
	}{
		{
			name:      "valid padding",
			data:      []byte{1, 2, 3, 4, 5, 5, 5, 5, 5},
			expectErr: false,
			expected:  []byte{1, 2, 3, 4},
		},
		{
			name:      "full block padding",
			data:      []byte{16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16},
			expectErr: false,
			expected:  []byte{},
		},
		{
			name:      "empty data",
			data:      []byte{},
			expectErr: true,
			expected:  nil,
		},
		{
			name:      "zero padding",
			data:      []byte{1, 2, 3, 0},
			expectErr: true,
			expected:  nil,
		},
		{
			name:      "invalid padding length",
			data:      []byte{1, 2, 3, 10},
			expectErr: true,
			expected:  nil,
		},
		{
			name:      "inconsistent padding",
			data:      []byte{1, 2, 3, 3, 2},
			expectErr: true,
			expected:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := pkcs5Unpadding(tt.data)

			if tt.expectErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if !bytes.Equal(result, tt.expected) {
					t.Errorf("expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

func TestEncryptDecrypt(t *testing.T) {
	secret := []byte("my-secret-key...")
	c := newCrypt(secret)

	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "empty data",
			data: []byte{},
		},
		{
			name: "short data",
			data: []byte("hello"),
		},
		{
			name: "data equal to block size",
			data: make([]byte, 16),
		},
		{
			name: "data larger than block size",
			data: []byte("this is a longer message that spans multiple blocks"),
		},
		{
			name: "binary data",
			data: []byte{0, 1, 2, 3, 255, 254, 253, 252},
		},
		{
			name: "large data",
			data: make([]byte, 1024),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fill large data with random bytes
			if len(tt.data) == 1024 {
				rand.Read(tt.data)
			}

			// Encrypt
			encrypted, err := c.Encrypt(tt.data)
			if err != nil {
				t.Fatalf("encryption failed: %v", err)
			}

			// Verify encrypted data is different from original (unless original is empty)
			if len(tt.data) > 0 && bytes.Equal(encrypted, tt.data) {
				t.Error("encrypted data is same as original")
			}

			// Verify encrypted data includes IV (at least block size)
			if len(encrypted) < 16 {
				t.Errorf("encrypted data too short: %d bytes", len(encrypted))
			}

			// Decrypt
			decrypted, err := c.Decrypt(encrypted)
			if err != nil {
				t.Fatalf("decryption failed: %v", err)
			}

			// Verify decrypted matches original
			if !bytes.Equal(decrypted, tt.data) {
				t.Errorf("decrypted data doesn't match original.\nOriginal:  %v\nDecrypted: %v", tt.data, decrypted)
			}
		})
	}
}

func TestEncryptionIsRandomized(t *testing.T) {
	secret := []byte("test-secret.....")
	c := newCrypt(secret)
	data := []byte("same data every time")

	// Encrypt the same data multiple times
	encrypted1, err := c.Encrypt(data)
	if err != nil {
		t.Fatalf("encryption 1 failed: %v", err)
	}

	encrypted2, err := c.Encrypt(data)
	if err != nil {
		t.Fatalf("encryption 2 failed: %v", err)
	}

	// Encrypted results should be different due to random IV
	if bytes.Equal(encrypted1, encrypted2) {
		t.Error("encryption is not randomized - same input produced same output")
	}

	// But both should decrypt to the same original data
	decrypted1, err := c.Decrypt(encrypted1)
	if err != nil {
		t.Fatalf("decryption 1 failed: %v", err)
	}

	decrypted2, err := c.Decrypt(encrypted2)
	if err != nil {
		t.Fatalf("decryption 2 failed: %v", err)
	}

	if !bytes.Equal(decrypted1, data) || !bytes.Equal(decrypted2, data) {
		t.Error("decryption failed to recover original data")
	}
}

func TestDecryptInvalidData(t *testing.T) {
	secret := []byte("test-secret.....")
	c := newCrypt(secret)

	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "too short",
			data: []byte("short"),
		},
		{
			name: "not multiple of block size",
			data: make([]byte, 17), // 16 + 1
		},
		{
			name: "corrupted padding",
			data: func() []byte {
				// Create valid encrypted data then corrupt it
				original := []byte("test data")
				encrypted, _ := c.Encrypt(original)
				// Corrupt the last byte (padding)
				encrypted[len(encrypted)-1] = 255
				return encrypted
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.Decrypt(tt.data)
			if err == nil {
				t.Error("expected decryption to fail but it succeeded")
			}
		})
	}
}

func TestDifferentSecretsCannotDecrypt(t *testing.T) {
	data := []byte("secret message")

	c1 := newCrypt([]byte("secret1........."))
	c2 := newCrypt([]byte("secret2........."))

	// Encrypt with first secret
	encrypted, err := c1.Encrypt(data)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	// Try to decrypt with second secret
	_, err = c2.Decrypt(encrypted)
	if err == nil {
		t.Error("decryption with wrong secret should fail")
	}
}

func BenchmarkEncrypt(b *testing.B) {
	secret := []byte("benchmark-secret")
	c := newCrypt(secret)
	data := make([]byte, 1024)
	rand.Read(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Encrypt(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecrypt(b *testing.B) {
	secret := []byte("benchmark-secret")
	c := newCrypt(secret)
	data := make([]byte, 1024)
	rand.Read(data)

	encrypted, err := c.Encrypt(data)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Decrypt(encrypted)
		if err != nil {
			b.Fatal(err)
		}
	}
}
