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

func (s *HubHandler) GetHighestModno(ctx context.Context, req *connect.Request[v1hub.GetHighestModnoRequest]) (*connect.Response[types.Int64Value], error) {
	sel := req.Msg.Selector

	query, err := opSelectorToQuery(sel)
	if err != nil {
		return nil, err
	}

	var modno int64
	if err := s.oplog.ForEach(query, indexutil.CollectAll(), func(op *v1.Operation) error {
		if op.Modno > modno {
			modno = op.Modno
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("oplog query: %w", err)
	}

	return connect.NewResponse(&types.Int64Value{
		Value: modno,
	}), nil
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

func (s *HubHandler) GetConfigRequest(ctx context.Context, req *connect.Request[v1hub.GetConfigRequest], resp *connect.ServerStream[v1.Config]) error {
	cfg, err := s.config.Get()
	if err != nil {
		return fmt.Errorf("get config: %w", err)
	}

	transform := func(cfg *v1.Config) *v1.Config {
		return cfg
	}

	// send initial configuration
	if err := resp.Send(transform(cfg)); err != nil {
		return fmt.Errorf("send initial config: %w", err)
	}

	changeNotify := make(chan struct{}, 1)

	onChange := func(*v1.Config) {
		select {
		case changeNotify <- struct{}{}:
		default:
		}
	}
	s.config.Subscribe(onChange)
	defer s.config.Unsubscribe(onChange)

	// send any changes to the configuration as they occur
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-changeNotify:
			cfg, err := s.config.Get() // get the latest config after the change
			if err != nil {
				return fmt.Errorf("get config: %w", err)
			}
			if err := resp.Send(transform(cfg)); err != nil {
				return fmt.Errorf("send config: %w", err)
			}
		}
	}
}
