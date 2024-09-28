package logstore

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"slices"
	"sync"
	"testing"
	"time"
)

func TestReadWrite(t *testing.T) {
	t.Parallel()

	ls, err := NewLogStore(t.TempDir())
	if err != nil {
		t.Fatalf("new log writer failed: %v", err)
	}
	defer ls.Close()

	w, err := ls.Create("test", 0)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if _, err := w.Write([]byte("hello, world")); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// assert that the file is on disk at this point
	entries := getInprogressEntries(t, ls)
	if len(entries) != 1 {
		t.Fatalf("unexpected number of inprogress entries: %d", len(entries))
	}

	if err := w.Close(); err != nil {
		t.Fatalf("close writer failed: %v", err)
	}

	r, err := ls.Open("test")
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(data) != "hello, world" {
		t.Fatalf("unexpected content: %s", data)
	}

	entries = getInprogressEntries(t, ls)
	if len(entries) != 0 {
		t.Fatalf("unexpected number of inprogress entries: %d", len(entries))
	}
}

func TestHugeReadWrite(t *testing.T) {
	t.Parallel()

	ls, err := NewLogStore(t.TempDir())
	if err != nil {
		t.Fatalf("new log writer failed: %v", err)
	}
	defer ls.Close()

	w, err := ls.Create("test", 0)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	data := bytes.Repeat([]byte("hello, world\n"), 1<<15)
	if _, err := w.Write(data); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer failed: %v", err)
	}

	r, err := ls.Open("test")
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}

	readData := bytes.NewBuffer(nil)
	if _, err := io.Copy(readData, r); err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if !bytes.Equal(readData.Bytes(), data) {
		t.Fatalf("unexpected content")
	}

	if err := r.Close(); err != nil {
		t.Fatalf("close reader failed: %v", err)
	}
}

func TestReadWhileWrite(t *testing.T) {
	t.Parallel()

	ls, err := NewLogStore(t.TempDir())
	if err != nil {
		t.Fatalf("new log writer failed: %v", err)
	}
	defer ls.Close()

	w, err := ls.Create("test", 0)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	r, err := ls.Open("test")
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	data := bytes.NewBuffer(nil)
	wantData := bytes.NewBuffer(nil)

	var wg sync.WaitGroup
	var readn int64
	var readerr error
	wg.Add(1)
	go func() {
		defer r.Close()
		readn, readerr = io.Copy(data, r)
		wg.Done()
	}()

	for i := 0; i < 100; i++ {
		str := fmt.Sprintf("hello, world %d\n", i)
		wantData.WriteString(str)

		if _, err := w.Write([]byte(str)); err != nil {
			t.Fatalf("write failed: %v", err)
		}

		if i%2 == 0 {
			time.Sleep(2 * time.Millisecond)
		}
	}

	fmt.Printf("trying to close writer from test...")
	if err := w.Close(); err != nil {
		t.Fatalf("close writer failed: %v", err)
	}

	wg.Wait()

	// check that the asynchronous read completed successfully
	if readerr != nil {
		t.Fatalf("read failed: %v", readerr)
	}
	if readn == 0 || readn != int64(wantData.Len()) {
		t.Fatalf("unexpected read length: %d", readn)
	}
	if !bytes.Equal(data.Bytes(), wantData.Bytes()) {
		t.Fatalf("unexpected content: %s", data.Bytes())
	}

	// check that the finalized data matches expectations
	var finalizedData bytes.Buffer
	r2, err := ls.Open("test")
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	if _, err := io.Copy(&finalizedData, r2); err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if !bytes.Equal(finalizedData.Bytes(), wantData.Bytes()) {
		t.Fatalf("unexpected content: %s", finalizedData.Bytes())
	}

	if err := r2.Close(); err != nil {
		t.Fatalf("close reader failed: %v", err)
	}
}

func getInprogressEntries(t *testing.T, ls *LogStore) []os.DirEntry {
	entries, err := os.ReadDir(ls.inprogressDir)
	if err != nil {
		t.Fatalf("read dir failed: %v", err)
	}

	entries = slices.DeleteFunc(entries, func(e os.DirEntry) bool { return e.IsDir() })
	return entries
}
