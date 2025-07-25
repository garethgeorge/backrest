package tunnel

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
)

// NewInsecureHttpClient creates a new HTTP client that allows insecure connections (HTTP/2 over plain TCP).
// This is useful for testing or when you don't need TLS.
func NewInsecureHttpClient() connect.HTTPClient {
	return &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(network, addr string, _ *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
			IdleConnTimeout: 300 * time.Second,
			ReadIdleTimeout: 60 * time.Second,
		},
	}
}

func NewWrappedStreamClient(stream *WrappedStream) connect.HTTPClient {
	return &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(network, addr string, _ *tls.Config) (net.Conn, error) {
				// Note that the addr doesn't matter at all when using a wrapped stream.
				return stream.Dial()
			},
		},
	}
}
