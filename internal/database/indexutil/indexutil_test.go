package indexutil

import (
	"fmt"
	"testing"

	"go.etcd.io/bbolt"
)

func TestIndexing(t *testing.T) {
	db, err := bbolt.Open(t.TempDir() + "/test.boltdb", 0600, nil)
	if err != nil {
		t.Fatalf("error opening database: %s", err)
	}

	if err := db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucket([]byte("test"))
		if err != nil {
			return fmt.Errorf("error creating bucket: %s", err)
		}
		for id := 0; id < 100; id += 1 {
			if err := IndexByteValue(b, []byte("document"), int64(id)); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("db.Update error: %v", err)
	}
	
	if err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("test"))
		ids := IndexSearchByteValue(b, []byte("document")).ToSlice()
		if len(ids) != 100 {
			t.Errorf("want 100 ids, got %d", len(ids))
		}
		ids = IndexSearchByteValue(b, []byte("other")).ToSlice()
		if len(ids) != 0 {
			t.Errorf("want 0 ids, got %d", len(ids))
		}
		return nil
	}); err != nil {
		t.Fatalf("db.View error: %v", err)
	}
}
