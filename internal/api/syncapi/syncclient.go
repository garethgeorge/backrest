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
				c.peer.GetInitialPairingSecret(),
				nil, // client never handles unknown peers
			)
			cmdStream.SendErrorAndTerminate(err)
		}()

		connectErr := cmdStream.ConnectStream(ctx, c.client.Sync(ctx))
		if connectErr != nil {
			c.l.Sugar().Infof("lost stream connection to peer %q (%s): %v", c.peer.InstanceId, c.peer.Keyid, connectErr)
			var syncErr *SyncError
			state := c.mgr.peerStateManager.GetPeerState(c.peer.Keyid).Clone()
			if state == nil {
				state = newPeerState(c.peer.InstanceId, c.peer.Keyid)
			}
			state.LastHeartbeat = time.Now()
			if errors.As(connectErr, &syncErr) {
				state.ConnectionState = syncErr.State
				state.ConnectionStateMessage = syncErr.Message.Error()
			} else {
				state.ConnectionState = v1sync.ConnectionState_CONNECTION_STATE_ERROR_INTERNAL
				state.ConnectionStateMessage = connectErr.Error()
			}
			c.mgr.peerStateManager.SetPeerState(c.peer.Keyid, state)
		}

		wg.Wait()

		// Reset reconnect backoff if the session lasted long enough to be considered a real success,
		// rather than a handshake that failed immediately. Using reconnectDelay as the threshold means
		// any session that ran at least one full retry window counts as stable.
		if time.Since(lastConnect) >= c.reconnectDelay {
			c.reconnectAttempts = 0
		}

		delay := c.reconnectDelay - time.Since(lastConnect)
		if c.reconnectAttempts > 0 {
			backoff := time.Duration(1<<min(c.reconnectAttempts, 5)) * c.reconnectDelay // 2^reconnectAttempts, max 32
			delay += backoff
		}
		if delay < 0 {
			delay = 0
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

	oplogSubscription *oplog.Subscription // set while subscribed; unsubscribed in OnConnectionDisconnected.
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

func (c *syncSessionHandlerClient) canForwardMeta(meta oplog.OpMetadata) bool {
	if meta.OriginalID != 0 {
		return false // don't forward ops received from other peers
	}
	if _, ok := c.canForwardReposSet[meta.RepoGUID]; ok {
		return true
	}
	if meta.PlanID != "" {
		if _, ok := c.canForwardPlansSet[meta.PlanID]; ok {
			return true
		}
	}
	return false
}

func (c *syncSessionHandlerClient) sendManifest(stream *bidiSyncCommandStream) (int, error) {
	var opIDs, modnos []int64
	if err := c.oplog.QueryMetadata(oplog.Query{}, func(meta oplog.OpMetadata) error {
		if c.canForwardMeta(meta) {
			opIDs = append(opIDs, meta.ID)
			modnos = append(modnos, meta.Modno)
		}
		return nil
	}); err != nil {
		return 0, fmt.Errorf("querying operation metadata for manifest: %w", err)
	}
	stream.Send(&v1sync.SyncStreamItem{
		Action: &v1sync.SyncStreamItem_OperationManifest{
			OperationManifest: &v1sync.SyncStreamItem_SyncActionOperationManifest{
				OpIds:  opIDs,
				Modnos: modnos,
			},
		},
	})
	return len(opIDs), nil
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

	// Clear the pairing secret from the known host entry now that pairing has succeeded.
	snapshotHosts := c.syncConfigSnapshot.config.GetMultihost().GetKnownHosts()
	khIdx := slices.IndexFunc(snapshotHosts, func(kh *v1.Multihost_Peer) bool {
		return kh.GetKeyid() == peer.GetKeyid()
	})
	if khIdx >= 0 && snapshotHosts[khIdx].GetInitialPairingSecret() != "" {
		if err := c.mgr.configMgr.Transform(func(cfg *v1.Config) (*v1.Config, error) {
			idx := slices.IndexFunc(cfg.GetMultihost().GetKnownHosts(), func(kh *v1.Multihost_Peer) bool {
				return kh.GetKeyid() == peer.GetKeyid()
			})
			if idx >= 0 {
				cfg.GetMultihost().GetKnownHosts()[idx].InitialPairingSecret = ""
			}
			cfg.Modno++
			return cfg, nil
		}); err != nil {
			c.l.Sugar().Warnf("failed to clear pairing secret after successful pairing: %v", err)
		} else {
			c.l.Sugar().Infof("cleared pairing secret for peer %q after successful connection", peer.InstanceId)
		}
	}

	// Send a heartbeat every interval to keep the connection alive.
	go sendHeartbeats(ctx, stream, env.MultihostHeartbeatInterval())

	// Forward a view of our config (if the peer is allowed to see it).
	repoCount, planCount, err := c.sendConfig(ctx, stream)
	if err != nil {
		return fmt.Errorf("send config to peer %q: %w", peer.InstanceId, err)
	}

	// Forward a list of the resources we're making available to the peer
	resRepoCount, resPlanCount, err := c.sendResourceList(ctx, stream)
	if err != nil {
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
	c.oplogSubscription = &oplogSubscription
	c.oplog.Subscribe(oplog.Query{}, c.oplogSubscription)

	// Send initial operation manifest to the server for reconciliation.
	opCount, err := c.sendManifest(stream)
	if err != nil {
		return fmt.Errorf("send manifest to peer %q: %w", peer.InstanceId, err)
	}

	c.l.Sugar().Infof("sent initial state to server: %d operations, %d repos, %d plans (config); %d repos, %d plans (resources)",
		opCount, repoCount, planCount, resRepoCount, resPlanCount)

	return nil
}

func (c *syncSessionHandlerClient) OnConnectionDisconnected() {
	if c.oplogSubscription != nil {
		c.oplog.Unsubscribe(c.oplogSubscription)
		c.oplogSubscription = nil
	}
}

func (c *syncSessionHandlerClient) HandleRequestResources(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionRequestResources) error {
	_, _, err := c.sendResourceList(ctx, stream)
	return err
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

func (c *syncSessionHandlerClient) HandleOperationManifest(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionOperationManifest) error {
	// Server re-requested a manifest (e.g. after reconnect). Respond with a fresh one.
	opCount, err := c.sendManifest(stream)
	if err != nil {
		return err
	}
	c.l.Sugar().Debugf("re-sent operation manifest with %d operations", opCount)
	return nil
}

func (c *syncSessionHandlerClient) HandleRequestOperationData(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionRequestOperationData) error {
	var batch []*v1.Operation
	send := func() {
		if len(batch) == 0 {
			return
		}
		stream.Send(&v1sync.SyncStreamItem{
			Action: &v1sync.SyncStreamItem_ReceiveOperations{
				ReceiveOperations: &v1sync.SyncStreamItem_SyncActionReceiveOperations{
					Event: &v1.OperationEvent{
						Event: &v1.OperationEvent_UpdatedOperations{
							UpdatedOperations: &v1.OperationList{Operations: batch},
						},
					},
				},
			},
		})
		batch = batch[:0]
	}
	for _, id := range item.GetOpIds() {
		op, err := c.oplog.Get(id)
		if err != nil {
			continue // may have been deleted between manifest and request
		}
		batch = append(batch, op)
		if len(batch) >= 256 {
			send()
		}
	}
	send()
	return nil
}

func (c *syncSessionHandlerClient) HandleReceiveOperations(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionReceiveOperations) error {
	return NewSyncErrorProtocol(errors.New("client should not receive ReceiveOperations messages, this is a host-only message"))
}

func (c *syncSessionHandlerClient) HandleReceiveResources(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionReceiveResources) error {
	c.l.Sugar().Debugf("received resource list from server: %d repos, %d plans", len(item.GetRepos()), len(item.GetPlans()))
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
	c.l.Sugar().Debugf("received remote config: %d repos, %d plans, modno=%d",
		len(item.GetConfig().GetRepos()), len(item.GetConfig().GetPlans()), item.GetConfig().GetModno())
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
	return c.mgr.configMgr.Transform(func(cfg *v1.Config) (*v1.Config, error) {
		snapshot := proto.Clone(cfg).(*v1.Config) // snapshot for change detection

		var plansNew, plansUpdated, plansUnchanged int
		for _, plan := range item.GetPlans() {
			if !c.permissions.CheckPermissionForPlan(plan.Id, permissions.PermsCanWriteConfiguration...) {
				return nil, NewSyncErrorAuth(fmt.Errorf("peer %q is not allowed to update plan %q", c.peer.InstanceId, plan.Id))
			}

			idx := slices.IndexFunc(cfg.Plans, func(p *v1.Plan) bool {
				return p.Id == plan.Id
			})
			if idx >= 0 {
				if proto.Equal(cfg.Plans[idx], plan) {
					c.l.Sugar().Debugf("received plan %s (unchanged)", plan.Id)
					plansUnchanged++
				} else {
					c.l.Sugar().Debugf("received plan %s (updated)", plan.Id)
					plansUpdated++
				}
				cfg.Plans[idx] = plan
			} else {
				c.l.Sugar().Debugf("received plan %s (new)", plan.Id)
				plansNew++
				cfg.Plans = append(cfg.Plans, plan)
			}
		}

		var reposNew, reposUpdated, reposUnchanged, reposSkipped int
		for _, repo := range item.GetRepos() {
			idx := slices.IndexFunc(cfg.Repos, func(r *v1.Repo) bool {
				return r.Guid == repo.Guid
			})

			// Permission check: accept if we have RECEIVE_SHARED_REPOS and the repo
			// is either new or already owned by this peer; otherwise require scoped write perms.
			isNewOrOwnedByPeer := idx < 0 || cfg.Repos[idx].GetOriginInstanceId() == c.peer.InstanceId
			allowed := (isNewOrOwnedByPeer && c.permissions.HasPermissionType(permissions.PermsCanReceiveSharedRepos...)) ||
				c.permissions.CheckPermissionForRepo(repo.Id, permissions.PermsCanWriteConfiguration...)
			if !allowed {
				return nil, NewSyncErrorAuth(fmt.Errorf("peer %q is not allowed to update repo %q", c.peer.InstanceId, repo.Id))
			}

			if idx >= 0 {
				if proto.Equal(cfg.Repos[idx], repo) {
					c.l.Sugar().Debugf("received repo %s (unchanged)", repo.Id)
					reposUnchanged++
				} else {
					c.l.Sugar().Debugf("received repo %s (updated)", repo.Id)
					reposUpdated++
				}
				cfg.Repos[idx] = repo
			} else {
				conflictIdx := slices.IndexFunc(cfg.Repos, func(r *v1.Repo) bool {
					return r.Id == repo.Id || r.Uri == repo.Uri
				})
				if conflictIdx >= 0 {
					c.l.Sugar().Warnf("received shared repo %q (guid %s) conflicts with existing local repo %q (guid %s), skipping", repo.Id, repo.Guid, cfg.Repos[conflictIdx].Id, cfg.Repos[conflictIdx].Guid)
					reposSkipped++
					continue
				}
				c.l.Sugar().Debugf("received repo %s (new)", repo.Id)
				reposNew++
				cfg.Repos = append(cfg.Repos, repo)
			}
		}

		var plansDeleted int
		for _, plan := range item.GetPlansToDelete() {
			if !c.permissions.CheckPermissionForPlan(plan, permissions.PermsCanWriteConfiguration...) {
				return nil, NewSyncErrorAuth(fmt.Errorf("peer %q is not allowed to delete plan %q", c.peer.InstanceId, plan))
			}

			idx := slices.IndexFunc(cfg.Plans, func(p *v1.Plan) bool {
				return p.Id == plan
			})
			if idx >= 0 {
				c.l.Sugar().Debugf("received plan deletion: %s", plan)
				plansDeleted++
				cfg.Plans = append(cfg.Plans[:idx], cfg.Plans[idx+1:]...)
			} else {
				c.l.Sugar().Warnf("received plan deletion request for non-existent plan %q, ignoring", plan)
			}
		}

		var reposDeleted int
		for _, repoID := range item.GetReposToDelete() {
			if !c.permissions.CheckPermissionForRepo(repoID, permissions.PermsCanWriteConfiguration...) {
				return nil, NewSyncErrorAuth(fmt.Errorf("peer %q is not allowed to delete repo %q", c.peer.InstanceId, repoID))
			}

			idx := slices.IndexFunc(cfg.Repos, func(r *v1.Repo) bool {
				return r.Id == repoID
			})
			if idx >= 0 {
				c.l.Sugar().Debugf("received repo deletion: %s", repoID)
				reposDeleted++
				cfg.Repos = append(cfg.Repos[:idx], cfg.Repos[idx+1:]...)
			} else {
				c.l.Sugar().Warnf("received repo deletion request for non-existent repo %q, ignoring", repoID)
			}
		}

		// Skip the update if nothing actually changed to avoid triggering a reconnect loop.
		hasChanges := !proto.Equal(cfg, snapshot)
		if hasChanges {
			cfg.Modno++
		}

		c.l.Sugar().Debugf("SetConfig from peer %q: repos(%d new, %d updated, %d unchanged, %d skipped, %d deleted) plans(%d new, %d updated, %d unchanged, %d deleted) — config %s",
			c.peer.GetInstanceId(),
			reposNew, reposUpdated, reposUnchanged, reposSkipped, reposDeleted,
			plansNew, plansUpdated, plansUnchanged, plansDeleted,
			map[bool]string{true: "updated", false: "unchanged"}[hasChanges])

		if !hasChanges {
			return nil, nil
		}
		return cfg, nil
	})
}

func (c *syncSessionHandlerClient) sendConfig(ctx context.Context, stream *bidiSyncCommandStream) (int, int, error) {
	localConfig := c.syncConfigSnapshot.config
	remoteConfig := &v1sync.RemoteConfig{
		Version: localConfig.Version,
		Modno:   localConfig.Modno,
	}

	for _, repo := range localConfig.Repos {
		if c.permissions.CheckPermissionForRepo(repo.Id, permissions.PermsCanViewConfiguration...) {
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

	return len(remoteConfig.Repos), len(remoteConfig.Plans), nil
}

func (c *syncSessionHandlerClient) sendResourceList(ctx context.Context, stream *bidiSyncCommandStream) (int, int, error) {
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

	return len(repoMetadatas), len(planMetadatas), nil
}
