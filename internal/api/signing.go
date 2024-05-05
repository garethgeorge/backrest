package api

import (
	"crypto"
	"crypto/hmac"
	"crypto/rand"
)

var (
	secret = make([]byte, 32)
)

func init() {
	n, err := rand.Read(secret)
	if n != 32 || err != nil {
		panic("failed to generate secret key")
	}
}

func generateSignature(data []byte) ([]byte, error) {
	h := hmac.New(crypto.SHA256.New, secret)
	if n, err := h.Write(data); n != len(data) || err != nil {
		panic("failed to write data to hmac")
	}
	return h.Sum(nil), nil
}
