package kvstore

import (
	"bytes"
	"strings"
	"testing"

	"zombiezen.com/go/sqlite/sqlitex"
)

func newTestDB(t *testing.T) *sqlitex.Pool {
	// Using a named in-memory database "file:test.db?mode=memory&cache=shared"
	// ensures that all connections in the pool share the same database.
	dbpool, err := sqlitex.Open("file:test.db?mode=memory&cache=shared", 0, 10)
	if err != nil {
		t.Fatalf("failed to open memory database: %v", err)
	}
	t.Cleanup(func() {
		if err := dbpool.Close(); err != nil {
			t.Logf("failed to close dbpool: %v", err)
		}
	})
	return dbpool
}

func TestSqliteKvStore(t *testing.T) {
	dbpool := newTestDB(t)
	store, err := NewSqliteKVStore(dbpool, "kv")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("Get non-existent", func(t *testing.T) {
		value, err := store.Get("non-existent")
		if err != nil {
			t.Fatal(err)
		}
		if value != nil {
			t.Errorf("expected nil, got %v", value)
		}
	})

	t.Run("Set and Get", func(t *testing.T) {
		key := "hello"
		value := []byte("world")
		if err := store.Set(key, value); err != nil {
			t.Fatal(err)
		}

		retrieved, err := store.Get(key)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(value, retrieved) {
			t.Errorf("expected %v, got %v", value, retrieved)
		}
	})

	t.Run("Set and Get empty value", func(t *testing.T) {
		key := "empty"
		value := []byte{}
		if err := store.Set(key, value); err != nil {
			t.Fatal(err)
		}

		retrieved, err := store.Get(key)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(value, retrieved) {
			t.Errorf("expected %v, got %v", value, retrieved)
		}
	})

	t.Run("Update value", func(t *testing.T) {
		key := "update"
		value1 := []byte("value1")
		value2 := []byte("value2")

		if err := store.Set(key, value1); err != nil {
			t.Fatal(err)
		}
		if err := store.Set(key, value2); err != nil {
			t.Fatal(err)
		}

		retrieved, err := store.Get(key)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(value2, retrieved) {
			t.Errorf("expected %v, got %v", value2, retrieved)
		}
	})

	t.Run("ForEach", func(t *testing.T) {
		if err := store.Set("prefix1/key1", []byte("value1")); err != nil {
			t.Fatal(err)
		}
		if err := store.Set("prefix1/key2", []byte("value2")); err != nil {
			t.Fatal(err)
		}
		if err := store.Set("prefix2/key3", []byte("value3")); err != nil {
			t.Fatal(err)
		}

		count := 0
		err := store.ForEach("prefix1", func(key string, value []byte) error {
			count++
			if !strings.HasPrefix(key, "prefix1") {
				t.Errorf("unexpected key: %s", key)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if count != 2 {
			t.Errorf("expected 2 keys, got %d", count)
		}
	})

	t.Run("ForEach empty prefix", func(t *testing.T) {
		count := 0
		err := store.ForEach("", func(key string, value []byte) error {
			count++
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if count != 6 {
			t.Errorf("expected 6 keys, got %d", count)
		}
	})

	t.Run("ForEach with wildcard", func(t *testing.T) {
		if err := store.Set("prefix_with%/key1", []byte("value1")); err != nil {
			t.Fatal(err)
		}
		if err := store.Set("prefix_with%/key2", []byte("value2")); err != nil {
			t.Fatal(err)
		}

		count := 0
		err := store.ForEach("prefix_with%", func(key string, value []byte) error {
			count++
			if !strings.HasPrefix(key, "prefix_with%") {
				t.Errorf("unexpected key: %s", key)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if count != 2 {
			t.Errorf("expected 2 keys, got %d", count)
		}
	})
}