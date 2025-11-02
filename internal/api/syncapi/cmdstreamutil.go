package syncapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"connectrpc.com/connect"
	"github.com/garethgeorge/backrest/gen/go/v1sync"
)

type syncCommandStreamTrait interface {
	Send(item *v1sync.SyncStreamItem) error
	Receive() (*v1sync.SyncStreamItem, error)
}

var _ syncCommandStreamTrait = (*connect.BidiStream[v1sync.SyncStreamItem, v1sync.SyncStreamItem])(nil)          // Ensure that connect.BidiStream implements syncCommandStreamTrait
var _ syncCommandStreamTrait = (*connect.BidiStreamForClient[v1sync.SyncStreamItem, v1sync.SyncStreamItem])(nil) // Ensure that connect.BidiStreamForClient implements syncCommandStreamTrait

type bidiSyncCommandStream struct {
	sendChan             chan *v1sync.SyncStreamItem
	recvChan             chan *v1sync.SyncStreamItem
	terminateWithErrChan chan error
}

func newBidiSyncCommandStream() *bidiSyncCommandStream {
	return &bidiSyncCommandStream{
		sendChan:             make(chan *v1sync.SyncStreamItem, 64), // Buffered channel to allow sending items without blocking
		recvChan:             make(chan *v1sync.SyncStreamItem, 1),
		terminateWithErrChan: make(chan error, 1),
	}
}

func (s *bidiSyncCommandStream) Send(item *v1sync.SyncStreamItem) {
	select {
	case s.sendChan <- item:
	default:
		// Try again with a timeout, if it fails, send an error to terminate the stream
		select {
		case s.sendChan <- item:
		case <-time.After(100 * time.Millisecond):
			s.SendErrorAndTerminate(NewSyncErrorDisconnected(errors.New("send channel is full, cannot send item")))
		}
	}
}

// SendErrorAndTerminate sends an error to the termination channel.
// If the error is nil, it terminates only.
func (s *bidiSyncCommandStream) SendErrorAndTerminate(err error) {
	select {
	case s.terminateWithErrChan <- err:
	default:
		// If the channel is full, we can't send the error, so we just ignore it.
		// This is a best-effort termination.
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
		return nil // Return nil if no item is received within the duration
	}
}

func (s *bidiSyncCommandStream) ConnectStream(ctx context.Context, stream syncCommandStreamTrait) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		for ctx.Err() == nil {
			if val, err := stream.Receive(); err != nil {
				s.SendErrorAndTerminate(NewSyncErrorDisconnected(fmt.Errorf("receiving item: %w", err)))
				break
			} else {
				s.recvChan <- val
			}
		}
		close(s.recvChan)
	}()

	for {
		select {
		case item := <-s.sendChan:
			if item == nil {
				continue
			}
			if err := stream.Send(item); err != nil {
				if errors.Is(err, io.EOF) {
					err = fmt.Errorf("connection failed or dropped: %w", err)
				}
				s.SendErrorAndTerminate(err)
				return err
			}
		case err := <-s.terminateWithErrChan:
			return err // Terminate the stream with the error or nil if no error was sent
		case <-ctx.Done():
			// Context is done, we should stop processing.
			return ctx.Err()
		}
	}
}
