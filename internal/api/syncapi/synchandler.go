package syncapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"
	"sync"
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

	mu sync.Mutex
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

	// Attach the 'Authorization' header to the sync stream as we create it.
	if authHeader, err := createAuthenticationHeader(snapshot.config.Instance, snapshot.identityKey); err != nil {
		zap.S().Errorf("failed to create authentication header: %v", err)
		return connect.NewError(connect.CodeInternal, fmt.Errorf("creating authentication header: %w", err))
	} else {
		stream.ResponseHeader().Set("Authorization", authHeader)
	}

	// Setup to read from the stream and handle commands
	// Note that runSync will perform the authentication check, which in this case is asserting that a peer is present.
	// The peer must be provided in the context by the authentication middleware.
	sessionHandler := newSyncHandlerServer(h.mgr, snapshot)
	cmdStream := newBidiSyncCommandStream()

	// Send a heartbeat packet to send the initial headers to the client and establish the connection.
	cmdStream.Send(&v1.SyncStreamItem{
		Action: &v1.SyncStreamItem_Heartbeat{},
	})

	go func() {
		err := runSync(
			ctx,
			snapshot.config.Instance,
			snapshot.identityKey,
			cmdStream,
			sessionHandler,
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

type syncSessionHandlerServer struct {
	unimplementedSyncSessionHandler

	mgr      *SyncManager
	snapshot syncConfigSnapshot

	peer        *v1.Multihost_Peer // The authorized client peer this handler is associated with, set during OnConnectionEstablished.
	permissions *permissions.PermissionSet

	opIDLru   *lru.Cache[int64, int64] // original ID -> local ID
	flowIDLru *lru.Cache[int64, int64] // original flow ID -> local flow ID

	requestedLogStreams map[string]struct{}
	activeLogStreams    map[string]io.WriteCloser

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

		requestedLogStreams: make(map[string]struct{}),
		activeLogStreams:    make(map[string]io.WriteCloser),
	}
}

var _ syncSessionHandler = (*syncSessionHandlerServer)(nil)

func (h *syncSessionHandlerServer) OnConnectionEstablished(ctx context.Context, stream *bidiSyncCommandStream, peer *v1.Multihost_Peer) error {
	// Check if the peer is already connected, and then store the connection.
	h.mgr.mu.Lock()
	if _, exists := h.mgr.sessionHandlerMap[peer.Keyid]; exists {
		h.mgr.mu.Unlock()
		return NewSyncErrorAuth(fmt.Errorf("client %q is already connected", peer.InstanceId))
	}
	h.mgr.sessionHandlerMap[peer.Keyid] = h
	h.mgr.mu.Unlock()

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
	if err := h.sendConfigToClient(stream, h.snapshot.config); err != nil {
		return err
	}

	return nil
}

func (h *syncSessionHandlerServer) OnConnectionClosed(ctx context.Context, stream *bidiSyncCommandStream) error {
	if h.peer != nil {
		zap.S().Infof("syncserver connection closed for client %q", h.peer.InstanceId)
		h.mgr.mu.Lock()
		delete(h.mgr.sessionHandlerMap, h.peer.Keyid)
		h.mgr.mu.Unlock()
	}

	// Close any active resources e.g. sinks for active log streams.
	for logID, logSink := range h.activeLogStreams {
		if err := logSink.Close(); err != nil {
			return fmt.Errorf("action SendLogData: closing log stream %q: %w", logID, err)
		}
	}

	return nil
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
		zap.L().Debug("syncserver received create operations", zap.Any("operations", event.CreatedOperations.GetOperations()))
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
		zap.Any("repos", item.GetRepos()),
		zap.Any("plans", item.GetPlans()))
	h.mgr.peerStateManager.UpdatePeerState(h.peer.Keyid, h.peer.InstanceId, func(peerState *PeerState) {
		if peerState == nil {
			return // this should not happen
		}

		repos := item.GetRepos()
		plans := item.GetPlans()
		for _, repo := range repos {
			peerState.KnownRepos[repo.Id] = repo
		}
		for _, plan := range plans {
			peerState.KnownPlans[plan.Id] = plan
		}
	})
	return nil
}

func (h *syncSessionHandlerServer) HandleSendLogData(ctx context.Context, stream *bidiSyncCommandStream, item *v1.SyncStreamItem_SyncActionSendLogData) error {
	h.mgr.mu.Lock()
	defer h.mgr.mu.Unlock()

	logID := item.GetLogId()
	if logID == "" {
		return NewSyncErrorProtocol(errors.New("action SendLogData: log ID is required"))
	}
	if _, exists := h.requestedLogStreams[logID]; !exists {
		return NewSyncErrorProtocol(fmt.Errorf("action SendLogData: log ID %q was not requested", logID))
	}

	var logSink io.WriteCloser
	if s, exists := h.activeLogStreams[logID]; !exists {
		// Check if there are too many active log streams
		if len(h.activeLogStreams) >= 16 {
			return NewSyncErrorProtocol(fmt.Errorf("action SendLogData: too many active log streams, limit is 16"))
		}

		// If the log stream is not active, we need to create a new one.
		f, err := h.mgr.logStore.Create(logID, 0, 24*3600) // 24 hour retention, will just be re-requested next time it's wanted.
		if err != nil {
			return fmt.Errorf("action SendLogData: creating log stream %q: %w", logID, err)
		}
		logSink = f
		h.activeLogStreams[logID] = logSink
	} else {
		logSink = s
	}

	if len(item.GetChunk()) == 0 {
		delete(h.activeLogStreams, logID)
		delete(h.requestedLogStreams, logID)
		if err := logSink.Close(); err != nil {
			return fmt.Errorf("action SendLogData: closing log stream %q: %w", logID, err)
		}
		return nil
	}

	if _, err := logSink.Write(item.GetChunk()); err != nil {
		return fmt.Errorf("action SendLogData: writing to log stream %q: %w", logID, err)
	}

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
			resourceListMsg.Repos = append(resourceListMsg.Repos, &v1.SyncRepoMetadata{Id: repo.Id, Guid: repo.Guid})
			allowedRepoIDs = append(allowedRepoIDs, repo.Id)
		}
	}
	for _, plan := range config.Plans {
		if h.permissions.CheckPermissionForPlan(plan.Id, v1.Multihost_Permission_PERMISSION_READ_CONFIG) {
			remoteConfig.Plans = append(remoteConfig.Plans, plan)
			resourceListMsg.Plans = append(resourceListMsg.Plans, &v1.SyncPlanMetadata{Id: plan.Id})
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
