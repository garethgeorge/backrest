package tunnel

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"sync"
)

type crypt struct {
	secret []byte
	block  cipher.Block
	once   sync.Once
	err    error
}

func newCrypt(secret []byte) *crypt {
	return &crypt{secret: secret}
}

func (c *crypt) init() {
	c.block, c.err = aes.NewCipher(c.secret)
}

// pkcs5Padding adds PKCS#5 padding to the data
func pkcs5Padding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := make([]byte, padding)
	for i := range padtext {
		padtext[i] = byte(padding)
	}
	return append(data, padtext...)
}

// pkcs5Unpadding removes PKCS#5 padding from the data
func pkcs5Unpadding(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("data is empty")
	}

	padding := int(data[len(data)-1])
	if padding > len(data) || padding == 0 {
		return nil, errors.New("invalid padding")
	}

	// Check that all padding bytes are correct
	for i := len(data) - padding; i < len(data); i++ {
		if data[i] != byte(padding) {
			return nil, errors.New("invalid padding")
		}
	}

	return data[:len(data)-padding], nil
}

func (c *crypt) Encrypt(data []byte) ([]byte, error) {
	c.once.Do(c.init)
	if c.err != nil {
		return nil, c.err
	}

	// Add PKCS#5 padding
	paddedData := pkcs5Padding(data, aes.BlockSize)

	// Create a random IV
	ciphertext := make([]byte, aes.BlockSize+len(paddedData))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	// Encrypt the data
	mode := cipher.NewCBCEncrypter(c.block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], paddedData)

	return ciphertext, nil
}

func (c *crypt) Decrypt(data []byte) ([]byte, error) {
	if len(data) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}

	c.once.Do(c.init)
	if c.err != nil {
		return nil, c.err
	}

	// Extract IV and ciphertext
	iv := data[:aes.BlockSize]
	ciphertext := data[aes.BlockSize:]

	// Check that ciphertext is a multiple of block size
	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, errors.New("ciphertext is not a multiple of the block size")
	}

	// Decrypt the data
	mode := cipher.NewCBCDecrypter(c.block, iv)
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	// Remove PKCS#5 padding
	return pkcs5Unpadding(plaintext)
}
