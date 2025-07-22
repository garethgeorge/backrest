package tunnel

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"connectrpc.com/connect"
	"github.com/garethgeorge/backrest/gen/go/v1sync"
	"github.com/hashicorp/go-multierror"
	"google.golang.org/protobuf/proto"
)

var ErrStreamClosed = errors.New("stream closed")

type stream interface {
	Send(item *v1sync.TunnelMessage) error
	Receive() (*v1sync.TunnelMessage, error)
	Close() error
}

type cryptedStream struct {
	stream
	crypt
}

func newCryptedStream(s stream, secret []byte) *cryptedStream {
	return &cryptedStream{
		stream: s,
		crypt: crypt{
			secret: secret,
		},
	}
}

func (cs *cryptedStream) Send(item *v1sync.TunnelMessage) error {
	if item.Data == nil {
		return cs.stream.Send(item)
	}
	bytes, err := proto.Marshal(item)
	if err != nil {
		return err
	}
	enc, err := cs.Encrypt(bytes)
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}
	return cs.stream.Send(&v1sync.TunnelMessage{
		Encrypted: enc,
	})
}

func (cs *cryptedStream) Receive() (*v1sync.TunnelMessage, error) {
	msg, err := cs.stream.Receive()
	if err != nil {
		return nil, err
	}
	dec, err := cs.Decrypt(msg.Encrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	var tm v1sync.TunnelMessage
	if err := proto.Unmarshal(dec, &tm); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	if len(tm.Encrypted) != 0 {
		return nil, fmt.Errorf("unexpected encrypted field in decrypted message")
	}
	return &tm, nil
}

type clientStream struct {
	sendMu    sync.Mutex
	receiveMu sync.Mutex
	stream    *connect.BidiStreamForClient[v1sync.TunnelMessage, v1sync.TunnelMessage]
	closed    atomic.Bool
}

func (s *clientStream) Send(item *v1sync.TunnelMessage) error {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	if s.closed.Load() {
		return connect.NewError(connect.CodeFailedPrecondition, ErrStreamClosed)
	}
	return s.stream.Send(item)
}

func (s *clientStream) Receive() (*v1sync.TunnelMessage, error) {
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
	var err error
	if e := s.stream.CloseResponse(); e != nil {
		err = multierror.Append(err, e)
	}
	s.receiveMu.Unlock()
	s.sendMu.Lock()
	if e := s.stream.CloseRequest(); e != nil {
		err = multierror.Append(err, e)
	}
	s.sendMu.Unlock()
	return err
}

type serverStream struct {
	sendMu    sync.Mutex
	receiveMu sync.Mutex
	stream    *connect.BidiStream[v1sync.TunnelMessage, v1sync.TunnelMessage]
	closed    atomic.Bool
}

func (s *serverStream) Send(item *v1sync.TunnelMessage) error {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	if s.closed.Load() {
		return connect.NewError(connect.CodeFailedPrecondition, ErrStreamClosed)
	}
	return s.stream.Send(item)
}

func (s *serverStream) Receive() (*v1sync.TunnelMessage, error) {
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
