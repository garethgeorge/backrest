package indexutil

import (
	"bytes"

	"github.com/garethgeorge/backrest/internal/oplog/bboltstore/serializationutil"
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
func IndexSearchByteValue(b *bolt.Bucket, value []byte) IndexIterator {
	return newSearchIterator(b, serializationutil.BytesToKey(value))
}

type IndexIterator interface {
	Next() (int64, bool)
}

type SeekableIndexIterator interface {
	IndexIterator
	Seek(int64) (int64, bool) // seek to the first recordId >= id and return it or return false.
}

type IndexSearchIterator struct {
	c      *bolt.Cursor
	k      []byte
	prefix []byte
}

var _ SeekableIndexIterator = &IndexSearchIterator{}

func newSearchIterator(b *bolt.Bucket, prefix []byte) IndexIterator {
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

func (i *IndexSearchIterator) Seek(id int64) (int64, bool) {
	seekTo := []byte{}
	seekTo = append(seekTo, i.prefix...)
	seekTo = append(seekTo, serializationutil.Itob(id)...)
	k, _ := i.c.Seek(seekTo)
	if k == nil || !bytes.HasPrefix(k, i.prefix) {
		return 0, false
	}
	id, err := serializationutil.Btoi(k[len(i.prefix):])
	if err != nil {
		return 0, false
	}
	return id, true
}

type JoinIterator struct {
	iters     []IndexIterator
	seekables []SeekableIndexIterator
}

func NewJoinIterator(iters ...IndexIterator) *JoinIterator {
	seekables := make([]SeekableIndexIterator, 0, len(iters))
	for _, iter := range iters {
		if seekable, ok := iter.(SeekableIndexIterator); ok {
			seekables = append(seekables, seekable)
		} else {
			seekables = append(seekables, nil)
		}
	}
	return &JoinIterator{
		iters:     iters,
		seekables: seekables,
	}
}

func (j *JoinIterator) Next() (int64, bool) {
	if len(j.iters) == 0 {
		return 0, false
	}

	nexts := make([]int64, len(j.iters))
	for idx, iter := range j.iters {
		id, ok := iter.Next()
		if !ok {
			return 0, false
		}
		nexts[idx] = id
	}

	for {
		var ok bool
		maxIdx := 0
		allSame := true
		for idx, id := range nexts {
			if id > nexts[maxIdx] {
				maxIdx = idx
			}
			if id != nexts[0] {
				allSame = false
			}
		}

		if allSame {
			return nexts[0], true
		}

		for idx, id := range nexts {
			if id == nexts[maxIdx] {
				continue
			}

			if j.seekables[idx] != nil {
				nexts[idx], ok = j.seekables[idx].Seek(nexts[maxIdx])
				if !ok {
					return 0, false
				}
			} else {
				nexts[idx], ok = j.iters[idx].Next()
				if !ok {
					return 0, false
				}
			}
		}
	}
}

type Collector func(IndexIterator) []int64

func CollectAll() Collector {
	return func(iter IndexIterator) []int64 {
		ids := make([]int64, 0, 100)
		for id, ok := iter.Next(); ok; id, ok = iter.Next() {
			ids = append(ids, id)
		}
		return ids
	}
}
