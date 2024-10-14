package api

import (
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"encoding/binary"
)

var (
	secret = make([]byte, 32)
)

func init() {
	n, err := rand.Read(secret)
	if n != 32 || err != nil {
		panic("failed to generate secret key; is /dev/urandom available?")
	}
}

func sign(data []byte) ([]byte, error) {
	h := hmac.New(crypto.SHA256.New, secret)
	if n, err := h.Write(data); n != len(data) || err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func signInt64(data int64) ([]byte, error) {
	dataBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(dataBytes, uint64(data))
	return sign(dataBytes)
}
