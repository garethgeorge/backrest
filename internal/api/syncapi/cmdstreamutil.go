package syncapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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
	sendChan             chan *v1sync.SyncStreamItem
	recvChan             chan *v1sync.SyncStreamItem
	terminateWithErrChan chan error
}

func newBidiSyncCommandStream() *bidiSyncCommandStream {
	return &bidiSyncCommandStream{
		sendChan:             make(chan *v1sync.SyncStreamItem, 256),
		recvChan:             make(chan *v1sync.SyncStreamItem, 1),
		terminateWithErrChan: make(chan error, 1),
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

func (s *bidiSyncCommandStream) SendErrorAndTerminate(err error) {
	select {
	case s.terminateWithErrChan <- err:
	default:
	}
}

func (s *bidiSyncCommandStream) ReadChannel() chan *v1sync.SyncStreamItem {
	return s.recvChan
}

func (s *bidiSyncCommandStream) ReceiveWithinDuration(d time.Duration) *v1sync.SyncStreamItem {
	select {
	case item := <-s.recvChan:
		return item
	case <-time.After(d):
		return nil
	}
}

// ConnectStream bridges the channel-based bidiSyncCommandStream to a real transport.
// It first performs an ECDH key exchange on the raw transport to establish an encrypted
// session, then starts the send/recv pump loop over the encrypted channel.
func (s *bidiSyncCommandStream) ConnectStream(ctx context.Context, stream syncCommandStreamTrait) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Perform ECDH key exchange on the raw transport before starting the pump.
	transport, err := establishEncryption(stream)
	if err != nil {
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
		case err := <-s.terminateWithErrChan:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// establishEncryption performs an ECDH key exchange on the raw transport and
// returns an encrypted stream wrapper. Each side generates an ephemeral ECDH P-256
// key pair, exchanges public keys, and derives a shared AES-256-GCM session key.
// The handshake (identity authentication) runs over the encrypted channel afterward.
func establishEncryption(stream syncCommandStreamTrait) (syncCommandStreamTrait, error) {
	keyPair, err := cryptoutil.GenerateECDHKeyPair()
	if err != nil {
		return nil, NewSyncErrorInternal(fmt.Errorf("generating ephemeral ECDH key: %w", err))
	}

	// Send our ephemeral ECDH public key
	if err := stream.Send(&v1sync.SyncStreamItem{
		Action: &v1sync.SyncStreamItem_EstablishSharedSecret{
			EstablishSharedSecret: &v1sync.SyncStreamItem_SyncEstablishSharedSecret{
				EcdhPublicKey: keyPair.Public.Bytes(),
			},
		},
	}); err != nil {
		return nil, NewSyncErrorProtocol(fmt.Errorf("sending ECDH public key: %w", err))
	}

	// Receive the peer's ephemeral ECDH public key
	peerMsg, err := stream.Receive()
	if err != nil {
		return nil, NewSyncErrorProtocol(fmt.Errorf("receiving ECDH public key: %w", err))
	}
	peerSecret := peerMsg.GetEstablishSharedSecret()
	if peerSecret == nil {
		return nil, NewSyncErrorProtocol(fmt.Errorf("expected ECDH key exchange, got %T", peerMsg.GetAction()))
	}

	peerECDHPub, err := cryptoutil.ParseECDHPublicKey(peerSecret.GetEcdhPublicKey())
	if err != nil {
		return nil, NewSyncErrorProtocol(fmt.Errorf("parsing peer ECDH public key: %w", err))
	}

	// Derive AES-256-GCM session key
	gcm, err := cryptoutil.DeriveSessionKey(keyPair.Private, peerECDHPub)
	if err != nil {
		return nil, NewSyncErrorProtocol(fmt.Errorf("deriving session key: %w", err))
	}

	// Determine nonce direction: side with smaller public key uses prefix 0x00
	localIsSmaller := bytes.Compare(keyPair.Public.Bytes(), peerECDHPub.Bytes()) < 0

	zap.L().Info("encrypted sync session established")

	return newEncryptedStream(stream, gcm, localIsSmaller), nil
}
