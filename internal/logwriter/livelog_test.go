package logwriter

import (
	"bytes"
	"testing"
)

func TestWriteThenRead(t *testing.T) {
	t.TempDir()

	logger, err := NewLiveLogger(t.TempDir())
	if err != nil {
		t.Fatalf("NewLiveLogger failed: %v", err)
	}

	writer, err := logger.NewWriter("test")
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	data := []byte("test")
	if _, err := writer.Write(data); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	ch, err := logger.Subscribe("test")
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	d := <-ch
	if string(d) != "test" {
		t.Fatalf("Read failed: expected test, got %s", string(d))
	}
}

func TestBigWriteThenRead(t *testing.T) {
	bigtext := genbytes(32 * 1000)
	logger, err := NewLiveLogger(t.TempDir())
	if err != nil {
		t.Fatalf("NewLiveLogger failed: %v", err)
	}

	writer, err := logger.NewWriter("test")
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	if _, err := writer.Write([]byte(bigtext)); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	writer.Close()

	ch, err := logger.Subscribe("test")
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	data := make([]byte, 0)
	for d := range ch {
		data = append(data, d...)
	}
	if !bytes.Equal(data, bigtext) {
		t.Fatalf("Read failed: expected %d bytes, got %d", len(bigtext), len(data))
	}
}

func TestWritingWhileReading(t *testing.T) {
	logger, err := NewLiveLogger(t.TempDir())
	if err != nil {
		t.Fatalf("NewLiveLogger failed: %v", err)
	}

	writer, err := logger.NewWriter("test")
	if err != nil {
		t.Fatalf("NewWriter failed: %v", err)
	}

	if _, err := writer.Write([]byte("test")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	ch, err := logger.Subscribe("test")
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	if r1 := <-ch; string(r1) != "test" {
		t.Fatalf("Read failed: expected test, got %s", string(r1))
	}

	go func() {
		writer.Write([]byte("test2"))
		writer.Close()
	}()

	if r2 := <-ch; string(r2) != "test2" {
		t.Fatalf("Read failed: expected test2, got %s", string(r2))
	}
}

func genbytes(length int) []byte {
	data := make([]byte, length)
	for i := 0; i < length; i++ {
		data[i] = 'a'
	}
	return data
}
