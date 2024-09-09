package serializationutil

import (
	"encoding/binary"
	"errors"
)

var ErrInvalidLength = errors.New("invalid length")

func Itob(v int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func Btoi(b []byte) (int64, error) {
	if len(b) != 8 {
		return 0, ErrInvalidLength
	}
	return int64(binary.BigEndian.Uint64(b)), nil
}

func Stob(v string) []byte {
	b := make([]byte, 0, len(v)+8)
	b = append(b, Itob(int64(len(v)))...)
	b = append(b, []byte(v)...)
	return b
}

func Btos(b []byte) (string, int64, error) {
	if len(b) < 8 {
		return "", 0, ErrInvalidLength
	}
	length, _ := Btoi(b[:8])
	if int64(len(b)) < 8+length {
		return "", 0, ErrInvalidLength
	}
	return string(b[8 : 8+length]), 8 + length, nil
}

func BytesToKey(b []byte) []byte {
	key := make([]byte, 0, 8+len(b))
	key = append(key, Itob(int64(len(b)))...)
	key = append(key, b...)
	return key
}
