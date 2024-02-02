package rotatinglog

import (
	"fmt"
	"strings"
	"testing"
)

func TestRotatingLog(t *testing.T) {
	log := NewRotatingLog(t.TempDir()+"/rotatinglog", 10)
	name, err := log.Write([]byte("test"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	data, err := log.Read(name)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if string(data) != "test" {
		t.Fatalf("Read failed: expected test, got %s", string(data))
	}
}

func TestRotatingLogMultipleEntries(t *testing.T) {
	log := NewRotatingLog(t.TempDir()+"/rotatinglog", 10)
	for i := 0; i < 10; i++ {
		name, err := log.Write([]byte(fmt.Sprintf("%d", i)))
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
		data, err := log.Read(name)
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}
		if fmt.Sprintf("%d", i) != string(data) {
			t.Fatalf("Read failed: expected %d, got %s", i, string(data))
		}
	}
}

func TestBigEntries(t *testing.T) {
	log := NewRotatingLog(t.TempDir()+"/rotatinglog", 10)
	for size := range []int{10, 100, 1234, 5938, 1023, 1025} {
		data := genstr(size)
		name, err := log.Write([]byte(data))
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
		read, err := log.Read(name)
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}
		if string(read) != data {
			t.Fatalf("Read failed: expected %s, got %s", data, string(read))
		}
	}
}

func genstr(size int) string {
	return strings.Repeat("a", size)
}
