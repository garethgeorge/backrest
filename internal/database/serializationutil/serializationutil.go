package serializationutil

import "encoding/binary"

func Itob(v int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func Btoi(b []byte) int64 {
	return int64(binary.BigEndian.Uint64(b))
}

func Stob(v string) []byte {
	var b []byte
	b = append(b, Itob(int64(len(v)))...)
	b = append(b, []byte(v)...)
	return b
}

func Btos(b []byte) (string, int64) {
	length := Btoi(b[:8])
	return string(b[8:8+length]), 8+length
}

func BytesToKey(b []byte) []byte {
	var key []byte
	key = append(key, Itob(int64(len(b)))...)
	key = append(key, b...)
	return key
}

func NormalizeSnapshotId(id string) string {
	return id[:8]
}