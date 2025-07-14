package tunnel

import "net/http"

// Okay we need implementations of two things, a net.Listener and a net.Conn over a socket.

// TunnelHandler is a server that implements the tunnel protocol, allows clients to connect and, when connected, will allow sending requests to connected clients.
type TunnelHandler struct {
}

// TunnelClient is a client that connects to a TunnelHandler and provides an http.Server that requests can be sent to.
type TunnelClient struct {
	server *http.Server
}

func NewTunnelClient(server *http.Server) *TunnelClient {
	return &TunnelClient{
		server: server,
	}
}

// Serve starts the HTTP server for the tunnel client and connects to the TunnelHandler that will be it's client.
func (t *TunnelClient) Serve() {
	return t.server
}
