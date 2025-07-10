package syncapi

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/garethgeorge/backrest/gen/go/types"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1/v1connect"
	"github.com/garethgeorge/backrest/internal/api/syncapi/permissions"
	"github.com/garethgeorge/backrest/internal/env"
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

	reconnectAttempts int
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
	c.mgr.peerStateManager.UpdatePeerState(peer.Keyid, peer.InstanceId, func(peerState *PeerState) {
		// this will create a new peer state if one doesn't exist
	})
	return c, nil
}

func (c *SyncClient) RunSync(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}

		lastConnect := time.Now()

		syncSessionHandler := newSyncHandlerClient(
			c.l,
			c.mgr,
			c.syncConfigSnapshot,
			c.oplog,
			c.peer,
		)

		cmdStream := newBidiSyncCommandStream()

		c.l.Sugar().Infof("connecting to peer %q (%s) at %s", c.peer.InstanceId, c.peer.Keyid, c.peer.GetInstanceUrl())

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := runSync(
				ctx,
				c.localInstanceID,
				c.syncConfigSnapshot.identityKey,
				cmdStream,
				syncSessionHandler,
				c.syncConfigSnapshot.config.GetMultihost().GetKnownHosts(),
			)
			cmdStream.SendErrorAndTerminate(err)
		}()

		if err := cmdStream.ConnectStream(ctx, c.client.Sync(ctx)); err != nil {
			c.l.Sugar().Infof("lost stream connection to peer %q (%s): %v", c.peer.InstanceId, c.peer.Keyid, err)
			var syncErr *SyncError
			c.mgr.peerStateManager.UpdatePeerState(c.peer.Keyid, c.peer.InstanceId, func(state *PeerState) {
				state.LastHeartbeat = time.Now()
				if errors.As(err, &syncErr) {
					state.ConnectionState = syncErr.State
					state.ConnectionStateMessage = syncErr.Message.Error()
				} else {
					state.ConnectionState = v1.SyncConnectionState_CONNECTION_STATE_ERROR_INTERNAL
					state.ConnectionStateMessage = err.Error()
				}
			})
		} else {
			c.reconnectAttempts = 0
			c.mgr.peerStateManager.UpdatePeerState(c.peer.Keyid, c.peer.InstanceId, func(state *PeerState) {
				state.ConnectionState = v1.SyncConnectionState_CONNECTION_STATE_DISCONNECTED
				state.ConnectionStateMessage = "disconnected"
				state.LastHeartbeat = time.Now()
			})
		}

		wg.Wait()

		delay := c.reconnectDelay - time.Since(lastConnect)
		if c.reconnectAttempts > 0 {
			backoff := time.Duration(1<<min(c.reconnectAttempts, 5)) * c.reconnectDelay // 2^reconnectAttempts, max 32
			delay += backoff
		}
		c.l.Sugar().Infof("disconnected, will retry after %v (attempt %d)", delay, c.reconnectAttempts)
		c.reconnectAttempts++
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return
		}
	}
}

// syncSessionHandlerClient is a syncSessionHandler implementation for clients.
type syncSessionHandlerClient struct {
	unimplementedSyncSessionHandler

	l                  *zap.Logger
	mgr                *SyncManager
	syncConfigSnapshot syncConfigSnapshot
	localInstanceID    string
	oplog              *oplog.OpLog

	peer        *v1.Multihost_Peer // The peer this handler is associated with, unset until OnConnectionEstablished is called.
	permissions *permissions.PermissionSet
}

func newSyncHandlerClient(
	l *zap.Logger,
	mgr *SyncManager,
	snapshot syncConfigSnapshot,
	oplog *oplog.OpLog,
	peer *v1.Multihost_Peer, // The peer this handler is associated with, must be set before calling OnConnectionEstablished.
) *syncSessionHandlerClient {
	return &syncSessionHandlerClient{
		l:                  l,
		mgr:                mgr,
		syncConfigSnapshot: snapshot,
		localInstanceID:    snapshot.config.Instance,
		oplog:              oplog,
		peer:               peer,
	}
}

var _ syncSessionHandler = (*syncSessionHandlerClient)(nil)

func (c *syncSessionHandlerClient) canForwardOperation(op *v1.Operation) bool {
	if op.GetOriginalInstanceKeyid() != "" || op.GetInstanceId() != c.localInstanceID {
		return false // only forward operations that were created by this instance
	}

	return (op.GetPlanId() != "" && c.permissions.CheckPermissionForPlan(op.GetPlanId(), v1.Multihost_Permission_PERMISSION_READ_OPERATIONS)) ||
		(op.GetRepoId() != "" && c.permissions.CheckPermissionForRepo(op.GetRepoId(), v1.Multihost_Permission_PERMISSION_READ_OPERATIONS))
}

func (c *syncSessionHandlerClient) OnConnectionEstablished(ctx context.Context, stream *bidiSyncCommandStream, peer *v1.Multihost_Peer) error {
	// A client expects to connect to a specific peer, so we check that the peer we connected to matches the one we expect.
	if !proto.Equal(c.peer, peer) {
		return NewSyncErrorAuth(fmt.Errorf("peer mismatch: expected %s (%s), got %s (%s)", c.peer.Keyid, c.peer.InstanceId, peer.Keyid, peer.InstanceId))
	}

	var err error
	c.peer = peer
	c.permissions, err = permissions.NewPermissionSet(peer.Permissions)
	if err != nil {
		return NewSyncErrorAuth(fmt.Errorf("creating permission set for peer %q: %w", peer.InstanceId, err))
	}

	c.l.Sugar().Infof("sync connection established with peer %q (%s)", peer.InstanceId, peer.Keyid)
	c.mgr.peerStateManager.UpdatePeerState(peer.Keyid, c.peer.InstanceId, func(peerState *PeerState) {
		peerState.ConnectionState = v1.SyncConnectionState_CONNECTION_STATE_CONNECTED
		peerState.ConnectionStateMessage = "connected"
		peerState.LastHeartbeat = time.Now()
	})

	// Send a heartbeat every 2 minutes to keep the connection alive.
	go sendHeartbeats(ctx, stream, env.MultihostHeartbeatInterval())

	localConfig := c.syncConfigSnapshot.config

	// start by forwarding the configuration and the resource lists the peer is allowed to see.
	{
		remoteConfig := &v1.RemoteConfig{
			Version: localConfig.Version,
			Modno:   localConfig.Modno,
		}
		resourceList := &v1.SyncStreamItem_SyncActionListResources{}
		for _, repo := range localConfig.Repos {
			if c.permissions.CheckPermissionForRepo(repo.Guid, v1.Multihost_Permission_PERMISSION_READ_CONFIG) {
				remoteConfig.Repos = append(remoteConfig.Repos, repo)
			}
			if c.permissions.CheckPermissionForRepo(repo.Guid, v1.Multihost_Permission_PERMISSION_READ_OPERATIONS, v1.Multihost_Permission_PERMISSION_READ_CONFIG) {
				resourceList.Repos = append(resourceList.Repos, &v1.SyncRepoMetadata{
					Id:   repo.Id,
					Guid: repo.Guid,
				})
			}
		}
		for _, plan := range localConfig.Plans {
			if c.permissions.CheckPermissionForPlan(plan.Id, v1.Multihost_Permission_PERMISSION_READ_CONFIG) {
				remoteConfig.Plans = append(remoteConfig.Plans, plan)
			}
			if c.permissions.CheckPermissionForPlan(plan.Id, v1.Multihost_Permission_PERMISSION_READ_OPERATIONS, v1.Multihost_Permission_PERMISSION_READ_CONFIG) {
				resourceList.Plans = append(resourceList.Plans, &v1.SyncPlanMetadata{
					Id: plan.Id,
				})
			}
		}

		stream.Send(&v1.SyncStreamItem{
			Action: &v1.SyncStreamItem_SendConfig{
				SendConfig: &v1.SyncStreamItem_SyncActionSendConfig{
					Config: remoteConfig,
				},
			},
		})
		stream.Send(&v1.SyncStreamItem{
			Action: &v1.SyncStreamItem_ListResources{
				ListResources: resourceList,
			},
		})
	}

	oplogSubscription := func(ops []*v1.Operation, event oplog.OperationEvent) {
		var opsToForward []*v1.Operation
		for _, op := range ops {
			if c.canForwardOperation(op) {
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

		stream.Send(&v1.SyncStreamItem{
			Action: &v1.SyncStreamItem_SendOperations{
				SendOperations: &v1.SyncStreamItem_SyncActionSendOperations{
					Event: eventProto,
				},
			},
		})
	}
	c.oplog.Subscribe(oplog.Query{}, &oplogSubscription)
	go func() {
		<-ctx.Done()
		c.oplog.Unsubscribe(&oplogSubscription)
	}()

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

			diffSel.OriginalInstanceKeyid = proto.String(c.syncConfigSnapshot.identityKey.KeyID())
			stream.Send(&v1.SyncStreamItem{
				Action: &v1.SyncStreamItem_DiffOperations{
					DiffOperations: &v1.SyncStreamItem_SyncActionDiffOperations{
						HaveOperationsSelector: diffSel,
						HaveOperationIds:       opIds,
						HaveOperationModnos:    opModnos,
					},
				},
			})
			return nil
		}

		reposToSync := []string{}
		plansToSync := []string{}

		// Start syncing operations for all repos and plans that the peer is allowed to read.
		for _, repo := range localConfig.Repos {
			if !c.permissions.CheckPermissionForRepo(repo.Id, v1.Multihost_Permission_PERMISSION_READ_OPERATIONS) {
				continue // skip repos that the peer is not allowed to read
			}
			diffSel := &v1.OpSelector{
				RepoGuid: proto.String(repo.Guid),
			}
			if err := startSync(diffSel); err != nil {
				c.l.Sugar().Errorf("failed to start sync for repo %q: %v", repo.Guid, err)
				stream.SendErrorAndTerminate(fmt.Errorf("start sync for repo %q: %w", repo.Guid, err))
				return
			}
			reposToSync = append(reposToSync, repo.Id)
		}

		for _, plan := range localConfig.Plans {
			if !c.permissions.CheckPermissionForPlan(plan.Id, v1.Multihost_Permission_PERMISSION_READ_OPERATIONS) {
				continue // skip plans that the peer is not allowed to read
			}
			if c.permissions.CheckPermissionForPlan(plan.Repo, v1.Multihost_Permission_PERMISSION_READ_OPERATIONS) {
				continue // skip the sync if we're already syncing the whole repo that it belongs to
			}
			diffSel := &v1.OpSelector{
				PlanId:   proto.String(plan.Id),
				RepoGuid: proto.String(plan.Repo),
			}
			if err := startSync(diffSel); err != nil {
				c.l.Sugar().Errorf("failed to start sync for plan %q: %v", plan.Id, err)
				stream.SendErrorAndTerminate(fmt.Errorf("start sync for plan %q: %w", plan.Id, err))
				return
			}
			plansToSync = append(plansToSync, plan.Id)
		}
		c.l.Sugar().Infof("triggered sync for repos %v and plans %v for local instance ID: %q, peer instance ID: %q",
			reposToSync, plansToSync, c.localInstanceID, c.peer.InstanceId)
	}()

	return nil
}

func (c *syncSessionHandlerClient) HandleHeartbeat(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionHeartbeat) error {
	c.mgr.peerStateManager.UpdatePeerState(c.peer.Keyid, c.peer.InstanceId, func(peerState *PeerState) {
		if peerState == nil {
			return // this should not happen
		}
		peerState.LastHeartbeat = time.Now()
	})
	return nil
}

func (c *syncSessionHandlerClient) HandleDiffOperations(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionDiffOperations) error {
	requestedOperations := item.GetRequestOperations()
	c.l.Sugar().Debugf("received operation request for operations: %v", requestedOperations)

	var deletedIDs []int64
	var sendOps []*v1.Operation

	sendOpsFunc := func() error {
		stream.Send(&v1.SyncStreamItem{
			Action: &v1.SyncStreamItem_SendOperations{
				SendOperations: &v1.SyncStreamItem_SyncActionSendOperations{
					Event: &v1.OperationEvent{
						Event: &v1.OperationEvent_CreatedOperations{
							CreatedOperations: &v1.OperationList{Operations: sendOps},
						},
					},
				},
			},
		})
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
		if !c.canForwardOperation(op) {
			c.l.Sugar().Warnf("skipping operation %d for repo %q, plan %q, not allowed to read by peer %q",
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
		stream.Send(&v1.SyncStreamItem{
			Action: &v1.SyncStreamItem_SendOperations{
				SendOperations: &v1.SyncStreamItem_SyncActionSendOperations{
					Event: &v1.OperationEvent{
						Event: &v1.OperationEvent_DeletedOperations{
							DeletedOperations: &types.Int64List{Values: deletedIDs},
						},
					},
				},
			},
		})
	}

	c.l.Debug("replied to an operations request", zap.Int("num_ops_requested", len(requestedOperations)), zap.Int("num_ops_sent", sentOps), zap.Int("num_ops_deleted", len(deletedIDs)))
	return nil
}

func (c *syncSessionHandlerClient) HandleSendOperations(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionSendOperations) error {
	return NewSyncErrorProtocol(errors.New("client should not receive SendOperations messages, this is a host-only message"))
}

func (c *syncSessionHandlerClient) HandleSendConfig(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionSendConfig) error {
	c.l.Sugar().Debugf("received remote config update")
	newRemoteConfig := item.Config
	if newRemoteConfig == nil {
		return NewSyncErrorProtocol(fmt.Errorf("received nil remote config"))
	}
	c.mgr.peerStateManager.UpdatePeerState(c.peer.Keyid, c.peer.InstanceId, func(peerState *PeerState) {
		if peerState == nil {
			return // this should not happen
		}
		peerState.Config = newRemoteConfig
	})
	return nil
}

func (c *syncSessionHandlerClient) HandleSetConfig(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionSetConfig) error {
	// Log the received config updates
	c.l.Sugar().Debugf("received SetConfig request from peer %q")

	// Fetch latest config from the config manager
	latestConfig, err := c.mgr.configMgr.Get()
	if err != nil {
		return fmt.Errorf("fetch latest config: %w", err)
	}

	latestConfig = proto.Clone(latestConfig).(*v1.Config) // Clone to avoid modifying the original config

	for _, plan := range item.GetPlans() {
		c.l.Sugar().Debugf("received plan update: %s", plan.Id)
		if !c.permissions.CheckPermissionForPlan(plan.Id, v1.Multihost_Permission_PERMISSION_READ_WRITE_CONFIG) {
			return NewSyncErrorAuth(fmt.Errorf("peer %q is not allowed to update plan %q", c.peer.InstanceId, plan.Id))
		}

		// Update the plan in the local config
		idx := slices.IndexFunc(latestConfig.Plans, func(p *v1.Plan) bool {
			return p.Id == plan.Id
		})
		if idx >= 0 {
			latestConfig.Plans[idx] = plan
		} else {
			latestConfig.Plans = append(latestConfig.Plans, plan)
		}
	}

	for _, repo := range item.GetRepos() {
		c.l.Sugar().Debugf("received repo update: %s", repo.Guid)
		if !c.permissions.CheckPermissionForRepo(repo.Id, v1.Multihost_Permission_PERMISSION_READ_WRITE_CONFIG) {
			return NewSyncErrorAuth(fmt.Errorf("peer %q is not allowed to update repo %q", c.peer.InstanceId, repo.Id))
		}

		// Update the repo in the local config
		idx := slices.IndexFunc(latestConfig.Repos, func(r *v1.Repo) bool {
			return r.Guid == repo.Guid
		})
		if idx >= 0 {
			latestConfig.Repos[idx] = repo
		} else {
			latestConfig.Repos = append(latestConfig.Repos, repo)
		}
	}

	for _, plan := range item.GetPlansToDelete() {
		c.l.Sugar().Debugf("received plan deletion request: %s", plan)
		if !c.permissions.CheckPermissionForPlan(plan, v1.Multihost_Permission_PERMISSION_READ_WRITE_CONFIG) {
			return NewSyncErrorAuth(fmt.Errorf("peer %q is not allowed to delete plan %q", c.peer.InstanceId, plan))
		}

		// Remove the plan from the local config
		idx := slices.IndexFunc(latestConfig.Plans, func(p *v1.Plan) bool {
			return p.Id == plan
		})
		if idx >= 0 {
			latestConfig.Plans = append(latestConfig.Plans[:idx], latestConfig.Plans[idx+1:]...)
		} else {
			c.l.Sugar().Warnf("received plan deletion request for non-existent plan %q, ignoring", plan)
		}
	}

	for _, repo := range item.GetReposToDelete() {
		c.l.Sugar().Debugf("received repo deletion request: %s", repo)
		if !c.permissions.CheckPermissionForRepo(repo, v1.Multihost_Permission_PERMISSION_READ_WRITE_CONFIG) {
			return NewSyncErrorAuth(fmt.Errorf("peer %q is not allowed to delete repo %q", c.peer.InstanceId, repo))
		}

		// Remove the repo from the local config
		idx := slices.IndexFunc(latestConfig.Repos, func(r *v1.Repo) bool {
			return r.Id == repo
		})
		if idx >= 0 {
			latestConfig.Repos = append(latestConfig.Repos[:idx], latestConfig.Repos[idx+1:]...)
		} else {
			c.l.Sugar().Warnf("received repo deletion request for non-existent repo %q, ignoring", repo)
		}
	}

	// Update the local config with the new changes
	latestConfig.Modno++
	if err := c.mgr.configMgr.Update(latestConfig); err != nil {
		return fmt.Errorf("set updated config: %w", err)
	}

	return nil
}

func (c *syncSessionHandlerClient) HandleListResources(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionListResources) error {
	c.l.Sugar().Debugf("received ListResources request from peer %q", c.peer.InstanceId)
	return nil
}
