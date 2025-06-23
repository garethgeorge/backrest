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
	"github.com/garethgeorge/backrest/internal/config"
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

	initialConfig := snapshot.config
	identityKey := snapshot.identityKey

	receiveError := make(chan error)
	receive := make(chan *v1.SyncStreamItem)
	send := make(chan *v1.SyncStreamItem, 1)
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
	zap.S().Debugf("syncserver a client connected, broadcast handshake as %v", initialConfig.Instance)
	handshakePacket, err := createHandshakePacket(initialConfig.Instance, identityKey)
	if err != nil {
		zap.S().Warnf("syncserver failed to create handshake packet: %v", err)
		return connect.NewError(connect.CodeInternal, errors.New("couldn't build handshake packet, check server logs"))
	}
	if err := stream.Send(handshakePacket); err != nil {
		return err
	}

	// Try to read the handshake packet from the client.
	// TODO: perform this handshake in a header as a pre-flight before opening the stream.
	handshakeMsg, err := tryReceiveWithinDuration(ctx, receive, receiveError, 5*time.Second)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("handshake packet not received: %w", err))
	}
	handshake := handshakeMsg.GetHandshake()
	if _, err := verifyHandshakePacket(handshakeMsg); err != nil {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("verify handshake packet: %w", err))
	}
	clientInstanceID := string(handshake.GetInstanceId().Payload)

	var authorizedClientPeer *v1.Multihost_Peer
	authorizedClientPeerIdx := slices.IndexFunc(initialConfig.Multihost.GetAuthorizedClients(), func(peer *v1.Multihost_Peer) bool {
		return peer.InstanceId == clientInstanceID
	})
	if authorizedClientPeerIdx == -1 {
		// TODO: check the key signature of the handshake message here.
		zap.S().Warnf("syncserver rejected a connection from client instance ID %q because it is not authorized", clientInstanceID)
		return connect.NewError(connect.CodePermissionDenied, errors.New("client is not an authorized peer"))
	} else {
		authorizedClientPeer = initialConfig.Multihost.AuthorizedClients[authorizedClientPeerIdx]
	}

	if !authorizedClientPeer.KeyidVerified {
		return errors.New("authorized keyid must be verified prior to establishing connection")
	} else if err := authorizeHandshakeAsPeer(handshakeMsg, authorizedClientPeer); err != nil {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("rejected authorization as peer %v: %w", authorizedClientPeer.InstanceId, err))
	}

	// TODO: implement key handshake and verification
	// key handshake flow is
	// 1. both ends send their public keys and key ids
	// 2. key ids are checked against values stored in config and against the public key exchanged. E.g. it must match the hash of the key.
	// 3. start communicating.

	zap.S().Infof("syncserver accepted a connection from client instance ID %q", authorizedClientPeer.InstanceId)
	opIDLru, _ := lru.New[int64, int64](4096)   // original ID -> local ID
	flowIDLru, _ := lru.New[int64, int64](1024) // original flow ID -> local flow ID

	insertOrUpdate := func(op *v1.Operation) error {
		op.OriginalInstanceKeyid = authorizedClientPeer.Keyid
		op.OriginalId = op.Id
		op.OriginalFlowId = op.FlowId
		op.Id = 0
		op.FlowId = 0

		var ok bool
		if op.Id, ok = opIDLru.Get(op.OriginalId); !ok {
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
				opIDLru.Add(foundOp.Id, foundOp.Id)
			}
		}
		if op.FlowId, ok = flowIDLru.Get(op.OriginalFlowId); !ok {
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
				flowIDLru.Add(op.OriginalFlowId, flowOp.FlowId)
			}
		}

		return h.mgr.oplog.Set(op)
	}

	deleteByOriginalID := func(originalID int64) error {
		var foundOp *v1.Operation
		if err := h.mgr.oplog.Query(oplog.Query{}.
			SetOriginalInstanceKeyid(authorizedClientPeer.Keyid).
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

	sendConfigToClient := func(config *v1.Config) error {
		remoteConfig := &v1.RemoteConfig{}
		var allowedRepoIDs []string
		for _, repo := range config.Repos {
			if slices.Contains(repo.AllowedPeerInstanceIds, clientInstanceID) {
				allowedRepoIDs = append(allowedRepoIDs, repo.Id)
				remoteConfig.Repos = append(remoteConfig.Repos, protoutil.RepoToRemoteRepo(repo))
			}
		}

		zap.S().Debugf("syncserver determined client %v is allowlisted for repos %v", clientInstanceID, allowedRepoIDs)

		// Send the config, this is the first meaningful packet the client will receive.
		// Once configuration is received, the client will start sending diffs.
		if err := stream.Send(&v1.SyncStreamItem{
			Action: &v1.SyncStreamItem_SendConfig{
				SendConfig: &v1.SyncStreamItem_SyncActionSendConfig{
					Config: remoteConfig,
				},
			},
		}); err != nil {
			return fmt.Errorf("sending config to client: %w", err)
		}
		return nil
	}

	handleSyncCommand := func(item *v1.SyncStreamItem) error {
		switch action := item.Action.(type) {
		case *v1.SyncStreamItem_SendConfig:
			return errors.New("clients can not push configs to server")
		case *v1.SyncStreamItem_DiffOperations:
			diffSel := action.DiffOperations.GetHaveOperationsSelector()
			if diffSel == nil {
				return connect.NewError(connect.CodeInvalidArgument, errors.New("action DiffOperations: selector is required"))
			}

			// The diff selector _must_ be scoped to the instance ID of the client.
			if diffSel.GetInstanceId() != clientInstanceID {
				return connect.NewError(connect.CodePermissionDenied, errors.New("action DiffOperations: instance ID mismatch in diff selector"))
			}

			// The diff selector _must_ specify a repo the client has access to
			repo := config.FindRepoByGUID(initialConfig, diffSel.GetRepoGuid())
			if repo == nil {
				zap.S().Warnf("syncserver action DiffOperations: client %q tried to diff with repo %q that does not exist", clientInstanceID, diffSel.GetRepoGuid())
				return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("action DiffOperations: repo %q not found", diffSel.GetRepoGuid()))
			}
			if !slices.Contains(repo.GetAllowedPeerInstanceIds(), clientInstanceID) {
				zap.S().Warnf("syncserver action DiffOperations: client %q tried to diff with repo %q that they are not allowed to access", clientInstanceID, repo.Id)
				return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("action DiffOperations: client is not allowed to access repo %q", repo.Id))
			}

			// These are required to be the same length for a pairwise zip.
			if len(action.DiffOperations.HaveOperationIds) != len(action.DiffOperations.HaveOperationModnos) {
				return connect.NewError(connect.CodeInvalidArgument, errors.New("action DiffOperations: operation IDs and modnos must be the same length"))
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

			remoteMetadata := make([]oplog.OpMetadata, len(action.DiffOperations.HaveOperationIds))
			for i, id := range action.DiffOperations.HaveOperationIds {
				remoteMetadata[i] = oplog.OpMetadata{
					ID:    id,
					Modno: action.DiffOperations.HaveOperationModnos[i],
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
				zap.String("client_instance_id", clientInstanceID),
				zap.Any("query", diffSelQuery),
				zap.Int("request_due_to_modno", requestDueToModno),
				zap.Int("request_local_but_not_remote", requestMissingLocal),
				zap.Int("request_remote_but_not_local", requestMissingRemote),
				zap.Int("request_ids_total", len(requestIDs)),
			)
			if len(requestIDs) > 0 {
				zap.L().Debug("syncserver sending request operations to client", zap.String("client_instance_id", clientInstanceID), zap.Any("request_ids", requestIDs))
				if err := stream.Send(&v1.SyncStreamItem{
					Action: &v1.SyncStreamItem_DiffOperations{
						DiffOperations: &v1.SyncStreamItem_SyncActionDiffOperations{
							RequestOperations: requestIDs,
						},
					},
				}); err != nil {
					return fmt.Errorf("sending request operations: %w", err)
				}
			}

			return nil
		case *v1.SyncStreamItem_SendOperations:
			switch event := action.SendOperations.GetEvent().Event.(type) {
			case *v1.OperationEvent_CreatedOperations:
				zap.L().Debug("syncserver received created operations", zap.Any("operations", event.CreatedOperations.GetOperations()))
				for _, op := range event.CreatedOperations.GetOperations() {
					if err := insertOrUpdate(op); err != nil {
						return fmt.Errorf("action SendOperations: operation event create %+v: %w", op, err)
					}
				}
			case *v1.OperationEvent_UpdatedOperations:
				zap.L().Debug("syncserver received update operations", zap.Any("operations", event.UpdatedOperations.GetOperations()))
				for _, op := range event.UpdatedOperations.GetOperations() {
					if err := insertOrUpdate(op); err != nil {
						return fmt.Errorf("action SendOperations: operation event update %+v: %w", op, err)
					}
				}
			case *v1.OperationEvent_DeletedOperations:
				zap.L().Debug("syncserver received delete operations", zap.Any("operations", event.DeletedOperations.GetValues()))
				for _, id := range event.DeletedOperations.GetValues() {
					if err := deleteByOriginalID(id); err != nil {
						return fmt.Errorf("action SendOperations: operation event delete %d: %w", id, err)
					}
				}
			case *v1.OperationEvent_KeepAlive:
			default:
				return connect.NewError(connect.CodeInvalidArgument, errors.New("action SendOperations: unknown event type"))
			}
		default:
			return connect.NewError(connect.CodeInvalidArgument, errors.New("unknown action type"))
		}

		return nil
	}

	// subscribe to our own configuration for changes
	configWatchCh := h.mgr.configMgr.Watch()
	defer h.mgr.configMgr.StopWatching(configWatchCh)
	sendConfigToClient(initialConfig)

	for {
		select {
		case err := <-receiveError:
			zap.S().Debugf("syncserver receive error from client %q: %v", authorizedClientPeer.InstanceId, err)
			return err
		case sendItem, ok := <-send: // note: send channel should only be used when sending from a different goroutine than the main loop
			if !ok {
				return nil
			}
			if err := stream.Send(sendItem); err != nil {
				return err
			}
		case item, ok := <-receive:
			if !ok {
				return nil
			}
			if err := handleSyncCommand(item); err != nil {
				return err
			}
		case <-configWatchCh:
			newConfig, err := h.mgr.configMgr.Get()
			if err != nil {
				zap.S().Warnf("syncserver failed to get the newest config: %v", err)
				continue
			}
			sendConfigToClient(newConfig)
		case <-ctx.Done():
			zap.S().Infof("syncserver client %q disconnected", authorizedClientPeer.InstanceId)
			return ctx.Err()
		}
	}
}
