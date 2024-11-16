package syncengine

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/garethgeorge/backrest/gen/go/types"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1/v1connect"
	"github.com/garethgeorge/backrest/internal/oplog"
)

type ServerConnInfo struct {
	Status   v1.SyncStreamItem_ConnectionState
	Message  string
	LastSeen time.Time // only valid for disconnected streams.
}

type SyncClient struct {
	localInstanceID string
	oplog           *oplog.OpLog
	client          v1connect.BackrestSyncServiceClient
	reconnectDelay  time.Duration

	connectedRepos map[string]*v1.SyncConnectionInfo
	lastFullSync   map[string]time.Time
}

func NewSyncClient(localInstanceID, remoteInstanceURL string, oplog *oplog.OpLog) *SyncClient {
	client := v1connect.NewBackrestSyncServiceClient(
		http.DefaultClient,
		remoteInstanceURL,
	)

	return &SyncClient{
		reconnectDelay: 60 * time.Second,
		client:         client,
		oplog:          oplog,
	}
}

func (c *SyncClient) setStatusForRepos(repos []*v1.Repo, state v1.SyncStreamItem_ConnectionState, message string) {
	for _, repo := range repos {
		c.connectedRepos[repo.GetId()] = &v1.SyncConnectionInfo{
			RepoId:          repo.GetId(),
			ConnectionState: state,
			Message:         message,
			LastActivityMs:  time.Now().UnixMilli(),
		}
	}
}

func (c *SyncClient) RunSyncForRepos(ctx context.Context, repos []*v1.Repo) {
	for {
		if ctx.Err() != nil {
			break // context is done
		}

		c.setStatusForRepos(repos, v1.SyncStreamItem_CONNECTION_STATE_PENDING, "connecting")
		if err := c.runSyncForReposInternal(ctx, repos); err != nil {
			c.setStatusForRepos(repos, v1.SyncStreamItem_CONNECTION_STATE_DISCONNECTED, err.Error())
		} else {
			c.setStatusForRepos(repos, v1.SyncStreamItem_CONNECTION_STATE_DISCONNECTED, "connection closed")
		}

		time.Sleep(c.reconnectDelay) // sleep before reconnecting, by default 60 seconds but the server can adjust this.
	}
}

func (c *SyncClient) runSyncForReposInternal(ctx context.Context, repos []*v1.Repo) error {
	stream := c.client.Sync(ctx)

	ctx, cancelWithError := context.WithCancelCause(ctx)

	receive := make(chan *v1.SyncStreamItem, 1)
	send := make(chan *v1.SyncStreamItem, 100)

	go func() {
		for {
			item, err := stream.Receive()
			if err != nil {
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
		_, keyID, keySecret, err := ParseRemoteRepoURI(repo.GetUri())
		if err != nil {
			return fmt.Errorf("parse repo URI %q: %w", repo.GetUri(), err)
		}
		if err := stream.Send(&v1.SyncStreamItem{
			Action: &v1.SyncStreamItem_ConnectRepo{
				ConnectRepo: &v1.SyncStreamItem_SyncActionConnectRepo{
					RepoId:    repo.GetId(),
					KeyId:     keyID,
					KeySecret: keySecret,
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
