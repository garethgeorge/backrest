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

func btos(b []byte) (string, int64) {
	length := btoi(b[:8])
	return string(b[8:8+length]), 8+length
}

func normalizeSnapshotId(id string) string {
	return id[:8]
}