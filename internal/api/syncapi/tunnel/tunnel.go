package tunnel

import (
	"context"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"os"
	"slices"
	"sync"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1/v1connect"
	"github.com/garethgeorge/backrest/internal/env"
	"go.uber.org/zap"
)

var (
	heartbeatInterval = env.MultihostHeartbeatInterval()
)

// Okay we need implementations of two things, a net.Listener and a net.Conn over a socket.

// TunnelHandler is a server that implements the tunnel protocol, allows clients to connect and, when connected, will allow sending requests to connected clients.
type TunnelHandler struct {
	sessionMu sync.Mutex // Mutex to protect the conns map
	sessions  map[string]*tunnelHandlerSession
}

type tunnelHandlerSession struct {
	clientID string

	sessionCloseOnce sync.Once     // Ensures the session is closed only once
	sessionClose     chan struct{} // Channel to signal that the session is closed

	// streamMu protects the stream and is used to ensure that only one goroutine can read/write to the stream at a time.
	streamMu sync.Mutex
	stream   *connect.BidiStream[v1.TunnelMessage, v1.TunnelMessage]

	connsMu    sync.Mutex            // Mutex to protect the conns map
	conns      map[int64]*tunnelConn // guarded by connsMu, maps connection IDs to tunnelConn instances
	nextConnId int64                 // guarded by connsMu, used to generate unique connection IDs
}

func (s *tunnelHandlerSession) Close() {
	s.connsMu.Lock()
	defer s.connsMu.Unlock()
	for _, conn := range s.conns {
		conn.Close() // Close all connections in the session
	}
	s.sessionCloseOnce.Do(func() {
		close(s.sessionClose) // Signal that the session is closed
	})
}

var _ v1connect.TunnelServiceHandler = (*TunnelHandler)(nil)

func NewTunnelHandler() *TunnelHandler {
	return &TunnelHandler{
		sessions: make(map[string]*tunnelHandlerSession),
	}
}

func (h *TunnelHandler) Tunnel(ctx context.Context, stream *connect.BidiStream[v1.TunnelMessage, v1.TunnelMessage]) error {
	// Receive initial handshake message to identify the client.
	msg, err := stream.Receive()
	if err != nil {
		return fmt.Errorf("failed to receive initial message: %w", err)
	}
	if msg.ClientId == "" {
		return fmt.Errorf("client ID is required in the initial message")
	}

	clientID := msg.ClientId
	h.sessionMu.Lock()
	session, ok := h.sessions[clientID]
	if ok {
		return fmt.Errorf("client %q is already connected", clientID)
	}
	session = &tunnelHandlerSession{
		clientID:     clientID,
		stream:       stream,
		conns:        make(map[int64]*tunnelConn),
		sessionClose: make(chan struct{}),
	}
	h.sessions[clientID] = session
	h.sessionMu.Unlock()

	// Cleanup the session when the stream ends.
	defer func() {
		h.sessionMu.Lock()
		delete(h.sessions, clientID) // Remove the session when done
		h.sessionMu.Unlock()
		session.Close()
	}()

	// create an interval timer, used to efficiently provide a clock for reads, broken into 10 quantums
	// meaning cancellations are +/- 10ms from the deadline.
	ioDeadline := 100 * time.Millisecond
	ticker := time.NewTicker(ioDeadline / 10) // 10 ticks per deadline
	defer ticker.Stop()

	for {
		msg, err := stream.Receive()
		if err != nil {
			if err != io.EOF {
				return fmt.Errorf("failed to read from stream: %w", err)
			}
			break // EOF indicates the stream is closed, we can exit the loop.
		}

		if msg.ConnId < 0 {
			// Negative connection IDs are healthcheck messages, we can ignore them.
			continue
		}

		session.connsMu.Lock()
		conn, ok := session.conns[msg.ConnId]
		if msg.Close {
			if ok {
				conn.Close() // Close the connection if it exists
			}
			delete(session.conns, msg.ConnId) // Remove the connection from the map
			session.connsMu.Unlock()
			continue
		}
		session.connsMu.Unlock()
		if !ok {
			zap.S().Debugf("client sent packet for unknown connection ID %d, connections must always be initiated by the server", msg.ConnId)
			continue
		}
		if len(msg.Data) == 0 {
			continue // Skip empty data messages
		}

		quantums := 0
		select {
		case conn.readsChan <- msg.Data: // Try to write first.
		case <-conn.closedChan:
			// If the connection is closed, we can pass on the write.
		case <-ticker.C:
			// Close the channel if it's not able to respond to reads within the deadline (prevent head of line blocking).
			if quantums >= 10 {
				conn.Close()
			}
			quantums++
		}
	}
	return nil
}

func (h *TunnelHandler) Dial(clientID string) (net.Conn, error) {
	h.sessionMu.Lock()
	session, ok := h.sessions[clientID]
	if !ok {
		h.sessionMu.Unlock()
		return nil, fmt.Errorf("client %q not connected: %w", clientID, net.ErrClosed)
	}
	h.sessionMu.Unlock()

	session.connsMu.Lock()
	session.nextConnId++
	connID := session.nextConnId
	newConn := newTunnelConn(connID)
	session.conns[connID] = newConn
	session.connsMu.Unlock()

	// acquire the session's stream lock and send an empty initial message to initialize the connection
	session.streamMu.Lock()
	defer session.streamMu.Unlock()
	if err := session.stream.Send(&v1.TunnelMessage{
		ConnId: connID,
		Data:   nil, // nil data indicates the connection is being established
	}); err != nil {
		return nil, fmt.Errorf("failed to send initial message for connection %d: %w", connID, err)
	}

	// start management goroutine for the new connection
	go func() {
		for {
			select {
			case <-session.sessionClose:
				return // Exit without any cleanup, nothing is listening anymore.
			case <-newConn.closedChan:
				session.streamMu.Lock()
				// Try to send a close message to the stream.
				session.stream.Send(&v1.TunnelMessage{
					ConnId: connID,
					Data:   nil,
					Close:  true,
				})
				session.streamMu.Unlock()
				return
			case data := <-newConn.writeChan:
				session.streamMu.Lock()
				if err := session.stream.Send(&v1.TunnelMessage{
					ConnId: connID,
					Data:   data.bytes,
				}); err != nil {
					data.resultChan <- err
				}
				session.streamMu.Unlock()
			}
		}
	}()

	return newConn, nil
}

// TunnelClient is a client that connects to a TunnelHandler and provides an http.Server that requests can be sent to.
type TunnelClient struct {
	server   *http.Server
	clientID string

	ReconnectDelay time.Duration

	connectionLimit int64 // The maximum number of connections that can be established at once.

	Logger *zap.Logger
}

func NewTunnelClient(server *http.Server, clientID string) *TunnelClient {
	return &TunnelClient{
		server:          server,
		clientID:        clientID,
		ReconnectDelay:  60 * time.Second,
		connectionLimit: 100, // Default connection limit, can be adjusted as needed.
		Logger:          zap.L().Named("TunnelClient-" + clientID),
	}
}

func (t *TunnelClient) Shutdown(ctx context.Context) error {
	if t.server != nil {
		return t.server.Shutdown(ctx)
	}
	return nil
}

// Serve starts the HTTP server for the tunnel client and connects to the TunnelHandler that will be it's client.
func (t *TunnelClient) Serve(ctx context.Context, addr string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	client := v1connect.NewTunnelServiceClient(newInsecureClient(), addr)
	tunnelListener := newTunnelListener()
	go func() {
		attempts := 0
		for {
			connectTime := time.Now()

			stream := client.Tunnel(ctx)
			if err := t.provideConnsFromStream(stream, tunnelListener); err != nil {
				zap.L().Error("Failed to provide connections from stream", zap.Error(err))
			}

			attempts++
			connectionLife := time.Since(connectTime)
			waitToReconnect := t.ReconnectDelay - connectionLife
			if waitToReconnect > 0 {
				time.Sleep(waitToReconnect)
			}
		}
	}()

	return t.server.Serve(tunnelListener)
}

func (t *TunnelClient) ServeOnce(ctx context.Context, addr string) error {
	ctx, cancel := context.WithCancel(ctx)
	t.Logger.Info("Starting TunnelClient server", zap.String("addr", addr))
	defer cancel()
	client := v1connect.NewTunnelServiceClient(newInsecureClient(), addr)
	stream := client.Tunnel(ctx)
	tunnelListener := newTunnelListener()
	defer tunnelListener.Close()

	go func() {
		if err := t.provideConnsFromStream(stream, tunnelListener); err != nil {
			tunnelListener.Close() // Close the listener if there's an error
			zap.L().Error("Failed to provide connections from stream", zap.Error(err))
			return
		}
		t.server.Shutdown(ctx)
	}()

	t.Logger.Info("Starting TunnelClient server", zap.String("addr", t.server.Addr))

	return t.server.Serve(tunnelListener)
}

func (t *TunnelClient) provideConnsFromStream(stream *connect.BidiStreamForClient[v1.TunnelMessage, v1.TunnelMessage], tunnelListener *tunnelListener) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Send an initial handshake message to the server to identify the client.
	if err := stream.Send(&v1.TunnelMessage{
		ClientId: t.clientID,
	}); err != nil {
		return fmt.Errorf("failed to send initial handshake message: %w", err)
	}

	t.Logger.Info("Starting to provide connections from stream")

	var sendMu sync.Mutex
	var connsMu sync.Mutex
	conns := make(map[int64]*tunnelConn)
	defer func() {
		t.Logger.Sugar().Debugf("Stream ending, closing all open connections for tunnel client, these were: %v", slices.Collect(maps.Keys(conns)))
		for _, conn := range conns {
			conn.Close() // Close all connections when the stream ends
		}
	}()

	// newConn must be called with connsMu held.
	newConn := func(connId int64) *tunnelConn {
		conn := newTunnelConn(connId)
		conns[connId] = conn

		go func() {
			for {
				select {
				case data := <-conn.writeChan:
					sendMu.Lock()
					err := stream.Send(&v1.TunnelMessage{
						ConnId: connId,
						Data:   data.bytes,
					})
					sendMu.Unlock()
					data.resultChan <- err
				case <-conn.closedChan:
					sendMu.Lock()
					if err := stream.Send(&v1.TunnelMessage{
						ConnId: connId,
						Data:   nil, // nil data indicates the connection is closed
						Close:  true,
					}); err != nil {
						sendMu.Unlock()
						return
					}
					sendMu.Unlock()
				}
			}
		}()
		t.Logger.Sugar().Debugf("Server established new connection with ID %d", connId)
		tunnelListener.ProvideConn(conn)
		return conn
	}

	// create an interval timer, used to efficiently provide a clock for reads, broken into 10 quantums
	// meaning cancellations are +/- 10ms from the deadline.
	ioDeadline := 100 * time.Millisecond
	ticker := time.NewTicker(ioDeadline / 10) // 10 ticks per deadline
	defer ticker.Stop()

	// Schedule healthcheck messages to be sent every 30 seconds.
	healthcheckTicker := time.NewTicker(heartbeatInterval)
	defer healthcheckTicker.Stop()
	go func() {
		sendCheck := func() {
			sendMu.Lock()
			defer sendMu.Unlock()
			stream.Send(&v1.TunnelMessage{
				ConnId: -1, // Healthcheck message, negative connection ID
				Data:   nil,
				Close:  false,
			})
		}
		sendCheck() // Send an initial healthcheck immediately

		for {
			select {
			case <-ctx.Done():
				return
			case <-healthcheckTicker.C:
				sendCheck() // Send healthcheck every 30 seconds
			}
		}
	}()

	// Loop and read messages from the stream.
	for {
		msg, err := stream.Receive()
		if err != nil {
			t.Logger.Sugar().Errorf("Failed to read from stream: %v", err)
			if err != io.EOF {
				return err
			}
			break
		}
		if msg.ConnId < 0 {
			// connection IDs below zero are healthcheck messages, we can ignore them.
			continue
		}

		connsMu.Lock()
		conn, ok := conns[msg.ConnId]
		if msg.Close {
			if ok {
				conn.Close() // Close the connection if it exists
			}
			delete(conns, msg.ConnId) // Remove the connection from the map
			connsMu.Unlock()
			continue
		} else if !ok {
			if int64(len(conns)) >= t.connectionLimit {
				connsMu.Unlock()
				return connect.NewError(connect.CodeResourceExhausted, os.ErrNotExist) // Return an error if the connection limit is reached
			}
			conn = newConn(msg.ConnId) // Create a new connection if it doesn't exist
		}
		connsMu.Unlock()

		if len(msg.Data) == 0 {
			continue // Skip empty data messages
		}

		quantums := 0
		select {
		case conn.readsChan <- msg.Data: // Try to write first.
		case <-conn.closedChan:
			// If the connection is closed, we can pass on the write.
		case <-ticker.C:
			// Close the channel if it's not able to respond to reads within the deadline (prevent head of line blocking).
			if quantums >= 10 {
				conn.Close()
			}
			quantums++
		}
	}

	return nil
}

type tunnelConn struct {
	connId int64

	readMu     sync.Mutex
	readBuffer []byte
	readsChan  chan []byte
	writeChan  chan struct {
		connId     int64
		bytes      []byte
		resultChan chan error
	}
	closedChanOnce sync.Once
	closedChan     chan struct{}

	deadlineChanMu     sync.Mutex    // mutex to protect the deadline channels
	readDeadlineTimer  *time.Timer   // timer for read deadline
	readDeadlineChan   chan struct{} // channel to signal read deadline, swapped on deadline set for a different channel
	writeDeadlineTimer *time.Timer   // timer for write deadline
	writeDeadlineChan  chan struct{} // channel to signal write deadline, swapped on deadline set for a different channel
}

var _ net.Conn = (*tunnelConn)(nil)

func newTunnelConn(connId int64) *tunnelConn {
	return &tunnelConn{
		connId:    connId,
		readsChan: make(chan []byte, 1), // Buffered channel to hold read data
		writeChan: make(chan struct {
			connId     int64
			bytes      []byte
			resultChan chan error
		}, 1),
		closedChan:        make(chan struct{}),
		readDeadlineChan:  make(chan struct{}),
		writeDeadlineChan: make(chan struct{}),
	}
}

func (c *tunnelConn) Read(b []byte) (n int, err error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	c.deadlineChanMu.Lock()
	deadlineChan := c.readDeadlineChan
	c.deadlineChanMu.Unlock()

	if len(c.readBuffer) == 0 {
		select {
		case <-c.closedChan:
			return 0, net.ErrClosed
		case <-deadlineChan:
			return 0, os.ErrDeadlineExceeded // net.Conn interface requires us to return os.ErrDeadlineExceeded on read deadline
		case data := <-c.readsChan:
			c.readBuffer = data
		}
	}

	if len(c.readBuffer) == 0 {
		return 0, io.EOF
	}

	n = copy(b, c.readBuffer)
	c.readBuffer = c.readBuffer[n:]

	if len(c.readBuffer) == 0 {
		c.readsChan <- nil // Signal that the read buffer is empty
	}

	return n, nil
}

func (c *tunnelConn) Write(b []byte) (n int, err error) {
	resultChan := make(chan error, 1)
	c.writeChan <- struct {
		connId     int64
		bytes      []byte
		resultChan chan error
	}{
		connId:     c.connId,
		bytes:      b,
		resultChan: resultChan,
	}

	c.deadlineChanMu.Lock()
	deadlineChan := c.writeDeadlineChan
	c.deadlineChanMu.Unlock()

	select {
	case err := <-resultChan:
		if err != nil {
			return 0, err
		}
		return len(b), nil
	case <-c.closedChan:
		return 0, net.ErrClosed
	case <-deadlineChan:
		return 0, os.ErrDeadlineExceeded // net.Conn interface requires us to return os.ErrDeadlineExceeded on write deadline
	}
}

func (c *tunnelConn) Close() error {
	c.closedChanOnce.Do(func() {
		close(c.closedChan) // Close the channel to signal that the connection is closed
	})
	c.deadlineChanMu.Lock()
	if c.readDeadlineTimer != nil {
		// More about cleaning up resources than logical correctness.
		c.readDeadlineTimer.Stop()
		c.readDeadlineTimer = nil
	}
	if c.writeDeadlineTimer != nil {
		// More about cleaning up resources than logical correctness.
		c.writeDeadlineTimer.Stop()
		c.writeDeadlineTimer = nil
	}
	c.deadlineChanMu.Unlock()
	return nil
}

func (c *tunnelConn) LocalAddr() net.Addr {
	// Return the local address of the tunnel connection
	return &net.TCPAddr{
		IP:   net.IPv4zero,
		Port: 0,
	}
}

func (c *tunnelConn) RemoteAddr() net.Addr {
	// Return the remote address of the tunnel connection
	return &net.TCPAddr{
		IP:   net.IPv4zero,
		Port: 0,
	}
}

func (c *tunnelConn) SetDeadline(t time.Time) error {
	c.SetReadDeadline(t)
	c.SetWriteDeadline(t)
	return nil
}

func (c *tunnelConn) SetReadDeadline(t time.Time) error {
	c.deadlineChanMu.Lock()
	defer c.deadlineChanMu.Unlock()
	if c.readDeadlineTimer != nil {
		c.readDeadlineTimer.Stop()
	}
	if t.IsZero() {
		c.readDeadlineTimer = nil
		return nil
	}
	c.readDeadlineTimer = time.AfterFunc(time.Until(t), func() {
		close(c.readDeadlineChan)
	})
	c.readDeadlineChan = make(chan struct{}) // Reset the channel to ensure it is ready for the next deadline
	return nil
}

func (c *tunnelConn) SetWriteDeadline(t time.Time) error {
	c.deadlineChanMu.Lock()
	defer c.deadlineChanMu.Unlock()
	if c.writeDeadlineTimer != nil {
		c.writeDeadlineTimer.Stop()
	}
	if t.IsZero() {
		c.writeDeadlineTimer = nil
		return nil
	}

	c.writeDeadlineTimer = time.AfterFunc(time.Until(t), func() {
		close(c.writeDeadlineChan)
	})
	c.writeDeadlineChan = make(chan struct{}) // Reset the channel to ensure it is ready for the next deadline
	return nil
}
