package syncapi

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/garethgeorge/backrest/gen/go/types"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1/v1connect"
	"github.com/garethgeorge/backrest/internal/oplog"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
)

type SyncClient struct {
	localInstanceID string
	oplog           *oplog.OpLog
	client          v1connect.BackrestSyncServiceClient
	reconnectDelay  time.Duration
	l               *zap.Logger
	connectionState v1.SyncConnectionState
}

func newInsecureClient() *http.Client {
	return &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(network, addr string, _ *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
			IdleConnTimeout: 300 * time.Second,
			ReadIdleTimeout: 60 * time.Second,
		},
	}
}

func NewSyncClient(localInstanceID, remoteInstanceURL string, oplog *oplog.OpLog) *SyncClient {
	urlToUse := backrestRemoteUrlToHTTPUrl(remoteInstanceURL)
	client := v1connect.NewBackrestSyncServiceClient(
		newInsecureClient(),
		urlToUse,
	)

	return &SyncClient{
		localInstanceID: localInstanceID,
		reconnectDelay:  60 * time.Second,
		client:          client,
		oplog:           oplog,
		l:               zap.L().Named("syncclient").With(zap.String("peer", remoteInstanceURL)),
	}
}

func (c *SyncClient) RunSyncForRepos(ctx context.Context, repos []*v1.Repo) {
	for {
		if ctx.Err() != nil {
			return
		}

		lastConnect := time.Now()
		c.setStatusForRepos(repos, v1.SyncStreamItem_CONNECTION_STATE_PENDING, "connecting")
		if err := c.runSyncForReposInternal(ctx, repos); err != nil {
			c.l.Warn("sync error", zap.Error(err))
			c.setStatusForRepos(repos, v1.SyncStreamItem_CONNECTION_STATE_DISCONNECTED, err.Error())
		} else {
			c.setStatusForRepos(repos, v1.SyncStreamItem_CONNECTION_STATE_DISCONNECTED, "connection closed")
		}

		delay := c.reconnectDelay - time.Since(lastConnect)
		c.l.Info("lost connection, retrying after delay", zap.Duration("delay", delay))
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return
		}
	}
}

func (c *SyncClient) runSyncForReposInternal(ctx context.Context, repos []*v1.Repo) error {
	c.l.Info("connecting to sync server")
	stream := c.client.Sync(ctx)

	ctx, cancelWithError := context.WithCancelCause(ctx)

	receive := make(chan *v1.SyncStreamItem, 1)
	send := make(chan *v1.SyncStreamItem, 100)

	go func() {
		for {
			item, err := stream.Receive()
			if err != nil {
				c.l.Debug("receive error from sync stream, this is typically due to connection loss", zap.Error(err))
				close(receive)
				return
			}
			receive <- item
		}
	}()

	// Broadcast initial packet containing the protocol version and instance ID.
	if err := stream.Send(&v1.SyncStreamItem{
		Action: &v1.SyncStreamItem_Handshake{
			Handshake: &v1.SyncStreamItem_SyncActionHandshake{
				ProtocolVersion: SyncProtocolVersion,
				InstanceId:      c.localInstanceID,
			},
		},
	}); err != nil {
		return err
	}

	// Wait for the handshake packet from the server.
	serverInstanceID := ""
	if msg, ok := <-receive; ok {
		handshake := msg.GetHandshake()
		if handshake == nil {
			return connect.NewError(connect.CodeInvalidArgument, errors.New("handshake packet must be sent first"))
		}

		serverInstanceID = handshake.GetInstanceId()
		if serverInstanceID == "" {
			return connect.NewError(connect.CodeInvalidArgument, errors.New("instance ID is required"))
		}

		if handshake.GetProtocolVersion() != SyncProtocolVersion {
			return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("unsupported peer protocol version, got %d, expected %d", handshake.GetProtocolVersion(), SyncProtocolVersion))
		}
	} else {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("no packets received"))
	}

	// Send the list of repos to connect to the server.
	for _, repo := range repos {
		if err := stream.Send(&v1.SyncStreamItem{
			Action: &v1.SyncStreamItem_ConnectRepo{
				ConnectRepo: &v1.SyncStreamItem_SyncActionConnectRepo{
					RepoId: repo.GetId(),
				},
			},
		}); err != nil {
			return fmt.Errorf("send connect repo: %w", err)
		}
	}

	didSubscribeOplog := false

	oplogSubscription := func(ops []*v1.Operation, event oplog.OperationEvent) {
		var opsToForward []*v1.Operation
		for _, op := range ops {
			if connInfo, ok := c.connectedRepos[op.GetRepoId()]; ok && connInfo.ConnectionState == v1.SyncStreamItem_CONNECTION_STATE_CONNECTED {
				opsToForward = append(opsToForward, op)
			}
		}

		if len(opsToForward) == 0 {
			return
		}

		var eventProto *v1.OperationEvent
		if event == oplog.OPERATION_ADDED {
			eventProto = &v1.OperationEvent{
				Event: &v1.OperationEvent_CreatedOperations{
					CreatedOperations: &v1.OperationList{Operations: opsToForward},
				},
			}
		} else if event == oplog.OPERATION_UPDATED {
			eventProto = &v1.OperationEvent{
				Event: &v1.OperationEvent_UpdatedOperations{
					UpdatedOperations: &v1.OperationList{Operations: opsToForward},
				},
			}
		} else if event == oplog.OPERATION_DELETED {
			ids := make([]int64, len(opsToForward))
			for i, op := range opsToForward {
				ids[i] = op.GetId()
			}
			eventProto = &v1.OperationEvent{
				Event: &v1.OperationEvent_DeletedOperations{
					DeletedOperations: &types.Int64List{Values: ids},
				},
			}
		}

		select {
		case send <- &v1.SyncStreamItem{
			Action: &v1.SyncStreamItem_SendOperations{
				SendOperations: &v1.SyncStreamItem_SyncActionSendOperations{
					Event: eventProto,
				},
			},
		}:
		default:
			cancelWithError(fmt.Errorf("operation send buffer overflow"))
		}
	}

	forwardOperationMetadata := func(repoID string) {
		zap.L().Info("TODO: implement metadata forwarding!")
	}

	handleSyncCommand := func(item *v1.SyncStreamItem) error {
		switch action := item.Action.(type) {
		case *v1.SyncStreamItem_UpdateConnectionState:
			update := action.UpdateConnectionState
			c.connectedRepos[update.RepoId] = &v1.SyncConnectionInfo{
				RepoId:          update.RepoId,
				ConnectionState: update.State,
				Message:         update.Message,
				InstanceId:      serverInstanceID,
				LastActivityMs:  time.Now().UnixMilli(),
			}

			if update.State == v1.SyncStreamItem_CONNECTION_STATE_CONNECTED {
				if !didSubscribeOplog {
					didSubscribeOplog = true
					c.oplog.Subscribe(oplog.Query{}, &oplogSubscription)
				}

				forwardOperationMetadata(update.RepoId)
			}
		case *v1.SyncStreamItem_Throttle:
			c.reconnectDelay = time.Duration(action.Throttle.GetDelayMs()) * time.Millisecond
		default:
			return fmt.Errorf("unknown action: %v", action)
		}
		return nil
	}

	for {
		select {
		case item, ok := <-receive:
			if !ok {
				return nil
			}
			if err := handleSyncCommand(item); err != nil {
				return err
			}
		case sendItem, ok := <-send:
			if !ok {
				return nil
			}
			if err := stream.Send(sendItem); err != nil {
				return err
			}
		case <-ctx.Done():
			if didSubscribeOplog {
				c.oplog.Unsubscribe(&oplogSubscription)
			}

			return ctx.Err()
		}
	}

}
