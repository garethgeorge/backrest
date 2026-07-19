package api

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/env"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/protoutil"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	// summaryHistoryDays is the dashboard window: today plus the prior N-1 days.
	summaryHistoryDays = 30

	// summaryChartBackups caps how many recent backups each summary's chart includes.
	summaryChartBackups = 60

	// overdueGraceFactor pads a schedule's nominal period before a day is flagged
	// overdue, tolerating scheduler jitter and backup run time.
	overdueGraceFactor = 1.25
)

func (s *BackrestHandler) GetSummaryDashboard(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[v1.SummaryDashboardResponse], error) {
	cfg, err := s.config.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	now := time.Now()
	cutoffMidnight := localMidnight(now).AddDate(0, 0, -(summaryHistoryDays - 1))

	// One accumulator per repo and per plan; each operation in the pass below is
	// dispatched to at most one of each.
	repoAccs := make(map[string]*summaryAcc) // keyed by repo GUID
	planAccs := make(map[string]*summaryAcc) // keyed by plan ID
	for _, repo := range cfg.Repos {
		repoAccs[repo.GetGuid()] = newSummaryAcc(cutoffMidnight)
	}
	for _, plan := range cfg.Plans {
		planAccs[plan.Id] = newSummaryAcc(cutoffMidnight)
	}
	// Walk every operation for this instance, newest to oldest, dispatching backup
	// and indexed-snapshot evidence to its plan's and its repo's accumulator.
	if err := s.oplog.Query(oplog.Query{}.SetInstanceID(cfg.Instance).SetReversed(true), func(op *v1.Operation) error {
		if backupOp := op.GetOperationBackup(); backupOp != nil {
			if acc, ok := planAccs[op.PlanId]; ok {
				acc.observeBackup(op, backupOp)
			}
			if acc, ok := repoAccs[op.RepoGuid]; ok {
				acc.observeBackup(op, backupOp)
			}
		}
		if indexOp := op.GetOperationIndexSnapshot(); indexOp != nil {
			if acc, ok := planAccs[op.PlanId]; ok {
				acc.observeIndexedSnapshot(indexOp)
			}
			if acc, ok := repoAccs[op.RepoGuid]; ok {
				acc.observeIndexedSnapshot(indexOp)
			}
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to query operations: %w", err)
	}

	response := &v1.SummaryDashboardResponse{
		ConfigPath: env.ConfigFilePath(),
		DataPath:   env.DataDir(),
	}
	for _, repo := range cfg.Repos {
		response.RepoSummaries = append(response.RepoSummaries,
			repoAccs[repo.GetGuid()].finalize(repo.Id, now, repoAllowedStaleness(cfg, repo.Id, now)))
	}
	for _, plan := range cfg.Plans {
		response.PlanSummaries = append(response.PlanSummaries,
			planAccs[plan.Id].finalize(plan.Id, now, allowedStaleness(plan.Schedule, now)))
	}

	return connect.NewResponse(response), nil
}

// allowedStaleness converts a schedule to the maximum acceptable gap between
// OK backups, or 0 when there is no expectation (disabled or unparseable).
func allowedStaleness(sched *v1.Schedule, now time.Time) time.Duration {
	period, err := protoutil.NominalPeriod(sched, now)
	if err != nil {
		if !errors.Is(err, protoutil.ErrScheduleDisabled) {
			zap.S().Warnf("summary dashboard: nominal period: %v", err)
		}
		return 0
	}
	return time.Duration(float64(period) * overdueGraceFactor)
}

// repoAllowedStaleness is the widest allowed staleness among plans targeting
// the repo: a repo day is overdue only when even the slowest plan should have run.
func repoAllowedStaleness(cfg *v1.Config, repoID string, now time.Time) time.Duration {
	var widest time.Duration
	for _, plan := range cfg.Plans {
		if plan.Repo != repoID {
			continue
		}
		if a := allowedStaleness(plan.Schedule, now); a > widest {
			widest = a
		}
	}
	return widest
}

// localMidnight truncates t to midnight in its own location.
func localMidnight(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// summaryDayAcc accumulates per-day backup stats for the dashboard history strip.
type summaryDayAcc struct {
	bytesAdded   int64
	bytesScanned int64
	statusCounts map[v1.OperationStatus]int64
}

type indexedSnapshotEvidence struct {
	startTime time.Time
	summary   *v1.SnapshotSummary
}

// summaryAcc accumulates the backup operations for one plan or repo, observed
// newest to oldest, into a dashboard summary.
type summaryAcc struct {
	cutoffMidnight time.Time

	backupsExamined  int64
	bytesScanned30   int64
	bytesAdded30     int64
	backupsFailed30  int64
	backupsSuccess30 int64
	backupsWarning30 int64
	nextBackupTime   int64
	protectedBytes   int64
	backupChart      *v1.SummaryDashboardResponse_BackupChart

	// Per-day accumulators keyed by the day's local-midnight unix millis. oldestDay
	// tracks the earliest in-window day with a backup; reachedCutoff means backups
	// exist beyond the window, so the history strip spans the full window.
	perDay        map[int64]*summaryDayAcc
	oldestDay     time.Time
	reachedCutoff bool

	// Times of in-window OK backups (success or warning, not a dry run), which
	// reset the staleness clock, plus the most recent OK backup before the window.
	okBackupDates            []time.Time
	lastOkBackupBeforeWindow time.Time

	// Indexed snapshots are durable evidence of a successful backup. They fill
	// history gaps when the corresponding backup operation has been collected.
	backupSnapshotIDs   map[string]struct{}
	indexedSnapshotByID map[string]indexedSnapshotEvidence
}

func newSummaryAcc(cutoffMidnight time.Time) *summaryAcc {
	return &summaryAcc{
		cutoffMidnight:      cutoffMidnight,
		backupChart:         &v1.SummaryDashboardResponse_BackupChart{},
		perDay:              make(map[int64]*summaryDayAcc),
		backupSnapshotIDs:   make(map[string]struct{}),
		indexedSnapshotByID: make(map[string]indexedSnapshotEvidence),
	}
}

func (a *summaryAcc) observeBackup(op *v1.Operation, backupOp *v1.OperationBackup) {
	if op.SnapshotId != "" {
		a.backupSnapshotIDs[op.SnapshotId] = struct{}{}
	}
	startTime := time.UnixMilli(op.UnixTimeStartMs)
	opMidnight := localMidnight(startTime)
	// Dry runs don't reset the staleness clock, matching the scheduler's view.
	isOkBackup := (op.Status == v1.OperationStatus_STATUS_SUCCESS ||
		op.Status == v1.OperationStatus_STATUS_WARNING) && !backupOp.DryRun

	// Backups older than the window only contribute the staleness anchor; walking
	// newest-first, the first OK backup seen here is the most recent.
	if opMidnight.Before(a.cutoffMidnight) {
		a.reachedCutoff = true
		if isOkBackup && a.lastOkBackupBeforeWindow.IsZero() {
			a.lastOkBackupBeforeWindow = startTime
		}
		return
	}
	if op.GetStatus() == v1.OperationStatus_STATUS_PENDING {
		a.nextBackupTime = op.UnixTimeStartMs
		return
	}
	a.backupsExamined++

	switch op.Status {
	case v1.OperationStatus_STATUS_SUCCESS:
		a.backupsSuccess30++
	case v1.OperationStatus_STATUS_ERROR:
		a.backupsFailed30++
	case v1.OperationStatus_STATUS_WARNING:
		a.backupsWarning30++
	}

	if isOkBackup {
		a.okBackupDates = append(a.okBackupDates, startTime)
	}

	summary := backupOp.GetLastStatus().GetSummary()
	if summary != nil {
		a.bytesScanned30 += summary.TotalBytesProcessed
		a.bytesAdded30 += summary.DataAdded
	}

	// protected_bytes: the most recent (first seen) good backup's total size.
	if a.protectedBytes == 0 && summary != nil && isOkBackup {
		a.protectedBytes = summary.TotalBytesProcessed
	}

	// Update the per-day aggregate for this backup's day.
	dayMs := opMidnight.UnixMilli()
	acc := a.perDay[dayMs]
	if acc == nil {
		acc = &summaryDayAcc{statusCounts: make(map[v1.OperationStatus]int64)}
		a.perDay[dayMs] = acc
	}
	acc.statusCounts[op.Status]++
	if summary != nil {
		acc.bytesAdded += summary.DataAdded
		acc.bytesScanned += summary.TotalBytesProcessed
	}
	if a.oldestDay.IsZero() || opMidnight.Before(a.oldestDay) {
		a.oldestDay = opMidnight
	}

	if len(a.backupChart.TimestampMs) < summaryChartBackups {
		duration := op.UnixTimeEndMs - op.UnixTimeStartMs
		if duration <= 1000 {
			duration = 1000
		}

		a.backupChart.FlowId = append(a.backupChart.FlowId, op.FlowId)
		a.backupChart.TimestampMs = append(a.backupChart.TimestampMs, op.UnixTimeStartMs)
		a.backupChart.DurationMs = append(a.backupChart.DurationMs, duration)
		a.backupChart.Status = append(a.backupChart.Status, op.Status)
		a.backupChart.BytesAdded = append(a.backupChart.BytesAdded, summary.GetDataAdded())
	}
}

func (a *summaryAcc) observeIndexedSnapshot(indexOp *v1.OperationIndexSnapshot) {
	if indexOp.Forgot {
		return
	}
	snapshot := indexOp.Snapshot
	if snapshot == nil || snapshot.Id == "" {
		return
	}
	a.indexedSnapshotByID[snapshot.Id] = indexedSnapshotEvidence{
		startTime: time.UnixMilli(snapshot.UnixTimeMs),
		summary:   snapshot.Summary,
	}
}

func (a *summaryAcc) applyIndexedSnapshotFallbacks() {
	for snapshotID, evidence := range a.indexedSnapshotByID {
		if _, ok := a.backupSnapshotIDs[snapshotID]; ok {
			continue
		}

		opMidnight := localMidnight(evidence.startTime)
		if opMidnight.Before(a.cutoffMidnight) {
			a.reachedCutoff = true
			if a.lastOkBackupBeforeWindow.IsZero() || evidence.startTime.After(a.lastOkBackupBeforeWindow) {
				a.lastOkBackupBeforeWindow = evidence.startTime
			}
			continue
		}

		a.okBackupDates = append(a.okBackupDates, evidence.startTime)
		dayMs := opMidnight.UnixMilli()
		acc := a.perDay[dayMs]
		if acc == nil {
			acc = &summaryDayAcc{statusCounts: make(map[v1.OperationStatus]int64)}
			a.perDay[dayMs] = acc
		}
		acc.statusCounts[v1.OperationStatus_STATUS_SUCCESS]++
		if evidence.summary != nil {
			acc.bytesAdded += evidence.summary.DataAdded
			acc.bytesScanned += evidence.summary.TotalBytesProcessed
		}
		if a.oldestDay.IsZero() || opMidnight.Before(a.oldestDay) {
			a.oldestDay = opMidnight
		}
	}
}

// finalize builds the summary proto. allowedStaleness > 0 enables overdue
// detection on the day history.
func (a *summaryAcc) finalize(id string, now time.Time, allowedStaleness time.Duration) *v1.SummaryDashboardResponse_Summary {
	a.applyIndexedSnapshotFallbacks()
	todayMidnight := localMidnight(now)

	backupsExamined := a.backupsExamined
	if backupsExamined == 0 {
		backupsExamined = 1 // prevent division by zero for avg calculations
	}

	// OK backups ascending: these reset the staleness clock. The pre-window
	// anchor counts only when the window has no OK backup of its own, so a fully
	// stalled plan still flags every day, while an overdue stretch that ended
	// before the window's first OK backup is resolved history and stays muted.
	okBackupDates := a.okBackupDates
	if len(okBackupDates) == 0 && !a.lastOkBackupBeforeWindow.IsZero() {
		okBackupDates = append(okBackupDates, a.lastOkBackupBeforeWindow)
	}
	slices.SortFunc(okBackupDates, func(x, y time.Time) int { return x.Compare(y) })

	// Flatten the per-day map into buckets ordered oldest-first, one per consecutive day
	// from the oldest active day (or the window start, if older backups exist) through
	// today; absent days get empty buckets. The client matches buckets to days by their
	// distance from the newest bucket, and renders days before the span as "before start".
	start := todayMidnight
	if a.reachedCutoff {
		start = a.cutoffMidnight
	} else if !a.oldestDay.IsZero() {
		start = a.oldestDay
	}
	var history []*v1.SummaryDashboardResponse_DayStatusBucket
	var lastOkBackup time.Time
	nextOkBackup := 0
	for day := start; !day.After(todayMidnight); day = day.AddDate(0, 0, 1) {
		bucket := &v1.SummaryDashboardResponse_DayStatusBucket{
			TimestampMs: day.UnixMilli(),
		}
		if acc := a.perDay[day.UnixMilli()]; acc != nil {
			bucket.BytesAdded = acc.bytesAdded
			bucket.BytesScanned = acc.bytesScanned
			for status, count := range acc.statusCounts {
				bucket.StatusCounts = append(bucket.StatusCounts, &v1.SummaryDashboardResponse_StatusAndCount{
					Status: status,
					Count:  count,
				})
			}
		}

		// A day is overdue when, at its close (clamped to now for today), the newest
		// OK backup is older than the allowed staleness AND the day had no OK backup
		// of its own. Days before the first OK backup ever are exempt: the scheduler
		// anchors to the first real run. Requiring the day itself to be empty keeps
		// sub-daily schedules honest: an hourly plan whose last run was at noon still
		// backed the day up even though midnight is well past the allowed gap, so the
		// day must render as backed-up, not overdue.
		if allowedStaleness > 0 {
			checkpoint := day.AddDate(0, 0, 1)
			if checkpoint.After(now) {
				checkpoint = now
			}
			for nextOkBackup < len(okBackupDates) && !okBackupDates[nextOkBackup].After(checkpoint) {
				lastOkBackup = okBackupDates[nextOkBackup]
				nextOkBackup++
			}
			// lastOkBackup <= checkpoint <= day+1, so lastOkBackup.Before(day) is
			// exactly "the newest OK backup landed on an earlier day" — this day had none.
			bucket.Overdue = !lastOkBackup.IsZero() &&
				lastOkBackup.Before(day) &&
				checkpoint.Sub(lastOkBackup) > allowedStaleness
		}

		history = append(history, bucket)
	}

	return &v1.SummaryDashboardResponse_Summary{
		Id:                        id,
		BytesScannedLast_30Days:   a.bytesScanned30,
		BytesAddedLast_30Days:     a.bytesAdded30,
		BackupsFailed_30Days:      a.backupsFailed30,
		BackupsWarningLast_30Days: a.backupsWarning30,
		BackupsSuccessLast_30Days: a.backupsSuccess30,
		BytesScannedAvg:           a.bytesScanned30 / backupsExamined,
		BytesAddedAvg:             a.bytesAdded30 / backupsExamined,
		NextBackupTimeMs:          a.nextBackupTime,
		RecentBackups:             a.backupChart,
		ProtectedBytes:            a.protectedBytes,
		HistoryLast_30Days:        history,
	}
}
