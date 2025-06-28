package syncapi

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1/v1connect"
)

type BackrestSyncStateHandler struct {
	v1connect.UnimplementedBackrestSyncStateServiceHandler
	mgr *SyncManager
}

var _ v1connect.BackrestSyncStateServiceHandler = &BackrestSyncStateHandler{}

func NewBackrestSyncStateHandler(mgr *SyncManager) *BackrestSyncStateHandler {
	return &BackrestSyncStateHandler{
		mgr: mgr,
	}
}

func (h *BackrestSyncStateHandler) GetKnownHostSyncStateStream(ctx context.Context, req *connect.Request[v1.SyncStateStreamRequest], stream *connect.ServerStream[v1.SyncStateStreamItem]) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	snapshot := h.mgr.getSyncConfigSnapshot()
	if snapshot == nil {
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("sync is not configured"))
	}

	messagesToSend := make(chan *v1.SyncStateStreamItem, 100) // Buffered channel to allow sending items without blocking

	peerStateToMsg := func(peerState *PeerState) *v1.SyncStateStreamItem {
		return &v1.SyncStateStreamItem{
			PeerInstanceId: peerState.InstanceID,
			PeerKeyid:      peerState.KeyID,
			State:          peerState.ConnectionState,
			StatusMessage:  peerState.ConnectionStateMessage,
		}
	}

	// Start a goroutine to listen for state changes and send them to the stream
	go func() {
		onStateChangeChan := h.mgr.knownHostPeerStates.onStateChanged.Subscribe()
		defer h.mgr.knownHostPeerStates.onStateChanged.Unsubscribe(onStateChangeChan)
		for {
			select {
			case <-ctx.Done():
				return // Exit the goroutine if the context is done
			case peerState, ok := <-onStateChangeChan:
				if !ok {
					return // Channel closed, exit the goroutine
				}
				select {
				case messagesToSend <- peerStateToMsg(peerState):
				default:
					// If the channel is full, wait for 100 milliseconds before cancelling
					select {
					case messagesToSend <- peerStateToMsg(peerState):
					case <-time.After(100 * time.Millisecond):
						cancel()
						return
					}
				}
			}
		}
	}()

	// Send the initial state of all known host peer states
	for _, peerState := range h.mgr.knownHostPeerStates.GetAllPeerStates() {
		if err := stream.Send(peerStateToMsg(peerState)); err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case item, ok := <-messagesToSend:
			if !ok {
				return nil
			}
			if err := stream.Send(item); err != nil {
				return connect.NewError(connect.CodeInternal, err)
			}
		}
	}
}
