package tunnel

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/garethgeorge/backrest/gen/go/v1sync"
	"github.com/garethgeorge/backrest/gen/go/v1sync/v1syncconnect"
	"github.com/garethgeorge/backrest/internal/testutil"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func newHelloHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Hello, World!"))
	})
	return mux
}

type sampleHandler struct {
	logger   *zap.Logger
	streams  []*WrappedStream
	provider *ConnectionProvider
}

var _ v1syncconnect.TunnelServiceHandler = (*sampleHandler)(nil)

func (sh *sampleHandler) Tunnel(ctx context.Context, stream *connect.BidiStream[v1sync.TunnelMessage, v1sync.TunnelMessage]) error {
	wrapped := NewWrappedStream(stream, WithLogger(sh.logger))
	wrapped.ProvideConnectionsTo(sh.provider)
	sh.streams = append(sh.streams, wrapped)
	return wrapped.HandlePackets(ctx)
}

func serveForTest(name string, t *testing.T, server *http.Server, listener net.Listener) {
	t.Helper()
	go func() {
		t.Logf("Starting server %s with listener", name)
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			t.Errorf("Server %s failed to start: %v", name, err)
		}
	}()
	t.Cleanup(func() {
		t.Logf("Shutting down server %s", name)
		deadline, cancel := context.WithDeadline(context.Background(), time.Now().Add(2*time.Second))
		defer cancel()
		if err := server.Shutdown(deadline); err != nil {
			t.Errorf("Failed to close server %s: %v", name, err)
		} else {
			t.Logf("Server %s closed successfully", name)
		}
	})
}

func listenAndServeForTest(name string, t *testing.T, server *http.Server) {
	t.Helper()
	go func() {
		t.Logf("Starting server %s on %s", name, server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Errorf("Server %s failed to start: %v", name, err)
		} else {
			t.Logf("Server %s closed successfully", name)
		}
	}()
	t.Cleanup(func() {
		t.Logf("Shutting down server %s on %s", name, server.Addr)
		if err := server.Shutdown(context.Background()); err != nil {
			t.Errorf("Failed to close server %s: %v", name, err)
		}
	})
}

func waitForConnectionReady(ctx context.Context, t *testing.T, wrapped *WrappedStream) {
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for connection to be ready: %v", ctx.Err())
		default:
			if wrapped.IsReady() {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestConnect(t *testing.T) {
	t.Skip("skipping syncapi tests")
	ctx, cancel := testutil.WithDeadlineFromTest(t, context.Background())
	defer cancel()

	// Construct the service we want to provide
	provider := NewConnectionProvider(10 /* bufferSize */)
	sampleHandler := &sampleHandler{
		logger:   testutil.NewTestLogger(t).Named("sample-handler"),
		provider: provider,
	}

	server := &http.Server{
		Handler: newHelloHandler(),
	}
	serveForTest("HTTPServer", t, server, provider)

	// Serve the sample handler
	mux := http.NewServeMux()
	mux.Handle(v1syncconnect.NewTunnelServiceHandler(sampleHandler))
	grpcServer := &http.Server{
		Addr:    testutil.AllocOpenBindAddr(t),
		Handler: h2c.NewHandler(mux, &http2.Server{}), // h2c is HTTP/2 without TLS for grpc-connect support.
	}
	listenAndServeForTest("gRPCServer", t, grpcServer)

	// Create a client and connect to the server
	client := v1syncconnect.NewTunnelServiceClient(NewInsecureHttpClient(), "http://"+grpcServer.Addr)
	stream := client.Tunnel(ctx)
	wrapped := NewWrappedStreamFromClient(stream, WithLogger(testutil.NewTestLogger(t).Named("client")))
	go func() {
		t.Log("Client stream started")
		if err := wrapped.HandlePackets(ctx); err != nil {
			t.Errorf("Failed to handle packets: %v", err)
		}
		t.Log("Client stream ended")
	}()

	waitForConnectionReady(ctx, t, wrapped)

	// Attempt to connect to the server
	conn, err := wrapped.Dial()
	if err != nil {
		t.Fatalf("Failed to dial server: %v", err)
	}

	// Write an HTTP request to the connection
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
	t.Logf("Received response: %s", resp.Status)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	t.Logf("Response body: %s", body)

	t.Logf("Closing connection with ID %d", conn.(*connState).connId)
	if err := conn.Close(); err != nil {
		t.Fatalf("Failed to close connection: %v", err)
	}
	t.Logf("Calling wrapped.Shutdown()")
	if err := wrapped.Shutdown(); err != nil {
		t.Fatalf("Failed to shutdown wrapped stream: %v", err)
	}
	t.Log("Test completed successfully")
}
