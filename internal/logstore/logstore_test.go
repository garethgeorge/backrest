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

	w, err := ls.Create("test", 0, 0)
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

	w, err := ls.Create("test", 0, 0)
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

	w, err := ls.Create("test", 0, 0)
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

func TestCreateMany(t *testing.T) {
	t.Parallel()

	ls, err := NewLogStore(t.TempDir())
	if err != nil {
		t.Fatalf("new log writer failed: %v", err)
	}
	defer ls.Close()

	const n = 10
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("test%d", i)
		w, err := ls.Create(name, 0, 0)
		if err != nil {
			t.Fatalf("create %q failed: %v", name, err)
		}
		if _, err := w.Write([]byte(fmt.Sprintf("hello, world %d", i))); err != nil {
			t.Fatalf("write failed: %v", err)
		}
		if err := w.Close(); err != nil {
			t.Fatalf("close writer failed: %v", err)
		}
	}

	entries := getInprogressEntries(t, ls)
	if len(entries) != 0 {
		t.Fatalf("unexpected number of inprogress entries: %d", len(entries))
	}

	for i := 0; i < n; i++ {
		name := fmt.Sprintf("test%d", i)
		r, err := ls.Open(name)
		if err != nil {
			t.Fatalf("open %q failed: %v", name, err)
		}
		data, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		if string(data) != fmt.Sprintf("hello, world %d", i) {
			t.Fatalf("unexpected content: %s", data)
		}
		if err := r.Close(); err != nil {
			t.Fatalf("close reader failed: %v", err)
		}
	}
}

func TestReopenStore(t *testing.T) {
	d := t.TempDir()
	{
		ls, err := NewLogStore(d)
		if err != nil {
			t.Fatalf("new log writer failed: %v", err)
		}

		w, err := ls.Create("test", 0, 0)
		if err != nil {
			t.Fatalf("create failed: %v", err)
		}

		if _, err := w.Write([]byte("hello, world")); err != nil {
			t.Fatalf("write failed: %v", err)
		}

		if err := w.Close(); err != nil {
			t.Fatalf("close writer failed: %v", err)
		}

		// confirm that the file is on disk
		r, err := ls.Open("test")
		if err != nil {
			t.Fatalf("open first store failed: %v", err)
		}
		r.Close()

		if err := ls.Close(); err != nil {
			t.Fatalf("close log store failed: %v", err)
		}

	}

	{
		ls, err := NewLogStore(d)
		if err != nil {
			t.Fatalf("new log writer failed: %v", err)
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
		if err := r.Close(); err != nil {
			t.Fatalf("close reader failed: %v", err)
		}

		if err := ls.Close(); err != nil {
			t.Fatalf("close log store failed: %v", err)
		}
	}
}

func TestFindLogsWithParent(t *testing.T) {
	t.Parallel()

	ls, err := NewLogStore(t.TempDir())
	if err != nil {
		t.Fatalf("new log store failed: %v", err)
	}
	defer ls.Close()

	// Setup: Create logs with different parent operation IDs
	type logEntry struct {
		id       string
		parentID int64
		content  string
	}

	logs := []logEntry{
		{"log1", 100, "content for log1"},
		{"log2", 100, "content for log2"},
		{"log3", 100, "content for log3"},
		{"log4", 200, "content for log4"},
		{"log5", 200, "content for log5"},
		{"log6", 0, "content for log6"},
		{"log7", 300, "content for log7"},
	}

	// Create all the logs
	for _, entry := range logs {
		w, err := ls.Create(entry.id, entry.parentID, 0)
		if err != nil {
			t.Fatalf("create log %q failed: %v", entry.id, err)
		}
		if _, err := w.Write([]byte(entry.content)); err != nil {
			t.Fatalf("write to log %q failed: %v", entry.id, err)
		}
		if err := w.Close(); err != nil {
			t.Fatalf("close log %q failed: %v", entry.id, err)
		}
	}

	// Test cases
	tests := []struct {
		name         string
		parentOpID   int64
		expectedLogs []string
	}{
		{
			name:         "find logs for parent 100",
			parentOpID:   100,
			expectedLogs: []string{"log1", "log2", "log3"},
		},
		{
			name:         "find logs for parent 200",
			parentOpID:   200,
			expectedLogs: []string{"log4", "log5"},
		},
		{
			name:         "find logs for parent 0",
			parentOpID:   0,
			expectedLogs: []string{"log6"},
		},
		{
			name:         "find logs for parent 300",
			parentOpID:   300,
			expectedLogs: []string{"log7"},
		},
		{
			name:         "find logs for non-existent parent",
			parentOpID:   999,
			expectedLogs: []string{},
		},
		{
			name:         "find logs for negative parent ID",
			parentOpID:   -1,
			expectedLogs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			foundLogs, err := ls.FindLogsWithParent(tt.parentOpID)
			if err != nil {
				t.Fatalf("FindLogsWithParent failed: %v", err)
			}

			if len(foundLogs) != len(tt.expectedLogs) {
				t.Fatalf("expected %d logs for parent %d, got %d", len(tt.expectedLogs), tt.parentOpID, len(foundLogs))
			}

			// Sort both slices for comparison
			slices.Sort(foundLogs)
			expectedSorted := make([]string, len(tt.expectedLogs))
			copy(expectedSorted, tt.expectedLogs)
			slices.Sort(expectedSorted)

			if !slices.Equal(foundLogs, expectedSorted) {
				t.Fatalf("expected logs %v for parent %d, got %v", expectedSorted, tt.parentOpID, foundLogs)
			}
		})
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
