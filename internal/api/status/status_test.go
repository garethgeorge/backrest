package status

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/config"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/oplog/memstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const instance = "test-instance"

// fakeStore counts queries and can be made to fail or hang.
type fakeStore struct {
	*memstore.MemStore
	queries atomic.Int64
	err     error
	block   chan struct{}
}

func (s *fakeStore) Query(q oplog.Query, f func(*v1.Operation) error) error {
	s.queries.Add(1)
	if s.block != nil {
		<-s.block
	}
	if s.err != nil {
		return s.err
	}
	return s.MemStore.Query(q, f)
}

type testOp struct {
	plan     string
	instance string
	ago      time.Duration // start time, before now
	status   v1.OperationStatus
	backup   bool
}

func newHandler(t *testing.T, ops ...testOp) (*Handler, *fakeStore) {
	t.Helper()
	store := &fakeStore{MemStore: memstore.NewMemStore()}
	now := time.Now()
	for _, o := range ops {
		op := &v1.Operation{
			InstanceId:      o.instance,
			PlanId:          o.plan,
			RepoId:          "repo",
			RepoGuid:        "repo-guid",
			UnixTimeStartMs: now.Add(-o.ago).UnixMilli(),
			UnixTimeEndMs:   now.Add(-o.ago + 5*time.Minute).UnixMilli(),
			Status:          o.status,
		}
		if o.backup {
			op.Op = &v1.Operation_OperationBackup{OperationBackup: &v1.OperationBackup{}}
		} else {
			op.Op = &v1.Operation_OperationForget{OperationForget: &v1.OperationForget{}}
		}
		require.NoError(t, store.Add(op))
	}
	opLog, err := oplog.NewOpLog(store)
	require.NoError(t, err)

	cfg := &config.MemoryStore{Config: &v1.Config{
		Instance: instance,
		Plans:    []*v1.Plan{{Id: "plan-a"}, {Id: "plan-b"}, {Id: "plan-c"}},
	}}
	return NewHandler(true, cfg, opLog), store
}

func get(t *testing.T, h *Handler, url string) (int, Response, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
	var resp Response
	if rec.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	}
	return rec.Code, resp, rec.Body.String()
}

// Operations are added oldest first: the memstore orders by ID, the real store by start time.
var history = []testOp{
	{plan: "plan-a", instance: instance, ago: 40 * 24 * time.Hour, status: v1.OperationStatus_STATUS_SUCCESS, backup: true},
	{plan: "plan-a", instance: instance, ago: 48 * time.Hour, status: v1.OperationStatus_STATUS_SUCCESS, backup: true},
	{plan: "plan-a", instance: instance, ago: 3 * time.Hour, status: v1.OperationStatus_STATUS_WARNING, backup: true},
	{plan: "plan-a", instance: instance, ago: 2 * time.Hour, status: v1.OperationStatus_STATUS_SUCCESS, backup: true},
	{plan: "plan-a", instance: instance, ago: 90 * time.Minute, status: v1.OperationStatus_STATUS_SUCCESS, backup: false},
	{plan: "plan-a", instance: "other-instance", ago: 80 * time.Minute, status: v1.OperationStatus_STATUS_SUCCESS, backup: true},
	{plan: "plan-a", instance: instance, ago: 60 * time.Minute, status: v1.OperationStatus_STATUS_ERROR, backup: true},
	{plan: "plan-b", instance: instance, ago: 35 * 24 * time.Hour, status: v1.OperationStatus_STATUS_SUCCESS, backup: true},
	{plan: "plan-b", instance: instance, ago: 10 * time.Minute, status: v1.OperationStatus_STATUS_ERROR, backup: true},
}

func TestDisabled(t *testing.T) {
	h, store := newHandler(t, history...)
	h.enabled = false

	code, _, _ := get(t, h, "/statusz")
	assert.Equal(t, http.StatusNotFound, code)
	assert.Zero(t, store.queries.Load())
}

func TestAllPlans(t *testing.T) {
	h, _ := newHandler(t, history...)
	code, resp, _ := get(t, h, "/statusz")
	require.Equal(t, http.StatusOK, code)
	require.Len(t, resp.Plans, 3)

	// Last success: the backup that started 2 h ago and ended 5 min later. Not
	// counted: the warning, the forget, the other instance's backup, the
	// failures, and the success older than 30 days.
	a := resp.Plans["plan-a"]
	require.NotNil(t, a.SecondsSinceLastSuccess)
	assert.InDelta(t, (115 * time.Minute).Seconds(), *a.SecondsSinceLastSuccess, 5)
	assert.Equal(t, int64(2), a.SuccessesLast30Days)

	// Only success is older than 30 days: reported, but not counted.
	b := resp.Plans["plan-b"]
	require.NotNil(t, b.SecondsSinceLastSuccess)
	assert.InDelta(t, (35*24*time.Hour - 5*time.Minute).Seconds(), *b.SecondsSinceLastSuccess, 5)
	assert.Equal(t, int64(0), b.SuccessesLast30Days)

	// Never ran.
	c := resp.Plans["plan-c"]
	assert.Nil(t, c.SecondsSinceLastSuccess)
	assert.Equal(t, int64(0), c.SuccessesLast30Days)
}

func TestNeverSucceededIsNull(t *testing.T) {
	h, _ := newHandler(t)
	_, _, body := get(t, h, "/statusz?plan=plan-c")
	assert.JSONEq(t, `{"plans":{"plan-c":{"seconds_since_last_success":null,"successes_last_30_days":0}}}`, body)
}

func TestPlanFilter(t *testing.T) {
	h, _ := newHandler(t, history...)

	code, resp, _ := get(t, h, "/statusz?plan=plan-b")
	require.Equal(t, http.StatusOK, code)
	assert.Len(t, resp.Plans, 1)
	assert.Contains(t, resp.Plans, "plan-b")

	code, resp, _ = get(t, h, "/statusz?plan=plan-a&plan=plan-c")
	require.Equal(t, http.StatusOK, code)
	assert.Len(t, resp.Plans, 2)
	assert.Contains(t, resp.Plans, "plan-a")
	assert.Contains(t, resp.Plans, "plan-c")

	code, _, body := get(t, h, "/statusz?plan=plan-a&plan=no-such-plan")
	assert.Equal(t, http.StatusNotFound, code)
	assert.NotContains(t, body, "no-such-plan")
}

func TestQueryErrorIsNotLeaked(t *testing.T) {
	h, store := newHandler(t, history...)
	store.err = errors.New("secret detail /var/lib/backrest")

	code, _, body := get(t, h, "/statusz")
	assert.Equal(t, http.StatusServiceUnavailable, code)
	assert.JSONEq(t, `{"error":"status unavailable"}`, body)
}

func TestTimeoutRunsOneQueryAtATime(t *testing.T) {
	h, store := newHandler(t, history...)
	h.timeout = 50 * time.Millisecond
	store.block = make(chan struct{})

	start := time.Now()
	code, _, body := get(t, h, "/statusz")
	assert.Less(t, time.Since(start), time.Second)
	assert.Equal(t, http.StatusServiceUnavailable, code)
	assert.JSONEq(t, `{"error":"status unavailable"}`, body)

	// The hanging query is still running: a second request waits for it
	// instead of starting another one.
	code, _, _ = get(t, h, "/statusz")
	assert.Equal(t, http.StatusServiceUnavailable, code)
	assert.Equal(t, int64(1), store.queries.Load())

	// Once it returns, its result is used.
	close(store.block)
	h.timeout = time.Second
	code, resp, _ := get(t, h, "/statusz")
	assert.Equal(t, http.StatusOK, code)
	assert.Len(t, resp.Plans, 3)
}

func TestCache(t *testing.T) {
	h, store := newHandler(t, history...)

	get(t, h, "/statusz")
	get(t, h, "/statusz?plan=plan-a")
	assert.Equal(t, int64(3), store.queries.Load(), "one query per plan, then served from the cache")

	h.cacheTTL = 0
	get(t, h, "/statusz")
	assert.Equal(t, int64(6), store.queries.Load(), "an expired cache queries again")
}
