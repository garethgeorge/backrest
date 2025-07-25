package tunnel

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	"github.com/garethgeorge/backrest/gen/go/v1sync"
	"go.uber.org/zap"
)

type WrappedStreamOptions func(*WrappedStream)

func WithLogger(logger *zap.Logger) WrappedStreamOptions {
	return func(ws *WrappedStream) {
		ws.logger = logger
	}
}

func WithHeartbeatInterval(interval time.Duration) WrappedStreamOptions {
	return func(ws *WrappedStream) {
		ws.heartbeatInterval = interval
	}
}

type WrappedStream struct {
	isClient bool

	stream stream
	logger *zap.Logger

	provider          *ConnectionProvider // is nil until ProvideConnectionsTo is called
	heartbeatInterval time.Duration       // interval for heartbeat messages, if set

	// pool of connections, every connection is bidirectional so can be initiated from either side.
	connsMu sync.Mutex
	conns   map[int64]*connState

	lastConnID atomic.Int64

	handlingPackets atomic.Bool
	streamStopped   atomic.Bool

	sharedSecretCh chan struct{}
	sharedSecret   []byte
}

func newWrappedStreamInternal(stream stream, isClient bool, opts ...WrappedStreamOptions) *WrappedStream {
	ws := &WrappedStream{
		isClient:          isClient,
		stream:            stream,
		heartbeatInterval: 30 * time.Second,
		conns:             make(map[int64]*connState),
		sharedSecretCh:    make(chan struct{}),
	}
	for _, opt := range opts {
		opt(ws)
	}
	if isClient {
		ws.lastConnID.Store(1) // Use odd numbered connection IDs for client-initiated connections.
	} else {
		ws.lastConnID.Store(2) // Use even numbered connection IDs for server-initiated connections.
	}
	return ws
}

func NewWrappedStream(stream *connect.BidiStream[v1sync.TunnelMessage, v1sync.TunnelMessage], opts ...WrappedStreamOptions) *WrappedStream {
	return newWrappedStreamInternal(&serverStream{
		stream: stream,
	}, false, opts...)
}

func NewWrappedStreamFromClient(stream *connect.BidiStreamForClient[v1sync.TunnelMessage, v1sync.TunnelMessage], opts ...WrappedStreamOptions) *WrappedStream {
	return newWrappedStreamInternal(&clientStream{
		stream: stream,
	}, true, opts...)
}

func (ws *WrappedStream) allocConnID() int64 {
	return ws.lastConnID.Add(2)
}

func (ws *WrappedStream) getSharedSecret() ([]byte, error) {
	<-ws.sharedSecretCh
	if ws.sharedSecret == nil {
		return nil, fmt.Errorf("shared secret not available")
	}
	return ws.sharedSecret, nil
}

func (ws *WrappedStream) IsReady() bool {
	return ws.handlingPackets.Load() && !ws.streamStopped.Load()
}

func (ws *WrappedStream) Dial() (net.Conn, error) {
	if !ws.handlingPackets.Load() {
		return nil, fmt.Errorf("cannot dial before handling packets")
	}

	secret, err := ws.getSharedSecret()
	if err != nil {
		return nil, fmt.Errorf("get shared secret: %w", err)
	}

	connID := ws.allocConnID()
	new := newConnState(ws.stream, connID, secret, ws.logger)
	if err := new.sendOpenPacket(); err != nil {
		return nil, fmt.Errorf("send open packet: %w", err)
	}
	ws.connsMu.Lock()
	defer ws.connsMu.Unlock()
	ws.conns[connID] = new
	return new, nil
}

func (ws *WrappedStream) ProvideConnectionsTo(provider *ConnectionProvider) {
	ws.provider = provider
}

func (ws *WrappedStream) sendHeartbeats(ctx context.Context, stream stream) {
	if ws.heartbeatInterval <= 0 || !ws.isClient {
		return
	}

	ticker := time.NewTicker(ws.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if ws.logger != nil {
				ws.logger.Debug("sending heartbeat")
			}
			if err := stream.Send(&v1sync.TunnelMessage{
				ConnId: -1, // handshake packet
			}); err != nil && ws.logger != nil {
				ws.logger.Error("failed to send heartbeat", zap.Error(err))
			}
		}
	}
}

func (ws *WrappedStream) HandlePackets(ctx context.Context) error {
	if ws.handlingPackets.Swap(true) {
		return fmt.Errorf("already handling packets")
	}
	defer ws.handlingPackets.Store(false)
	if ws.streamStopped.Load() {
		return fmt.Errorf("stream already stopped")
	}

	// TODO: optimization, it is generally secure and performant have a singleton key that is generated once and reused for all connections.
	// the only risk w/this approach is if the key were somehow leaked all related connections would be compromised. The risk is low in this case since
	// the key is only kept in memory for the lifetime of the process, and ideally is a second line of defense after TLS.
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generate key for handshake packet: %w", err)
	}

	if err := ws.stream.Send(&v1sync.TunnelMessage{
		ConnId:           -100, // handshake packet
		PubkeyEcdhX25519: key.PublicKey().Bytes(),
	}); err != nil {
		return fmt.Errorf("send handshake packet: %w", err)
	}

	go func() {
		timeoutTimer := time.NewTimer(5 * time.Second)
		defer timeoutTimer.Stop()
		select {
		case <-timeoutTimer.C:
			if ws.logger != nil {
				ws.logger.Warn("timeout waiting for handshake response")
			}
			ws.Shutdown()
		case <-ws.sharedSecretCh:
			if ws.logger != nil {
				ws.logger.Info("handshake response received, shared secret established")
			}
		}
	}()

	// receive handshake packet
	handshake, err := ws.stream.Receive()
	if err != nil {
		return fmt.Errorf("receive handshake packet: %w", err)
	}
	if handshake.GetConnId() != -100 {
		return fmt.Errorf("expected handshake packet with connId -100, got %d", handshake.GetConnId())
	}
	if len(handshake.PubkeyEcdhX25519) == 0 {
		return fmt.Errorf("handshake packet does not contain public key")
	}
	peerKey, err := ecdh.X25519().NewPublicKey(handshake.PubkeyEcdhX25519)
	if err != nil {
		return fmt.Errorf("parse peer public key: %w", err)
	}
	sharedSecret, err := key.ECDH(peerKey)
	if err != nil {
		return fmt.Errorf("compute shared key: %w", err)
	}
	ws.sharedSecret = sharedSecret
	close(ws.sharedSecretCh)

	// cryptedStream := newCryptedStream(ws.stream, ws.sharedSecret)
	cryptedStream := ws.stream

	newConn := func(connId int64) *connState {
		if ws.logger != nil {
			ws.logger.Info("new tunnel connection", zap.Int64("connId", connId))
		}
		new := newConnState(ws.stream, connId, ws.sharedSecret, ws.logger)
		ws.conns[connId] = new
		ws.provider.ProvideConn(new)
		return new
	}

	headOfLineBlockingTimer := time.NewTimer(0)
	defer headOfLineBlockingTimer.Stop()

	// send heartbeats in a separate goroutine if heartbeat interval is set
	go ws.sendHeartbeats(ctx, cryptedStream)

	for {
		msg, err := cryptedStream.Receive()
		if err != nil {
			if ws.handlingPackets.Load() {
				return nil
			}
			return fmt.Errorf("receive message: %w", err)
		}

		connId := msg.GetConnId()
		if connId <= 0 {
			// negative IDs are reserved for healthchecks and control messages, ignored.
			continue
		}

		if msg.Close {
			if ws.logger != nil {
				ws.logger.Info("closing connection", zap.Int64("connId", connId))
			}
			ws.connsMu.Lock()
			if conn, exists := ws.conns[connId]; exists {
				if err := conn.Close(); err != nil && ws.logger != nil {
					ws.logger.Error("failed to close connection", zap.Int64("connId", connId), zap.Error(err))
				}
				delete(ws.conns, connId)
			}
			ws.connsMu.Unlock()
			continue
		}

		ws.connsMu.Lock()
		conn, exists := ws.conns[connId]
		if !exists {
			if msg.Seqno != 0 {
				if ws.logger != nil {
					ws.logger.Warn("received message for unknown connection", zap.Int64("connId", connId), zap.Int64("seqno", msg.Seqno))
				}
				continue
			}
			if ws.provider == nil {
				if ws.logger != nil {
					ws.logger.Warn("received open packet for unknown connection, but no provider is set, ignoring the connection", zap.Int64("connId", connId))
				}
				continue
			}

			conn = newConn(connId)
		}
		ws.connsMu.Unlock()

		if len(msg.Data) == 0 {
			continue
		}

		headOfLineBlockingTimer.Reset(100 * time.Millisecond)
		select {
		case <-conn.closedCh:
			if ws.logger != nil {
				ws.logger.Warn("received message for closed connection", zap.Int64("connId", connId), zap.Int64("seqno", msg.Seqno))
			}
		case conn.reads <- msg.Data:
			if ws.logger != nil {
				ws.logger.Debug("received data on tunnel connection", zap.Int64("connId", connId), zap.Int64("seqno", msg.Seqno), zap.Int("dataLength", len(msg.Data)))
			}
		case <-headOfLineBlockingTimer.C:
			if ws.logger != nil {
				ws.logger.Warn("head-of-line blocking detected, no reads available for connection", zap.Int64("connId", connId), zap.Int64("seqno", msg.Seqno))
			}
			conn.Close()
		case <-ctx.Done():
			// Close all open connections when the context is done.
			ws.connsMu.Lock()
			if ws.logger != nil {
				ws.logger.Info("context done, closing all connections")
			}
			for _, c := range ws.conns {
				if err := c.Close(); err != nil && ws.logger != nil {
					ws.logger.Error("failed to close connection on context done", zap.Int64("connId", c.connId), zap.Error(err))
				}
			}
			ws.connsMu.Unlock()
			return nil
		}
	}
}

func (ws *WrappedStream) Shutdown() error {
	if ws.streamStopped.Swap(true) {
		if ws.logger != nil {
			ws.logger.Warn("wrapped stream already shutdown")
		}
		return nil
	}
	return ws.stream.Close()
}
