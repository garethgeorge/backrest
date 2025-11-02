package syncapi

import (
	"context"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1sync"
	"github.com/garethgeorge/backrest/gen/go/v1sync/v1syncconnect"
)

type BackrestSyncStateHandler struct {
	v1syncconnect.UnimplementedBackrestSyncStateServiceHandler
	mgr *SyncManager
}

var _ v1syncconnect.BackrestSyncStateServiceHandler = &BackrestSyncStateHandler{}

func NewBackrestSyncStateHandler(mgr *SyncManager) *BackrestSyncStateHandler {
	return &BackrestSyncStateHandler{
		mgr: mgr,
	}
}

func (h *BackrestSyncStateHandler) GetPeerSyncStatesStream(ctx context.Context, req *connect.Request[v1sync.SyncStateStreamRequest], stream *connect.ServerStream[v1sync.PeerState]) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	// Subscribe to the peer state changes
	onStateChangeChan := h.mgr.peerStateManager.OnStateChanged().Subscribe()

	messagesToSend := make(chan *v1sync.PeerState, 100) // Buffered channel to allow sending items without blocking

	sendAllInList := func(peers []*v1.Multihost_Peer) {
		for _, peerState := range peers {
			state := h.mgr.peerStateManager.GetPeerState(peerState.Keyid)
			if state == nil {
				continue // Skip if the peer state is not found
			}
			messagesToSend <- peerStateToProto(state)
		}
	}

	sendAll := func(config *v1.Config) {
		sendAllInList(config.GetMultihost().GetKnownHosts())
		sendAllInList(config.GetMultihost().GetAuthorizedClients())
	}

	// Start a goroutine to listen for state changes and send them to the stream
	go func() {
		config, err := h.mgr.configMgr.Get()
		if err != nil {
			cancel(err)
			return
		}

		// Send initial states for all known hosts and authorized clients
		sendAll(config)

		if !req.Msg.Subscribe {
			cancel(nil)
		}

		defer h.mgr.peerStateManager.OnStateChanged().Unsubscribe(onStateChangeChan)
		for {
			select {
			case <-ctx.Done():
				return // Exit the goroutine if the context is done

			case peerState, ok := <-onStateChangeChan:
				if !ok {
					return // Channel closed, exit the goroutine
				}

				select {
				case messagesToSend <- peerStateToProto(peerState):
				default:
					// If the channel is full, wait for 100 milliseconds before cancelling
					select {
					case messagesToSend <- peerStateToProto(peerState):
					case <-time.After(100 * time.Millisecond):
						cancel(nil)
						return
					}
				}
			}
		}
	}()

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
