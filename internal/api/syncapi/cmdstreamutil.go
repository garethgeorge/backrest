package syncapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	"github.com/garethgeorge/backrest/gen/go/v1sync"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
	"go.uber.org/zap"
)

type syncCommandStreamTrait interface {
	Send(item *v1sync.SyncStreamItem) error
	Receive() (*v1sync.SyncStreamItem, error)
}

var _ syncCommandStreamTrait = (*connect.BidiStream[v1sync.SyncStreamItem, v1sync.SyncStreamItem])(nil)
var _ syncCommandStreamTrait = (*connect.BidiStreamForClient[v1sync.SyncStreamItem, v1sync.SyncStreamItem])(nil)

type bidiSyncCommandStream struct {
	sendChan chan *v1sync.SyncStreamItem
	recvChan chan *v1sync.SyncStreamItem

	// done is closed exactly once when the stream is terminated. Readers can
	// observe termination by selecting on it; the cause (if any) is stored in
	// terminateErr.
	done         chan struct{}
	doneOnce     sync.Once
	terminateErr atomic.Pointer[error]
}

func newBidiSyncCommandStream() *bidiSyncCommandStream {
	return &bidiSyncCommandStream{
		sendChan: make(chan *v1sync.SyncStreamItem, 256),
		recvChan: make(chan *v1sync.SyncStreamItem, 1),
		done:     make(chan struct{}),
	}
}

func (s *bidiSyncCommandStream) Send(item *v1sync.SyncStreamItem) {
	select {
	case s.sendChan <- item:
	default:
		select {
		case s.sendChan <- item:
		case <-time.After(100 * time.Millisecond):
			s.SendErrorAndTerminate(NewSyncErrorDisconnected(errors.New("send channel is full, cannot send item")))
		}
	}
}

// SendErrorAndTerminate marks the stream as terminated. The first call wins:
// its err (if non-nil) is the one returned by Err. Subsequent calls are no-ops.
// Safe to call from any goroutine; non-blocking.
func (s *bidiSyncCommandStream) SendErrorAndTerminate(err error) {
	s.doneOnce.Do(func() {
		if err != nil {
			errCopy := err
			s.terminateErr.Store(&errCopy)
		}
		close(s.done)
	})
}

// Err returns the termination error, or nil if the stream has not been
// terminated or was terminated without an error.
func (s *bidiSyncCommandStream) Err() error {
	if errPtr := s.terminateErr.Load(); errPtr != nil {
		return *errPtr
	}
	return nil
}

// Done returns a channel that is closed when the stream is terminated.
func (s *bidiSyncCommandStream) Done() <-chan struct{} {
	return s.done
}

func (s *bidiSyncCommandStream) ReadChannel() chan *v1sync.SyncStreamItem {
	return s.recvChan
}

// ReceiveWithinDuration waits up to d for the next stream item. The returned
// error explains why no item arrived: ctx.Err() if ctx is cancelled, the
// stream's termination error (which may itself be nil) if the stream is
// terminated, or context.DeadlineExceeded if d elapses first. A nil item with
// a nil error means the stream was terminated cleanly with no cause.
func (s *bidiSyncCommandStream) ReceiveWithinDuration(ctx context.Context, d time.Duration) (*v1sync.SyncStreamItem, error) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case item, ok := <-s.recvChan:
		if !ok {
			return nil, s.Err()
		}
		return item, nil
	case <-s.done:
		return nil, s.Err()
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return nil, context.DeadlineExceeded
	}
}

// ConnectStream bridges the channel-based bidiSyncCommandStream to a real transport.
// It first performs a post-quantum KEM handshake on the raw transport to
// establish an encrypted session, then starts the send/recv pump loop over the
// encrypted channel. isInitiator must be true on the side that opens the
// connection (the client) and false on the side that accepts it (the server).
func (s *bidiSyncCommandStream) ConnectStream(ctx context.Context, stream syncCommandStreamTrait, isInitiator bool) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Perform the PQ KEM handshake on the raw transport before starting the pump.
	transport, err := establishEncryption(stream, isInitiator)
	if err != nil {
		// Signal termination so any goroutine parked in ReceiveWithinDuration
		// (e.g. runSync waiting for the handshake reply) wakes up immediately
		// instead of waiting out its full timeout.
		s.SendErrorAndTerminate(err)
		return err
	}

	go func() {
		defer close(s.recvChan)
		for {
			val, err := transport.Receive()
			if err != nil {
				s.SendErrorAndTerminate(NewSyncErrorDisconnected(fmt.Errorf("receiving item: %w", err)))
				return
			}
			select {
			case s.recvChan <- val:
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case item := <-s.sendChan:
			if item == nil {
				continue
			}
			if err := transport.Send(item); err != nil {
				if errors.Is(err, io.EOF) {
					err = fmt.Errorf("connection failed or dropped: %w", err)
				}
				s.SendErrorAndTerminate(err)
				return err
			}
		case <-s.done:
			return s.Err()
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// establishEncryption performs a post-quantum KEM handshake on the raw
// transport and returns an encrypted stream wrapper. The KEM ciphersuite is
// hard-pinned to TransportProtocolVersion and tied to the wire format.
//
// Flow: the initiator generates an ephemeral hybrid (ML-KEM-1024 + ECDH-P384)
// HPKE keypair and sends its public key. The responder encapsulates against
// it and replies with the encapsulation. Both sides derive an AES-256-GCM
// session key via the HPKE Export interface. The handshake (identity
// authentication) runs over the encrypted channel afterward.
//
// Initiators use nonce prefix 0x00; responders use 0x01 — fixed by role to
// avoid nonce reuse. isInitiator must be true on the connecting side (client)
// and false on the accepting side (server).
func establishEncryption(stream syncCommandStreamTrait, isInitiator bool) (syncCommandStreamTrait, error) {
	if isInitiator {
		recipient, pubBytes, err := cryptoutil.NewTransportRecipient()
		if err != nil {
			return nil, NewSyncErrorInternal(fmt.Errorf("generating ephemeral KEM key: %w", err))
		}

		if err := stream.Send(&v1sync.SyncStreamItem{
			Action: &v1sync.SyncStreamItem_EstablishSharedSecret{
				EstablishSharedSecret: &v1sync.SyncStreamItem_SyncEstablishSharedSecret{
					ProtocolVersion: cryptoutil.TransportProtocolVersion,
					KemPublicKey:    pubBytes,
				},
			},
		}); err != nil {
			return nil, NewSyncErrorDisconnected(fmt.Errorf("sending KEM public key: %w", err))
		}

		peerMsg, err := stream.Receive()
		if err != nil {
			return nil, NewSyncErrorDisconnected(fmt.Errorf("receiving KEM encapsulation: %w", err))
		}
		peerSecret := peerMsg.GetEstablishSharedSecret()
		if peerSecret == nil {
			return nil, NewSyncErrorProtocol(fmt.Errorf("expected KEM key exchange, got %T", peerMsg.GetAction()))
		}
		if peerSecret.GetProtocolVersion() != cryptoutil.TransportProtocolVersion {
			return nil, NewSyncErrorProtocol(fmt.Errorf("unsupported transport protocol version %d (this build requires v%d, post-quantum)", peerSecret.GetProtocolVersion(), cryptoutil.TransportProtocolVersion))
		}
		if len(peerSecret.GetKemEncapsulation()) == 0 {
			return nil, NewSyncErrorProtocol(errors.New("responder did not send KEM encapsulation"))
		}

		sess, err := recipient.Decapsulate(peerSecret.GetKemEncapsulation())
		if err != nil {
			return nil, NewSyncErrorProtocol(fmt.Errorf("decapsulating KEM: %w", err))
		}

		zap.L().Info("encrypted sync session established (initiator)")
		return newEncryptedStream(stream, sess), nil
	}

	peerMsg, err := stream.Receive()
	if err != nil {
		return nil, NewSyncErrorDisconnected(fmt.Errorf("receiving KEM public key: %w", err))
	}
	peerSecret := peerMsg.GetEstablishSharedSecret()
	if peerSecret == nil {
		return nil, NewSyncErrorProtocol(fmt.Errorf("expected KEM key exchange, got %T", peerMsg.GetAction()))
	}
	if peerSecret.GetProtocolVersion() != cryptoutil.TransportProtocolVersion {
		return nil, NewSyncErrorProtocol(fmt.Errorf("unsupported transport protocol version %d (this build requires v%d, post-quantum)", peerSecret.GetProtocolVersion(), cryptoutil.TransportProtocolVersion))
	}
	if len(peerSecret.GetKemPublicKey()) == 0 {
		return nil, NewSyncErrorProtocol(errors.New("initiator did not send KEM public key"))
	}

	enc, sess, err := cryptoutil.EncapsulateToTransport(peerSecret.GetKemPublicKey())
	if err != nil {
		return nil, NewSyncErrorProtocol(fmt.Errorf("encapsulating to KEM public key: %w", err))
	}

	if err := stream.Send(&v1sync.SyncStreamItem{
		Action: &v1sync.SyncStreamItem_EstablishSharedSecret{
			EstablishSharedSecret: &v1sync.SyncStreamItem_SyncEstablishSharedSecret{
				ProtocolVersion:  cryptoutil.TransportProtocolVersion,
				KemEncapsulation: enc,
			},
		},
	}); err != nil {
		return nil, NewSyncErrorDisconnected(fmt.Errorf("sending KEM encapsulation: %w", err))
	}

	zap.L().Info("encrypted sync session established (responder)")
	return newEncryptedStream(stream, sess), nil
}
