package tunnel

import (
	"crypto/tls"
	"net"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/net/http2"
)

// tunnelListener is a net.Listener that makes it possible to feed net.Conn instances into it.
type tunnelListener struct {
	doClose  sync.Once
	isOpen   chan struct{}
	nextConn chan net.Conn
}

func newTunnelListener() *tunnelListener {
	return &tunnelListener{
		isOpen:   make(chan struct{}),
		nextConn: make(chan net.Conn),
	}
}

var _ net.Listener = (*tunnelListener)(nil)

func (l *tunnelListener) Accept() (net.Conn, error) {
	select {
	case <-l.isOpen:
		// If the listener is closed, we don't accept new connections.
		return nil, net.ErrClosed
	case conn := <-l.nextConn:
		return conn, nil
	}
}

func (l *tunnelListener) Close() error {
	l.doClose.Do(func() {
		close(l.isOpen) // Close the channel to signal that the listener is closed
	})
	return nil
}

func (l *tunnelListener) Addr() net.Addr {
	return &net.TCPAddr{
		IP:   net.IPv4zero,
		Port: 0, // Port is not used in this implementation
	}
}

func (l *tunnelListener) ProvideConn(conn net.Conn) {
	select {
	case <-l.isOpen:
		// If the listener is closed, we don't accept new connections.
		conn.Close()
		return
	case l.nextConn <- conn:
	}
}

// newInsecureClient creates a new HTTP client that allows insecure connections (HTTP/2 over plain TCP).
// This is useful for testing or when you don't need TLS.
func newInsecureClient() *http.Client {
	return &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(network, addr string, _ *tls.Config) (net.Conn, error) {
				zap.L().Sugar().Debugf("Dialing %s on %s", network, addr)
				return net.Dial(network, addr)
			},
			IdleConnTimeout: 300 * time.Second,
			ReadIdleTimeout: 60 * time.Second,
		},
	}
}
