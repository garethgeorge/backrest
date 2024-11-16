package syncengine

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"connectrpc.com/connect"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1/v1connect"
	"github.com/garethgeorge/backrest/internal/api"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/oplog"
)

const SyncProtocolVersion = 1

type BackrestSyncHandler struct {
	config config.ConfigStore
	oplog  *oplog.OpLog
}

var _ v1connect.BackrestSyncServiceHandler = &BackrestSyncHandler{}

func NewBackrestSyncHandler(config config.ConfigStore, oplog *oplog.OpLog) *BackrestSyncHandler {
	return &BackrestSyncHandler{
		config: config,
		oplog:  oplog,
	}
}

func (h *BackrestSyncHandler) Sync(ctx context.Context, stream *connect.BidiStream[v1.SyncStreamItem, v1.SyncStreamItem]) error {
	// TODO: this request can be very long lived, we must periodically refresh the config
	initialConfig, err := h.config.Get()
	if err != nil {
		return err
	}

	receive := make(chan *v1.SyncStreamItem, 1)
	send := make(chan *v1.SyncStreamItem, 1)
	go func() {
		for {
			item, err := stream.Receive()
			if err != nil {
				break
			}
			receive <- item
		}
		close(receive)
	}()

	// Broadcast initial packet containing the protocol version and instance ID.
	if err := stream.Send(&v1.SyncStreamItem{
		Action: &v1.SyncStreamItem_Handshake{
			Handshake: &v1.SyncStreamItem_SyncActionHandshake{
				ProtocolVersion: SyncProtocolVersion,
				InstanceId:      initialConfig.Instance,
			},
		},
	}); err != nil {
		return err
	}

	// Try to read the handshake packet from the client.
	clientInstanceID := ""
	if msg, ok := <-receive; ok {
		handshake := msg.GetHandshake()
		if handshake == nil {
			return connect.NewError(connect.CodeInvalidArgument, errors.New("handshake packet must be sent first"))
		}

		clientInstanceID = handshake.GetInstanceId()
		if clientInstanceID == "" {
			return connect.NewError(connect.CodeInvalidArgument, errors.New("instance ID is required"))
		}
	} else {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("no packets received"))
	}

	// After receiving handshake packet, start processing commands
	connectedRepos := make(map[string]struct{})

	insertOrUpdate := func(op *v1.Operation) error {
		var foundOp *v1.Operation
		if err := h.oplog.Query(oplog.Query{OriginalID: op.Id, InstanceID: op.InstanceId}, func(o *v1.Operation) error {
			foundOp = o
			return nil
		}); err != nil {
			return fmt.Errorf("mapping remote ID to local ID: %w", err)
		}

		if foundOp == nil {
			op.OriginalId = op.Id
			op.Id = 0
			return h.oplog.Add(op)
		} else {
			op.OriginalId = op.Id
			op.Id = foundOp.Id
			return h.oplog.Update(op)
		}
	}

	handleSyncCommand := func(item *v1.SyncStreamItem) error {
		switch action := item.Action.(type) {
		case *v1.SyncStreamItem_ConnectRepo:
			// TODO: enforce authentication here
			// Auth should check credentials and also the instance ID of the client.
			if _, ok := connectedRepos[action.ConnectRepo.RepoId]; ok {
				return connect.NewError(connect.CodeAlreadyExists, errors.New("client is already connected to repo"))
			}

			connectedRepos[action.ConnectRepo.RepoId] = struct{}{}

			if err := stream.Send(&v1.SyncStreamItem{
				Action: &v1.SyncStreamItem_UpdateConnectionState{
					UpdateConnectionState: &v1.SyncStreamItem_SyncActionUpdateConnectionState{
						RepoId: action.ConnectRepo.RepoId,
						State:  v1.SyncStreamItem_CONNECTION_STATE_CONNECTED,
					},
				},
			}); err != nil {
				return fmt.Errorf("action ConnectRepo: send connection state reply: %w", err)
			}
		case *v1.SyncStreamItem_DiffOperations:
			diffSel := action.DiffOperations.GetHaveOperationsSelector()

			if diffSel == nil {
				return connect.NewError(connect.CodeInvalidArgument, errors.New("action DiffOperations: selector is required"))
			}
			// The diff selector _must_ be scoped to the instance ID of the client.
			if diffSel.GetInstanceId() != clientInstanceID {
				return connect.NewError(connect.CodePermissionDenied, errors.New("action DiffOperations: instance ID mismatch in diff selector"))
			}
			// The diff selector _must_ specify a repo we are subscribed to.
			if _, ok := connectedRepos[diffSel.GetRepoId()]; !ok {
				return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("action DiffOperations: client is not subscribed to repo %s", diffSel.GetRepoId()))
			}
			// These are required to be the same length for a pairwise zip.
			if len(action.DiffOperations.HaveOperationIds) != len(action.DiffOperations.HaveOperationModnos) {
				return connect.NewError(connect.CodeInvalidArgument, errors.New("action DiffOperations: operation IDs and modnos must be the same length"))
			}

			diffSelQuery, err := api.OpSelectorToQuery(diffSel)
			if err != nil {
				return fmt.Errorf("action DiffOperations: converting diff selector to query: %w", err)
			}

			localMetadata := []oplog.OpMetadata{}
			if err := h.oplog.QueryMetadata(diffSelQuery, func(metadata oplog.OpMetadata) error {
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

			requestIDs := []int64{}

			// This is a simple O(n) diff algorithm that compares the local and remote metadata vectors.
			localIndex := 0
			remoteIndex := 0
			for localIndex < len(localMetadata) && remoteIndex < len(remoteMetadata) {
				local := localMetadata[localIndex]
				remote := remoteMetadata[remoteIndex]

				if local.OriginalID == remote.ID {
					if local.Modno != remote.Modno {
						requestIDs = append(requestIDs, local.ID)
					}
					localIndex++
					remoteIndex++
				} else if local.OriginalID < remote.ID {
					// the ID is found locally not remotely, request it and see if we get a delete event back
					// from the client indicating that the operation was deleted.
					requestIDs = append(requestIDs, local.OriginalID)
					localIndex++
				} else {
					// the ID is found remotely not locally, request it for initial sync.
					requestIDs = append(requestIDs, remote.ID)
					remoteIndex++
				}
			}
			for localIndex < len(localMetadata) {
				requestIDs = append(requestIDs, localMetadata[localIndex].OriginalID)
				localIndex++
			}
			for remoteIndex < len(remoteMetadata) {
				requestIDs = append(requestIDs, remoteMetadata[remoteIndex].ID)
				remoteIndex++
			}

			if len(requestIDs) > 0 {
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
				for _, op := range event.CreatedOperations.GetOperations() {
					if err := insertOrUpdate(op); err != nil {
						return fmt.Errorf("action SendOperations: operation event create: %w", err)
					}
				}
			case *v1.OperationEvent_UpdatedOperations:
				for _, op := range event.UpdatedOperations.GetOperations() {
					if err := insertOrUpdate(op); err != nil {
						return fmt.Errorf("action SendOperations: operation event update: %w", err)
					}
				}
			case *v1.OperationEvent_DeletedOperations:
				if err := h.oplog.Delete(event.DeletedOperations.GetValues()...); err != nil {
					return fmt.Errorf("action SendOperations: operation event delete: %w", err)
				}
			case *v1.OperationEvent_KeepAlive:
			default:
				return connect.NewError(connect.CodeInvalidArgument, errors.New("action SendOperations: unknown event type"))
			}
		default:
			return connect.NewError(connect.CodeInvalidArgument, errors.New("Unknown action type"))
		}

		return nil
	}

	for {
		select {
		case item, ok := <-receive:
			if !ok {
				return nil
			}

			if err := handleSyncCommand(item); err != nil {
				return err
			}
		case sendItem, ok := <-send:
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
