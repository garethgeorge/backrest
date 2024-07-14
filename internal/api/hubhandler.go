package api

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/garethgeorge/backrest/gen/go/types"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1hub"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/oplog/indexutil"
	"github.com/garethgeorge/backrest/internal/orchestrator"
	"github.com/garethgeorge/backrest/internal/rotatinglog"
)

type HubHandler struct {
	v1hub.UnimplementedHubServer

	config       *config.ConfigManager
	orchestrator *orchestrator.Orchestrator
	oplog        *oplog.OpLog
	logStore     *rotatinglog.RotatingLog
}

func NewHubHandler(config *config.ConfigManager, orchestrator *orchestrator.Orchestrator, oplog *oplog.OpLog, logStore *rotatinglog.RotatingLog) *HubHandler {
	return &HubHandler{}
}

func (s *HubHandler) GetHighestModno(ctx context.Context, req *connect.Request[v1hub.GetHighestModnoRequest]) (*connect.Response[], error) {
	sel := req.Msg.Selector

	query, err := opSelectorToQuery(sel)
	if err != nil {
		return nil, err
	}

	var modno int64
	err := s.oplog.ForEach(query, indexutil.CollectAll(), func(op *v1.Operation) {
		if op.Modno > modno {
			modno = op.Modno
		}
	})
	if err != nil {
		return nil, fmt.Errorf("oplog query: %w", err)
	}

	return connect.NewResponse(types.Int64Value(modno)), nil
}

func (s *HubHandler) SyncOperations(ctx context.Context, req *connect.ClientStream[v1hub.OpSyncMetadata], resp *connect.ServerStream[v1hub.OpSyncMetadata]) error {
	// clients are expected to push operations to the hub
	for req.Receive() {
		opMeta := req.Msg()

		existing, err := s.oplog.Get(opMeta.Id)
		if err != nil && !errors.Is(err, oplog.ErrNotExist) {
			return fmt.Errorf("oplog get: %w", err)
		}

		if opMeta.Operation != nil {
			if err := s.oplog.ApplySyncMetadata(opMeta); err != nil {
				return fmt.Errorf("oplog apply sync metadata: %w", err)
			}
			continue
		}

		if existing == nil {
			// signal that the hub has no version of this operation
			resp.Send(&v1hub.OpSyncMetadata{
				Id:    opMeta.Id,
				Modno: -1,
			})
		} else if opMeta.Modno > existing.Modno {
			// signal that the hub has an older version of this operation
			resp.Send(&v1hub.OpSyncMetadata{
				Id:    opMeta.Id,
				Modno: existing.Modno,
			})
		}
	}

	return nil
}
