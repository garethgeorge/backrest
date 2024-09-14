package textstore

import (
	"bytes"
	"fmt"
	"io"
	"math/rand/v2"
	"testing"

	crand "crypto/rand"

	"github.com/google/go-cmp/cmp"
)

func TestCreate(t *testing.T) {
	store, err := NewSqliteTextStore("file::memory:?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestWriteRead(t *testing.T) {
	t.Parallel()
	store, err := NewSqliteTextStore("file::memory:?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	// defer store.Close()

	const id = "test"
	w, err := store.Create(id)
	if err != nil {
		t.Fatal(fmt.Errorf("create %q: %v", id, err))
	}
	if _, err := w.Write([]byte("hello")); err != nil {
		t.Fatal(fmt.Errorf("write: %v", err))
	}
	if err := w.Close(); err != nil {
		t.Fatal(fmt.Errorf("close: %v", err))
	}

	// now open for read
	r, err := store.Open(id)
	if err != nil {
		t.Fatal(fmt.Errorf("open %q: %v", id, err))
	}
	defer r.Close()

	buf := make([]byte, 5)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatal(fmt.Errorf("read: %v", err))
	}
	if n != 5 {
		t.Fatalf("read: got %d bytes, want 5", n)
	}
	if string(buf) != "hello" {
		t.Fatalf("read: got %q, want %q", buf, "hello")
	}
}

func TestVeryBigWrite(t *testing.T) {
	t.Parallel()
	big := bytes.Repeat([]byte("a"), 1<<20)

	store, err := NewSqliteTextStore("file::memory:?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	const id = "test2"
	w, err := store.Create(id)
	if err != nil {
		t.Fatalf("create %q: %v", id, err)
	}

	if _, err := w.Write(big); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	r, err := store.Open(id)
	if err != nil {
		t.Fatalf("open %q: %v", id, err)
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if diff := cmp.Diff(string(big), string(data)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestReadWriteCorrectness(t *testing.T) {
	t.Parallel()
	store, err := NewSqliteTextStore("file::memory:?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	const id = "test3"
	w, err := store.Create(id)
	if err != nil {
		t.Fatalf("create %q: %v", id, err)
	}

	want := bytes.NewBuffer(nil)

	for i := 0; i < 1000; i++ {
		seg := []byte(fmt.Sprintf("hello-%d\n", i))
		want.Write(seg)
		if _, err := w.Write(seg); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	r, err := store.Open(id)
	if err != nil {
		t.Fatalf("open %q: %v", id, err)
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if diff := cmp.Diff(want.String(), string(data)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRandomReadWriteSizes(t *testing.T) {
	tcs := []struct {
		id        string
		count     int
		sizeFn    func(i int) int
		verifyMod int
	}{
		{"1_byte_writes", 32 * 1024, func(i int) int { return 1 }, -1},
		{"3_byte_writes", 32 * 1024, func(i int) int { return 3 }, -1},
		{"1024_byte_writes", 128, func(i int) int { return 1024 }, 32},
		{"random_byte_writes", 1024, func(i int) int { return rand.IntN(4096) + 1 }, 128},
		{"chunk_sized_writes", 1024, func(i int) int { return sqliteWriterChunkSize }, 128},
	}

	for _, tc := range tcs {
		t.Run(tc.id, func(t *testing.T) {
			t.Parallel()
			store, err := NewSqliteTextStore("file::memory:?mode=memory&cache=shared")
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()

			expected := bytes.NewBuffer(nil)
			w, err := store.Create(tc.id)
			if err != nil {
				t.Fatalf("create %q: %v", tc.id, err)
			}
			defer w.Close()

			for i := 0; i < tc.count; i++ {
				size := tc.sizeFn(i)
				data := make([]byte, size)
				crand.Read(data)
				expected.Write(data)
				if n, err := w.Write(data); err != nil {
					t.Errorf("write: %v", err)
				} else if n != size {
					t.Errorf("write: got %d bytes, want %d", n, size)
				}

				if tc.verifyMod > 0 && i%tc.verifyMod == 0 {
					r, err := store.openInternal(tc.id, false)
					if err != nil {
						t.Fatalf("open %q: %v", tc.id, err)
					}

					data, err := io.ReadAll(r)
					if err != nil {
						t.Fatalf("read: %v", err)
					}

					if diff := cmp.Diff(expected.String(), string(data)); diff != "" {
						t.Fatalf("mismatch (-want +got):\n%s", diff)
					}

					if err := r.Close(); err != nil {
						t.Fatalf("close: %v", err)
					}
				}
			}
		})
	}

}
