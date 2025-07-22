package tunnel

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"
	"golang.org/x/net/http2"
)

// NewInsecureHttpClient creates a new HTTP client that allows insecure connections (HTTP/2 over plain TCP).
// This is useful for testing or when you don't need TLS.
func NewInsecureHttpClient() *http.Client {
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
