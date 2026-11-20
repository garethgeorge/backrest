// Package status serves /statusz: for each plan, how long ago its last
// successful backup finished and how many backups succeeded in the last 30
// days. The endpoint needs no authentication, so it is disabled unless
// switched on, and it reveals nothing beyond plan IDs and those two numbers.
package status

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/oplog"
	"go.uber.org/zap"
)

const (
	successWindow = 30 * 24 * time.Hour
	queryTimeout  = 10 * time.Second
	cacheTTL      = time.Minute
)

var errTimeout = errors.New("operation log query timed out")

type Response struct {
	Plans map[string]PlanStatus `json:"plans"`
}

type PlanStatus struct {
	// Null when the plan has no successful backup in the operation log.
	SecondsSinceLastSuccess *int64 `json:"seconds_since_last_success"`
	SuccessesLast30Days     int64  `json:"successes_last_30_days"`
}

// planStats holds absolute times, so a cached value still gives an exact
// "seconds since" at the moment of each request.
type planStats struct {
	lastSuccessMs int64 // 0 when there is none
	successes30d  int64
}

type computation struct {
	done  chan struct{} // closed once stats and err are set
	stats map[string]planStats
	err   error
}

type Handler struct {
	enabled  bool
	config   config.ConfigStore
	oplog    *oplog.OpLog
	timeout  time.Duration
	cacheTTL time.Duration

	mu       sync.Mutex
	cached   map[string]planStats
	cachedAt time.Time
	running  *computation
}

func NewHandler(enabled bool, cfg config.ConfigStore, opLog *oplog.OpLog) *Handler {
	return &Handler{
		enabled:  enabled,
		config:   cfg,
		oplog:    opLog,
		timeout:  queryTimeout,
		cacheTTL: cacheTTL,
	}
}

// ServeHTTP answers with all plans, or only those named by one or more
// ?plan=<id> parameters (404 if any of them is unknown).
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.enabled {
		http.NotFound(w, r)
		return
	}

	stats, err := h.stats()
	if err != nil {
		// The details go to the log only: the caller is not authenticated.
		zap.L().Error("statusz: computing plan status", zap.Error(err))
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "status unavailable"})
		return
	}

	planIDs := r.URL.Query()["plan"]
	if len(planIDs) == 0 {
		for id := range stats {
			planIDs = append(planIDs, id)
		}
	}

	now := time.Now()
	resp := Response{Plans: make(map[string]PlanStatus, len(planIDs))}
	for _, id := range planIDs {
		s, ok := stats[id]
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown plan"})
			return
		}
		status := PlanStatus{SuccessesLast30Days: s.successes30d}
		if s.lastSuccessMs != 0 {
			seconds := max(0, int64(now.Sub(time.UnixMilli(s.lastSuccessMs)).Seconds()))
			status.SecondsSinceLastSuccess = &seconds
		}
		resp.Plans[id] = status
	}
	writeJSON(w, http.StatusOK, resp)
}

// stats returns the per-plan numbers, computed at most once per cacheTTL.
// Only one computation runs at a time: requests arriving meanwhile wait for
// it instead of starting their own, so unauthenticated callers can't pile up
// queries on the database, not even while it hangs.
func (h *Handler) stats() (map[string]planStats, error) {
	h.mu.Lock()
	if h.cached != nil && time.Since(h.cachedAt) < h.cacheTTL {
		stats := h.cached
		h.mu.Unlock()
		return stats, nil
	}
	run := h.running
	if run == nil {
		run = &computation{done: make(chan struct{})}
		h.running = run
		go func() {
			run.stats, run.err = h.compute()
			h.mu.Lock()
			if run.err == nil {
				h.cached, h.cachedAt = run.stats, time.Now()
			}
			h.running = nil
			h.mu.Unlock()
			close(run.done)
		}()
	}
	h.mu.Unlock()

	// OpLog.Query takes no context, so the timeout can only stop the waiting,
	// not the query itself.
	select {
	case <-run.done:
		return run.stats, run.err
	case <-time.After(h.timeout):
		return nil, errTimeout
	}
}

func (h *Handler) compute() (map[string]planStats, error) {
	cfg, err := h.config.Get()
	if err != nil {
		return nil, fmt.Errorf("get config: %w", err)
	}

	cutoffMs := time.Now().Add(-successWindow).UnixMilli()
	stats := make(map[string]planStats, len(cfg.Plans))
	for _, plan := range cfg.Plans {
		var s planStats
		// Newest first. Stops once past the window with a success found; a plan
		// without any success is read to the end of its history.
		q := oplog.Query{}.SetInstanceID(cfg.Instance).SetPlanID(plan.Id).SetReversed(true)
		err := h.oplog.Query(q, func(op *v1.Operation) error {
			if op.UnixTimeStartMs < cutoffMs && s.lastSuccessMs != 0 {
				return oplog.ErrStopIteration
			}
			if op.GetOperationBackup() == nil || op.Status != v1.OperationStatus_STATUS_SUCCESS {
				return nil
			}
			if s.lastSuccessMs == 0 {
				s.lastSuccessMs = max(op.UnixTimeEndMs, op.UnixTimeStartMs)
			}
			if op.UnixTimeStartMs >= cutoffMs {
				s.successes30d++
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("query operations of plan %q: %w", plan.Id, err)
		}
		stats[plan.Id] = s
	}
	return stats, nil
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(body)
}
