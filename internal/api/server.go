package api

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/garethgeorge/resticui/gen/go/types"
	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	"github.com/garethgeorge/resticui/internal/config"
	"github.com/garethgeorge/resticui/internal/oplog"
	"github.com/garethgeorge/resticui/internal/oplog/indexutil"
	"github.com/garethgeorge/resticui/internal/orchestrator"
	"github.com/garethgeorge/resticui/internal/protoutil"
	"github.com/garethgeorge/resticui/internal/resticinstaller"
	"github.com/garethgeorge/resticui/pkg/restic"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	*v1.UnimplementedResticUIServer
	config       config.ConfigStore
	orchestrator *orchestrator.Orchestrator
	oplog        *oplog.OpLog
}

var _ v1.ResticUIServer = &Server{}

func NewServer(config config.ConfigStore, orchestrator *orchestrator.Orchestrator, oplog *oplog.OpLog) *Server {
	s := &Server{
		config:       config,
		orchestrator: orchestrator,
		oplog:        oplog,
	}

	return s
}

// GetConfig implements GET /v1/config
func (s *Server) GetConfig(ctx context.Context, empty *emptypb.Empty) (*v1.Config, error) {
	return s.config.Get()
}

// SetConfig implements POST /v1/config
func (s *Server) SetConfig(ctx context.Context, c *v1.Config) (*v1.Config, error) {
	existing, err := s.config.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to check current config: %w", err)
	}

	// Compare and increment modno
	if existing.Modno != c.Modno {
		return nil, errors.New("config modno mismatch, reload and try again")
	}
	c.Modno += 1

	if err := s.config.Update(c); err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	newConfig, err := s.config.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get newly set config: %w", err)
	}
	s.orchestrator.ApplyConfig(newConfig)
	return newConfig, nil
}

// AddRepo implements POST /v1/config/repo, it includes validation that the repo can be initialized.
func (s *Server) AddRepo(ctx context.Context, repo *v1.Repo) (*v1.Config, error) {
	c, err := s.config.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	c = proto.Clone(c).(*v1.Config)
	c.Repos = append(c.Repos, repo)

	if err := config.ValidateConfig(c); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	bin, err := resticinstaller.FindOrInstallResticBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to find or install restic binary: %w", err)
	}

	r := restic.NewRepo(bin, repo)
	// use background context such that the init op can try to complete even if the connection is closed.
	if err := r.Init(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to init repo: %w", err)
	}

	zap.L().Debug("Updating config")
	if err := s.config.Update(c); err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	s.orchestrator.ApplyConfig(c)

	return c, nil
}

// ListSnapshots implements POST /v1/snapshots
func (s *Server) ListSnapshots(ctx context.Context, query *v1.ListSnapshotsRequest) (*v1.ResticSnapshotList, error) {
	repo, err := s.orchestrator.GetRepo(query.RepoId)
	if err != nil {
		return nil, fmt.Errorf("failed to get repo: %w", err)
	}

	var snapshots []*restic.Snapshot
	if query.PlanId != "" {
		var plan *v1.Plan
		plan, err = s.orchestrator.GetPlan(query.PlanId)
		if err != nil {
			return nil, fmt.Errorf("failed to get plan %q: %w", query.PlanId, err)
		}
		snapshots, err = repo.SnapshotsForPlan(ctx, plan)
	} else {
		snapshots, err = repo.Snapshots(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	// Transform the snapshots and return them.
	var rs []*v1.ResticSnapshot
	for _, snapshot := range snapshots {
		rs = append(rs, protoutil.SnapshotToProto(snapshot))
	}

	return &v1.ResticSnapshotList{
		Snapshots: rs,
	}, nil
}

func (s *Server) ListSnapshotFiles(ctx context.Context, query *v1.ListSnapshotFilesRequest) (*v1.ListSnapshotFilesResponse, error) {
	repo, err := s.orchestrator.GetRepo(query.RepoId)
	if err != nil {
		return nil, fmt.Errorf("failed to get repo: %w", err)
	}

	entries, err := repo.ListSnapshotFiles(ctx, query.SnapshotId, query.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshot files: %w", err)
	}

	return &v1.ListSnapshotFilesResponse{
		Path:    query.Path,
		Entries: entries,
	}, nil
}

// GetOperationEvents implements GET /v1/events/operations
func (s *Server) GetOperationEvents(_ *emptypb.Empty, stream v1.ResticUI_GetOperationEventsServer) error {
	errorChan := make(chan error)
	defer close(errorChan)
	callback := func(eventType oplog.EventType, op *v1.Operation) {
		var eventTypeMapped v1.OperationEventType
		switch eventType {
		case oplog.EventTypeOpCreated:
			eventTypeMapped = v1.OperationEventType_EVENT_CREATED
		case oplog.EventTypeOpUpdated:
			eventTypeMapped = v1.OperationEventType_EVENT_UPDATED
		default:
			zap.L().Error("Unknown event type", zap.Int("eventType", int(eventType)))
			eventTypeMapped = v1.OperationEventType_EVENT_UNKNOWN
		}

		event := &v1.OperationEvent{
			Type:      eventTypeMapped,
			Operation: op,
		}

		go func() {
			if err := stream.Send(event); err != nil {
				errorChan <- fmt.Errorf("failed to send event: %w", err)
			}
		}()
	}
	s.oplog.Subscribe(&callback)
	defer s.oplog.Unsubscribe(&callback)
	select {
	case <-stream.Context().Done():
		return nil
	case err := <-errorChan:
		return err
	}
}

func (s *Server) GetOperations(ctx context.Context, req *v1.GetOperationsRequest) (*v1.OperationList, error) {
	collector := indexutil.CollectAll()

	if req.LastN != 0 {
		collector = indexutil.CollectLastN(int(req.LastN))
	}

	var err error
	var ops []*v1.Operation
	if req.RepoId != "" && req.PlanId != "" {
		return nil, errors.New("cannot specify both repoId and planId")
	} else if req.PlanId != "" {
		ops, err = s.oplog.GetByPlan(req.PlanId, collector)
	} else if req.RepoId != "" {
		ops, err = s.oplog.GetByRepo(req.RepoId, collector)
	} else if req.SnapshotId != "" {
		ops, err = s.oplog.GetBySnapshotId(req.SnapshotId, collector)
	} else if len(req.Ids) > 0 {
		ops = make([]*v1.Operation, 0, len(req.Ids))
		for i, id := range req.Ids {
			op, err := s.oplog.Get(id)
			if err != nil {
				return nil, fmt.Errorf("failed to get operation %d: %w", i, err)
			}
			ops = append(ops, op)
		}
	} else {
		ops, err = s.oplog.GetAll()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get operations: %w", err)
	}

	return &v1.OperationList{
		Operations: ops,
	}, nil
}

func (s *Server) Backup(ctx context.Context, req *types.StringValue) (*emptypb.Empty, error) {
	plan, err := s.orchestrator.GetPlan(req.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan %q: %w", req.Value, err)
	}
	s.orchestrator.ScheduleTask(orchestrator.NewOneofBackupTask(s.orchestrator, plan, time.Now()))
	return &emptypb.Empty{}, nil
}

func (s *Server) PathAutocomplete(ctx context.Context, path *types.StringValue) (*types.StringList, error) {
	ents, err := os.ReadDir(path.Value)
	if errors.Is(err, os.ErrNotExist) {
		return &types.StringList{}, nil
	} else if err != nil {
		return nil, err
	}

	var paths []string
	for _, ent := range ents {
		paths = append(paths, ent.Name())
	}

	return &types.StringList{Values: paths}, nil
}
