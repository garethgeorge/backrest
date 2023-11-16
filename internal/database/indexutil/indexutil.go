package indexutil

import (
	"bytes"

	"github.com/garethgeorge/resticui/internal/database/serializationutil"
	bolt "go.etcd.io/bbolt"
)

// IndexByteValue indexes a value and recordId tuple creating multimap from value to lists of associated recordIds.
func IndexByteValue(b *bolt.Bucket, value []byte, recordId int64) error {
	key := serializationutil.BytesToKey(value)
	key = append(key, serializationutil.Itob(recordId)...)
	return b.Put(key, []byte{})
}

// IndexSearchByteValue searches the index given a value and returns an iterator over the associated recordIds.
func IndexSearchByteValue(b *bolt.Bucket, value []byte) *IndexSearchIterator {
	return newSearchIterator(b, serializationutil.BytesToKey(value))
}

type IndexSearchIterator struct {
	c *bolt.Cursor
	k []byte
	prefix []byte
}

func newSearchIterator(b *bolt.Bucket, prefix []byte) *IndexSearchIterator {
	c := b.Cursor()
	k, _ := c.Seek(prefix)
	return &IndexSearchIterator{
		c: c,
		k: k,
		prefix: prefix,
	}
}

func (i *IndexSearchIterator) Next() (int64, bool) {
	if i.k == nil || !bytes.HasPrefix(i.k, i.prefix) {
		return 0, false
	}
	id := serializationutil.Btoi(i.k[len(i.prefix):])
	i.k, _ = i.c.Next()
	return id, true
}

func (i *IndexSearchIterator) ToSlice() []int64 {
	var ids []int64
	for id, ok := i.Next(); ok; id, ok = i.Next() {
		ids = append(ids, id)
	}
	return ids
}