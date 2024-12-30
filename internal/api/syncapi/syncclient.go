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
		l:               zap.L().Named(fmt.Sprintf("syncclient %q", peer.GetInstanceId())),
	}, nil
}

func (c *SyncClient) setConnectionState(state v1.SyncConnectionState, message string) {
	c.mu.Lock()
	c.connectionStatus = state
	c.connectionStatusMessage = message
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
			c.setConnectionState(v1.SyncConnectionState_CONNECTION_STATE_DISCONNECTED, err.Error())
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

	ctx, cancelWithError := context.WithCancelCause(ctx)

	receiveError := make(chan error, 1)
	receive := make(chan *v1.SyncStreamItem, 1)
	send := make(chan *v1.SyncStreamItem, 100)

	go func() {
		for {
			item, err := stream.Receive()
			if err != nil {
				receiveError <- err
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
		// note: the error checking w/streams in connectrpc is fairly awkward.
		// If write returns an EOF error, we are expected to call stream.Receive()
		// to get the unmarshalled network failure.
		if !errors.Is(err, io.EOF) {
			c.setConnectionState(v1.SyncConnectionState_CONNECTION_STATE_ERROR_PROTOCOL, err.Error())
			return err
		} else {
			_, err2 := stream.Receive()
			c.setConnectionState(v1.SyncConnectionState_CONNECTION_STATE_DISCONNECTED, err.Error())
			return err2
		}
	}
	c.setConnectionState(v1.SyncConnectionState_CONNECTION_STATE_CONNECTED, "connected")

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

	if serverInstanceID != c.peer.InstanceId {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("server instance ID %q does not match expected peer instance ID %q", serverInstanceID, c.peer.InstanceId))
	}

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
				return fmt.Errorf("received nil remote config")
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

			// load the local config so that we can index the remote repos into any local repos that reference their URIs
			// e.g. backrest:<instance-id> format URI.
			localConfig, err := c.mgr.configMgr.Get()
			if err != nil {
				return fmt.Errorf("get local config: %w", err)
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
					InstanceId: proto.String(c.localInstanceID),
					RepoId:     proto.String(repo.GetId()),
					RepoGuid:   proto.String(repo.GetGuid()),
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
					return fmt.Errorf("action sync config: send diff operations: %w", err)
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
					return fmt.Errorf("action diff operations: send create operations: %w", err)
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
					return fmt.Errorf("remote requested operation for repo %q for which sync was never initiated", op.GetRepoId())
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
					return fmt.Errorf("action diff operations: send delete operations: %w", err)
				}
			}

			c.l.Debug("replied to an operations request", zap.Int("num_ops_requested", len(requestedOperations)), zap.Int("num_ops_sent", sentOps), zap.Int("num_ops_deleted", len(deletedIDs)))
		case *v1.SyncStreamItem_Throttle:
			c.reconnectDelay = time.Duration(action.Throttle.GetDelayMs()) * time.Millisecond
		default:
			return fmt.Errorf("unknown action: %v", action)
		}
		return nil
	}

	for {
		select {
		case err := <-receiveError:
			return fmt.Errorf("connection terminated with error: %w", err)
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
			return ctx.Err()
		}
	}
}
