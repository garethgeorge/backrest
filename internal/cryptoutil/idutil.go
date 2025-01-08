package cryptoutil

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
)

var (
	DefaultIDBits = 256
)

func RandomUint64() (uint64, error) {
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(b), nil
}

func MustRandomUint64() uint64 {
	id, err := RandomUint64()
	if err != nil {
		panic(err)
	}
	return id
}

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
	if len(id)*4 < bits {
		return id
	}
	return id[:bits/4]
}
