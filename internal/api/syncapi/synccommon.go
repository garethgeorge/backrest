package syncapi

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
	"go.uber.org/zap"
)

var maxSignatureAge = 5 * time.Minute

func runSync(
	ctx context.Context,
	localInstanceID string,
	localKey *cryptoutil.PrivateKey,
	commandStream *bidiSyncCommandStream,
	handler syncSessionHandler,
) error {
	peer := PeerFromContext(ctx)
	peerPublicKey := PeerPublicKeyFromContext(ctx)
	if peer == nil || peerPublicKey == nil {
		return NewSyncErrorAuth(fmt.Errorf("peer not found in context, ensure authentication middleware is applied before sync handlers"))
	}

	if err := handler.OnConnectionEstablished(ctx, commandStream, peer); err != nil {
		return err
	}

	defer func() {
		if err := handler.OnConnectionClosed(ctx, commandStream); err != nil {
			zap.L().Error("error handling connection closed", zap.Error(err))
		}
	}()

	for item := range commandStream.ReadChannel() {
		switch item.GetAction().(type) {
		case *v1.SyncStreamItem_Heartbeat:
			if err := handler.HandleHeartbeat(ctx, commandStream, item.GetHeartbeat()); err != nil {
				return fmt.Errorf("handling heartbeat: %w", err)
			}
		case *v1.SyncStreamItem_DiffOperations:
			if err := handler.HandleDiffOperations(ctx, commandStream, item.GetDiffOperations()); err != nil {
				return fmt.Errorf("handling diff operations: %w", err)
			}
		case *v1.SyncStreamItem_SendOperations:
			if err := handler.HandleSendOperations(ctx, commandStream, item.GetSendOperations()); err != nil {
				return fmt.Errorf("handling send operations: %w", err)
			}
		case *v1.SyncStreamItem_SendConfig:
			if err := handler.HandleSendConfig(ctx, commandStream, item.GetSendConfig()); err != nil {
				return fmt.Errorf("handling send config: %w", err)
			}
		case *v1.SyncStreamItem_SetConfig:
			if err := handler.HandleSetConfig(ctx, commandStream, item.GetSetConfig()); err != nil {
				return fmt.Errorf("handling set config: %w", err)
			}
		case *v1.SyncStreamItem_ListResources:
			if err := handler.HandleListResources(ctx, commandStream, item.GetListResources()); err != nil {
				return fmt.Errorf("handling list resources: %w", err)
			}
		case *v1.SyncStreamItem_Throttle:
			if err := handler.HandleThrottle(ctx, commandStream, item.GetThrottle()); err != nil {
				return fmt.Errorf("handling throttle: %w", err)
			}
		case *v1.SyncStreamItem_GetLog:
			if err := handler.HandleGetLog(ctx, commandStream, item.GetGetLog()); err != nil {
				return fmt.Errorf("handling get log: %w", err)
			}
		case *v1.SyncStreamItem_SendLogData:
			if err := handler.HandleSendLogData(ctx, commandStream, item.GetSendLogData()); err != nil {
				return fmt.Errorf("handling send log data: %w", err)
			}
		default:
			return NewSyncErrorProtocol(fmt.Errorf("unknown action type %T in sync stream item", item.GetAction()))
		}
	}
	return nil
}

func tryReceiveWithinDuration(ctx context.Context, receiveChan chan *v1.SyncStreamItem, receiveErrChan chan error, timeout time.Duration) (*v1.SyncStreamItem, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	select {
	case item := <-receiveChan:
		return item, nil
	case err := <-receiveErrChan:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// sendHeartbeats sends a heartbeat message to the stream at regular intervals.
// This is useful for keeping the connection alive and ensuring that the peer is still responsive.
func sendHeartbeats(ctx context.Context, stream *bidiSyncCommandStream, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stream.Send(&v1.SyncStreamItem{
				Action: &v1.SyncStreamItem_Heartbeat{
					Heartbeat: &v1.SyncStreamItem_SyncActionHeartbeat{},
				},
			})
		case <-ctx.Done():
			return
		}
	}
}

// syncSessionHandler is a stateful handler for the messages within the context of a sync stream session.
// the handler does not need to be thread safe as it is guaranteed to be called from a single thread.
type syncSessionHandler interface {
	// OnConnectionEstablished is called when a new sync connection is established, provides the peer information once identified.
	OnConnectionEstablished(ctx context.Context, stream *bidiSyncCommandStream, peer *v1.Multihost_Peer) error
	// OnConnectionClosed is called when the sync connection is closed, e.g. when the peer disconnects. Guaranteed to be called after OnConnectionEstablished.
	OnConnectionClosed(ctx context.Context, stream *bidiSyncCommandStream) error
	// Handle* methods are called for each action type in the sync stream.
	HandleHeartbeat(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionHeartbeat) error
	HandleDiffOperations(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionDiffOperations) error
	HandleSendOperations(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionSendOperations) error
	HandleSendConfig(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionSendConfig) error
	HandleSetConfig(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionSetConfig) error
	HandleListResources(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionListResources) error
	HandleThrottle(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionThrottle) error
	HandleGetLog(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionGetLog) error
	HandleSendLogData(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionSendLogData) error
}

type unimplementedSyncSessionHandler struct{}

func (h *unimplementedSyncSessionHandler) OnConnectionEstablished(ctx context.Context, stream *bidiSyncCommandStream, peer *v1.Multihost_Peer) error {
	return nil // no-op by default.
}

func (h *unimplementedSyncSessionHandler) OnConnectionClosed(ctx context.Context, stream *bidiSyncCommandStream) error {
	return nil // no-op by default.
}

func (h *unimplementedSyncSessionHandler) HandleHeartbeat(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionHeartbeat) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleHeartbeat not implemented"))
}

func (h *unimplementedSyncSessionHandler) HandleDiffOperations(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionDiffOperations) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleDiffOperations not implemented"))
}

func (h *unimplementedSyncSessionHandler) HandleSendOperations(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionSendOperations) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleSendOperations not implemented"))
}

func (h *unimplementedSyncSessionHandler) HandleSendConfig(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionSendConfig) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleSendConfig not implemented"))
}

func (h *unimplementedSyncSessionHandler) HandleSetConfig(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionSetConfig) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleSetConfig not implemented"))
}

func (h *unimplementedSyncSessionHandler) HandleListResources(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionListResources) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleListResources not implemented"))
}

func (h *unimplementedSyncSessionHandler) HandleThrottle(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionThrottle) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleThrottle not implemented"))
}

func (h *unimplementedSyncSessionHandler) HandleGetLog(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionGetLog) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleGetLog not implemented"))
}

func (h *unimplementedSyncSessionHandler) HandleSendLogData(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionSendLogData) error {
	return NewSyncErrorProtocol(fmt.Errorf("HandleSendLogData not implemented"))
}
