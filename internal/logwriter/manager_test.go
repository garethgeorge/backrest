package logwriter

import "testing"

func TestLogLifecycle(t *testing.T) {
	mgr, err := NewLogManager(t.TempDir(), 10)
	if err != nil {
		t.Fatalf("NewLogManager failed: %v", err)
	}

	id, w, err := mgr.NewLiveWriter("test")
	if err != nil {
		t.Fatalf("NewLiveWriter failed: %v", err)
	}

	ch, err := mgr.Subscribe(id)
	if err != nil {
		t.Fatalf("Subscribe to live log %q failed: %v", id, err)
	}

	contents := "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum."
	if _, err := w.Write([]byte(contents)); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	w.Close()

	if data := <-ch; string(data) != contents {
		t.Fatalf("Read failed: expected %q, got %q", contents, string(data))
	}

	finalID, err := mgr.Finalize(id)
	if err != nil {
		t.Fatalf("Finalize failed: %v", err)
	}

	finalCh, err := mgr.Subscribe(finalID)
	if err != nil {
		t.Fatalf("Subscribe to finalized log %q failed: %v", finalID, err)
	}

	if data := <-finalCh; string(data) != contents {
		t.Fatalf("Read failed: expected %q, got %q", contents, string(data))
	}
}
