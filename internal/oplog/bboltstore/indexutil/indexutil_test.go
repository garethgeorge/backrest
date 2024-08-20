package indexutil

import (
	"fmt"
	"reflect"
	"testing"

	"go.etcd.io/bbolt"
)

func TestIndexing(t *testing.T) {
	db, err := bbolt.Open(t.TempDir()+"/test.boltdb", 0600, nil)
	if err != nil {
		t.Fatalf("error opening database: %s", err)
	}
	defer db.Close()

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
		ids := CollectAll()(IndexSearchByteValue(b, []byte("document")))
		if len(ids) != 100 {
			t.Errorf("want 100 ids, got %d", len(ids))
		}
		ids = CollectAll()(IndexSearchByteValue(b, []byte("other")))
		if len(ids) != 0 {
			t.Errorf("want 0 ids, got %d", len(ids))
		}
		return nil
	}); err != nil {
		t.Fatalf("db.View error: %v", err)
	}
}

func TestIndexJoin(t *testing.T) {
	// Arrange
	db, err := bbolt.Open(t.TempDir()+"/test.boltdb", 0600, nil)
	if err != nil {
		t.Fatalf("error opening database: %s", err)
	}
	defer db.Close()

	if err := db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucket([]byte("test"))
		if err != nil {
			return fmt.Errorf("error creating bucket: %s", err)
		}
		for id := 0; id < 150; id += 1 {
			if err := IndexByteValue(b, []byte("document"), int64(id)); err != nil {
				return err
			}
		}

		for id := 0; id < 100; id += 2 {
			if err := IndexByteValue(b, []byte("other"), int64(id)); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		t.Fatalf("db.Update error: %v", err)
	}

	if err := db.View(func(tx *bbolt.Tx) error {
		// Act
		b := tx.Bucket([]byte("test"))
		ids := CollectAll()(NewJoinIterator(IndexSearchByteValue(b, []byte("document")), IndexSearchByteValue(b, []byte("other"))))

		// Assert
		if len(ids) != 50 {
			t.Errorf("want 50 ids, got %d", len(ids))
		}

		wantIds := []int64{}
		for id := 0; id < 100; id += 2 {
			wantIds = append(wantIds, int64(id))
		}

		if !reflect.DeepEqual(ids, wantIds) {
			t.Errorf("want %v, got %v", wantIds, ids)
		}

		return nil
	}); err != nil {
		t.Fatalf("db.View error: %v", err)
	}
}
