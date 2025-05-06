package webui

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
)

func TestEmbedNotEmpty(t *testing.T) {
	files, err := content.ReadDir(contentPrefix)
	if err != nil {
		t.Fatalf("expected embedded files for WebUI, got error: %v", err)
	}

	if len(files) == 0 {
		t.Fatalf("expected >0 embedded files for WebUI, got %d", len(files))
	}
}

func TestServeIndexNoGzip(t *testing.T) {
	handler := Handler()

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Accept-Encoding", "")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Windows doesn't have the gzip binary, so we skip compression during the
	// go:generate step on Windows
	if runtime.GOOS != "windows" && rr.Header().Get("Content-Encoding") != "" {
		t.Errorf("handler returned wrong content encoding: got %v want %v",
			rr.Header().Get("Content-Encoding"), "")
	}
}

func TestServeIndexGzip(t *testing.T) {
	handler := Handler()

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Accept-Encoding", "gzip")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Windows doesn't have the gzip binary, so we skip compression during the
	// go:generate step on Windows
	if runtime.GOOS != "windows" && rr.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("handler returned wrong content encoding: got %v want %v",
			rr.Header().Get("Content-Encoding"), "gzip")
	}
}
