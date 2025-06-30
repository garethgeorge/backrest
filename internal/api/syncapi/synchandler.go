package syncapi

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1/v1connect"
	"github.com/garethgeorge/backrest/internal/api/syncapi/permissions"
	"github.com/garethgeorge/backrest/internal/env"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/protoutil"
	lru "github.com/hashicorp/golang-lru/v2"
	"go.uber.org/zap"
)

const SyncProtocolVersion = 1

type BackrestSyncHandler struct {
	v1connect.UnimplementedBackrestSyncServiceHandler
	mgr *SyncManager
}

var _ v1connect.BackrestSyncServiceHandler = &BackrestSyncHandler{}

func NewBackrestSyncHandler(mgr *SyncManager) *BackrestSyncHandler {
	return &BackrestSyncHandler{
		mgr: mgr,
	}
}

func (h *BackrestSyncHandler) Sync(ctx context.Context, stream *connect.BidiStream[v1.SyncStreamItem, v1.SyncStreamItem]) error {
	// TODO: this request can be very long lived, we must periodically refresh the config
	// e.g. to disconnect a client if its access is revoked.
	snapshot := h.mgr.getSyncConfigSnapshot()
	if snapshot == nil {
		return connect.NewError(connect.CodePermissionDenied, errors.New("sync server is not configured"))
	}

	sessionHandler := newSyncHandlerServer(h.mgr, snapshot)
	cmdStream := newBidiSyncCommandStream()

	go func() {
		err := runSync(
			ctx,
			snapshot.config.Instance,
			snapshot.identityKey,
			cmdStream,
			sessionHandler,
			snapshot.config.GetMultihost().GetAuthorizedClients(),
		)
		cmdStream.SendErrorAndTerminate(err)
	}()

	if err := cmdStream.ConnectStream(ctx, stream); err != nil {
		if sessionHandler.peer != nil {
			zap.S().Errorf("sync handler stream error for client %q: %v", sessionHandler.peer.InstanceId, err)
			h.mgr.peerStateManager.UpdatePeerState(sessionHandler.peer.Keyid, sessionHandler.peer.InstanceId, func(peerState *PeerState) {
				peerState.LastHeartbeat = time.Now()
				peerState.ConnectionState = v1.SyncConnectionState_CONNECTION_STATE_DISCONNECTED
				peerState.ConnectionStateMessage = err.Error()
				var syncErr *SyncError
				if errors.As(err, &syncErr) {
					peerState.ConnectionState = syncErr.State
					peerState.ConnectionStateMessage = syncErr.Message.Error()
				}
			})
		} else {
			zap.S().Errorf("sync handler stream error for unestablished session: %v", err)
		}

		var syncErr *SyncError
		if errors.As(err, &syncErr) {
			switch syncErr.State {
			case v1.SyncConnectionState_CONNECTION_STATE_ERROR_AUTH:
				return connect.NewError(connect.CodePermissionDenied, syncErr.Message)
			case v1.SyncConnectionState_CONNECTION_STATE_ERROR_PROTOCOL:
				return connect.NewError(connect.CodeInvalidArgument, syncErr.Message)
			default:
				return connect.NewError(connect.CodeInternal, syncErr.Message)
			}
		}
		return connect.NewError(connect.CodeInternal, err)
	} else {
		h.mgr.peerStateManager.UpdatePeerState(sessionHandler.peer.Keyid, sessionHandler.peer.InstanceId, func(peerState *PeerState) {
			peerState.LastHeartbeat = time.Now()
			peerState.ConnectionState = v1.SyncConnectionState_CONNECTION_STATE_DISCONNECTED
			peerState.ConnectionStateMessage = "disconnected"
		})
		zap.S().Infof("sync handler stream closed for client %q", sessionHandler.peer.InstanceId)
	}

	return nil
}

// syncSessionHandlerServer is a syncSessionHandler implementation for servers.
type syncSessionHandlerServer struct {
	unimplementedSyncSessionHandler

	mgr      *SyncManager
	snapshot syncConfigSnapshot

	peer        *v1.Multihost_Peer // The authorized client peer this handler is associated with, set during OnConnectionEstablished.
	permissions *permissions.PermissionSet

	opIDLru   *lru.Cache[int64, int64] // original ID -> local ID
	flowIDLru *lru.Cache[int64, int64] // original flow ID -> local flow ID

	configWatchCh chan struct{} // Channel for configuration updates
}

func newSyncHandlerServer(mgr *SyncManager, snapshot *syncConfigSnapshot) *syncSessionHandlerServer {
	opIDLru, _ := lru.New[int64, int64](4096)   // original ID -> local ID
	flowIDLru, _ := lru.New[int64, int64](1024) // original flow ID -> local flow ID

	return &syncSessionHandlerServer{
		mgr:       mgr,
		snapshot:  *snapshot,
		opIDLru:   opIDLru,
		flowIDLru: flowIDLru,
	}
}

var _ syncSessionHandler = (*syncSessionHandlerServer)(nil)

func (h *syncSessionHandlerServer) OnConnectionEstablished(ctx context.Context, stream *bidiSyncCommandStream, peer *v1.Multihost_Peer) error {
	// Verify that the peer is in our authorized clients list
	authorizedClientPeerIdx := slices.IndexFunc(h.snapshot.config.Multihost.GetAuthorizedClients(), func(p *v1.Multihost_Peer) bool {
		return p.InstanceId == peer.InstanceId && p.Keyid == peer.Keyid
	})
	if authorizedClientPeerIdx == -1 {
		zap.S().Warnf("syncserver rejected a connection from client instance ID %q because it is not authorized", peer.InstanceId)
		return NewSyncErrorAuth(errors.New("client is not an authorized peer"))
	}

	h.peer = h.snapshot.config.Multihost.AuthorizedClients[authorizedClientPeerIdx]

	var err error
	h.permissions, err = permissions.NewPermissionSet(h.peer.GetPermissions())
	if err != nil {
		zap.S().Warnf("syncserver failed to create permission set for client %q: %v", peer.InstanceId, err)
		return NewSyncErrorInternal(fmt.Errorf("failed to create permission set for client %q: %w", peer.InstanceId, err))
	}

	if !h.peer.KeyidVerified {
		return NewSyncErrorAuth(fmt.Errorf("client %q is not visually verified, please verify the key ID %q", peer.InstanceId, h.peer.Keyid))
	}

	// Configure the state for the connected peer.
	h.mgr.peerStateManager.UpdatePeerState(h.peer.Keyid, peer.InstanceId, func(peerState *PeerState) {
		peerState.ConnectionStateMessage = "connected"
		peerState.ConnectionState = v1.SyncConnectionState_CONNECTION_STATE_CONNECTED
		peerState.LastHeartbeat = time.Now()
	})

	zap.S().Infof("syncserver accepted a connection from client instance ID %q", h.peer.InstanceId)

	// start a heartbeat thread
	go sendHeartbeats(ctx, stream, env.MultihostHeartbeatInterval())

	// subscribe to our own configuration for changes
	h.configWatchCh = h.mgr.configMgr.OnChange.Subscribe()
	go func() {
		defer h.mgr.configMgr.OnChange.Unsubscribe(h.configWatchCh)
		for {
			select {
			case <-h.configWatchCh:
				newConfig, err := h.mgr.configMgr.Get()
				if err != nil {
					zap.S().Warnf("syncserver failed to get the newest config: %v", err)
					continue
				}
				if err := h.sendConfigToClient(stream, newConfig); err != nil {
					zap.S().Errorf("failed to send updated config to client: %v", err)
					stream.SendErrorAndTerminate(fmt.Errorf("sending updated config: %w", err))
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Send initial configuration to client
	return h.sendConfigToClient(stream, h.snapshot.config)
}

func (h *syncSessionHandlerServer) HandleHeartbeat(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionHeartbeat) error {
	h.mgr.peerStateManager.UpdatePeerState(h.peer.Keyid, h.peer.InstanceId, func(peerState *PeerState) {
		if peerState == nil {
			return // this should not happen
		}
		peerState.LastHeartbeat = time.Now()
	})
	return nil
}

func (h *syncSessionHandlerServer) HandleDiffOperations(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionDiffOperations) error {
	diffSel := item.GetHaveOperationsSelector()
	if diffSel == nil {
		return NewSyncErrorProtocol(errors.New("action DiffOperations: selector is required"))
	}

	// The diff selector _must_ select operations owned by the client's keyid, otherwise there are no restrictions.
	if diffSel.GetOriginalInstanceKeyid() != h.peer.Keyid {
		return NewSyncErrorProtocol(fmt.Errorf("action DiffOperations: selector must select operations owned by the client's keyid %q, got %q", h.peer.Keyid, diffSel.GetOriginalInstanceKeyid()))
	}

	// These are required to be the same length for a pairwise zip.
	if len(item.HaveOperationIds) != len(item.HaveOperationModnos) {
		return NewSyncErrorProtocol(errors.New("action DiffOperations: operation IDs and modnos must be the same length"))
	}

	diffSelQuery, err := protoutil.OpSelectorToQuery(diffSel)
	if err != nil {
		return fmt.Errorf("action DiffOperations: converting diff selector to query: %w", err)
	}

	localMetadata := []oplog.OpMetadata{}
	if err := h.mgr.oplog.QueryMetadata(diffSelQuery, func(metadata oplog.OpMetadata) error {
		if metadata.OriginalID == 0 {
			return nil // skip operations that didn't come from a remote
		}
		localMetadata = append(localMetadata, metadata)
		return nil
	}); err != nil {
		return fmt.Errorf("action DiffOperations: querying local metadata: %w", err)
	}
	sort.Slice(localMetadata, func(i, j int) bool {
		return localMetadata[i].OriginalID < localMetadata[j].OriginalID
	})

	remoteMetadata := make([]oplog.OpMetadata, len(item.HaveOperationIds))
	for i, id := range item.HaveOperationIds {
		remoteMetadata[i] = oplog.OpMetadata{
			ID:    id,
			Modno: item.HaveOperationModnos[i],
		}
	}
	sort.Slice(remoteMetadata, func(i, j int) bool {
		return remoteMetadata[i].ID < remoteMetadata[j].ID
	})

	requestDueToModno := 0
	requestMissingRemote := 0
	requestMissingLocal := 0
	requestIDs := []int64{}

	// This is a simple O(n) diff algorithm that compares the local and remote metadata vectors.
	localIndex := 0
	remoteIndex := 0
	for localIndex < len(localMetadata) && remoteIndex < len(remoteMetadata) {
		local := localMetadata[localIndex]
		remote := remoteMetadata[remoteIndex]

		if local.OriginalID == remote.ID {
			if local.Modno != remote.Modno {
				requestIDs = append(requestIDs, local.OriginalID)
				requestDueToModno++
			}
			localIndex++
			remoteIndex++
		} else if local.OriginalID < remote.ID {
			// the ID is found locally not remotely, request it and see if we get a delete event back
			// from the client indicating that the operation was deleted.
			requestIDs = append(requestIDs, local.OriginalID)
			localIndex++
			requestMissingLocal++
		} else {
			// the ID is found remotely not locally, request it for initial sync.
			requestIDs = append(requestIDs, remote.ID)
			remoteIndex++
			requestMissingRemote++
		}
	}
	for localIndex < len(localMetadata) {
		requestIDs = append(requestIDs, localMetadata[localIndex].OriginalID)
		localIndex++
		requestMissingLocal++
	}
	for remoteIndex < len(remoteMetadata) {
		requestIDs = append(requestIDs, remoteMetadata[remoteIndex].ID)
		remoteIndex++
		requestMissingRemote++
	}

	zap.L().Debug("syncserver diff operations with client metadata",
		zap.String("client_instance_id", h.peer.InstanceId),
		zap.Any("query", diffSelQuery),
		zap.Int("request_due_to_modno", requestDueToModno),
		zap.Int("request_local_but_not_remote", requestMissingLocal),
		zap.Int("request_remote_but_not_local", requestMissingRemote),
		zap.Int("request_ids_total", len(requestIDs)),
	)
	if len(requestIDs) > 0 {
		zap.L().Debug("syncserver sending request operations to client", zap.String("client_instance_id", h.peer.InstanceId), zap.Any("request_ids", requestIDs))
		stream.Send(&v1.SyncStreamItem{
			Action: &v1.SyncStreamItem_DiffOperations{
				DiffOperations: &v1.SyncStreamItem_SyncActionDiffOperations{
					RequestOperations: requestIDs,
				},
			},
		})
	}

	return nil
}

func (h *syncSessionHandlerServer) HandleSendOperations(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionSendOperations) error {
	switch event := item.GetEvent().Event.(type) {
	case *v1.OperationEvent_CreatedOperations:
		zap.L().Debug("syncserver received created operations", zap.Any("operations", event.CreatedOperations.GetOperations()))
		for _, op := range event.CreatedOperations.GetOperations() {
			if err := h.insertOrUpdate(op); err != nil {
				return fmt.Errorf("action SendOperations: operation event create %+v: %w", op, err)
			}
		}
	case *v1.OperationEvent_UpdatedOperations:
		zap.L().Debug("syncserver received update operations", zap.Any("operations", event.UpdatedOperations.GetOperations()))
		for _, op := range event.UpdatedOperations.GetOperations() {
			if err := h.insertOrUpdate(op); err != nil {
				return fmt.Errorf("action SendOperations: operation event update %+v: %w", op, err)
			}
		}
	case *v1.OperationEvent_DeletedOperations:
		zap.L().Debug("syncserver received delete operations", zap.Any("operations", event.DeletedOperations.GetValues()))
		for _, id := range event.DeletedOperations.GetValues() {
			if err := h.deleteByOriginalID(id); err != nil {
				return fmt.Errorf("action SendOperations: operation event delete %d: %w", id, err)
			}
		}
	case *v1.OperationEvent_KeepAlive:
	default:
		return NewSyncErrorProtocol(errors.New("action SendOperations: unknown event type"))
	}
	return nil
}

func (h *syncSessionHandlerServer) HandleSendConfig(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionSendConfig) error {
	h.mgr.peerStateManager.UpdatePeerState(h.peer.Keyid, h.peer.InstanceId, func(peerState *PeerState) {
		if peerState == nil {
			return // this should not happen
		}
		peerState.Config = item.GetConfig()
	})
	return nil
}

func (h *syncSessionHandlerServer) HandleListResources(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionListResources) error {
	zap.L().Debug("syncserver received resource list from client", zap.String("client_instance_id", h.peer.InstanceId),
		zap.Any("repos", item.GetRepoIds()),
		zap.Any("plans", item.GetPlanIds()))
	h.mgr.peerStateManager.UpdatePeerState(h.peer.Keyid, h.peer.InstanceId, func(peerState *PeerState) {
		if peerState == nil {
			return // this should not happen
		}

		repos := item.GetRepoIds()
		plans := item.GetPlanIds()
		for _, repoID := range repos {
			peerState.KnownRepos[repoID] = struct{}{}
		}
		for _, planID := range plans {
			peerState.KnownPlans[planID] = struct{}{}
		}
	})
	return nil
}

func (h *syncSessionHandlerServer) insertOrUpdate(op *v1.Operation) error {
	op.OriginalInstanceKeyid = h.peer.Keyid
	op.OriginalId = op.Id
	op.OriginalFlowId = op.FlowId
	op.Id = 0
	op.FlowId = 0

	var ok bool
	if op.Id, ok = h.opIDLru.Get(op.OriginalId); !ok {
		var foundOp *v1.Operation
		if err := h.mgr.oplog.Query(oplog.Query{}.
			SetOriginalInstanceKeyid(op.OriginalInstanceKeyid).
			SetOriginalID(op.OriginalId), func(o *v1.Operation) error {
			foundOp = o
			return nil
		}); err != nil {
			return fmt.Errorf("mapping remote ID to local ID: %w", err)
		}
		if foundOp != nil {
			op.Id = foundOp.Id
			h.opIDLru.Add(foundOp.Id, foundOp.Id)
		}
	}
	if op.FlowId, ok = h.flowIDLru.Get(op.OriginalFlowId); !ok {
		var flowOp *v1.Operation
		if err := h.mgr.oplog.Query(oplog.Query{}.
			SetOriginalInstanceKeyid(op.OriginalInstanceKeyid).
			SetOriginalFlowID(op.OriginalFlowId), func(o *v1.Operation) error {
			flowOp = o
			return nil
		}); err != nil {
			return fmt.Errorf("mapping remote flow ID to local ID: %w", err)
		}
		if flowOp != nil {
			op.FlowId = flowOp.FlowId
			h.flowIDLru.Add(op.OriginalFlowId, flowOp.FlowId)
		}
	}

	return h.mgr.oplog.Set(op)
}

func (h *syncSessionHandlerServer) deleteByOriginalID(originalID int64) error {
	var foundOp *v1.Operation
	if err := h.mgr.oplog.Query(oplog.Query{}.
		SetOriginalInstanceKeyid(h.peer.Keyid).
		SetOriginalID(originalID), func(o *v1.Operation) error {
		foundOp = o
		return nil
	}); err != nil {
		return fmt.Errorf("mapping remote ID to local ID: %w", err)
	}

	if foundOp == nil {
		zap.S().Debugf("syncserver received delete for non-existent operation %v", originalID)
		return nil
	}

	return h.mgr.oplog.Delete(foundOp.Id)
}

func (h *syncSessionHandlerServer) sendConfigToClient(stream *bidiSyncCommandStream, config *v1.Config) error {
	remoteConfig := &v1.RemoteConfig{
		Version: config.Version,
		Modno:   config.Modno,
	}
	resourceListMsg := &v1.SyncStreamItem_SyncActionListResources{}
	var allowedRepoIDs []string
	var allowedPlanIDs []string
	for _, repo := range config.Repos {
		if h.permissions.CheckPermissionForRepo(repo.Id, v1.Multihost_Permission_PERMISSION_READ_CONFIG) {
			remoteConfig.Repos = append(remoteConfig.Repos, repo)
			resourceListMsg.RepoIds = append(resourceListMsg.RepoIds, repo.Id)
			allowedRepoIDs = append(allowedRepoIDs, repo.Id)
		}
	}
	for _, plan := range config.Plans {
		if h.permissions.CheckPermissionForPlan(plan.Id, v1.Multihost_Permission_PERMISSION_READ_CONFIG) {
			remoteConfig.Plans = append(remoteConfig.Plans, plan)
			resourceListMsg.PlanIds = append(resourceListMsg.PlanIds, plan.Id)
			allowedPlanIDs = append(allowedPlanIDs, plan.Id)
		}
	}
	zap.S().Debugf("syncserver determined client %v is allowlisted to read configs for repos %v and plans %v", h.peer.InstanceId, allowedRepoIDs, allowedPlanIDs)

	// Send the config, this is the first meaningful packet the client will receive.
	// Once configuration is received, the client will start sending diffs.
	stream.Send(&v1.SyncStreamItem{
		Action: &v1.SyncStreamItem_SendConfig{
			SendConfig: &v1.SyncStreamItem_SyncActionSendConfig{
				Config: remoteConfig,
			},
		},
	})

	// Send the updated list of resources that the client can access.
	stream.Send(&v1.SyncStreamItem{
		Action: &v1.SyncStreamItem_ListResources{
			ListResources: resourceListMsg,
		},
	})
	return nil
}
