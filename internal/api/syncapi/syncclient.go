package syncapi

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/garethgeorge/backrest/gen/go/types"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1/v1connect"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/oplog"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
)

type SyncClient struct {
	mgr             *SyncManager
	localInstanceID string
	peer            *v1.Multihost_Peer
	oplog           *oplog.OpLog
	client          v1connect.BackrestSyncServiceClient
	reconnectDelay  time.Duration
	l               *zap.Logger

	// mutable properties
	mu                      sync.Mutex
	remoteConfigStore       RemoteConfigStore
	connectionStatus        v1.SyncConnectionState
	connectionStatusMessage string
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

func NewSyncClient(mgr *SyncManager, localInstanceID string, peer *v1.Multihost_Peer, oplog *oplog.OpLog) (*SyncClient, error) {
	if peer.GetInstanceUrl() == "" {
		return nil, errors.New("peer instance URL is required")
	}

	client := v1connect.NewBackrestSyncServiceClient(
		newInsecureClient(),
		peer.GetInstanceUrl(),
	)

	return &SyncClient{
		mgr:             mgr,
		localInstanceID: localInstanceID,
		peer:            peer,
		reconnectDelay:  60 * time.Second,
		client:          client,
		oplog:           oplog,
		l:               zap.L().Named("syncclient").With(zap.String("peer", peer.GetInstanceUrl())),
	}, nil
}

func (c *SyncClient) setConnectionState(state v1.SyncConnectionState, message string) {
	c.mu.Lock()
	c.connectionStatus = state
	c.mu.Unlock()
}

func (c *SyncClient) GetConnectionState() (v1.SyncConnectionState, string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connectionStatus, c.connectionStatusMessage
}

func (c *SyncClient) RunSync(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}

		lastConnect := time.Now()

		c.setConnectionState(v1.SyncConnectionState_CONNECTION_STATE_PENDING, "connection pending")

		if err := c.runSyncInternal(ctx); err != nil {
			c.l.Error("sync error", zap.Error(err))
			c.setConnectionState(v1.SyncConnectionState_CONNECTION_STATE_DISCONNECTED, err.Error())
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

func (c *SyncClient) runSyncInternal(ctx context.Context) error {
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
	// TODO: do this in a header instead of as a part of the stream.
	if err := stream.Send(&v1.SyncStreamItem{
		Action: &v1.SyncStreamItem_Handshake{
			Handshake: &v1.SyncStreamItem_SyncActionHandshake{
				ProtocolVersion: SyncProtocolVersion,
				InstanceId: &v1.SignedMessage{
					Payload:   []byte(c.localInstanceID),
					Signature: []byte("TOOD: inject a valid signature"),
					Keyid:     "TODO: inject a valid key ID",
				},
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

		serverInstanceID = string(handshake.GetInstanceId().GetPayload())
		if serverInstanceID == "" {
			return connect.NewError(connect.CodeInvalidArgument, errors.New("instance ID is required"))
		}

		if handshake.GetProtocolVersion() != SyncProtocolVersion {
			return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("unsupported peer protocol version, got %d, expected %d", handshake.GetProtocolVersion(), SyncProtocolVersion))
		}
	} else {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("no packets received"))
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

	haveRunSync := make(map[string]struct{}) // repo ID -> have run sync?

	handleSyncCommand := func(item *v1.SyncStreamItem) error {
		switch action := item.Action.(type) {
		case *v1.SyncStreamItem_SendConfig:
			newRemoteConfig := action.SendConfig.Config
			if err := c.mgr.remoteConfigStore.Update(c.peer.InstanceId); err != nil {
				return fmt.Errorf("update remote config store with latest config: %w", err)
			}

			localConfig, err := c.mgr.configMgr.Get()
			if err != nil {
				return fmt.Errorf("get local config: %w", err)
			}

			for _, repo := range newRemoteConfig.GetRepos() {
				_, ok := haveRunSync[repo.GetId()]
				if ok {
					continue
				}
				haveRunSync[repo.GetId()] = struct{}{}
				localRepoConfig := config.FindRepo(localConfig, repo.GetId())
				repo.Hooks = nil // we don't accept any hooks from the server. This could be used to execute arbitrary code on the client.
				instanceID, err := InstanceForBackrestURI(localRepoConfig.Uri)
				if err != nil || instanceID != c.peer.InstanceId {
					c.l.Sugar().Debugf("ignoring remote repo config %q/%q because the local repo with the same name specifies URI %q which does not reference the peer providing this config", c.peer.InstanceId, repo.GetId(), localRepoConfig.BackrestURI)
					continue
				}

				// Load operation metadata and send the initial diff state.
				var opIds []int64
				var opModnos []int64
				if err := c.oplog.QueryMetadata(oplog.Query{
					InstanceID: c.localInstanceID,
					RepoID:     repo.GetId(),
				}, func(op oplog.OpMetadata) error {
					opIds = append(opIds, op.ID)
					opModnos = append(opModnos, op.Modno)
					return nil
				}); err != nil {
					return fmt.Errorf("action sync config: query oplog for repo %q: %w", repo.GetId(), err)
				}

				if err := stream.Send(&v1.SyncStreamItem{
					Action: &v1.SyncStreamItem_DiffOperations{
						DiffOperations: &v1.SyncStreamItem_SyncActionDiffOperations{
							HaveOperationsSelector: &v1.OpSelector{
								InstanceId: c.localInstanceID,
								RepoId:     repo.GetId(),
							},
							HaveOperationIds:    opIds,
							HaveOperationModnos: opModnos,
						},
					},
				}); err != nil {
					return fmt.Errorf("action sync config: send diff operations: %w", err)
				}
			}
		case *v1.SyncStreamItem_DiffOperations:
			requestedOperations := action.DiffOperations.GetRequestOperations()

			var deletedIDs []int64
			var sendOps []*v1.Operation
			var ops []*v1.Operation
			for _, opID := range requestedOperations {
				op, err := c.oplog.Get(opID)
				if err != nil {
					if errors.Is(err, oplog.ErrNotExist) {
						deletedIDs = append(deletedIDs, opID)
						continue
					}
					c.l.Sugar().Warnf("action diff operations, failed to fetch a requested operation %d: %v", opID, err)
					continue // skip this operation
				}

				_, ok := haveRunSync[op.GetRepoId()]
				if !ok {
					// this should never happen if sync is working correctly. Would probably indicate oplog corruption?
					return fmt.Errorf("remote requested operation for repo %q for which sync was never initiated", op.GetRepoId())
				}

				sendOps = append(ops, op)
			}

			if len(deletedIDs) > 0 {
				if err := stream.Send(&v1.SyncStreamItem{
					Action: &v1.SyncStreamItem_SendOperations{
						SendOperations: &v1.SyncStreamItem_SyncActionSendOperations{
							Event: &v1.OperationEvent{
								Event: &v1.OperationEvent_DeletedOperations{
									DeletedOperations: &types.Int64List{Values: deletedIDs},
								},
							},
						},
					},
				}); err != nil {
					return fmt.Errorf("action diff operations: send delete operations: %w", err)
				}
			}

			if len(sendOps) > 0 {
				if err := stream.Send(&v1.SyncStreamItem{
					Action: &v1.SyncStreamItem_SendOperations{
						SendOperations: &v1.SyncStreamItem_SyncActionSendOperations{
							Event: &v1.OperationEvent{
								Event: &v1.OperationEvent_CreatedOperations{
									CreatedOperations: &v1.OperationList{Operations: sendOps},
								},
							},
						},
					},
				}); err != nil {
					return fmt.Errorf("action diff operations: send create operations: %w", err)
				}
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
		case sendItem, ok := <-send: // note: send channel should only be used when sending from a different goroutine than the main loop
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
