package tunnel

import (
	"bufio"
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/garethgeorge/backrest/gen/go/v1/v1connect"
	"github.com/garethgeorge/backrest/internal/testutil"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func TestFailsToConnect(t *testing.T) {
	server := &http.Server{
		Handler: http.NotFoundHandler(),
	}

	t.Cleanup(func() {
		if err := server.Close(); err != nil {
			t.Errorf("Failed to close server: %v", err)
		}
	})

	client := NewTunnelClient(server, "helloclient")
	if client == nil {
		t.Fatal("Expected NewTunnelClient to return a non-nil client")
	}

	if err := client.ServeOnce(context.Background(), "localhost:9443"); err == nil {
		t.Error("Expected Serve to fail with a closed server, but it succeeded")
	} else {
		t.Logf("Serve failed as expected: %v", err)
	}
}

func TestCanConnect(t *testing.T) {
	testutil.InstallZapLogger(t)
	ctx, cancel := testutil.WithDeadlineFromTest(t, context.Background())
	defer cancel()

	handler := NewTunnelHandler()

	mux := http.NewServeMux()
	mux.Handle(v1connect.NewTunnelServiceHandler(handler))

	server := &http.Server{
		Addr:    testutil.AllocOpenBindAddr(t),
		Handler: h2c.NewHandler(mux, &http2.Server{}), // h2c is HTTP/2 without TLS for grpc-connect support.
	}
	t.Cleanup(func() {
		if err := server.Shutdown(context.Background()); err != nil {
			t.Errorf("Failed to close server: %v", err)
		}
	})

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Errorf("Server failed to start: %v", err)
		}
	}()

	clientServer := &http.Server{
		Addr: server.Addr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// OK handler
			w.WriteHeader(http.StatusOK)
		}),
	}

	client := NewTunnelClient(clientServer, "helloclient")
	client.ReconnectDelay = 1000 * time.Millisecond // Set a short delay for testing purposes
	go func() {
		if err := client.Serve(ctx, "http://"+server.Addr); err != nil && err != http.ErrServerClosed {
			t.Errorf("Server failed to serve: %v", err)
		}
	}()

	time.Sleep(1 * time.Second) // Wait for the client to connect

	// Now try dialing the client from the server
	conn, err := handler.Dial("helloclient")
	if err != nil {
		t.Fatalf("Failed to dial client: %v", err)
	}
	defer conn.Close()

	// Write an HTTP request to the conn
	req, err := http.NewRequest("GET", "http://example.com", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	if err := req.Write(conn); err != nil {
		t.Fatalf("Failed to write request: %v", err)
	}

	// Read the response
	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status OK, got %s", resp.Status)
	}
}
