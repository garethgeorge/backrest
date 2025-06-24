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
	"github.com/garethgeorge/backrest/internal/config"
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
	remoteConfigStore       RemoteConfigStore
	connectionStatus        v1.SyncConnectionState
	connectionStatusMessage string

	// sync state subscribers are channels that can be used to notify
	// subscribers about changes in the sync state.
	subscribers []chan struct{}
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
	for _, subscriber := range c.subscribers {
		select {
		case subscriber <- struct{}{}:
		default:
			// If the subscriber channel is full, we skip sending the update.
			// This is to prevent blocking the sync client if a subscriber is not reading updates.
		}
	}
	c.mu.Unlock()
}

func (c *SyncClient) GetConnectionState() (v1.SyncConnectionState, string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connectionStatus, c.connectionStatusMessage
}

func (c *SyncClient) SubscribeToSyncStateUpdates() chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()

	ch := make(chan struct{}, 1)
	c.subscribers = append(c.subscribers, ch)
	return ch
}

func (c *SyncClient) UnsubscribeFromSyncStateUpdates(ch chan struct{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range c.subscribers {
		if c.subscribers[i] == ch {
			// Swap with last element and truncate slice
			lastIdx := len(c.subscribers) - 1
			c.subscribers[i] = c.subscribers[lastIdx]
			c.subscribers = c.subscribers[:lastIdx]
			close(ch)
			return
		}
	}
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

	c.l.Debug("received handshake packet from server", zap.String("server_instance_id", serverInstanceID))

	// haveRunSync tracks which repo GUIDs we've initiated a sync for with the server.
	// operation requests (from the server) are ignored if the GUID is not allowlisted in this map.
	haveRunSync := make(map[string]struct{})

	oplogSubscription := func(ops []*v1.Operation, event oplog.OperationEvent) {
		var opsToForward []*v1.Operation
		for _, op := range ops {
			if _, ok := haveRunSync[op.GetRepoGuid()]; ok {
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

	handleSyncCommand := func(item *v1.SyncStreamItem) error {
		switch action := item.Action.(type) {
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

			// remove any repo IDs that are no longer in the config, our access has been revoked.
			remoteRepoGUIDs := make(map[string]struct{})
			for _, repo := range newRemoteConfig.Repos {
				remoteRepoGUIDs[repo.GetGuid()] = struct{}{}
			}
			for repoID := range haveRunSync {
				if _, ok := remoteRepoGUIDs[repoID]; !ok {
					delete(haveRunSync, repoID)
				}
			}

			for _, repo := range newRemoteConfig.Repos {
				_, ok := haveRunSync[repo.GetGuid()]
				if ok {
					continue
				}
				localRepoConfig := config.FindRepoByGUID(localConfig, repo.GetGuid())
				if localRepoConfig == nil {
					c.l.Sugar().Debugf("ignoring remote repo config %q/%q because no local repo has the same GUID %q", c.peer.InstanceId, repo.GetId())
					continue
				}
				instanceID, err := InstanceForBackrestURI(localRepoConfig.Uri)
				if err != nil || instanceID != c.peer.InstanceId {
					c.l.Sugar().Debugf("ignoring remote repo config %q/%q because the local repo (%q) with the same GUID specifies URI %q (instance ID %q) which does not reference the peer providing this config", c.peer.InstanceId, repo.GetId(), localRepoConfig.Id, localRepoConfig.Guid, instanceID)
					continue
				}

				diffSel := &v1.OpSelector{
					InstanceId:            proto.String(c.localInstanceID),
					OriginalInstanceKeyid: proto.String(c.syncConfigSnapshot.identityKey.KeyID()),
					RepoGuid:              proto.String(repo.GetGuid()),
				}

				diffQuery, err := protoutil.OpSelectorToQuery(diffSel)
				if err != nil {
					return fmt.Errorf("convert operation selector to query: %w", err)
				}

				haveRunSync[repo.GetGuid()] = struct{}{}

				// Load operation metadata and send the initial diff state.
				var opIds []int64
				var opModnos []int64
				if err := c.oplog.QueryMetadata(diffQuery, func(op oplog.OpMetadata) error {
					opIds = append(opIds, op.ID)
					opModnos = append(opModnos, op.Modno)
					return nil
				}); err != nil {
					return fmt.Errorf("action sync config: query oplog for repo %q: %w", repo.GetId(), err)
				}

				c.l.Sugar().Infof("initiating operation history sync for repo %q", repo.GetId())

				if err := stream.Send(&v1.SyncStreamItem{
					Action: &v1.SyncStreamItem_DiffOperations{
						DiffOperations: &v1.SyncStreamItem_SyncActionDiffOperations{
							HaveOperationsSelector: diffSel,
							HaveOperationIds:       opIds,
							HaveOperationModnos:    opModnos,
						},
					},
				}); err != nil {
					return &SyncError{
						State:   v1.SyncConnectionState_CONNECTION_STATE_DISCONNECTED,
						Message: fmt.Errorf("action sync config: send diff operations: %w", err),
					}
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
				if op.GetInstanceId() != c.localInstanceID {
					c.l.Sugar().Warnf("action diff operations, requested operation %d is not from this instance, this shouldn't happen with a wellbehaved server", opID)
					continue // skip operations that are not from this instance e.g. an "index snapshot" picking up snapshots created by another instance.
				}

				_, ok := haveRunSync[op.RepoGuid]
				if !ok {
					// this should never happen if sync is working correctly. Would probably indicate oplog or our access was revoked.
					// Error out and re-initiate sync.
					return &SyncError{
						State:   v1.SyncConnectionState_CONNECTION_STATE_ERROR_PROTOCOL,
						Message: fmt.Errorf("remote requested operation for repo %q for which sync was never initiated", op.GetRepoId()),
					}
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
		case *v1.SyncStreamItem_Throttle:
			c.reconnectDelay = time.Duration(action.Throttle.GetDelayMs()) * time.Millisecond
		default:
			return &SyncError{
				State:   v1.SyncConnectionState_CONNECTION_STATE_ERROR_PROTOCOL,
				Message: fmt.Errorf("unknown action: %v", action),
			}
		}
		return nil
	}

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
