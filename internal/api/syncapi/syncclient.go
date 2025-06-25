package syncapi

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/garethgeorge/backrest/gen/go/types"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1/v1connect"
	"github.com/garethgeorge/backrest/internal/api/syncapi/permissions"
	"github.com/garethgeorge/backrest/internal/eventemitter"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/protoutil"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"google.golang.org/protobuf/proto"
)

type SyncClient struct {
	mgr *SyncManager

	syncConfigSnapshot syncConfigSnapshot
	localInstanceID    string
	peer               *v1.Multihost_Peer
	oplog              *oplog.OpLog
	client             v1connect.BackrestSyncServiceClient
	reconnectDelay     time.Duration
	l                  *zap.Logger

	// mutable properties
	mu                      sync.Mutex
	connectionStatus        v1.SyncConnectionState
	connectionStatusMessage string

	// event emitters
	OnStateChange eventemitter.EventEmitter[struct{}]
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

func NewSyncClient(
	mgr *SyncManager,
	snapshot syncConfigSnapshot,
	peer *v1.Multihost_Peer,
	oplog *oplog.OpLog,
) (*SyncClient, error) {
	if peer.GetInstanceUrl() == "" {
		return nil, errors.New("peer instance URL is required")
	}

	client := v1connect.NewBackrestSyncServiceClient(
		newInsecureClient(),
		peer.GetInstanceUrl(),
	)

	c := &SyncClient{
		mgr:                mgr,
		syncConfigSnapshot: snapshot,
		localInstanceID:    snapshot.config.Instance,
		peer:               peer,
		reconnectDelay:     mgr.syncClientRetryDelay,
		client:             client,
		oplog:              oplog,
		l:                  zap.L().Named(fmt.Sprintf("syncclient for %q", peer.GetInstanceId())),
	}
	c.setConnectionState(v1.SyncConnectionState_CONNECTION_STATE_DISCONNECTED, "starting up")
	return c, nil
}

func (c *SyncClient) setConnectionState(state v1.SyncConnectionState, message string) {
	c.mu.Lock()
	c.connectionStatus = state
	c.connectionStatusMessage = message
	c.OnStateChange.Emit(struct{}{})
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
			c.l.Sugar().Errorf("sync error: %v", err)
			var syncErr *SyncError
			if errors.As(err, &syncErr) {
				c.setConnectionState(syncErr.State, syncErr.Message.Error())
			} else {
				c.setConnectionState(v1.SyncConnectionState_CONNECTION_STATE_ERROR_INTERNAL, err.Error())
			}
		}

		delay := c.reconnectDelay - time.Since(lastConnect)
		c.l.Sugar().Infof("disconnected, will retry after %v", delay)
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

	localConfig := c.syncConfigSnapshot.config
	localIdentityKey := c.syncConfigSnapshot.identityKey

	ctx, cancelWithError := context.WithCancelCause(ctx)
	defer cancelWithError(nil)

	receiveError := make(chan error)
	receive := make(chan *v1.SyncStreamItem)
	send := make(chan *v1.SyncStreamItem, 100)

	go func() {
		for {
			item, err := stream.Receive()
			if err != nil {
				receiveError <- err
				break
			}
			receive <- item
		}
		close(receive)
	}()

	// Broadcast initial packet containing the protocol version and instance ID.
	// TODO: do this in a header instead of as a part of the stream.
	handshakePacket, err := createHandshakePacket(c.localInstanceID, localIdentityKey)
	if err != nil {
		return fmt.Errorf("create handshake packet: %w", err)
	}

	if err := stream.Send(handshakePacket); err != nil {
		// note: the error checking w/streams in connectrpc is fairly awkward.
		// If write returns an EOF error, we are expected to call stream.Receive()
		// to get the unmarshalled network failure.
		if !errors.Is(err, io.EOF) {
			return &SyncError{
				State:   v1.SyncConnectionState_CONNECTION_STATE_ERROR_PROTOCOL,
				Message: fmt.Errorf("send handshake packet: %w", err),
			}
		} else {
			_, err2 := stream.Receive()
			return &SyncError{
				State:   v1.SyncConnectionState_CONNECTION_STATE_DISCONNECTED,
				Message: err2,
			}
		}
	}
	c.setConnectionState(v1.SyncConnectionState_CONNECTION_STATE_CONNECTED, "connected")

	c.l.Debug("sent handshake packet, now waiting for server handshake", zap.String("local_instance_id", c.localInstanceID), zap.String("host_instance_id", c.peer.InstanceId))

	// Wait for the handshake packet from the server.
	handshakeMsg, err := tryReceiveWithinDuration(ctx, receive, receiveError, 5*time.Second)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, &SyncError{
			State:   v1.SyncConnectionState_CONNECTION_STATE_ERROR_AUTH,
			Message: fmt.Errorf("read error before handshake packet: %v", err),
		})
	}
	if _, err := verifyHandshakePacket(handshakeMsg); err != nil {
		return connect.NewError(connect.CodeInvalidArgument, &SyncError{
			State:   v1.SyncConnectionState_CONNECTION_STATE_ERROR_AUTH,
			Message: fmt.Errorf("verify handshake packet: %v", err),
		})
	}
	if err := authorizeHandshakeAsPeer(handshakeMsg, c.peer); err != nil {
		return connect.NewError(connect.CodeInvalidArgument, &SyncError{
			State:   v1.SyncConnectionState_CONNECTION_STATE_ERROR_AUTH,
			Message: fmt.Errorf("authorize handshake packet: %v", err),
		})
	}
	serverInstanceID := c.peer.InstanceId

	peerPerms, err := permissions.NewPermissionSet(c.peer.Permissions)
	if err != nil {
		return &SyncError{
			State:   v1.SyncConnectionState_CONNECTION_STATE_ERROR_AUTH,
			Message: fmt.Errorf("parse peer permissions: %w", err),
		}
	}
	c.l.Debug("received handshake packet from server", zap.String("server_instance_id", serverInstanceID))

	// start by forwarding the configuration and the resource lists the peer is allowed to see.
	{
		remoteConfig := &v1.RemoteConfig{
			Version: localConfig.Version,
			Modno:   localConfig.Modno,
		}
		resourceList := &v1.SyncStreamItem_SyncActionListResources{}
		for _, repo := range localConfig.Repos {
			if peerPerms.CheckPermissionForRepo(repo.Guid, v1.Multihost_Permission_PERMISSION_READ_CONFIG) {
				remoteConfig.Repos = append(remoteConfig.Repos, repo)
				resourceList.RepoIds = append(resourceList.RepoIds, repo.Id)
			}
		}
		for _, plan := range localConfig.Plans {
			if peerPerms.CheckPermissionForPlan(plan.Id, v1.Multihost_Permission_PERMISSION_READ_CONFIG) {
				remoteConfig.Plans = append(remoteConfig.Plans, plan)
				resourceList.PlanIds = append(resourceList.PlanIds, plan.Id)
			}
		}

		if err := stream.Send(&v1.SyncStreamItem{
			Action: &v1.SyncStreamItem_SendConfig{
				SendConfig: &v1.SyncStreamItem_SyncActionSendConfig{
					Config: remoteConfig,
				},
			},
		}); err != nil {
			return fmt.Errorf("send initial config: %w", err)
		}
		if err := stream.Send(&v1.SyncStreamItem{
			Action: &v1.SyncStreamItem_ListResources{
				ListResources: resourceList,
			},
		}); err != nil {
			return fmt.Errorf("send initial resource list: %w", err)
		}
	}

	canForwardOperation := func(op *v1.Operation) bool {
		if op.GetOriginalInstanceKeyid() != "" || op.GetInstanceId() != c.localInstanceID {
			return false // only forward operations that were created by this instance
		}

		return (op.GetPlanId() != "" && peerPerms.CheckPermissionForPlan(op.GetPlanId(), v1.Multihost_Permission_PERMISSION_READ_OPERATIONS)) ||
			(op.GetRepoGuid() != "" && peerPerms.CheckPermissionForRepo(op.GetRepoGuid(), v1.Multihost_Permission_PERMISSION_READ_OPERATIONS))
	}

	oplogSubscription := func(ops []*v1.Operation, event oplog.OperationEvent) {
		var opsToForward []*v1.Operation
		for _, op := range ops {
			if canForwardOperation(op) {
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
	c.oplog.Subscribe(oplog.Query{}, &oplogSubscription)
	defer c.oplog.Unsubscribe(&oplogSubscription)

	// Initiate sync on a background thread, this allows us to handle responses while still sending the initial diff state.
	// This is a slow operation and we don't want to block the main loop waiting for it to complete and potentially forcing incomming messages to buffer or drop.
	go func() {

		startSync := func(diffSel *v1.OpSelector) error {
			c.l.Sugar().Infof("starting sync with diffselector: %v", diffSel)

			diffQuery, err := protoutil.OpSelectorToQuery(diffSel)
			if err != nil {
				return fmt.Errorf("convert operation selector to query: %w", err)
			}
			// Load operation metadata and send the initial diff state.
			var opIds []int64
			var opModnos []int64
			if err := c.oplog.QueryMetadata(diffQuery, func(op oplog.OpMetadata) error {
				opIds = append(opIds, op.ID)
				opModnos = append(opModnos, op.Modno)
				return nil
			}); err != nil {
				return fmt.Errorf("query oplog with selector %v: %w", diffSel, err)
			}

			diffQuery.OriginalInstanceKeyid = proto.String(c.syncConfigSnapshot.identityKey.KeyID())
			send <- &v1.SyncStreamItem{
				Action: &v1.SyncStreamItem_DiffOperations{
					DiffOperations: &v1.SyncStreamItem_SyncActionDiffOperations{
						HaveOperationsSelector: diffSel,
						HaveOperationIds:       opIds,
						HaveOperationModnos:    opModnos,
					},
				},
			}
			return nil
		}
		// Start syncing operations for all repos and plans that the peer is allowed to read.
		for _, repo := range localConfig.Repos {
			if !peerPerms.CheckPermissionForRepo(repo.Guid, v1.Multihost_Permission_PERMISSION_READ_OPERATIONS) {
				continue // skip repos that the peer is not allowed to read
			}
			diffSel := &v1.OpSelector{
				RepoGuid: proto.String(repo.Guid),
			}
			if err := startSync(diffSel); err != nil {
				c.l.Sugar().Errorf("failed to start sync for repo %q: %v", repo.Guid, err)
				cancelWithError(fmt.Errorf("start sync for repo %q: %w", repo.Guid, err))
				return
			}
		}

		for _, plan := range localConfig.Plans {
			if !peerPerms.CheckPermissionForPlan(plan.Id, v1.Multihost_Permission_PERMISSION_READ_OPERATIONS) {
				continue // skip plans that the peer is not allowed to read
			}
			if peerPerms.CheckPermissionForPlan(plan.Repo, v1.Multihost_Permission_PERMISSION_READ_OPERATIONS) {
				continue // skip the sync if we're already syncing the whole repo that it belongs to
			}
			diffSel := &v1.OpSelector{
				PlanId:   proto.String(plan.Id),
				RepoGuid: proto.String(plan.Repo),
			}
			if err := startSync(diffSel); err != nil {
				c.l.Sugar().Errorf("failed to start sync for plan %q: %v", plan.Id, err)
				cancelWithError(fmt.Errorf("start sync for plan %q: %w", plan.Id, err))
				return
			}
		}
	}()

	handleSyncCommand := func(item *v1.SyncStreamItem) error {
		switch action := item.Action.(type) {
		case *v1.SyncStreamItem_ListResources:
			c.l.Sugar().Debugf("received resource list for peer %q, repo IDs: %v, plan IDs: %v",
				c.peer.InstanceId, action.ListResources.RepoIds, action.ListResources.PlanIds)
			// TODO: update this in the peer info store for this peer eventually.
		case *v1.SyncStreamItem_SendConfig:
			c.l.Sugar().Debugf("received remote config update")
			newRemoteConfig := action.SendConfig.Config
			if err := c.mgr.remoteConfigStore.Update(c.peer.InstanceId, newRemoteConfig); err != nil {
				return fmt.Errorf("update remote config store with latest config: %w", err)
			}

			if newRemoteConfig == nil {
				return &SyncError{
					State:   v1.SyncConnectionState_CONNECTION_STATE_ERROR_PROTOCOL,
					Message: fmt.Errorf("received nil remote config"),
				}
			}
		case *v1.SyncStreamItem_DiffOperations:
			requestedOperations := action.DiffOperations.GetRequestOperations()
			c.l.Sugar().Debugf("received operation request for operations: %v", requestedOperations)

			var deletedIDs []int64
			var sendOps []*v1.Operation

			sendOpsFunc := func() error {
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
					sendOps = sendOps[:0]
					return &SyncError{
						State:   v1.SyncConnectionState_CONNECTION_STATE_DISCONNECTED,
						Message: fmt.Errorf("action diff operations: send create operations: %w", err),
					}
				}
				c.l.Sugar().Debugf("sent %d operations", len(sendOps))
				sendOps = sendOps[:0]
				return nil
			}

			sentOps := 0
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
				if !canForwardOperation(op) {
					c.l.Sugar().Debugf("skipping operation %d for repo %q, plan %q, not allowed to read by peer %q",
						op.GetId(), op.GetRepoGuid(), op.GetPlanId(), c.peer.InstanceId)
					continue // skip operations that the peer is not allowed to read
				}
				sendOps = append(sendOps, op)
				sentOps += 1
				if len(sendOps) >= 256 {
					if err := sendOpsFunc(); err != nil {
						return err
					}
				}
			}

			if len(sendOps) > 0 {
				if err := sendOpsFunc(); err != nil {
					return err
				}
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
					return &SyncError{
						State:   v1.SyncConnectionState_CONNECTION_STATE_DISCONNECTED,
						Message: fmt.Errorf("action diff operations: send delete operations: %w", err),
					}
				}
			}

			c.l.Debug("replied to an operations request", zap.Int("num_ops_requested", len(requestedOperations)), zap.Int("num_ops_sent", sentOps), zap.Int("num_ops_deleted", len(deletedIDs)))

		case *v1.SyncStreamItem_Heartbeat:
			// TODO: handle the heartbeat messages, currently we just ignore them.
		case *v1.SyncStreamItem_Throttle:
			c.reconnectDelay = time.Duration(action.Throttle.GetDelayMs()) * time.Millisecond
		default:
			return &SyncError{
				State:   v1.SyncConnectionState_CONNECTION_STATE_ERROR_PROTOCOL,
				Message: fmt.Errorf("unknown action: %T - %v", action, action),
			}
		}
		return nil
	}

	// start a heartbeat thread
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				send <- &v1.SyncStreamItem{
					Action: &v1.SyncStreamItem_Heartbeat{
						Heartbeat: &v1.SyncStreamItem_SyncActionHeartbeat{},
					},
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case err := <-receiveError:
			return &SyncError{
				State:   v1.SyncConnectionState_CONNECTION_STATE_DISCONNECTED,
				Message: fmt.Errorf("connection terminated with error: %w", err),
			}
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
				return &SyncError{
					State:   v1.SyncConnectionState_CONNECTION_STATE_DISCONNECTED,
					Message: err,
				}
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
