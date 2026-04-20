package syncapi

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1sync"
	"github.com/garethgeorge/backrest/gen/go/v1sync/v1syncconnect"
	"github.com/garethgeorge/backrest/internal/api/syncapi/permissions"
	"github.com/garethgeorge/backrest/internal/env"
	"github.com/garethgeorge/backrest/internal/oplog"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const SyncProtocolVersion = 1

type BackrestSyncHandler struct {
	v1syncconnect.UnimplementedBackrestSyncServiceHandler
	mgr    *SyncManager
	mapper *remoteOpIDMapper
}

var _ v1syncconnect.BackrestSyncServiceHandler = &BackrestSyncHandler{}

func NewBackrestSyncHandler(mgr *SyncManager) *BackrestSyncHandler {
	mapper, _ := newRemoteOpIDMapper(mgr.oplog, 4096) // error can be ignored, it just checks for valid size
	return &BackrestSyncHandler{
		mgr:    mgr,
		mapper: mapper,
	}
}

func (h *BackrestSyncHandler) Sync(ctx context.Context, stream *connect.BidiStream[v1sync.SyncStreamItem, v1sync.SyncStreamItem]) error {
	// TODO: this request can be very long lived, we must periodically refresh the config
	// e.g. to disconnect a client if its access is revoked.
	snapshot := h.mgr.getSyncConfigSnapshot()
	if snapshot == nil {
		return connect.NewError(connect.CodePermissionDenied, errors.New("sync server is not configured"))
	}

	sessionHandler := newSyncHandlerServer(h.mgr, snapshot, h.mapper)
	cmdStream := newBidiSyncCommandStream()

	go func() {
		err := runSync(
			ctx,
			snapshot.config.Instance,
			snapshot.identityKey,
			cmdStream,
			sessionHandler,
			snapshot.config.GetMultihost().GetAuthorizedClients(),
			"", // server never sends a pairing secret
			h.handleUnknownPeerPairing(snapshot),
		)
		cmdStream.SendErrorAndTerminate(err)
	}()

	if err := cmdStream.ConnectStream(ctx, stream); err != nil {
		zap.S().Errorf("sync handler stream error: %v", err)
		var syncErr *SyncError
		if errors.As(err, &syncErr) {
			switch syncErr.State {
			case v1sync.ConnectionState_CONNECTION_STATE_ERROR_AUTH:
				return connect.NewError(connect.CodePermissionDenied, syncErr.Message)
			case v1sync.ConnectionState_CONNECTION_STATE_ERROR_PROTOCOL:
				return connect.NewError(connect.CodeInvalidArgument, syncErr.Message)
			default:
				return connect.NewError(connect.CodeInternal, syncErr.Message)
			}
		}

		if sessionHandler.peer != nil {
			peerState := h.mgr.peerStateManager.GetPeerState(sessionHandler.peer.Keyid).Clone()
			if peerState == nil {
				peerState = newPeerState(sessionHandler.peer.InstanceId, sessionHandler.peer.Keyid)
			}
			if syncErr != nil {
				peerState.ConnectionState = syncErr.State
				peerState.ConnectionStateMessage = syncErr.Message.Error()
			} else if errors.Is(err, context.Canceled) {
				peerState.ConnectionState = v1sync.ConnectionState_CONNECTION_STATE_DISCONNECTED
				peerState.ConnectionStateMessage = "lost connection"
			} else {
				peerState.ConnectionState = v1sync.ConnectionState_CONNECTION_STATE_DISCONNECTED
				peerState.ConnectionStateMessage = fmt.Sprintf("disconnected: %v", err)
			}
			peerState.LastHeartbeat = time.Now()
			h.mgr.peerStateManager.SetPeerState(sessionHandler.peer.Keyid, peerState)
		}
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

	mapper *remoteOpIDMapper

	l *zap.Logger
}

func newSyncHandlerServer(mgr *SyncManager, snapshot *syncConfigSnapshot, mapper *remoteOpIDMapper) *syncSessionHandlerServer {
	return &syncSessionHandlerServer{
		mgr:      mgr,
		snapshot: *snapshot,
		mapper:   mapper,
		l:        zap.L().Named("syncserver handler for unknown peer"),
	}
}

var _ syncSessionHandler = (*syncSessionHandlerServer)(nil)

func (h *syncSessionHandlerServer) OnConnectionEstablished(ctx context.Context, stream *bidiSyncCommandStream, peer *v1.Multihost_Peer) error {
	// Verify that the peer is in our authorized clients list
	authorizedClientPeerIdx := slices.IndexFunc(h.snapshot.config.Multihost.GetAuthorizedClients(), func(p *v1.Multihost_Peer) bool {
		return p.InstanceId == peer.InstanceId && p.Keyid == peer.Keyid
	})
	if authorizedClientPeerIdx == -1 {
		h.l.Sugar().Warnf("rejected a connection from client instance ID %q because it is not authorized", peer.InstanceId)
		return NewSyncErrorAuth(errors.New("client is not an authorized peer"))
	}

	h.peer = h.snapshot.config.Multihost.AuthorizedClients[authorizedClientPeerIdx]
	h.l = zap.L().Named(fmt.Sprintf("syncserver handler for peer %q", h.peer.InstanceId))

	var err error
	h.permissions, err = permissions.NewPermissionSet(h.peer.GetPermissions())
	if err != nil {
		h.l.Sugar().Warnf("failed to create permission set for client %q: %v", peer.InstanceId, err)
		return NewSyncErrorInternal(fmt.Errorf("failed to create permission set for client %q: %w", peer.InstanceId, err))
	}

	// Configure the state for the connected peer.
	peerState := newPeerState(peer.InstanceId, h.peer.Keyid)
	peerState.ConnectionStateMessage = "connected"
	peerState.ConnectionState = v1sync.ConnectionState_CONNECTION_STATE_CONNECTED
	peerState.LastHeartbeat = time.Now()
	h.mgr.peerStateManager.SetPeerState(h.peer.Keyid, peerState)

	h.l.Sugar().Infof("accepted a connection from client instance ID %q", h.peer.InstanceId)

	// Register this peer's stream handle so the API layer can send messages to it.
	h.mgr.registerConnectedPeer(h.peer.Keyid, &connectedPeerHandle{
		stream:      stream,
		peer:        h.peer,
		permissions: h.permissions,
	})

	// start a heartbeat thread
	go sendHeartbeats(ctx, stream, env.MultihostHeartbeatInterval())

	// subscribe to our own configuration for changes
	go func() {
		configWatchCh := h.mgr.configMgr.OnChange.Subscribe()
		defer h.mgr.configMgr.OnChange.Unsubscribe(configWatchCh)
		for {
			select {
			case <-configWatchCh:
				newConfig, err := h.mgr.configMgr.Get()
				if err != nil {
					h.l.Sugar().Warnf("failed to get config on change: %v, disconnecting client", err)
					stream.SendErrorAndTerminate(nil)
					return
				}

				// Check if this peer is still authorized
				peerIdx := slices.IndexFunc(newConfig.Multihost.GetAuthorizedClients(), func(p *v1.Multihost_Peer) bool {
					return p.InstanceId == h.peer.InstanceId && p.Keyid == h.peer.Keyid
				})
				if peerIdx == -1 {
					h.l.Sugar().Infof("disconnecting client %q: no longer authorized", h.peer.InstanceId)
					stream.SendErrorAndTerminate(nil)
					return
				}

				// Check if permissions changed by comparing the proto peer definition
				updatedPeer := newConfig.Multihost.AuthorizedClients[peerIdx]
				if !proto.Equal(h.peer, updatedPeer) {
					h.l.Sugar().Infof("disconnecting client %q: peer configuration changed", h.peer.InstanceId)
					stream.SendErrorAndTerminate(nil)
					return
				}

				// Permissions unchanged — send updated config and shared repos to client
				h.l.Sugar().Debugf("config changed, sending updated config to client %q", h.peer.InstanceId)
				if err := h.sendConfigToClient(stream, newConfig); err != nil {
					h.l.Sugar().Warnf("failed to send updated config to client %q: %v", h.peer.InstanceId, err)
				}
				h.sendSharedReposToClient(stream, newConfig)
			case <-ctx.Done():
				return
			}
		}
	}()

	if err := h.sendConfigToClient(stream, h.snapshot.config); err != nil {
		return NewSyncErrorInternal(fmt.Errorf("sending initial config to client: %w", err))
	}

	// Push shared repos to the client
	h.sendSharedReposToClient(stream, h.snapshot.config)

	// send initial request for operation sync
	if err := h.sendOperationSyncRequest(stream); err != nil {
		return NewSyncErrorInternal(fmt.Errorf("sending initial operation sync request: %w", err))
	}

	return nil
}

func (h *syncSessionHandlerServer) OnConnectionDisconnected() {
	if h.peer != nil {
		h.mgr.unregisterConnectedPeer(h.peer.Keyid)
	}
}

func (h *syncSessionHandlerServer) HandleHeartbeat(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionHeartbeat) error {
	peerState := h.mgr.peerStateManager.GetPeerState(h.peer.Keyid).Clone()
	if peerState == nil {
		return NewSyncErrorInternal(fmt.Errorf("peer state for %q not found", h.peer.Keyid))
	}
	peerState.LastHeartbeat = time.Now()
	h.mgr.peerStateManager.SetPeerState(h.peer.Keyid, peerState)
	return nil
}

func (h *syncSessionHandlerServer) HandleReceiveOperations(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionReceiveOperations) error {
	switch event := item.GetEvent().Event.(type) {
	case *v1.OperationEvent_CreatedOperations:
		h.l.Debug("received created operations", zap.Any("operations", event.CreatedOperations.GetOperations()))
		for _, op := range event.CreatedOperations.GetOperations() {
			if err := h.insertOrUpdate(op, false /* isUpdate */); err != nil {
				return fmt.Errorf("action ReceiveOperations: operation event create %+v: %w", op, err)
			}
		}
	case *v1.OperationEvent_UpdatedOperations:
		h.l.Debug("received update operations", zap.Any("operations", event.UpdatedOperations.GetOperations()))
		for _, op := range event.UpdatedOperations.GetOperations() {
			if err := h.insertOrUpdate(op, true /* isUpdate */); err != nil {
				return fmt.Errorf("action ReceiveOperations: operation event update %+v: %w", op, err)
			}
		}
	case *v1.OperationEvent_DeletedOperations:
		h.l.Debug("received delete operations", zap.Any("operations", event.DeletedOperations.GetValues()))
		for _, id := range event.DeletedOperations.GetValues() {
			if err := h.deleteByOriginalID(id); err != nil {
				return fmt.Errorf("action ReceiveOperations: operation event delete %d: %w", id, err)
			}
		}
	case *v1.OperationEvent_KeepAlive:
	default:
		return NewSyncErrorProtocol(errors.New("action ReceiveOperations: unknown event type"))
	}
	return nil
}

func (h *syncSessionHandlerServer) HandleReceiveConfig(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionReceiveConfig) error {
	peerState := h.mgr.peerStateManager.GetPeerState(h.peer.Keyid).Clone()
	if peerState == nil {
		return NewSyncErrorInternal(fmt.Errorf("peer state for %q not found", h.peer.Keyid))
	}
	peerState.Config = item.GetConfig()
	h.mgr.peerStateManager.SetPeerState(h.peer.Keyid, peerState)
	return nil
}

func (h *syncSessionHandlerServer) HandleReceiveResources(ctx context.Context, stream *bidiSyncCommandStream, item *v1sync.SyncStreamItem_SyncActionReceiveResources) error {
	h.l.Debug("received resource list from client",
		zap.Any("repos", item.GetRepos()),
		zap.Any("plans", item.GetPlans()))
	peerState := h.mgr.peerStateManager.GetPeerState(h.peer.Keyid).Clone()
	if peerState == nil {
		return NewSyncErrorInternal(fmt.Errorf("peer state for %q not found", h.peer.Keyid))
	}
	repos := item.GetRepos()
	plans := item.GetPlans()
	for _, repo := range repos {
		peerState.KnownRepos[repo.Id] = repo
	}
	for _, plan := range plans {
		peerState.KnownPlans[plan.Id] = plan
	}
	h.mgr.peerStateManager.SetPeerState(h.peer.Keyid, peerState)
	return nil
}

func (h *syncSessionHandlerServer) insertOrUpdate(op *v1.Operation, isUpdate bool) error {
	// Returns a localOpID and localFlowID or 0 if not found in which case a new ID will be assigned by the insert.
	localOpID, localFlowID, err := h.mapper.TranslateOpIdAndFlowID(h.peer.Keyid, op.Id, op.FlowId)
	if err != nil {
		return fmt.Errorf("translating operation ID and flow ID: %w", err)
	}
	op.OriginalInstanceKeyid = h.peer.Keyid
	op.OriginalId = op.Id
	op.OriginalFlowId = op.FlowId
	op.Id = localOpID
	op.FlowId = localFlowID
	if op.Id == 0 {
		if isUpdate {
			h.l.Sugar().Warnf("received update for non-existent operation %+v, inserting instead", op)
		}
		op.Modno = 0
		return h.mgr.oplog.Add(op)
	} else {
		if !isUpdate {
			h.l.Sugar().Warnf("received insert for existing operation %+v, updating instead", op)
		}
		return h.mgr.oplog.Update(op)
	}
}

func (h *syncSessionHandlerServer) deleteByOriginalID(originalID int64) error {
	foundOp, err := h.mgr.oplog.FindOneMetadata(oplog.Query{}.
		SetOriginalInstanceKeyid(h.peer.Keyid).
		SetOriginalID(originalID))
	if err != nil {
		return fmt.Errorf("finding operation metadata: %w", err)
	}
	if foundOp.ID == 0 {
		h.l.Sugar().Debugf("received delete for non-existent operation %v", originalID)
		return nil
	}
	return h.mgr.oplog.Delete(foundOp.ID)
}

func (h *syncSessionHandlerServer) sendConfigToClient(stream *bidiSyncCommandStream, config *v1.Config) error {
	remoteConfig := &v1sync.RemoteConfig{
		Version: config.Version,
		Modno:   config.Modno,
	}
	resourceListMsg := &v1sync.SyncStreamItem_SyncActionReceiveResources{}
	var allowedRepoIDs []string
	var allowedPlanIDs []string
	for _, repo := range config.Repos {
		if h.permissions.CheckPermissionForRepo(repo.Id, permissions.PermsCanViewConfiguration...) {
			remoteConfig.Repos = append(remoteConfig.Repos, repo)
			resourceListMsg.Repos = append(resourceListMsg.Repos, &v1sync.RepoMetadata{
				Id:   repo.Id,
				Guid: repo.Guid,
			})
			allowedRepoIDs = append(allowedRepoIDs, repo.Id)
		}
	}
	for _, plan := range config.Plans {
		if h.permissions.CheckPermissionForPlan(plan.Id, permissions.PermsCanViewConfiguration...) {
			remoteConfig.Plans = append(remoteConfig.Plans, plan)
			resourceListMsg.Plans = append(resourceListMsg.Plans, &v1sync.PlanMetadata{
				Id: plan.Id,
			})
			allowedPlanIDs = append(allowedPlanIDs, plan.Id)
		}
	}
	h.l.Sugar().Debugf("determined client %v is allowlisted to read configs for repos %v and plans %v", h.peer.InstanceId, allowedRepoIDs, allowedPlanIDs)

	// Send the config, this is the first meaningful packet the client will receive.
	stream.Send(&v1sync.SyncStreamItem{
		Action: &v1sync.SyncStreamItem_ReceiveConfig{
			ReceiveConfig: &v1sync.SyncStreamItem_SyncActionReceiveConfig{
				Config: remoteConfig,
			},
		},
	})

	// Send the updated list of resources that the client can access.
	stream.Send(&v1sync.SyncStreamItem{
		Action: &v1sync.SyncStreamItem_ReceiveResources{
			ReceiveResources: resourceListMsg,
		},
	})
	return nil
}

// sendSharedReposToClient sends repos marked as shared to the client via SetConfig.
// This pushes repo configurations to the client so they are added to the client's local config.
func (h *syncSessionHandlerServer) sendSharedReposToClient(stream *bidiSyncCommandStream, config *v1.Config) {
	var sharedRepos []*v1.Repo
	for _, repo := range config.Repos {
		if repo.GetShared() && h.permissions.CheckPermissionForRepo(repo.Id, permissions.PermsCanViewConfiguration...) {
			repoCopy := proto.Clone(repo).(*v1.Repo)
			repoCopy.OriginInstanceId = config.Instance
			sharedRepos = append(sharedRepos, repoCopy)
		}
	}

	if len(sharedRepos) == 0 {
		return
	}

	h.l.Sugar().Debugf("sending %d shared repos to client %q", len(sharedRepos), h.peer.InstanceId)
	stream.Send(&v1sync.SyncStreamItem{
		Action: &v1sync.SyncStreamItem_SetConfig{
			SetConfig: &v1sync.SyncStreamItem_SyncActionSetConfig{
				Repos: sharedRepos,
			},
		},
	})
}

// ValidatePairingSecret checks a pairing secret against a list of pairing tokens.
// Returns the matching token if valid, or an error explaining why validation failed.
// This is a pure function with no side effects, making it easy to test exhaustively.
func ValidatePairingSecret(secret string, tokens []*v1.Multihost_PairingToken, now time.Time) (*v1.Multihost_PairingToken, error) {
	if secret == "" {
		return nil, fmt.Errorf("empty pairing secret")
	}
	for _, token := range tokens {
		if token.Secret != secret {
			continue
		}
		if token.ExpiresAtUnix > 0 && now.Unix() > token.ExpiresAtUnix {
			return nil, fmt.Errorf("pairing token %q has expired", token.Label)
		}
		if token.MaxUses > 0 && token.Uses >= token.MaxUses {
			return nil, fmt.Errorf("pairing token %q has reached its maximum number of uses (%d)", token.Label, token.MaxUses)
		}
		return token, nil
	}
	return nil, fmt.Errorf("no matching pairing token found")
}

// handleUnknownPeerPairing returns an onUnknownPeerFunc that validates a pairing secret
// from the handshake, adds the client to authorized_clients in the config, and consumes the token.
// The peer is added to the config BEFORE runSync proceeds with its normal authorization check,
// ensuring that runSync's hard gate (peer must be in authorized_clients) is never bypassed.
func (h *BackrestSyncHandler) handleUnknownPeerPairing(snapshot *syncConfigSnapshot) onUnknownPeerFunc {
	return func(handshake *v1sync.SyncStreamItem) (*v1.Multihost_Peer, error) {
		pairingSecret := handshake.GetHandshake().GetPairingSecret()
		if pairingSecret == "" {
			return nil, fmt.Errorf("unknown peer and no pairing secret provided")
		}

		// Defense-in-depth: re-verify the handshake signature to ensure the client
		// holds the private key for the public key it presents. This is already checked
		// by verifyHandshakePacket in runSync, but we verify again here since this is
		// a security-critical path that adds a new authorized client.
		if _, err := verifyHandshakePacket(handshake); err != nil {
			return nil, fmt.Errorf("handshake signature verification failed: %w", err)
		}

		peerKeyID := handshake.GetHandshake().GetPublicKey().GetKeyid()
		peerInstanceID := string(handshake.GetHandshake().GetInstanceId().GetPayload())

		// Fetch the latest config for token validation (not the snapshot, which may be stale).
		cfg, err := h.mgr.configMgr.Get()
		if err != nil {
			return nil, fmt.Errorf("failed to get config: %w", err)
		}

		// Validate the pairing secret against stored tokens.
		token, err := ValidatePairingSecret(pairingSecret, cfg.GetMultihost().GetPairingTokens(), time.Now())
		if err != nil {
			zap.S().Warnf("rejected pairing attempt from %q (%s): %v", peerInstanceID, peerKeyID, err)
			return nil, err
		}

		// Token is valid — add the client to authorized_clients and consume the token.
		newPeer := &v1.Multihost_Peer{
			InstanceId:  peerInstanceID,
			Keyid:       peerKeyID,
			Permissions: token.Permissions,
		}
		cfg.Multihost.AuthorizedClients = append(cfg.Multihost.AuthorizedClients, newPeer)

		// Consume the token: increment uses, remove if exhausted.
		token.Uses++
		if token.MaxUses > 0 && token.Uses >= token.MaxUses {
			cfg.Multihost.PairingTokens = slices.DeleteFunc(cfg.Multihost.PairingTokens, func(t *v1.Multihost_PairingToken) bool {
				return t.Secret == token.Secret
			})
		}

		cfg.Modno++
		if err := h.mgr.configMgr.Update(cfg); err != nil {
			return nil, fmt.Errorf("failed to save paired client: %w", err)
		}

		zap.S().Infof("successfully paired client %q (%s) via pairing token %q", peerInstanceID, peerKeyID, token.Label)
		return newPeer, nil
	}
}

func (h *syncSessionHandlerServer) sendOperationSyncRequest(stream *bidiSyncCommandStream) error {
	highestID, highestModno, err := h.mgr.oplog.GetHighestOpIDAndModno(oplog.Query{}.SetOriginalInstanceKeyid(h.peer.Keyid))
	if err != nil {
		return fmt.Errorf("getting highest opid and modno: %w", err)
	}
	stream.Send(&v1sync.SyncStreamItem{
		Action: &v1sync.SyncStreamItem_RequestOperations{
			RequestOperations: &v1sync.SyncStreamItem_SyncActionRequestOperations{
				HighOpid:  highestID,
				HighModno: highestModno,
			},
		},
	})
	h.l.Sugar().Debugf("requested operations from client starting at opID %d and modno %d", highestID, highestModno)
	return nil
}
