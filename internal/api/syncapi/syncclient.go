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
	"github.com/garethgeorge/backrest/gen/go/v1sync"
	"github.com/garethgeorge/backrest/gen/go/v1sync/v1syncconnect"
	"github.com/garethgeorge/backrest/internal/api/syncapi/permissions"
	"github.com/garethgeorge/backrest/internal/env"
	"github.com/garethgeorge/backrest/internal/oplog"
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
	client             v1syncconnect.BackrestSyncServiceClient
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

	client := v1syncconnect.NewBackrestSyncServiceClient(
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
	c.mgr.peerStateManager.SetPeerState(peer.Keyid, newPeerState(peer.InstanceId, peer.Keyid))
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
			state := c.mgr.peerStateManager.GetPeerState(c.peer.Keyid).Clone()
			if state == nil {
				state = newPeerState(c.peer.InstanceId, c.peer.Keyid)
			}
			state.LastHeartbeat = time.Now()
			if errors.As(err, &syncErr) {
				state.ConnectionState = syncErr.State
				state.ConnectionStateMessage = syncErr.Message.Error()
			} else {
				state.ConnectionState = v1sync.ConnectionState_CONNECTION_STATE_ERROR_INTERNAL
				state.ConnectionStateMessage = err.Error()
			}
			c.mgr.peerStateManager.SetPeerState(c.peer.Keyid, state)
		} else {
			c.reconnectAttempts = 0
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

	canForwardReposSet map[string]struct{}
	canForwardPlansSet map[string]struct{}
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

		canForwardReposSet: make(map[string]struct{}),
		canForwardPlansSet: make(map[string]struct{}),
	}
}

var _ syncSessionHandler = (*syncSessionHandlerClient)(nil)

func (c *syncSessionHandlerClient) applyPermissions() {
	for _, plan := range c.syncConfigSnapshot.config.Plans {
		if c.permissions.CheckPermissionForPlan(plan.Id, v1.Multihost_Permission_PERMISSION_READ_OPERATIONS) {
			c.canForwardPlansSet[plan.Id] = struct{}{}
		}
	}
	for _, repo := range c.syncConfigSnapshot.config.Repos {
		if c.permissions.CheckPermissionForRepo(repo.Id, v1.Multihost_Permission_PERMISSION_READ_OPERATIONS) {
			c.canForwardReposSet[repo.Guid] = struct{}{}
		}
	}
}

func (c *syncSessionHandlerClient) canForwardOperation(op *v1.Operation) bool {
	if op.GetOriginalInstanceKeyid() != "" || op.GetInstanceId() != c.localInstanceID {
		return false // only forward operations that were created by this instance
	}
	if _, ok := c.canForwardReposSet[op.GetRepoGuid()]; ok {
		return true
	}
	if op.GetPlanId() != "" {
		if _, ok := c.canForwardPlansSet[op.GetPlanId()]; ok {
			return true
		}
	}
	return false
}

func (c *syncSessionHandlerClient) OnConnectionEstablished(ctx context.Context, stream *bidiSyncCommandStream, peer *v1.Multihost_Peer) error {
	// A client expects to connect to a specific peer, so we check that the peer we connected to matches the one we expect.
	if !proto.Equal(c.peer, peer) {
		return NewSyncErrorAuth(fmt.Errorf("peer mismatch: expected %s (%s), got %s (%s)", c.peer.Keyid, c.peer.InstanceId, peer.Keyid, peer.InstanceId))
	}

	// Set the peer and permissions for this connection.
	var err error
	c.peer = peer
	c.permissions, err = permissions.NewPermissionSet(peer.Permissions)
	if err != nil {
		return NewSyncErrorAuth(fmt.Errorf("creating permission set for peer %q: %w", peer.InstanceId, err))
	}
	c.applyPermissions()

	c.l.Sugar().Infof("sync connection established with peer %q (%s)", peer.InstanceId, peer.Keyid)
	peerState := c.mgr.peerStateManager.GetPeerState(peer.Keyid).Clone()
	if peerState == nil {
		peerState = newPeerState(c.peer.InstanceId, peer.Keyid)
	}
	peerState.ConnectionState = v1sync.ConnectionState_CONNECTION_STATE_CONNECTED
	peerState.ConnectionStateMessage = "connected"
	peerState.LastHeartbeat = time.Now()
	c.mgr.peerStateManager.SetPeerState(peer.Keyid, peerState)

	// Send a heartbeat every interval to keep the connection alive.
	go sendHeartbeats(ctx, stream, env.MultihostHeartbeatInterval())

	// Forward a view of our config (if the peer is allowed to see it).
	if err := c.sendConfig(ctx, stream); err != nil {
		return fmt.Errorf("send config to peer %q: %w", peer.InstanceId, err)
	}

	// Forward a list of the resources we're making available to the peer
	if err := c.sendResourceList(ctx, stream); err != nil {
		return fmt.Errorf("send resource list to peer %q: %w", peer.InstanceId, err)
	}

	// Subscribe to oplog updates and forward relevant operations to the peer.
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
		switch event {
		case oplog.OPERATION_ADDED:
			eventProto = &v1.OperationEvent{
				Event: &v1.OperationEvent_CreatedOperations{
					CreatedOperations: &v1.OperationList{Operations: opsToForward},
				},
			}
		case oplog.OPERATION_UPDATED:
			eventProto = &v1.OperationEvent{
				Event: &v1.OperationEvent_UpdatedOperations{
					UpdatedOperations: &v1.OperationList{Operations: opsToForward},
				},
			}
		case oplog.OPERATION_DELETED:
			ids := make([]int64, len(opsToForward))
			for i, op := range opsToForward {
				ids[i] = op.GetId()
			}
			eventProto = &v1.OperationEvent{
				Event: &v1.OperationEvent_DeletedOperations{
					DeletedOperations: &types.Int64List{Values: ids},
				},
			}
		default:
			stream.SendErrorAndTerminate(fmt.Errorf("unknown oplog event type: %v", event))
			return
		}

		stream.Send(&v1sync.SyncStreamItem{
			Action: &v1sync.SyncStreamItem_ReceiveOperations{
				ReceiveOperations: &v1sync.SyncStreamItem_SyncActionReceiveOperations{
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
	return nil
}

func (c *syncSessionHandlerClient) HandleRequestResources(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionRequestResources) error {
	return c.sendResourceList(ctx, stream)
}

func (c *syncSessionHandlerClient) HandleHeartbeat(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionHeartbeat) error {
	peerState := c.mgr.peerStateManager.GetPeerState(c.peer.Keyid).Clone()
	if peerState == nil {
		return NewSyncErrorInternal(fmt.Errorf("peer state not found for peer %q", c.peer.InstanceId))
	}
	peerState.LastHeartbeat = time.Now()
	c.mgr.peerStateManager.SetPeerState(c.peer.Keyid, peerState)
	return nil
}

func (c *syncSessionHandlerClient) HandleRequestOperations(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionRequestOperations) error {
	highModno := item.GetHighModno()
	highOpid := item.GetHighOpid()
	c.l.Sugar().Debugf("received operation request for high_modno: %d, high_opid: %d", highModno, highOpid)

	var batch []*v1.Operation

	send := func() error {
		if len(batch) == 0 {
			return nil
		}

		// find new and updated operations
		var newOps []*v1.Operation
		var updatedOps []*v1.Operation
		for _, op := range batch {
			if op.GetId() > highOpid {
				newOps = append(newOps, op)
			} else {
				updatedOps = append(updatedOps, op)
			}
		}

		// send new and updated operations
		if len(newOps) > 0 {
			stream.Send(&v1sync.SyncStreamItem{
				Action: &v1sync.SyncStreamItem_ReceiveOperations{
					ReceiveOperations: &v1sync.SyncStreamItem_SyncActionReceiveOperations{
						Event: &v1.OperationEvent{
							Event: &v1.OperationEvent_CreatedOperations{
								CreatedOperations: &v1.OperationList{Operations: batch},
							},
						},
					},
				},
			})
		}
		if len(updatedOps) > 0 {
			stream.Send(&v1sync.SyncStreamItem{
				Action: &v1sync.SyncStreamItem_ReceiveOperations{
					ReceiveOperations: &v1sync.SyncStreamItem_SyncActionReceiveOperations{
						Event: &v1.OperationEvent{
							Event: &v1.OperationEvent_UpdatedOperations{
								UpdatedOperations: &v1.OperationList{Operations: updatedOps},
							},
						},
					},
				},
			})
		}

		batch = batch[:0]
		return nil
	}

	c.oplog.Query(oplog.Query{}.SetModnoGte(highModno), func(op *v1.Operation) error {
		if !c.canForwardOperation(op) {
			return nil // skip operations that the peer is not allowed to read
		}

		batch = append(batch, op)
		if len(batch) >= 256 {
			if err := send(); err != nil {
				return err
			}
		}
		return nil
	})

	if err := send(); err != nil {
		return err
	}

	return nil
}

func (c *syncSessionHandlerClient) HandleReceiveOperations(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionReceiveOperations) error {
	return NewSyncErrorProtocol(errors.New("client should not receive ReceiveOperations messages, this is a host-only message"))
}

func (c *syncSessionHandlerClient) HandleReceiveResources(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionReceiveResources) error {
	c.l.Debug("received resource list from server",
		zap.Any("repos", item.GetRepos()),
		zap.Any("plans", item.GetPlans()))
	peerState := c.mgr.peerStateManager.GetPeerState(c.peer.Keyid).Clone()
	if peerState == nil {
		return NewSyncErrorInternal(fmt.Errorf("peer state for %q not found", c.peer.Keyid))
	}
	repos := item.GetRepos()
	plans := item.GetPlans()
	for _, repo := range repos {
		peerState.KnownRepos[repo.Id] = repo
	}
	for _, plan := range plans {
		peerState.KnownPlans[plan.Id] = plan
	}
	c.mgr.peerStateManager.SetPeerState(c.peer.Keyid, peerState)
	return nil
}

// Note unused: there isn't a situation where the host would send its config for information, the host will only call 'SetConfig' to update the config.
func (c *syncSessionHandlerClient) HandleReceiveConfig(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionReceiveConfig) error {
	c.l.Sugar().Debugf("received remote config update")
	peerState := c.mgr.peerStateManager.GetPeerState(c.peer.Keyid).Clone()
	if peerState == nil {
		return NewSyncErrorInternal(fmt.Errorf("peer state for %q not found", c.peer.Keyid))
	}
	newRemoteConfig := item.Config
	if newRemoteConfig == nil {
		return NewSyncErrorProtocol(fmt.Errorf("received nil remote config"))
	}
	peerState.Config = newRemoteConfig
	c.mgr.peerStateManager.SetPeerState(c.peer.Keyid, peerState)
	return nil
}

func (c *syncSessionHandlerClient) HandleSetConfig(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionSetConfig) error {
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
		if !c.permissions.CheckPermissionForPlan(plan.Id, permissions.PermsCanWriteConfiguration...) {
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
		if !c.permissions.CheckPermissionForRepo(repo.Id, permissions.PermsCanWriteConfiguration...) {
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
		if !c.permissions.CheckPermissionForPlan(plan, permissions.PermsCanWriteConfiguration...) {
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

	for _, repoID := range item.GetReposToDelete() {
		c.l.Sugar().Debugf("received repo deletion request: %s", repoID)
		if !c.permissions.CheckPermissionForRepo(repoID, permissions.PermsCanWriteConfiguration...) {
			return NewSyncErrorAuth(fmt.Errorf("peer %q is not allowed to delete repo %q", c.peer.InstanceId, repoID))
		}

		// Remove the repo from the local config
		idx := slices.IndexFunc(latestConfig.Repos, func(r *v1.Repo) bool {
			return r.Id == repoID
		})
		if idx >= 0 {
			latestConfig.Repos = append(latestConfig.Repos[:idx], latestConfig.Repos[idx+1:]...)
		} else {
			c.l.Sugar().Warnf("received repo deletion request for non-existent repo %q, ignoring", repoID)
		}
	}

	// Update the local config with the new changes
	latestConfig.Modno++
	if err := c.mgr.configMgr.Update(latestConfig); err != nil {
		return fmt.Errorf("set updated config: %w", err)
	}

	return nil
}

func (c *syncSessionHandlerClient) sendConfig(ctx context.Context, stream *bidiSyncCommandStream) error {
	localConfig := c.syncConfigSnapshot.config
	remoteConfig := &v1sync.RemoteConfig{
		Version: localConfig.Version,
		Modno:   localConfig.Modno,
	}

	for _, repo := range localConfig.Repos {
		if c.permissions.CheckPermissionForRepo(repo.Guid, permissions.PermsCanViewConfiguration...) {
			remoteConfig.Repos = append(remoteConfig.Repos, repo)
		}
	}

	for _, plan := range localConfig.Plans {
		if c.permissions.CheckPermissionForPlan(plan.Id, permissions.PermsCanViewConfiguration...) {
			remoteConfig.Plans = append(remoteConfig.Plans, plan)
		}
	}

	stream.Send(&v1sync.SyncStreamItem{
		Action: &v1sync.SyncStreamItem_ReceiveConfig{
			ReceiveConfig: &v1sync.SyncStreamItem_SyncActionReceiveConfig{
				Config: remoteConfig,
			},
		},
	})

	return nil
}

func (c *syncSessionHandlerClient) sendResourceList(ctx context.Context, stream *bidiSyncCommandStream) error {
	repoMetadatas := []*v1sync.RepoMetadata{}
	planMetadatas := []*v1sync.PlanMetadata{}

	for _, repo := range c.syncConfigSnapshot.config.Repos {
		if c.permissions.CheckPermissionForRepo(repo.Id, permissions.PermsCanViewResources...) {
			repoMetadatas = append(repoMetadatas, &v1sync.RepoMetadata{
				Id:   repo.Id,
				Guid: repo.Guid,
			})
		}
	}

	for _, plan := range c.syncConfigSnapshot.config.Plans {
		if c.permissions.CheckPermissionForPlan(plan.Id, permissions.PermsCanViewResources...) {
			planMetadatas = append(planMetadatas, &v1sync.PlanMetadata{
				Id: plan.Id,
			})
		}
	}

	stream.Send(&v1sync.SyncStreamItem{
		Action: &v1sync.SyncStreamItem_ReceiveResources{
			ReceiveResources: &v1sync.SyncStreamItem_SyncActionReceiveResources{
				Repos: repoMetadatas,
				Plans: planMetadatas,
			},
		},
	})

	return nil
}
