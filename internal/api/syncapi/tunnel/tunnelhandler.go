package tunnel

import (
	"context"

	"connectrpc.com/connect"
	"github.com/garethgeorge/backrest/gen/go/v1sync"
	"github.com/garethgeorge/backrest/gen/go/v1sync/v1syncconnect"
	"go.uber.org/zap"
)

type TunnelHandler struct {
	logger   *zap.Logger
	streams  []*WrappedStream
	provider *ConnectionProvider
}

var _ v1syncconnect.TunnelServiceHandler = (*TunnelHandler)(nil)

func NewTunnelHandler(provider *ConnectionProvider) *TunnelHandler {
	return &TunnelHandler{
		logger:   zap.NewNop(),
		streams:  make([]*WrappedStream, 0),
		provider: provider,
	}
}

func (th *TunnelHandler) SetLogger(logger *zap.Logger) *TunnelHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	th.logger = logger.Named("tunnel-handler")
	return th
}

func (th *TunnelHandler) Tunnel(ctx context.Context, stream *connect.BidiStream[v1sync.TunnelMessage, v1sync.TunnelMessage]) error {
	wrapped := NewWrappedStream(stream, WithLogger(th.logger))
	wrapped.ProvideConnectionsTo(th.provider)
	th.streams = append(th.streams, wrapped)
	return wrapped.HandlePackets(ctx)
}
