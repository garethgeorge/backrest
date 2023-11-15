package oplog

import "encoding/binary"

func itob(v int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func btoi(b []byte) int64 {
	return int64(binary.BigEndian.Uint64(b))
}

func stob(v string) []byte {
	var b []byte
	b = append(b, itob(int64(len(v)))...)
	b = append(b, []byte(v)...)
	return b
}
