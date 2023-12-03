package orchestrator

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	"github.com/garethgeorge/resticui/internal/oplog/indexutil"
	"go.uber.org/zap"
)

// PruneTask tracks a forget operation.
type PruneTask struct {
	name         string
	orchestrator *Orchestrator // owning orchestrator
	plan         *v1.Plan
	linkSnapshot string // snapshot to link the task to.
	op           *v1.Operation
	at           *time.Time
	force        bool
}

var _ Task = &PruneTask{}

func NewOneofPruneTask(orchestrator *Orchestrator, plan *v1.Plan, linkSnapshot string, at time.Time, force bool) *PruneTask {
	return &PruneTask{
		orchestrator: orchestrator,
		plan:         plan,
		at:           &at,
		linkSnapshot: linkSnapshot,
		force:        force, // overrides the PrunePolicy's MaxFrequencyDays
	}
}

func (t *PruneTask) Name() string {
	return fmt.Sprintf("prune for plan %q", t.plan.Id)
}

func (t *PruneTask) Next(now time.Time) *time.Time {
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

func (t *PruneTask) getNextPruneTime(repo *RepoOrchestrator, policy *v1.PrunePolicy) (time.Time, error) {
	var lastPruneTime time.Time
	t.orchestrator.OpLog.ForEachByRepo(t.plan.Repo, indexutil.CollectLastN(1000), func(op *v1.Operation) error {
		if _, ok := op.Op.(*v1.Operation_OperationPrune); ok {
			lastPruneTime = time.Unix(0, op.UnixTimeStartMs*int64(time.Millisecond))
		}
		return nil
	})

	if repo.repoConfig.PrunePolicy != nil {
		return lastPruneTime.Add(time.Duration(repo.repoConfig.PrunePolicy.MaxFrequencyDays) * 24 * time.Hour), nil
	} else {
		return lastPruneTime.Add(7 * 24 * time.Hour), nil // default to 7 days.
	}
}

func (t *PruneTask) Run(ctx context.Context) error {
	t.op.UnixTimeStartMs = curTimeMillis()

	return WithOperation(t.orchestrator.OpLog, t.op, func() error {
		repo, err := t.orchestrator.GetRepo(t.plan.Repo)
		if err != nil {
			return fmt.Errorf("get repo %v: %w", t.plan.Repo, err)
		}

		opPrune := &v1.Operation_OperationPrune{
			OperationPrune: &v1.OperationPrune{},
		}
		t.op.Op = opPrune

		if !t.force {
			nextPruneTime, err := t.getNextPruneTime(repo, repo.repoConfig.PrunePolicy)
			if err != nil {
				return fmt.Errorf("get next prune time: %w", err)
			}
			if nextPruneTime.After(time.Now()) {
				opPrune.OperationPrune.Output = "Skipping prune operation.\nPrune will next run at (or after): " + nextPruneTime.String() + "\nAdjust prune policy's MaxFrequencyDays to increase or decrease the interval."
				return nil
			}
		}

		ctx, cancel := context.WithCancel(ctx)
		interval := time.NewTicker(1 * time.Second)
		defer interval.Stop()
		var buf synchronizedBuffer
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {

			defer wg.Done()
			for {
				select {
				case <-interval.C:
					output := buf.String()
					if len(output) > 8*1024 { // only provide live status upto the first 8K of output.
						output = output[:len(output)-8*1024]
					}

					if opPrune.OperationPrune.Output != output {
						opPrune.OperationPrune.Output = buf.String()

						if err := t.orchestrator.OpLog.Update(t.op); err != nil {
							zap.L().Error("update prune operation with status output", zap.Error(err))
						}
					}
				case <-ctx.Done():
					return
				}
			}
		}()

		if err := repo.Prune(ctx, &buf); err != nil {
			cancel()
			return fmt.Errorf("prune: %w", err)
		}
		cancel()
		wg.Wait()

		// TODO: it would be best to store the output in separate storage for large status data.
		output := buf.String()
		if len(output) > 8*1024 { // only provide live status upto the first 8K of output.
			output = output[:len(output)-8*1024]
		}
		t.op.Op = &v1.Operation_OperationPrune{
			OperationPrune: &v1.OperationPrune{
				Output: output,
			},
		}

		return nil
	})
}

func (t *PruneTask) Cancel(withStatus v1.OperationStatus) error {
	return nil
}

// synchronizedBuffer is used for collecting prune command's output
type synchronizedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (w *synchronizedBuffer) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.buf.Write(p)
}

func (w *synchronizedBuffer) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.buf.String()
}
