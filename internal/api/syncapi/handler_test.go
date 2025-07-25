package syncapi

import (
	"context"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1sync/v1syncconnect"
	"github.com/garethgeorge/backrest/internal/api/syncapi/tunnel"
	"github.com/garethgeorge/backrest/internal/logstore"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/oplog/memstore"
	"github.com/garethgeorge/backrest/internal/testutil"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/protobuf/proto"
)

func TestHandler(t *testing.T) {
	ctx, cancel := testutil.WithDeadlineFromTest(t, context.Background())
	defer cancel()

	handler := newHandlerForTest(t)
	handler.SetLogger(testutil.NewTestLogger(t))

	mux := http.NewServeMux()
	mux.Handle(v1syncconnect.NewSyncPeerServiceHandler(handler))

	// Construct the service we want to provide
	provider := tunnel.NewConnectionProvider(10 /* bufferSize */)
	server := &http.Server{
		Handler: h2c.NewHandler(mux, &http2.Server{}), // h2c is HTTP/2 without TLS for grpc-connect support.
	}
	testutil.ServeForTest("HTTPServer", t, server, provider)

	// Serve the tunnel service which is proxying to the provided service.
	tunnelServiceMux := http.NewServeMux()
	tunnelHandler := tunnel.NewTunnelHandler(provider).SetLogger(testutil.NewTestLogger(t))
	tunnelServiceMux.Handle(v1syncconnect.NewTunnelServiceHandler(tunnelHandler))
	grpcServer := &http.Server{
		Addr:    testutil.AllocOpenBindAddr(t),
		Handler: h2c.NewHandler(tunnelServiceMux, &http2.Server{}), // h2c is HTTP/2 without TLS for grpc-connect support.
	}
	testutil.ListenAndServeForTest("gRPCTunnelServer", t, grpcServer)

	// With that "simple" setup out of the way, we can try establishing a connection.
	// TODO: this really could be made more ergonomic
	tunnelClient := v1syncconnect.NewTunnelServiceClient(tunnel.NewInsecureHttpClient(), "http://"+grpcServer.Addr)
	stream := tunnelClient.Tunnel(ctx)
	wrapped := tunnel.NewWrappedStreamFromClient(stream, tunnel.WithLogger(testutil.NewTestLogger(t).Named("tunnel-client")))
	go func() {
		t.Log("Client stream started")
		if err := wrapped.HandlePackets(ctx); err != nil {
			t.Errorf("Failed to handle packets: %v", err)
		}
		t.Log("Client stream ended")
	}()

	// Spin until the connection is ready.
	for !wrapped.IsReady() {
		time.Sleep(10 * time.Millisecond)
	}

	t.Log("Connection is ready")

	client := v1syncconnect.NewSyncPeerServiceClient(tunnel.NewWrappedStreamClient(wrapped), "https://localhost:80")
	resp, err := client.GetOperationMetadata(ctx, connect.NewRequest(&v1.OpSelector{
		OriginalInstanceKeyid: proto.String("test-instance-key-id"),
	}))
	if err != nil {
		t.Fatalf("Failed to get operation metadata: %v", err)
	}
	t.Logf("Received response: %v", resp.Msg)
}

func newHandlerForTest(t *testing.T) *syncHandler {
	t.Helper()
	// Create a new oplog
	opstore := memstore.NewMemStore()
	oplog, err := oplog.NewOpLog(opstore)
	assert.NoError(t, err, "Failed to create oplog")

	// Create a new log store
	logStore, err := logstore.NewLogStore(t.TempDir())
	assert.NoError(t, err, "Failed to create log store")

	// Create the sync handler
	return NewSyncHandler(oplog, logStore)
}
