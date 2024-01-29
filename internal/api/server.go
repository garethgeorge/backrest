package api

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/garethgeorge/backrest/gen/go/types"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1/v1connect"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/oplog/indexutil"
	"github.com/garethgeorge/backrest/internal/orchestrator"
	"github.com/garethgeorge/backrest/internal/protoutil"
	"github.com/garethgeorge/backrest/internal/resticinstaller"
	"github.com/garethgeorge/backrest/pkg/restic"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	v1connect.UnimplementedBackrestHandler
	config       config.ConfigStore
	orchestrator *orchestrator.Orchestrator
	oplog        *oplog.OpLog
}

var _ v1connect.BackrestHandler = &Server{}

func NewServer(config config.ConfigStore, orchestrator *orchestrator.Orchestrator, oplog *oplog.OpLog) *Server {
	s := &Server{
		config:       config,
		orchestrator: orchestrator,
		oplog:        oplog,
	}

	return s
}

// GetConfig implements GET /v1/config
func (s *Server) GetConfig(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[v1.Config], error) {
	config, err := s.config.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	return connect.NewResponse(config), nil
}

// SetConfig implements POST /v1/config
func (s *Server) SetConfig(ctx context.Context, req *connect.Request[v1.Config]) (*connect.Response[v1.Config], error) {
	existing, err := s.config.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to check current config: %w", err)
	}

	// Compare and increment modno
	if existing.Modno != req.Msg.Modno {
		return nil, errors.New("config modno mismatch, reload and try again")
	}
	req.Msg.Modno += 1

	if err := s.config.Update(req.Msg); err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	newConfig, err := s.config.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get newly set config: %w", err)
	}
	if err := s.orchestrator.ApplyConfig(newConfig); err != nil {
		return nil, fmt.Errorf("failed to apply config: %w", err)
	}
	return connect.NewResponse(newConfig), nil
}

// AddRepo implements POST /v1/config/repo, it includes validation that the repo can be initialized.
func (s *Server) AddRepo(ctx context.Context, req *connect.Request[v1.Repo]) (*connect.Response[v1.Config], error) {
	c, err := s.config.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	c = proto.Clone(c).(*v1.Config)
	c.Repos = append(c.Repos, req.Msg)

	if err := config.ValidateConfig(c); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	bin, err := resticinstaller.FindOrInstallResticBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to find or install restic binary: %w", err)
	}

	r := restic.NewRepo(bin, req.Msg)

	// use background context such that the init op can try to complete even if the connection is closed.
	if err := r.Init(context.Background(), restic.WithPropagatedEnvVars(restic.EnvToPropagate...)); err != nil {
		return nil, fmt.Errorf("failed to init repo: %w", err)
	}

	zap.L().Debug("Updating config")
	if err := s.config.Update(c); err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	zap.L().Debug("Applying config")
	s.orchestrator.ApplyConfig(c)

	// index snapshots for the newly added repository.
	zap.L().Debug("Scheduling index snapshots task")
	s.orchestrator.ScheduleTask(orchestrator.NewOneoffIndexSnapshotsTask(s.orchestrator, req.Msg.Id, time.Now()), orchestrator.TaskPriorityInteractive+orchestrator.TaskPriorityIndexSnapshots)

	zap.L().Debug("Done add repo")
	return connect.NewResponse(c), nil
}

// ListSnapshots implements POST /v1/snapshots
func (s *Server) ListSnapshots(ctx context.Context, req *connect.Request[v1.ListSnapshotsRequest]) (*connect.Response[v1.ResticSnapshotList], error) {
	query := req.Msg
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

	return connect.NewResponse(&v1.ResticSnapshotList{
		Snapshots: rs,
	}), nil
}

func (s *Server) ListSnapshotFiles(ctx context.Context, req *connect.Request[v1.ListSnapshotFilesRequest]) (*connect.Response[v1.ListSnapshotFilesResponse], error) {
	query := req.Msg
	repo, err := s.orchestrator.GetRepo(query.RepoId)
	if err != nil {
		return nil, fmt.Errorf("failed to get repo: %w", err)
	}

	entries, err := repo.ListSnapshotFiles(ctx, query.SnapshotId, query.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshot files: %w", err)
	}

	return connect.NewResponse(&v1.ListSnapshotFilesResponse{
		Path:    query.Path,
		Entries: entries,
	}), nil
}

// GetOperationEvents implements GET /v1/events/operations
func (s *Server) GetOperationEvents(ctx context.Context, req *connect.Request[emptypb.Empty], resp *connect.ServerStream[v1.OperationEvent]) error {
	errorChan := make(chan error)
	defer close(errorChan)
	callback := func(oldOp *v1.Operation, newOp *v1.Operation) {
		var event *v1.OperationEvent
		if oldOp == nil && newOp != nil {
			event = &v1.OperationEvent{
				Type:      v1.OperationEventType_EVENT_CREATED,
				Operation: newOp,
			}
		} else if oldOp != nil && newOp != nil {
			event = &v1.OperationEvent{
				Type:      v1.OperationEventType_EVENT_UPDATED,
				Operation: newOp,
			}
		} else if oldOp != nil && newOp == nil {
			event = &v1.OperationEvent{
				Type:      v1.OperationEventType_EVENT_DELETED,
				Operation: oldOp,
			}
		} else {
			zap.L().Error("Unknown event type")
			return
		}

		if err := resp.Send(event); err != nil {
			errorChan <- fmt.Errorf("failed to send event: %w", err)
		}
	}
	s.oplog.Subscribe(&callback)
	defer s.oplog.Unsubscribe(&callback)
	select {
	case <-ctx.Done():
		return nil
	case err := <-errorChan:
		return err
	}
}

func (s *Server) GetOperations(ctx context.Context, req *connect.Request[v1.GetOperationsRequest]) (*connect.Response[v1.OperationList], error) {
	idCollector := indexutil.CollectAll()

	if req.Msg.LastN != 0 {
		idCollector = indexutil.CollectLastN(int(req.Msg.LastN))
	}

	var err error
	var ops []*v1.Operation
	opCollector := func(op *v1.Operation) error {
		ops = append(ops, op)
		return nil
	}
	if req.Msg.RepoId != "" && req.Msg.PlanId != "" {
		return nil, errors.New("cannot specify both repoId and planId")
	} else if req.Msg.PlanId != "" {
		err = s.oplog.ForEachByPlan(req.Msg.PlanId, idCollector, opCollector)
	} else if req.Msg.RepoId != "" {
		err = s.oplog.ForEachByRepo(req.Msg.RepoId, idCollector, opCollector)
	} else if req.Msg.SnapshotId != "" {
		err = s.oplog.ForEachBySnapshotId(req.Msg.SnapshotId, idCollector, opCollector)
	} else if len(req.Msg.Ids) > 0 {
		ops = make([]*v1.Operation, 0, len(req.Msg.Ids))
		for i, id := range req.Msg.Ids {
			op, err := s.oplog.Get(id)
			if err != nil {
				return nil, fmt.Errorf("failed to get operation %d: %w", i, err)
			}
			ops = append(ops, op)
		}
	} else {
		err = s.oplog.ForAll(opCollector)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get operations: %w", err)
	}

	return connect.NewResponse(&v1.OperationList{
		Operations: ops,
	}), nil
}

func (s *Server) IndexSnapshots(ctx context.Context, req *connect.Request[types.StringValue]) (*connect.Response[emptypb.Empty], error) {
	_, err := s.orchestrator.GetRepo(req.Msg.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to get repo %q: %w", req.Msg.Value, err)
	}

	s.orchestrator.ScheduleTask(orchestrator.NewOneoffIndexSnapshotsTask(s.orchestrator, req.Msg.Value, time.Now()), orchestrator.TaskPriorityInteractive+orchestrator.TaskPriorityIndexSnapshots)

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *Server) Backup(ctx context.Context, req *connect.Request[types.StringValue]) (*connect.Response[emptypb.Empty], error) {
	plan, err := s.orchestrator.GetPlan(req.Msg.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan %q: %w", req.Msg.Value, err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	s.orchestrator.ScheduleTask(orchestrator.NewOneoffBackupTask(s.orchestrator, plan, time.Now()), orchestrator.TaskPriorityInteractive, func(e error) {
		err = e
		wg.Done()
	})
	wg.Wait()
	return connect.NewResponse(&emptypb.Empty{}), err
}

func (s *Server) Forget(ctx context.Context, req *connect.Request[types.StringValue]) (*connect.Response[emptypb.Empty], error) {
	plan, err := s.orchestrator.GetPlan(req.Msg.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan %q: %w", req.Msg.Value, err)
	}

	at := time.Now()
	var wg sync.WaitGroup
	wg.Add(1)
	s.orchestrator.ScheduleTask(orchestrator.NewOneoffForgetTask(s.orchestrator, plan, "", at), orchestrator.TaskPriorityInteractive+orchestrator.TaskPriorityForget, func(e error) {
		err = e
		wg.Done()
	})
	s.orchestrator.ScheduleTask(orchestrator.NewOneoffIndexSnapshotsTask(s.orchestrator, plan.Repo, at), orchestrator.TaskPriorityInteractive+orchestrator.TaskPriorityIndexSnapshots)
	wg.Wait()
	return connect.NewResponse(&emptypb.Empty{}), err
}

func (s *Server) Prune(ctx context.Context, req *connect.Request[types.StringValue]) (*connect.Response[emptypb.Empty], error) {
	plan, err := s.orchestrator.GetPlan(req.Msg.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan %q: %w", req.Msg.Value, err)
	}

	at := time.Now()
	var wg sync.WaitGroup
	wg.Add(1)
	s.orchestrator.ScheduleTask(orchestrator.NewOneoffPruneTask(s.orchestrator, plan, "", at, true), orchestrator.TaskPriorityInteractive+orchestrator.TaskPriorityPrune, func(e error) {
		err = e
		wg.Done()
	})
	wg.Wait()

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *Server) Restore(ctx context.Context, req *connect.Request[v1.RestoreSnapshotRequest]) (*connect.Response[emptypb.Empty], error) {
	plan, err := s.orchestrator.GetPlan(req.Msg.PlanId)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan %q: %w", req.Msg.PlanId, err)
	}

	if req.Msg.Target == "" {
		req.Msg.Target = path.Join(os.Getenv("HOME"), "Downloads")
	}

	target := path.Join(req.Msg.Target, fmt.Sprintf("restic-restore-%v", time.Now().Format("2006-01-02T15-04-05")))
	_, err = os.Stat(target)
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("restore target dir %q already exists", req.Msg.Target)
	}

	at := time.Now()

	s.orchestrator.ScheduleTask(orchestrator.NewOneoffRestoreTask(s.orchestrator, orchestrator.RestoreTaskOpts{
		Plan:       plan,
		SnapshotId: req.Msg.SnapshotId,
		Path:       req.Msg.Path,
		Target:     target,
	}, at), orchestrator.TaskPriorityInteractive+orchestrator.TaskPriorityDefault)

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *Server) Unlock(ctx context.Context, req *connect.Request[types.StringValue]) (*connect.Response[emptypb.Empty], error) {
	repo, err := s.orchestrator.GetRepo(req.Msg.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to get repo %q: %w", req.Msg.Value, err)
	}

	if err := repo.Unlock(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to unlock repo %q: %w", req.Msg.Value, err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *Server) Cancel(ctx context.Context, req *connect.Request[types.Int64Value]) (*connect.Response[emptypb.Empty], error) {
	if err := s.orchestrator.CancelOperation(req.Msg.Value, v1.OperationStatus_STATUS_USER_CANCELLED); err != nil {
		return nil, err
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *Server) ClearHistory(ctx context.Context, req *connect.Request[v1.ClearHistoryRequest]) (*connect.Response[emptypb.Empty], error) {
	var err error
	var ids []int64
	opCollector := func(op *v1.Operation) error {
		if !req.Msg.OnlyFailed || op.Status == v1.OperationStatus_STATUS_ERROR {
			ids = append(ids, op.Id)
		}
		return nil
	}

	if req.Msg.RepoId != "" && req.Msg.PlanId != "" {
		return nil, errors.New("cannot specify both repoId and planId")
	} else if req.Msg.PlanId != "" {
		err = s.oplog.ForEachByPlan(req.Msg.PlanId, indexutil.CollectAll(), opCollector)
	} else if req.Msg.RepoId != "" {
		err = s.oplog.ForEachByRepo(req.Msg.RepoId, indexutil.CollectAll(), opCollector)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get operations to delete: %w", err)
	}

	if err := s.oplog.Delete(ids...); err != nil {
		return nil, fmt.Errorf("failed to delete operations: %w", err)
	}

	return connect.NewResponse(&emptypb.Empty{}), err
}

func (s *Server) GetOperationData(ctx context.Context, req *connect.Request[v1.OperationDataRequest]) (*connect.Response[types.BytesValue], error) {
	data, err := s.oplog.GetBigData(req.Msg.Id, req.Msg.Key)
	if err != nil {
		return nil, fmt.Errorf("get operation data: %w", err)
	}
	return connect.NewResponse(&types.BytesValue{Value: data}), nil
}

func (s *Server) PathAutocomplete(ctx context.Context, path *connect.Request[types.StringValue]) (*connect.Response[types.StringList], error) {
	ents, err := os.ReadDir(path.Msg.Value)
	if errors.Is(err, os.ErrNotExist) {
		return connect.NewResponse(&types.StringList{}), nil
	} else if err != nil {
		return nil, err
	}

	var paths []string
	for _, ent := range ents {
		paths = append(paths, ent.Name())
	}

	return connect.NewResponse(&types.StringList{Values: paths}), nil
}
