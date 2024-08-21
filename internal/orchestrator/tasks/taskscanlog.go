package tasks

import (
	"context"
	"fmt"
	"slices"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"go.uber.org/zap"
)

type ScanLogTask struct {
	didRun bool
}

var _ Task = &ScanLogTask{}

func (*ScanLogTask) Name() string {
	return "scan operation log"
}

func (t *ScanLogTask) Next(now time.Time, runner TaskRunner) (ScheduledTask, error) {
	if t.didRun {
		return NeverScheduledTask, nil
	}

	return ScheduledTask{
		Task:  &ScanLogTask{},
		RunAt: time.Now(),
	}, nil
}

func (*ScanLogTask) Run(ctx context.Context, st ScheduledTask, runner TaskRunner) error {
	oplog := runner.OpLog()

	var incompleteOpRepos []string
	if err := oplog.Scan(func(incomplete *v1.Operation) {
		incomplete.Status = v1.OperationStatus_STATUS_ERROR
		incomplete.DisplayMessage = "Failed, orchestrator killed while operation was in progress."

		if incomplete.RepoId != "" && !slices.Contains(incompleteOpRepos, incomplete.RepoId) {
			incompleteOpRepos = append(incompleteOpRepos, incomplete.RepoId)
		}
	}); err != nil {
		return fmt.Errorf("scan oplog: %w", err)
	}

	for _, repoId := range incompleteOpRepos {
		repo, err := runner.GetRepoOrchestrator(repoId)
		if err != nil {
			zap.L().Warn("repo not found for incomplete operation. Possibly just deleted.", zap.String("repo", repoId))
			continue
		}
		if err := repo.Unlock(context.Background()); err != nil {
			zap.L().Error("failed to unlock repo", zap.String("repo", repoId), zap.Error(err))
		}
	}

	return nil
}

func (*ScanLogTask) PlanID() string {
	return ""
}

func (*ScanLogTask) RepoID() string {
	return ""
}
