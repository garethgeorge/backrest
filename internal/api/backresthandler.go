package api

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path"
	"slices"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/garethgeorge/backrest/gen/go/types"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/gen/go/v1/v1connect"
	syncapi "github.com/garethgeorge/backrest/internal/api/syncapi"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
	"github.com/garethgeorge/backrest/internal/env"
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
	config           config.ConfigStore
	orchestrator     *orchestrator.Orchestrator
	oplog            *oplog.OpLog
	logStore         *logstore.LogStore
	peerStateManager syncapi.PeerStateManager
}

var _ v1connect.BackrestHandler = &BackrestHandler{}

func NewBackrestHandler(config config.ConfigStore, peerStateManager syncapi.PeerStateManager, orchestrator *orchestrator.Orchestrator, oplog *oplog.OpLog, logStore *logstore.LogStore) *BackrestHandler {
	s := &BackrestHandler{
		config:           config,
		orchestrator:     orchestrator,
		oplog:            oplog,
		logStore:         logStore,
		peerStateManager: peerStateManager,
	}

	return s
}

// GetConfig implements GET /v1/config
func (s *BackrestHandler) GetConfig(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[v1.Config], error) {
	c, err := s.config.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	return connect.NewResponse(config.SanitizeForNetwork(c)), nil
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

	// Rehydrate the network sanitized config
	rehydratedConfig := config.RehydrateNetworkSanitizedConfig(req.Msg, existing)

	for _, repo := range rehydratedConfig.Repos {
		if err := ensureSftpFlags(repo); err != nil {
			return nil, err
		}
	}

	if err := config.ValidateConfig(rehydratedConfig); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	rehydratedConfig.Modno++

	if err := s.config.Update(rehydratedConfig); err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	newConfig, err := s.config.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get newly set config: %w", err)
	}
	return connect.NewResponse(newConfig), nil
}

func (s *BackrestHandler) CheckRepoExists(ctx context.Context, req *connect.Request[v1.Repo]) (*connect.Response[types.BoolValue], error) {
	c, err := s.config.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	if req.Msg.GetSftpAutoAddToKnownHosts() {
		if err := s.addSftpHostKey(req.Msg.GetUri()); err != nil {
			zap.S().Warnf("failed to automatically add SFTP host key for %s: %v", req.Msg.GetUri(), err)
		}
	}

	// We need to build the repo with the correct flags for this one-off check.
	repoForCheck := proto.Clone(req.Msg).(*v1.Repo)
	if err := ensureSftpFlags(repoForCheck); err != nil {
		return nil, err
	}

	c = proto.Clone(c).(*v1.Config)
	if idx := slices.IndexFunc(c.Repos, func(r *v1.Repo) bool { return r.Id == req.Msg.Id }); idx != -1 {
		c.Repos[idx] = req.Msg
	} else {
		c.Repos = append(c.Repos, req.Msg)
	}

	if req.Msg.Guid == "" {
		req.Msg.Guid = cryptoutil.MustRandomID(cryptoutil.DefaultIDBits)
	}

	if err := config.ValidateConfig(c); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	bin, err := resticinstaller.FindOrInstallResticBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to find or install restic binary: %w", err)
	}

	r, err := repo.NewRepoOrchestrator(c, repoForCheck, bin)
	if err != nil {
		return nil, fmt.Errorf("failed to configure repo: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := r.Exists(ctx); err != nil {
		zap.S().Debugf("repo %q exists or not: %v", req.Msg.Id, err)
		if errors.Is(err, restic.ErrRepoNotFound) {
			zap.S().Debugf("repo %q does not exist", req.Msg.Id)
			return connect.NewResponse(&types.BoolValue{Value: false}), nil
		}
		return nil, err
	}
	return connect.NewResponse(&types.BoolValue{Value: true}), nil
}

// AddRepo implements POST /v1/config/repo, it includes validation that the repo can be initialized.
func (s *BackrestHandler) AddRepo(ctx context.Context, req *connect.Request[v1.Repo]) (*connect.Response[v1.Config], error) {
	c, err := s.config.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	newRepo := req.Msg

	if newRepo.GetSftpAutoAddToKnownHosts() {
		if err := s.addSftpHostKey(newRepo.GetUri()); err != nil {
			zap.S().Warnf("failed to automatically add SFTP host key for %s: %v", newRepo.GetUri(), err)
		}
	}

	if err := ensureSftpFlags(newRepo); err != nil {
		return nil, err
	}

	// Deep copy the configuration
	c = proto.Clone(c).(*v1.Config)

	// Add or implicit update the repo
	var oldRepo *v1.Repo
	if idx := slices.IndexFunc(c.Repos, func(r *v1.Repo) bool { return r.Id == newRepo.Id }); idx != -1 {
		oldRepo = c.Repos[idx]
		c.Repos[idx] = newRepo
	} else {
		c.Repos = append(c.Repos, newRepo)
	}

	// Ensure the Repo GUID is set to the correct value.
	// This is derived from 'restic cat config' for local repos.
	// For remote repos, the GUID is derived from the remote config's value for the repo.
	bin, err := resticinstaller.FindOrInstallResticBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to find or install restic binary: %w", err)
	}

	r, err := repo.NewRepoOrchestrator(c, newRepo, bin)
	if err != nil {
		return nil, fmt.Errorf("failed to configure repo: %w", err)
	}

	if err := r.Init(ctx); err != nil {
		return nil, fmt.Errorf("failed to init repo: %w", err)
	}

	guid, err := r.RepoGUID()
	zap.S().Debugf("GUID for repo %q is %q from restic", newRepo.Id, guid)
	if err != nil {
		return nil, fmt.Errorf("failed to get repo config: %w", err)
	}

	newRepo.Guid = guid

	if err := config.ValidateConfig(c); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	zap.L().Debug("updating config", zap.Int32("version", c.Version))
	if err := s.config.Update(c); err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	// If the GUID has changed, and we just successfully updated the config in storage, then we need to migrate the oplog.
	if oldRepo != nil && newRepo.Guid != oldRepo.Guid {
		migratedCount := 0

		q := oplog.Query{}.
			SetInstanceID(c.Instance)
		// we use RepoID here to _ensure_ we consolidate all operations to the most recent GUID.
		// this provides some resiliancy in the case of a previous partial update.
		q.DeprecatedRepoID = &oldRepo.Id
		if err := s.oplog.Transform(q, func(op *v1.Operation) (*v1.Operation, error) {
			op.RepoGuid = newRepo.Guid
			migratedCount++
			return op, nil
		}); err != nil {
			return nil, fmt.Errorf("failed to get operations for repo: %w", err)
		}

		zap.S().Infof("updated GUID for repo %q from %q to %q, migrated %d operations to reference the new GUID", newRepo.Id, oldRepo.Guid, newRepo.Guid, migratedCount)
	}

	// index snapshots for the newly added repository.
	zap.L().Debug("scheduling index snapshots task")
	s.orchestrator.ScheduleTask(tasks.NewOneoffIndexSnapshotsTask(newRepo, time.Now()), tasks.TaskPriorityInteractive+tasks.TaskPriorityIndexSnapshots)

	zap.L().Debug("done add repo")
	return connect.NewResponse(c), nil
}

func (s *BackrestHandler) RemoveRepo(ctx context.Context, req *connect.Request[types.StringValue]) (*connect.Response[v1.Config], error) {
	cfg, err := s.config.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	// Remove the repo from the configuration
	cfg.Repos = slices.DeleteFunc(cfg.Repos, func(r *v1.Repo) bool {
		return r.Id == req.Msg.Value
	})
	if err := s.config.Update(cfg); err != nil {
		return nil, fmt.Errorf("failed to update config: %w", err)
	}

	// Query for all operations for the repo
	q := oplog.Query{}.
		SetInstanceID(cfg.Instance)
	q.DeprecatedRepoID = &req.Msg.Value
	var opIDs []int64
	if err := s.oplog.Query(q, func(op *v1.Operation) error {
		opIDs = append(opIDs, op.Id)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to get operations for repo: %w", err)
	}

	// Delete operations referencing the repo from the oplog in batches
	for len(opIDs) > 0 {
		batchSize := 256
		if batchSize > len(opIDs) {
			batchSize = len(opIDs)
		}
		if err := s.oplog.Delete(opIDs[:batchSize]...); err != nil {
			return nil, fmt.Errorf("failed to delete operations: %w", err)
		}
		opIDs = opIDs[batchSize:]
	}

	return connect.NewResponse(cfg), nil
}

// addSftpHostKey parses an SFTP URI and adds the host's public key to the user's known_hosts file.
// This is equivalent to what `ssh-keyscan` does.
func (s *BackrestHandler) addSftpHostKey(uri string) error {
	if !strings.HasPrefix(uri, "sftp:") {
		return nil
	}
	uri = strings.TrimPrefix(uri, "sftp:")

	slashIdx := strings.Index(uri, "/")
	if slashIdx == -1 {
		slashIdx = len(uri)
	}

	authority := uri[:slashIdx]
	hostPart := authority
	if atIdx := strings.LastIndex(authority, "@"); atIdx != -1 {
		hostPart = authority[atIdx+1:]
	}

	host, _, err := net.SplitHostPort(hostPart)
	if err != nil {
		host = hostPart
	}

	if host == "" {
		return errors.New("could not parse host from sftp uri")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get home directory: %w", err)
	}
	knownHostsPath := path.Join(home, ".ssh", "known_hosts")

	if err := os.MkdirAll(path.Dir(knownHostsPath), 0700); err != nil {
		return err
	}

	checkCmd := exec.Command("ssh-keygen", "-F", host)
	if err := checkCmd.Run(); err == nil {
		zap.S().Debugf("SFTP host %s already in known_hosts", host)
		return nil
	}

	keyscanCmd := exec.Command("ssh-keyscan", "-H", host)
	keyOutput, err := keyscanCmd.Output()
	if err != nil {
		return fmt.Errorf("ssh-keyscan for host %s failed: %w", host, err)
	}

	f, err := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open known_hosts file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(keyOutput); err != nil {
		return fmt.Errorf("failed to write to known_hosts file: %w", err)
	}

	zap.S().Infof("Added SFTP host %s to known_hosts file at %s", host, knownHostsPath)
	return nil
}

func ensureSftpFlags(repo *v1.Repo) error {
	if !strings.HasPrefix(repo.GetUri(), "sftp:") {
		return nil
	}

	// Remove any existing sftp.args flags to avoid duplicates.
	repo.Flags = slices.DeleteFunc(repo.Flags, func(f string) bool {
		return strings.HasPrefix(f, "--option=sftp.args")
	})

	sftpArgs := "-oBatchMode=yes"
	argsChanged := false

	if repo.GetSftpIdentityFile() != "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("could not get home directory: %w", err)
		}
		keyPath := path.Join(home, ".ssh", repo.GetSftpIdentityFile())
		// We don't check for file existence here, because the config might be edited on a different machine.
		// Restic will fail later if the path is invalid, which is acceptable.
		sftpArgs += fmt.Sprintf(" -i %s", keyPath)
		argsChanged = true
	}

	if repo.GetSftpPort() != 0 && repo.GetSftpPort() != 22 {
		sftpArgs += fmt.Sprintf(" -p %d", repo.GetSftpPort())
		argsChanged = true
	}

	if argsChanged {
		repo.Flags = append(repo.Flags, fmt.Sprintf("--option=sftp.args='%s'", sftpArgs))
	}

	return nil
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
	q, err := protoutil.OpSelectorToQuery(req.Msg.Selector)
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
	// Ensure the repo is valid before scheduling the task
	repo, err := s.orchestrator.GetRepo(req.Msg.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to get repo %q: %w", req.Msg.Value, err)
	}

	// Schedule the indexing task
	if err := s.orchestrator.ScheduleTask(tasks.NewOneoffIndexSnapshotsTask(repo, time.Now()), tasks.TaskPriorityInteractive+tasks.TaskPriorityIndexSnapshots); err != nil {
		return nil, fmt.Errorf("failed to schedule indexing task: %w", err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *BackrestHandler) Backup(ctx context.Context, req *connect.Request[types.StringValue]) (*connect.Response[emptypb.Empty], error) {
	plan, err := s.orchestrator.GetPlan(req.Msg.Value)
	if err != nil {
		return nil, err
	}
	repo, err := s.orchestrator.GetRepo(plan.Repo)
	if err != nil {
		return nil, err
	}
	wait := make(chan struct{})
	if err := s.orchestrator.ScheduleTask(tasks.NewOneoffBackupTask(repo, plan, time.Now()), tasks.TaskPriorityInteractive, func(e error) {
		err = e
		close(wait)
	}); err != nil {
		return nil, err
	}
	<-wait
	return connect.NewResponse(&emptypb.Empty{}), err
}

func (s *BackrestHandler) Forget(ctx context.Context, req *connect.Request[v1.ForgetRequest]) (*connect.Response[emptypb.Empty], error) {
	at := time.Now()
	var err error

	repo, err := s.orchestrator.GetRepo(req.Msg.RepoId)
	if err != nil {
		return nil, err
	}

	if req.Msg.SnapshotId != "" && req.Msg.PlanId != "" && req.Msg.RepoId != "" {
		wait := make(chan struct{})
		if err := s.orchestrator.ScheduleTask(
			tasks.NewOneoffForgetSnapshotTask(repo, req.Msg.PlanId, 0, at, req.Msg.SnapshotId),
			tasks.TaskPriorityInteractive+tasks.TaskPriorityForget, func(e error) {
				err = e
				close(wait)
			}); err != nil {
			return nil, err
		}
		<-wait
	} else if req.Msg.RepoId != "" && req.Msg.PlanId != "" {
		wait := make(chan struct{})
		if err := s.orchestrator.ScheduleTask(
			tasks.NewOneoffForgetTask(repo, req.Msg.PlanId, 0, at),
			tasks.TaskPriorityInteractive+tasks.TaskPriorityForget, func(e error) {
				err = e
				close(wait)
			}); err != nil {
			return nil, err
		}
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

	repo, err := s.orchestrator.GetRepo(req.Msg.RepoId)
	if err != nil {
		return nil, err
	}

	priority := tasks.TaskPriorityInteractive
	switch req.Msg.Task {
	case v1.DoRepoTaskRequest_TASK_CHECK:
		task = tasks.NewCheckTask(repo, tasks.PlanForSystemTasks, true)
	case v1.DoRepoTaskRequest_TASK_PRUNE:
		task = tasks.NewPruneTask(repo, tasks.PlanForSystemTasks, true)
		priority |= tasks.TaskPriorityPrune
	case v1.DoRepoTaskRequest_TASK_STATS:
		task = tasks.NewStatsTask(repo, tasks.PlanForSystemTasks, true)
		priority |= tasks.TaskPriorityStats
	case v1.DoRepoTaskRequest_TASK_INDEX_SNAPSHOTS:
		task = tasks.NewOneoffIndexSnapshotsTask(repo, time.Now())
		priority |= tasks.TaskPriorityIndexSnapshots
	case v1.DoRepoTaskRequest_TASK_UNLOCK:
		repo, err := s.orchestrator.GetRepoOrchestrator(req.Msg.RepoId)
		if err != nil {
			return nil, err
		}
		if err := repo.Unlock(ctx); err != nil {
			return nil, fmt.Errorf("failed to unlock repo %q: %w", req.Msg.RepoId, err)
		}
		return connect.NewResponse(&emptypb.Empty{}), nil
	default:
		return nil, fmt.Errorf("unknown task %v", req.Msg.Task.String())
	}

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
	req.Msg.Target = strings.TrimSpace(req.Msg.Target)
	req.Msg.Path = strings.TrimSpace(req.Msg.Path)

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

	repo, err := s.orchestrator.GetRepo(req.Msg.RepoId)
	if err != nil {
		return nil, err
	}

	at := time.Now()
	s.orchestrator.ScheduleTask(tasks.NewOneoffRestoreTask(repo, req.Msg.PlanId, 0 /* flowID */, at, req.Msg.SnapshotId, req.Msg.Path, req.Msg.Target), tasks.TaskPriorityInteractive+tasks.TaskPriorityDefault)

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *BackrestHandler) RunCommand(ctx context.Context, req *connect.Request[v1.RunCommandRequest]) (*connect.Response[types.Int64Value], error) {
	cfg, err := s.config.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	repo := config.FindRepo(cfg, req.Msg.RepoId)
	if repo == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("repo %q not found", req.Msg.RepoId))
	}

	// group commands within the last 24 hours (or 256 operations) into the same flow ID
	var flowID int64
	if s.oplog.Query(oplog.Query{}.
		SetInstanceID(cfg.Instance).
		SetRepoGUID(repo.GetGuid()).
		SetLimit(256).
		SetReversed(true), func(op *v1.Operation) error {
		if op.GetOperationRunCommand() != nil && time.Since(time.UnixMilli(op.UnixTimeStartMs)) < 30*time.Minute {
			flowID = op.FlowId
		}
		return nil
	}) != nil {
		return nil, fmt.Errorf("failed to query operations")
	}

	task := tasks.NewOneoffRunCommandTask(repo, tasks.PlanForSystemTasks, flowID, time.Now(), req.Msg.Command)
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

	q, err := protoutil.OpSelectorToQuery(req.Msg.Selector)
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

	var errChan = make(chan error, 1)
	var sendChan = make(chan []byte, 1)
	var buffer bytes.Buffer
	var bufferMu sync.Mutex

	go func() {
		data := make([]byte, 4*1024)
		for {
			n, err := r.Read(data)
			if n == 0 {
				break
			} else if err != nil && err != io.EOF {
				errChan <- fmt.Errorf("failed to read log data: %w", err)
				close(errChan)
				break
			}
			bufferMu.Lock()
			buffer.Write(data[:n])
			if buffer.Len() > 128*1024 {
				sendChan <- bytes.Clone(buffer.Bytes())
				buffer.Reset()
			}
			bufferMu.Unlock()
		}

		if buffer.Len() > 0 {
			bufferMu.Lock()
			sendChan <- bytes.Clone(buffer.Bytes())
			buffer.Reset()
			bufferMu.Unlock()
		}
		close(sendChan)
	}()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case data, ok := <-sendChan:
			if !ok {
				return nil
			}
			if err := resp.Send(&types.BytesValue{Value: data}); err != nil {
				bufferMu.Lock()
				buffer.Write(data)
				bufferMu.Unlock()
				return err
			}
		case err := <-errChan:
			return err
		case <-ticker.C:
			bufferMu.Lock()
			if buffer.Len() > 0 {
				if err := resp.Send(&types.BytesValue{Value: bytes.Clone(buffer.Bytes())}); err != nil {
					bufferMu.Unlock()
					return err
				}
				buffer.Reset()
			}
			bufferMu.Unlock()
		}
	}

}

func (s *BackrestHandler) GetDownloadURL(ctx context.Context, req *connect.Request[types.Int64Value]) (*connect.Response[types.StringValue], error) {
	op, err := s.oplog.Get(req.Msg.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to get operation %v: %w", req.Msg.Value, err)
	}

	switch op.Op.(type) {
	case *v1.Operation_OperationIndexSnapshot:
	case *v1.Operation_OperationRestore:
	default:
		return nil, fmt.Errorf("operation %v is not a restore or snapshot operation", req.Msg.Value)
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

func (s *BackrestHandler) GetSummaryDashboard(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[v1.SummaryDashboardResponse], error) {
	config, err := s.config.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	generateSummaryHelper := func(id string, q oplog.Query) (*v1.SummaryDashboardResponse_Summary, error) {
		var backupsExamined int64
		var bytesScanned30 int64
		var bytesAdded30 int64
		var backupsFailed30 int64
		var backupsSuccess30 int64
		var backupsWarning30 int64
		var nextBackupTime int64
		backupChart := &v1.SummaryDashboardResponse_BackupChart{}

		s.oplog.Query(q, func(op *v1.Operation) error {
			t := time.UnixMilli(op.UnixTimeStartMs)

			if backupOp := op.GetOperationBackup(); backupOp != nil {
				if time.Since(t) > 30*24*time.Hour {
					return oplog.ErrStopIteration
				} else if op.GetStatus() == v1.OperationStatus_STATUS_PENDING {
					nextBackupTime = op.UnixTimeStartMs
					return nil
				}
				backupsExamined++

				if op.Status == v1.OperationStatus_STATUS_SUCCESS {
					backupsSuccess30++
				} else if op.Status == v1.OperationStatus_STATUS_ERROR {
					backupsFailed30++
				} else if op.Status == v1.OperationStatus_STATUS_WARNING {
					backupsWarning30++
				}

				if summary := backupOp.GetLastStatus().GetSummary(); summary != nil {
					bytesScanned30 += summary.TotalBytesProcessed
					bytesAdded30 += summary.DataAdded
				}

				// recent backups chart
				if len(backupChart.TimestampMs) < 60 { // only include the latest 90 backups in the chart
					duration := op.UnixTimeEndMs - op.UnixTimeStartMs
					if duration <= 1000 {
						duration = 1000
					}

					backupChart.FlowId = append(backupChart.FlowId, op.FlowId)
					backupChart.TimestampMs = append(backupChart.TimestampMs, op.UnixTimeStartMs)
					backupChart.DurationMs = append(backupChart.DurationMs, duration)
					backupChart.Status = append(backupChart.Status, op.Status)
					backupChart.BytesAdded = append(backupChart.BytesAdded, backupOp.GetLastStatus().GetSummary().GetDataAdded())
				}
			}

			return nil
		})

		if backupsExamined == 0 {
			backupsExamined = 1 // prevent division by zero for avg calculations
		}

		return &v1.SummaryDashboardResponse_Summary{
			Id:                        id,
			BytesScannedLast_30Days:   bytesScanned30,
			BytesAddedLast_30Days:     bytesAdded30,
			BackupsFailed_30Days:      backupsFailed30,
			BackupsWarningLast_30Days: backupsWarning30,
			BackupsSuccessLast_30Days: backupsSuccess30,
			BytesScannedAvg:           bytesScanned30 / backupsExamined,
			BytesAddedAvg:             bytesAdded30 / backupsExamined,
			NextBackupTimeMs:          nextBackupTime,
			RecentBackups:             backupChart,
		}, nil
	}

	response := &v1.SummaryDashboardResponse{
		ConfigPath: env.ConfigFilePath(),
		DataPath:   env.DataDir(),
	}

	for _, repo := range config.Repos {
		resp, err := generateSummaryHelper(repo.Id, oplog.Query{}.
			SetInstanceID(config.Instance).
			SetRepoGUID(repo.GetGuid()).
			SetReversed(true).
			SetLimit(1000))
		if err != nil {
			return nil, fmt.Errorf("summary for repo %q: %w", repo.Id, err)
		}

		response.RepoSummaries = append(response.RepoSummaries, resp)
	}

	for _, plan := range config.Plans {
		resp, err := generateSummaryHelper(plan.Id, oplog.Query{}.
			SetInstanceID(config.Instance).
			SetPlanID(plan.Id).
			SetReversed(true).
			SetLimit(1000))
		if err != nil {
			return nil, fmt.Errorf("summary for plan %q: %w", plan.Id, err)
		}

		response.PlanSummaries = append(response.PlanSummaries, resp)
	}

	return connect.NewResponse(response), nil
}
