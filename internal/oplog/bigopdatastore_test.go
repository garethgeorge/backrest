package oplog

import (
	"testing"
)

func TestBigOpLogDataStore(t *testing.T) {
	store := &BigOpDataStore{
		path: t.TempDir(),
	}

	if err := store.SetBigData(1, "test", []byte("hello world")); err != nil {
		t.Fatal(err)
	}

	if byte, err := store.GetBigData(1, "test"); err != nil || string(byte) != "hello world" {
		t.Fatalf("expected %q, got %q", "hello world", string(byte))
	}

	if err := store.SetBigData(1, "test", []byte("hello world 2")); err != nil {
		t.Fatal(err)
	}

	if byte, err := store.GetBigData(1, "test"); err != nil || string(byte) != "hello world 2" {
		t.Fatalf("expected %q, got %q", "hello world 2", string(byte))
	}

	if err := store.DeleteOperationData(1); err != nil {
		t.Fatal(err)
	}

	if _, err := store.GetBigData(1, "test"); err == nil {
		t.Fatalf("expected error, got nil")
	}
}
