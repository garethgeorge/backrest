package syncapi

import (
	"context"
	"errors"
	"sync"

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
	snapshot := h.mgr.getSyncConfigSnapshot()
	if snapshot == nil {
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("sync is not configured"))
	}

	messagesToSend := make(chan *v1.SyncStateStreamItem, 1)

	publishStateForClient := func(client *SyncClient) {
		if client == nil {
			return
		}

		state, stateMessage := client.GetConnectionState()
		item := &v1.SyncStateStreamItem{
			PeerInstanceId: client.peer.InstanceId,
			PeerKeyid:      client.peer.Keyid,
			State:          state,
			StatusMessage:  stateMessage,
		}

		messagesToSend <- item
	}

	syncClients := h.mgr.GetSyncClients()

	var wg sync.WaitGroup

	for _, client := range syncClients {
		wg.Add(1)
		go func(c *SyncClient) {
			defer wg.Done()
			publishStateForClient(client)
			if !req.Msg.Subscribe {
				return
			}

			ch := c.SubscribeToSyncStateUpdates()
			defer c.UnsubscribeFromSyncStateUpdates(ch)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ch:
					publishStateForClient(c)
				}
			}
		}(client)
	}

	go func() {
		wg.Wait()
		close(messagesToSend)
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case item, ok := <-messagesToSend:
			if !ok {
				return nil // Channel closed, no more items to send
			}
			if err := stream.Send(item); err != nil {
				return connect.NewError(connect.CodeInternal, err)
			}
		}
	}
}
