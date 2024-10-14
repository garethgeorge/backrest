package api

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"reflect"
	"slices"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/garethgeorge/backrest/gen/go/types"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1/v1connect"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/logstore"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/orchestrator"
	"github.com/garethgeorge/backrest/internal/orchestrator/repo"
	"github.com/garethgeorge/backrest/internal/orchestrator/tasks"
	"github.com/garethgeorge/backrest/internal/protoutil"
	"github.com/garethgeorge/backrest/internal/resticinstaller"
	"github.com/garethgeorge/backrest/pkg/restic"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type BackrestHandler struct {
	v1connect.UnimplementedBackrestHandler
	config       config.ConfigStore
	orchestrator *orchestrator.Orchestrator
	oplog        *oplog.OpLog
	logStore     *logstore.LogStore
}

var _ v1connect.BackrestHandler = &BackrestHandler{}

func NewBackrestHandler(config config.ConfigStore, orchestrator *orchestrator.Orchestrator, oplog *oplog.OpLog, logStore *logstore.LogStore) *BackrestHandler {
	s := &BackrestHandler{
		config:       config,
		orchestrator: orchestrator,
		oplog:        oplog,
		logStore:     logStore,
	}

	return s
}

// GetConfig implements GET /v1/config
func (s *BackrestHandler) GetConfig(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[v1.Config], error) {
	config, err := s.config.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	return connect.NewResponse(config), nil
}

// SetConfig implements POST /v1/config
func (s *BackrestHandler) SetConfig(ctx context.Context, req *connect.Request[v1.Config]) (*connect.Response[v1.Config], error) {
	existing, err := s.config.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to check current config: %w", err)
	}

	// Compare and increment modno
	if existing.Modno != req.Msg.Modno {
		return nil, errors.New("config modno mismatch, reload and try again")
	}

	if err := config.ValidateConfig(req.Msg); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
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
func (s *BackrestHandler) AddRepo(ctx context.Context, req *connect.Request[v1.Repo]) (*connect.Response[v1.Config], error) {
	c, err := s.config.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	// Deep copy the configuration
	c = proto.Clone(c).(*v1.Config)

	// Add or implicit update the repo
	if idx := slices.IndexFunc(c.Repos, func(r *v1.Repo) bool { return r.Id == req.Msg.Id }); idx != -1 {
		c.Repos[idx] = req.Msg
	} else {
		c.Repos = append(c.Repos, req.Msg)
	}

	if err := config.ValidateConfig(c); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	bin, err := resticinstaller.FindOrInstallResticBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to find or install restic binary: %w", err)
	}

	r, err := repo.NewRepoOrchestrator(c, req.Msg, bin)
	if err != nil {
		return nil, fmt.Errorf("failed to configure repo: %w", err)
	}

	// use background context such that the init op can try to complete even if the connection is closed.
	if err := r.Init(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to init repo: %w", err)
	}

	zap.L().Debug("updating config", zap.Int32("version", c.Version))
	if err := s.config.Update(c); err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	zap.L().Debug("applying config", zap.Int32("version", c.Version))
	s.orchestrator.ApplyConfig(c)

	// index snapshots for the newly added repository.
	zap.L().Debug("scheduling index snapshots task")
	s.orchestrator.ScheduleTask(tasks.NewOneoffIndexSnapshotsTask(req.Msg.Id, time.Now()), tasks.TaskPriorityInteractive+tasks.TaskPriorityIndexSnapshots)

	zap.L().Debug("done add repo")
	return connect.NewResponse(c), nil
}

// ListSnapshots implements POST /v1/snapshots
func (s *BackrestHandler) ListSnapshots(ctx context.Context, req *connect.Request[v1.ListSnapshotsRequest]) (*connect.Response[v1.ResticSnapshotList], error) {
	query := req.Msg
	repo, err := s.orchestrator.GetRepoOrchestrator(query.RepoId)
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

func (s *BackrestHandler) ListSnapshotFiles(ctx context.Context, req *connect.Request[v1.ListSnapshotFilesRequest]) (*connect.Response[v1.ListSnapshotFilesResponse], error) {
	query := req.Msg
	repo, err := s.orchestrator.GetRepoOrchestrator(query.RepoId)
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
func (s *BackrestHandler) GetOperationEvents(ctx context.Context, req *connect.Request[emptypb.Empty], resp *connect.ServerStream[v1.OperationEvent]) error {
	errChan := make(chan error, 1)
	events := make(chan *v1.OperationEvent, 100)

	timer := time.NewTicker(60 * time.Second)
	defer timer.Stop()

	callback := func(ops []*v1.Operation, eventType oplog.OperationEvent) {
		var event *v1.OperationEvent
		switch eventType {
		case oplog.OPERATION_ADDED:
			event = &v1.OperationEvent{
				Event: &v1.OperationEvent_CreatedOperations{
					CreatedOperations: &v1.OperationList{
						Operations: ops,
					},
				},
			}
		case oplog.OPERATION_UPDATED:
			event = &v1.OperationEvent{
				Event: &v1.OperationEvent_UpdatedOperations{
					UpdatedOperations: &v1.OperationList{
						Operations: ops,
					},
				},
			}
		case oplog.OPERATION_DELETED:
			ids := make([]int64, len(ops))
			for i, o := range ops {
				ids[i] = o.Id
			}

			event = &v1.OperationEvent{
				Event: &v1.OperationEvent_DeletedOperations{
					DeletedOperations: &types.Int64List{
						Values: ids,
					},
				},
			}
		default:
			zap.L().Error("Unknown event type")
		}

		select {
		case events <- event:
		default:
			select {
			case errChan <- errors.New("event buffer overflow, closing stream for client retry and catchup"):
			default:
			}
		}
	}

	s.oplog.Subscribe(oplog.SelectAll, &callback)
	defer func() {
		if err := s.oplog.Unsubscribe(&callback); err != nil {
			zap.L().Error("failed to unsubscribe from oplog", zap.Error(err))
		}
	}()

	for {
		select {
		case <-timer.C:
			if err := resp.Send(&v1.OperationEvent{
				Event: &v1.OperationEvent_KeepAlive{},
			}); err != nil {
				return err
			}
		case err := <-errChan:
			return err
		case <-ctx.Done():
			return nil
		case event := <-events:
			if err := resp.Send(event); err != nil {
				return err
			}
		}
	}
}

func (s *BackrestHandler) GetOperations(ctx context.Context, req *connect.Request[v1.GetOperationsRequest]) (*connect.Response[v1.OperationList], error) {
	q, err := opSelectorToQuery(req.Msg.Selector)
	if req.Msg.LastN != 0 {
		q.Reversed = true
		q.Limit = int(req.Msg.LastN)
	}
	if err != nil {
		return nil, err
	}

	var ops []*v1.Operation
	opCollector := func(op *v1.Operation) error {
		ops = append(ops, op)
		return nil
	}
	err = s.oplog.Query(q, opCollector)
	if err != nil {
		return nil, fmt.Errorf("failed to get operations: %w", err)
	}

	slices.SortFunc(ops, func(i, j *v1.Operation) int {
		if i.Id < j.Id {
			return -1
		}
		return 1
	})

	return connect.NewResponse(&v1.OperationList{
		Operations: ops,
	}), nil
}

func (s *BackrestHandler) IndexSnapshots(ctx context.Context, req *connect.Request[types.StringValue]) (*connect.Response[emptypb.Empty], error) {
	_, err := s.orchestrator.GetRepo(req.Msg.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to get repo %q: %w", req.Msg.Value, err)
	}

	s.orchestrator.ScheduleTask(tasks.NewOneoffIndexSnapshotsTask(req.Msg.Value, time.Now()), tasks.TaskPriorityInteractive+tasks.TaskPriorityIndexSnapshots)

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *BackrestHandler) Backup(ctx context.Context, req *connect.Request[types.StringValue]) (*connect.Response[emptypb.Empty], error) {
	plan, err := s.orchestrator.GetPlan(req.Msg.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan %q: %w", req.Msg.Value, err)
	}
	wait := make(chan struct{})
	s.orchestrator.ScheduleTask(tasks.NewOneoffBackupTask(plan, time.Now()), tasks.TaskPriorityInteractive, func(e error) {
		err = e
		close(wait)
	})
	<-wait
	return connect.NewResponse(&emptypb.Empty{}), err
}

func (s *BackrestHandler) Forget(ctx context.Context, req *connect.Request[v1.ForgetRequest]) (*connect.Response[emptypb.Empty], error) {
	at := time.Now()
	var err error
	if req.Msg.SnapshotId != "" && req.Msg.PlanId != "" && req.Msg.RepoId != "" {
		wait := make(chan struct{})
		s.orchestrator.ScheduleTask(
			tasks.NewOneoffForgetSnapshotTask(req.Msg.RepoId, req.Msg.PlanId, 0, at, req.Msg.SnapshotId),
			tasks.TaskPriorityInteractive+tasks.TaskPriorityForget, func(e error) {
				err = e
				close(wait)
			})
		<-wait
	} else if req.Msg.RepoId != "" && req.Msg.PlanId != "" {
		wait := make(chan struct{})
		s.orchestrator.ScheduleTask(
			tasks.NewOneoffForgetTask(req.Msg.RepoId, req.Msg.PlanId, 0, at),
			tasks.TaskPriorityInteractive+tasks.TaskPriorityForget, func(e error) {
				err = e
				close(wait)
			})
		<-wait
	} else {
		return nil, errors.New("must specify repoId and planId and (optionally) snapshotId")
	}
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s BackrestHandler) DoRepoTask(ctx context.Context, req *connect.Request[v1.DoRepoTaskRequest]) (*connect.Response[emptypb.Empty], error) {
	var task tasks.Task
	priority := tasks.TaskPriorityInteractive
	switch req.Msg.Task {
	case v1.DoRepoTaskRequest_TASK_CHECK:
		task = tasks.NewCheckTask(req.Msg.RepoId, tasks.PlanForSystemTasks, true)
	case v1.DoRepoTaskRequest_TASK_PRUNE:
		task = tasks.NewPruneTask(req.Msg.RepoId, tasks.PlanForSystemTasks, true)
		priority |= tasks.TaskPriorityPrune
	case v1.DoRepoTaskRequest_TASK_STATS:
		task = tasks.NewStatsTask(req.Msg.RepoId, tasks.PlanForSystemTasks, true)
		priority |= tasks.TaskPriorityStats
	case v1.DoRepoTaskRequest_TASK_INDEX_SNAPSHOTS:
		task = tasks.NewOneoffIndexSnapshotsTask(req.Msg.RepoId, time.Now())
		priority |= tasks.TaskPriorityIndexSnapshots
	case v1.DoRepoTaskRequest_TASK_UNLOCK:
		repo, err := s.orchestrator.GetRepoOrchestrator(req.Msg.RepoId)
		if err != nil {
			return nil, fmt.Errorf("failed to get repo %q: %w", req.Msg.RepoId, err)
		}
		if err := repo.Unlock(ctx); err != nil {
			return nil, fmt.Errorf("failed to unlock repo %q: %w", req.Msg.RepoId, err)
		}
		return connect.NewResponse(&emptypb.Empty{}), nil
	default:
		return nil, fmt.Errorf("unknown task %v", req.Msg.Task.String())
	}

	var err error
	wait := make(chan struct{})
	if err := s.orchestrator.ScheduleTask(task, priority, func(e error) {
		err = e
		close(wait)
	}); err != nil {
		return nil, err
	}
	<-wait
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *BackrestHandler) Restore(ctx context.Context, req *connect.Request[v1.RestoreSnapshotRequest]) (*connect.Response[emptypb.Empty], error) {
	if req.Msg.Target == "" {
		req.Msg.Target = path.Join(os.Getenv("HOME"), "Downloads", fmt.Sprintf("restic-restore-%v", time.Now().Format("2006-01-02T15-04-05")))
	}
	if req.Msg.Path == "" {
		req.Msg.Path = "/"
	}

	// prevent restoring to a directory that already exists
	if _, err := os.Stat(req.Msg.Target); err == nil {
		return nil, fmt.Errorf("target directory %q already exists", req.Msg.Target)
	}

	at := time.Now()
	s.orchestrator.ScheduleTask(tasks.NewOneoffRestoreTask(req.Msg.RepoId, req.Msg.PlanId, 0 /* flowID */, at, req.Msg.SnapshotId, req.Msg.Path, req.Msg.Target), tasks.TaskPriorityInteractive+tasks.TaskPriorityDefault)

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *BackrestHandler) RunCommand(ctx context.Context, req *connect.Request[v1.RunCommandRequest]) (*connect.Response[types.Int64Value], error) {
	// group commands within the last 24 hours (or 256 operations) into the same flow ID
	var flowID int64
	if s.oplog.Query(oplog.Query{RepoID: req.Msg.RepoId, Limit: 256, Reversed: true}, func(op *v1.Operation) error {
		if op.GetOperationRunCommand() != nil && time.Since(time.UnixMilli(op.UnixTimeStartMs)) < 30*time.Minute {
			flowID = op.FlowId
		}
		return nil
	}) != nil {
		return nil, fmt.Errorf("failed to query operations")
	}

	task := tasks.NewOneoffRunCommandTask(req.Msg.RepoId, tasks.PlanForSystemTasks, flowID, time.Now(), req.Msg.Command)
	st, err := s.orchestrator.CreateUnscheduledTask(task, tasks.TaskPriorityInteractive, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}
	if err := s.orchestrator.RunTask(context.Background(), st); err != nil {
		return nil, fmt.Errorf("failed to run command: %w", err)
	}

	return connect.NewResponse(&types.Int64Value{Value: st.Op.GetId()}), nil
}

func (s *BackrestHandler) Cancel(ctx context.Context, req *connect.Request[types.Int64Value]) (*connect.Response[emptypb.Empty], error) {
	if err := s.orchestrator.CancelOperation(req.Msg.Value, v1.OperationStatus_STATUS_USER_CANCELLED); err != nil {
		return nil, err
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *BackrestHandler) ClearHistory(ctx context.Context, req *connect.Request[v1.ClearHistoryRequest]) (*connect.Response[emptypb.Empty], error) {
	var err error
	var ids []int64

	opCollector := func(op *v1.Operation) error {
		if !req.Msg.OnlyFailed || op.Status == v1.OperationStatus_STATUS_ERROR {
			ids = append(ids, op.Id)
		}
		return nil
	}

	q, err := opSelectorToQuery(req.Msg.Selector)
	if err != nil {
		return nil, err
	}
	if err := s.oplog.Query(q, opCollector); err != nil {
		return nil, fmt.Errorf("failed to get operations to delete: %w", err)
	}

	if err := s.oplog.Delete(ids...); err != nil {
		return nil, fmt.Errorf("failed to delete operations: %w", err)
	}

	return connect.NewResponse(&emptypb.Empty{}), err
}

func (s *BackrestHandler) GetLogs(ctx context.Context, req *connect.Request[v1.LogDataRequest], resp *connect.ServerStream[types.BytesValue]) error {
	r, err := s.logStore.Open(req.Msg.Ref)
	if err != nil {
		if errors.Is(err, logstore.ErrLogNotFound) {
			resp.Send(&types.BytesValue{
				Value: []byte(fmt.Sprintf("file associated with log %v not found, it may have expired.", req.Msg.GetRef())),
			})
			return nil
		}
		return fmt.Errorf("get log data %v: %w", req.Msg.GetRef(), err)
	}
	go func() {
		<-ctx.Done()
		r.Close()
	}()

	var bufferMu sync.Mutex
	var buffer bytes.Buffer
	var errChan = make(chan error, 1)
	go func() {
		data := make([]byte, 4*1024)
		for {
			n, err := r.Read(data)
			if n == 0 {
				close(errChan)
				break
			} else if err != nil && err != io.EOF {
				errChan <- fmt.Errorf("failed to read log data: %w", err)
				close(errChan)
				return
			}
			bufferMu.Lock()
			buffer.Write(data[:n])
			bufferMu.Unlock()
		}
	}()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	flush := func() error {
		bufferMu.Lock()
		if buffer.Len() > 0 {
			if err := resp.Send(&types.BytesValue{Value: buffer.Bytes()}); err != nil {
				bufferMu.Unlock()
				return fmt.Errorf("failed to send log data: %w", err)
			}
			buffer.Reset()
		}
		bufferMu.Unlock()
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return flush()
		case err := <-errChan:
			_ = flush()
			return err
		case <-ticker.C:
			if err := flush(); err != nil {
				return err
			}
		}
	}

}

func (s *BackrestHandler) GetDownloadURL(ctx context.Context, req *connect.Request[types.Int64Value]) (*connect.Response[types.StringValue], error) {
	op, err := s.oplog.Get(req.Msg.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to get operation %v: %w", req.Msg.Value, err)
	}
	_, ok := op.Op.(*v1.Operation_OperationRestore)
	if !ok {
		return nil, fmt.Errorf("operation %v is not a restore operation", req.Msg.Value)
	}
	signature, err := signInt64(op.Id) // the signature authenticates the download URL. Note that the shared URL will be valid for any downloader.
	if err != nil {
		return nil, fmt.Errorf("failed to generate signature: %w", err)
	}
	return connect.NewResponse(&types.StringValue{
		Value: fmt.Sprintf("./download/%x-%s/", op.Id, hex.EncodeToString(signature)),
	}), nil
}

func (s *BackrestHandler) PathAutocomplete(ctx context.Context, path *connect.Request[types.StringValue]) (*connect.Response[types.StringList], error) {
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

// func (s *BackrestHandler) GetSummaryDashboard(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[v1.SummaryDashboardResponse], error) {
// 	// scan the oplog for each configured repo
// 	config, err := s.config.Get()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get config: %w", err)
// 	}

// 	response := &v1.SummaryDashboardResponse{

// 	for _, repo := range config.Repos {
// 		_, err := s.orchestrator.GetRepoOrchestrator(repo.Id)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to get repo %q: %w", repo.Id, err)
// 		}
// 	}

// 	return connect.NewResponse(dashboard), nil
// }

func opSelectorToQuery(sel *v1.OpSelector) (oplog.Query, error) {
	if sel == nil {
		return oplog.Query{}, errors.New("empty selector")
	}
	q := oplog.Query{
		RepoID:     sel.RepoId,
		PlanID:     sel.PlanId,
		SnapshotID: sel.SnapshotId,
		FlowID:     sel.FlowId,
	}
	if len(sel.Ids) > 0 && !reflect.DeepEqual(q, oplog.Query{}) {
		return oplog.Query{}, errors.New("cannot specify both query and ids")
	}
	q.OpIDs = sel.Ids
	return q, nil
}
