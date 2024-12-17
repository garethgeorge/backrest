package cryptoutil

import (
	"crypto/rand"
	"encoding/hex"
)

var (
	DefaultIDBits = 256
)

func RandomID(bits int) (string, error) {
	b := make([]byte, bits/8)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func MustRandomID(bits int) string {
	id, err := RandomID(bits)
	if err != nil {
		panic(err)
	}
	return id
}

func TruncateID(id string, bits int) string {
	if len(id)*8 < bits {
		return id
	}
	return id[:bits/8]
}
