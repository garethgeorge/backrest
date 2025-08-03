package testutil

import (
	"net"
	"testing"
)

func AllocOpenBindAddr(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()

	// Get the port number from the listener
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("failed to split host and port: %v", err)
	}

	return "127.0.0.1:" + port
}
