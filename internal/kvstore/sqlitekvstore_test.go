package kvstore

import (
	"bytes"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
)

func newTestDB(t testing.TB) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", memdb.TestDB(t))
	if err != nil {
		t.Fatalf("failed to open memory database: %v", err)
	}
	return db
}

func TestSqliteKvStore(t *testing.T) {
	dbpool := newTestDB(t)
	store, err := NewSqliteKVStore(dbpool, "kv")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("Get non-existent", func(t *testing.T) {
		value, err := store.Get("non-existent")
		if err != ErrNotExist {
			t.Errorf("expected ErrNotExist, got %v", err)
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

func BenchmarkSqliteKvStore_BulkInsert(b *testing.B) {
	dbpool := newTestDB(b)
	store, err := NewSqliteKVStore(dbpool, "benchmark_insert")
	if err != nil {
		b.Fatal(err)
	}

	// Pre-generate test data
	keys := make([]string, b.N)
	values := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		keys[i] = fmt.Sprintf("key-%d", i)
		values[i] = []byte(fmt.Sprintf("value-data-for-key-%d-with-some-longer-content-to-simulate-realistic-data", i))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if err := store.Set(keys[i], values[i]); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSqliteKvStore_BulkRetrieve(b *testing.B) {
	dbpool := newTestDB(b)
	store, err := NewSqliteKVStore(dbpool, "benchmark_retrieve")
	if err != nil {
		b.Fatal(err)
	}

	// Pre-populate the store with test data
	numKeys := 10000
	keys := make([]string, numKeys)
	expectedValues := make([][]byte, numKeys)
	for i := 0; i < numKeys; i++ {
		keys[i] = fmt.Sprintf("key-%d", i)
		expectedValues[i] = []byte(fmt.Sprintf("value-data-for-key-%d-with-some-longer-content-to-simulate-realistic-data", i))
		if err := store.Set(keys[i], expectedValues[i]); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		keyIndex := i % numKeys
		value, err := store.Get(keys[keyIndex])
		if err != nil {
			b.Fatal(err)
		}
		if !bytes.Equal(value, expectedValues[keyIndex]) {
			b.Fatalf("unexpected value for key %s", keys[keyIndex])
		}
	}
}

// Implement a benchmark that uses range scans with a filter that should only match a subset of keys
func BenchmarkSqliteKvStore_RangeScanWithFilter(b *testing.B) {
	dbpool := newTestDB(b)
	store, err := NewSqliteKVStore(dbpool, "benchmark_range_scan")
	if err != nil {
		b.Fatal(err)
	}

	// Pre-populate the store with test data
	numKeys := 10000
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key-%04d", i)
		value := []byte(fmt.Sprintf("value-data-for-key-%d", i))
		if err := store.Set(key, value); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		count := 0
		// pick a random block of keys in the range 0-9
		key := fmt.Sprintf("key-%d", i%10)
		err := store.ForEach(key, func(key string, value []byte) error {
			if !strings.HasPrefix(key, "key-") {
				return fmt.Errorf("unexpected key: %s", key)
			}
			count++
			return nil
		})
		if err != nil {
			b.Fatal(err)
		}
		if count != numKeys/10 {
			b.Fatalf("expected %d keys, got %d", numKeys/10, count)
		}
	}
}
