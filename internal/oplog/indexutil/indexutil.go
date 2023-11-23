package indexutil

import (
	"bytes"
	"sort"

	"github.com/garethgeorge/resticui/internal/oplog/serializationutil"
	bolt "go.etcd.io/bbolt"
)

// IndexByteValue indexes a value and recordId tuple creating multimap from value to lists of associated recordIds.
func IndexByteValue(b *bolt.Bucket, value []byte, recordId int64) error {
	key := serializationutil.BytesToKey(value)
	key = append(key, serializationutil.Itob(recordId)...)
	return b.Put(key, []byte{})
}

func IndexRemoveByteValue(b *bolt.Bucket, value []byte, recordId int64) error {
	key := serializationutil.BytesToKey(value)
	key = append(key, serializationutil.Itob(recordId)...)
	return b.Delete(key)
}

// IndexSearchByteValue searches the index given a value and returns an iterator over the associated recordIds.
func IndexSearchByteValue(b *bolt.Bucket, value []byte) *IndexSearchIterator {
	return newSearchIterator(b, serializationutil.BytesToKey(value))
}

type IndexSearchIterator struct {
	c      *bolt.Cursor
	k      []byte
	prefix []byte
}

func newSearchIterator(b *bolt.Bucket, prefix []byte) *IndexSearchIterator {
	c := b.Cursor()
	k, _ := c.Seek(prefix)
	return &IndexSearchIterator{
		c:      c,
		k:      k,
		prefix: prefix,
	}
}

func (i *IndexSearchIterator) Next() (int64, bool) {
	if i.k == nil || !bytes.HasPrefix(i.k, i.prefix) {
		return 0, false
	}
	id, err := serializationutil.Btoi(i.k[len(i.prefix):])
	if err != nil {
		// this sholud never happen, if it does it indicates database corruption.
		return 0, false
	}
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

type Collector func(*IndexSearchIterator) []int64

func CollectAll() Collector {
	return func(iter *IndexSearchIterator) []int64 {
		return iter.ToSlice()
	}
}

func CollectFirstN(firstN int) Collector {
	return func(iter *IndexSearchIterator) []int64 {
		ids := make([]int64, 0, firstN)
		for id, ok := iter.Next(); ok && len(ids) < firstN; id, ok = iter.Next() {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool {
			return ids[i] < ids[j]
		})
		return ids
	}
}

func CollectLastN(lastN int) Collector {
	return func(iter *IndexSearchIterator) []int64 {
		ids := make([]int64, lastN)
		count := 0
		for id, ok := iter.Next(); ok; id, ok = iter.Next() {
			ids[count%lastN] = id
			count += 1
		}
		if count < lastN {
			return ids[:count]
		}
		return ids
	}
}
