package tunnel

import (
	"errors"
	"sync"
	"sync/atomic"

	"connectrpc.com/connect"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/hashicorp/go-multierror"
)

var ErrStreamClosed = errors.New("stream closed")

type stream interface {
	Send(item *v1.TunnelMessage) error
	Receive() (*v1.TunnelMessage, error)
	Close() error

	IsClient() bool
}

type clientStream struct {
	sendMu    sync.Mutex
	receiveMu sync.Mutex
	stream    *connect.BidiStreamForClient[v1.TunnelMessage, v1.TunnelMessage]
	closed    atomic.Bool
}

func (s *clientStream) Send(item *v1.TunnelMessage) error {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	if s.closed.Load() {
		return connect.NewError(connect.CodeFailedPrecondition, ErrStreamClosed)
	}
	return s.stream.Send(item)
}

func (s *clientStream) Receive() (*v1.TunnelMessage, error) {
	s.receiveMu.Lock()
	defer s.receiveMu.Unlock()
	if s.closed.Load() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, ErrStreamClosed)
	}
	return s.stream.Receive()
}

// Close closes the request side of the stream, allowing the server to finish processing.
// It will block if Receive or Send are in progress.
func (s *clientStream) Close() error {
	s.closed.Store(true)
	s.receiveMu.Lock()
	err := s.stream.CloseResponse()
	s.receiveMu.Unlock()
	s.sendMu.Lock()
	err = multierror.Append(err, s.stream.CloseRequest())
	s.sendMu.Unlock()
	return err
}

func (s *clientStream) IsClient() bool {
	return true
}

type serverStream struct {
	sendMu    sync.Mutex
	receiveMu sync.Mutex
	stream    *connect.BidiStream[v1.TunnelMessage, v1.TunnelMessage]
	closed    atomic.Bool
}

func (s *serverStream) Send(item *v1.TunnelMessage) error {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	if s.closed.Load() {
		return connect.NewError(connect.CodeFailedPrecondition, ErrStreamClosed)
	}
	return s.stream.Send(item)
}

func (s *serverStream) Receive() (*v1.TunnelMessage, error) {
	s.receiveMu.Lock()
	defer s.receiveMu.Unlock()
	if s.closed.Load() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, ErrStreamClosed)
	}
	return s.stream.Receive()
}

func (s *serverStream) Close() error {
	s.receiveMu.Lock()
	s.sendMu.Lock()
	s.closed.Store(true)
	return nil
}

func (s *serverStream) IsClient() bool {
	return false
}
