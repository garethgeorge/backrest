package restic

import "testing"

func TestOutputCapture(t *testing.T) {
	c := newOutputCapturer(100)

	c.Write([]byte("hello"))

	if c.String() != "hello" {
		t.Errorf("expected 'hello', got '%s'", c.String())
	}
}

func TestOutputCaptureDrops(t *testing.T) {
	c := newOutputCapturer(2)

	c.Write([]byte("hello"))

	want := "h...[3 bytes dropped]...o"
	if c.String() != want {
		t.Errorf("expected '%s', got '%s'", want, c.String())
	}
}
