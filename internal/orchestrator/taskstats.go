package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog/indexutil"
	"go.uber.org/zap"
)

var statBytesThreshold int64 = 10 * 1024 * 1024 * 1024 // 10 GB added.

// StatsTask tracks a restic stats operation.
type StatsTask struct {
	TaskWithOperation
	plan         *v1.Plan
	linkSnapshot string // snapshot to link the task to (if any)
	at           *time.Time
}

var _ Task = &StatsTask{}

func NewOneoffStatsTask(orchestrator *Orchestrator, plan *v1.Plan, linkSnapshot string, at time.Time) *StatsTask {
	return &StatsTask{
		TaskWithOperation: TaskWithOperation{
			orch: orchestrator,
		},
		plan:         plan,
		at:           &at,
		linkSnapshot: linkSnapshot,
	}
}

func (t *StatsTask) Name() string {
	return fmt.Sprintf("stats for plan %q", t.plan.Id)
}

func (t *StatsTask) shouldRun() (bool, error) {
	var bytesSinceLastStat int64 = -1
	if err := t.orch.OpLog.ForEachByRepo(t.plan.Repo, indexutil.CollectLastN(50), func(op *v1.Operation) error {
		if _, ok := op.Op.(*v1.Operation_OperationStats); ok {
			bytesSinceLastStat = 0
		} else if backup, ok := op.Op.(*v1.Operation_OperationBackup); ok && backup.OperationBackup.LastStatus != nil {
			if summary, ok := backup.OperationBackup.LastStatus.Entry.(*v1.BackupProgressEntry_Summary); ok {
				bytesSinceLastStat += summary.Summary.DataAdded
			}
		}
		return nil
	}); err != nil {
		return false, fmt.Errorf("iterate oplog: %w", err)
	}

	zap.L().Debug("bytes since last stat", zap.Int64("bytes", bytesSinceLastStat), zap.String("repo", t.plan.Repo))
	if bytesSinceLastStat == -1 || bytesSinceLastStat > statBytesThreshold {
		zap.S().Debugf("bytes since last stat (%v) exceeds threshold (%v)", bytesSinceLastStat, statBytesThreshold)
		return true, nil
	}
	return false, nil
}

func (t *StatsTask) Next(now time.Time) *time.Time {
	shouldRun, err := t.shouldRun()
	if err != nil {
		zap.S().Errorf("task %v failed to check if it should run: %v", t.Name(), err)
	}
	if !shouldRun {
		return nil
	}

	ret := t.at
	if ret != nil {
		t.at = nil
		if err := t.setOperation(&v1.Operation{
			PlanId:          t.plan.Id,
			RepoId:          t.plan.Repo,
			SnapshotId:      t.linkSnapshot,
			UnixTimeStartMs: timeToUnixMillis(*ret),
			Status:          v1.OperationStatus_STATUS_PENDING,
			Op:              &v1.Operation_OperationStats{},
		}); err != nil {
			zap.S().Errorf("task %v failed to add operation to oplog: %v", t.Name(), err)
			return nil
		}
	}
	return ret
}

func (t *StatsTask) Run(ctx context.Context) error {
	if t.plan.Retention == nil {
		return errors.New("plan does not have a retention policy")
	}

	return t.runWithOpAndContext(ctx, func(ctx context.Context, op *v1.Operation) error {
		repo, err := t.orch.GetRepo(t.plan.Repo)
		if err != nil {
			return fmt.Errorf("get repo %q: %w", t.plan.Repo, err)
		}

		stats, err := repo.Stats(ctx)
		if err != nil {
			return fmt.Errorf("get stats: %w", err)
		}

		op.Op = &v1.Operation_OperationStats{
			OperationStats: &v1.OperationStats{
				Stats: stats,
			},
		}

		return err
	})
}
