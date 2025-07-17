package tunnel

import (
	"net"
	"sync"
)

// tunnelListener is a net.Listener that makes it possible to feed net.Conn instances into it.
type ConnectionProvider struct {
	doClose  sync.Once
	isOpen   chan struct{}
	nextConn chan net.Conn
}

var _ net.Listener = (*ConnectionProvider)(nil)

func NewConnectionProvider(bufferSize int) *ConnectionProvider {
	return &ConnectionProvider{
		isOpen:   make(chan struct{}),
		nextConn: make(chan net.Conn, bufferSize), // Buffered channel to hold incoming connections
	}
}

var _ net.Listener = (*ConnectionProvider)(nil)

func (l *ConnectionProvider) Accept() (net.Conn, error) {
	select {
	case <-l.isOpen:
		// If the listener is closed, we don't accept new connections.
		return nil, net.ErrClosed
	case conn := <-l.nextConn:
		return conn, nil
	}
}

func (l *ConnectionProvider) Close() error {
	l.doClose.Do(func() {
		close(l.isOpen) // Close the channel to signal that the listener is closed
	})
	return nil
}

func (l *ConnectionProvider) Addr() net.Addr {
	return &net.TCPAddr{
		IP:   net.IPv4zero,
		Port: 0, // Port is not used in this implementation
	}
}

func (l *ConnectionProvider) ProvideConn(conn net.Conn) {
	select {
	case <-l.isOpen:
		// If the listener is closed, we don't accept new connections.
		conn.Close()
		return
	case l.nextConn <- conn:
		// Successfully provided the connection
	}
}
