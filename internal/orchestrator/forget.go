package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	"go.uber.org/zap"
)

// ForgetTask tracks a forget operation.
type ForgetTask struct {
	name         string
	orchestrator *Orchestrator // owning orchestrator
	plan         *v1.Plan
	linkSnapshot string // snapshot to link the task to.
	op           *v1.Operation
	at           *time.Time
	cancel       atomic.Pointer[context.CancelFunc] // nil unless operation is running.
}

var _ Task = &ForgetTask{}

func NewOneofForgetTask(orchestrator *Orchestrator, plan *v1.Plan, linkSnapshot string, at time.Time) *ForgetTask {
	return &ForgetTask{
		orchestrator: orchestrator,
		plan:         plan,
		at:           &at,
		linkSnapshot: linkSnapshot,
	}
}

func (t *ForgetTask) Name() string {
	return fmt.Sprintf("forget for plan %q", t.plan.Id)
}

func (t *ForgetTask) Next(now time.Time) *time.Time {
	ret := t.at
	if ret != nil {
		t.at = nil
		t.op = &v1.Operation{
			PlanId:          t.plan.Id,
			RepoId:          t.plan.Repo,
			SnapshotId:      t.linkSnapshot,
			UnixTimeStartMs: timeToUnixMillis(*ret),
			Status:          v1.OperationStatus_STATUS_PENDING,
			Op:              &v1.Operation_OperationForget{},
		}
	}
	return ret
}

func (t *ForgetTask) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	t.cancel.Store(&cancel)
	defer t.cancel.Store(nil)

	if t.plan.Retention == nil {
		return errors.New("plan does not have a retention policy")
	}

	forgetOp := &v1.Operation_OperationForget{
		OperationForget: &v1.OperationForget{},
	}

	t.op.Op = forgetOp
	t.op.UnixTimeStartMs = curTimeMillis()

	var repo *RepoOrchestrator
	if err := WithOperation(t.orchestrator.OpLog, t.op, func() error {
		var err error
		repo, err = t.orchestrator.GetRepo(t.plan.Repo)
		if err != nil {
			return fmt.Errorf("get repo %q: %w", t.plan.Repo, err)
		}

		forgot, err := repo.Forget(ctx, t.plan)
		if err != nil {
			return fmt.Errorf("forget: %w", err)
		}

		forgetOp.OperationForget.Forget = append(forgetOp.OperationForget.Forget, forgot...)

		for _, forgot := range forgot {
			t.orchestrator.OpLog.ForEachBySnapshotId(forgot.Id, func(op *v1.Operation) error {
				return t.orchestrator.OpLog.Delete(op.Id)
			})

		}

		return nil
	}); err != nil {
		return err
	}

	if repo.repoConfig.PrunePolicy != nil {
		// TODO: schedule a prune task.
		zap.S().Warn("repo specified a prune policy, automatic pruning is not yet implemented.")
	}

	return nil
}

func (t *ForgetTask) Cancel(status v1.OperationStatus) error {
	if t.op == nil {
		return nil
	}

	cancel := t.cancel.Load()
	if cancel != nil && status == v1.OperationStatus_STATUS_USER_CANCELLED {
		(*cancel)() // try to interrupt the running operation.
	}

	t.op.Status = status
	t.op.UnixTimeEndMs = curTimeMillis()
	return t.orchestrator.OpLog.Update(t.op)
}
