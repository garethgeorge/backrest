package syncapi

import (
	"context"
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

func (h *BackrestSyncStateHandler) GetPeerSyncStatesStream(ctx context.Context, req *connect.Request[v1.SyncStateStreamRequest], stream *connect.ServerStream[v1.PeerState]) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	// Subscribe to the peer state changes
	onStateChangeChan := h.mgr.peerStateManager.OnStateChanged().Subscribe()

	messagesToSend := make(chan *v1.PeerState, 100) // Buffered channel to allow sending items without blocking

	peerStateToMsg := func(peerState *PeerState) *v1.PeerState {
		return &v1.PeerState{
			PeerInstanceId: peerState.InstanceID,
			PeerKeyid:      peerState.KeyID,
			State:          peerState.ConnectionState,
			StatusMessage:  peerState.ConnectionStateMessage,
		}
	}

	sendAll := func(config *v1.Config) {
		for _, peer := range config.GetMultihost().GetAuthorizedClients() {
			peerState := h.mgr.peerStateManager.GetPeerState(peer.Keyid)
			if peerState == nil {
				continue // Skip if no state is available for this peer
			}
			messagesToSend <- peerStateToMsg(peerState)
		}
		for _, peer := range config.GetMultihost().GetKnownHosts() {
			peerState := h.mgr.peerStateManager.GetPeerState(peer.Keyid)
			if peerState == nil {
				continue // Skip if no state is available for this peer
			}
			messagesToSend <- peerStateToMsg(peerState)
		}
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
				case messagesToSend <- peerStateToMsg(peerState):
				default:
					// If the channel is full, wait for 100 milliseconds before cancelling
					select {
					case messagesToSend <- peerStateToMsg(peerState):
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
