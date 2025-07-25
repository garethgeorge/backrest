package sqlitestore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/cryptoutil"
	"github.com/garethgeorge/backrest/internal/ioutil"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/protoutil"
	lru "github.com/hashicorp/golang-lru/v2"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/gofrs/flock"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

var ErrLocked = errors.New("sqlite db is locked")

type SqliteStore struct {
	dbpool    *sqlitex.Pool
	lastIDVal atomic.Int64
	dblock    *flock.Flock

	ogidCache *lru.TwoQueueCache[opGroupInfo, int64]

	tidyGroupsOnce sync.Once
}

var _ oplog.OpStore = (*SqliteStore)(nil)

func NewSqliteStore(db string) (*SqliteStore, error) {
	if err := os.MkdirAll(filepath.Dir(db), 0700); err != nil {
		return nil, fmt.Errorf("create sqlite db directory: %v", err)
	}
	dbpool, err := sqlitex.NewPool(db, sqlitex.PoolOptions{
		PoolSize: 16,
		Flags:    sqlite.OpenReadWrite | sqlite.OpenCreate | sqlite.OpenWAL,
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite pool: %v", err)
	}
	ogidCache, _ := lru.New2Q[opGroupInfo, int64](128)
	store := &SqliteStore{
		dbpool:    dbpool,
		dblock:    flock.New(db + ".lock"),
		ogidCache: ogidCache,
	}
	if locked, err := store.dblock.TryLock(); err != nil {
		return nil, fmt.Errorf("lock sqlite db: %v", err)
	} else if !locked {
		return nil, ErrLocked
	}
	if err := store.init(); err != nil {
		return nil, err
	}
	return store, nil
}

func NewMemorySqliteStore() (*SqliteStore, error) {
	dbpool, err := sqlitex.NewPool("file:"+cryptoutil.MustRandomID(64)+"?mode=memory&cache=shared", sqlitex.PoolOptions{
		PoolSize: 16,
		Flags:    sqlite.OpenReadWrite | sqlite.OpenCreate | sqlite.OpenURI,
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite pool: %v", err)
	}
	ogidCache, _ := lru.New2Q[opGroupInfo, int64](128)
	store := &SqliteStore{
		dbpool:    dbpool,
		ogidCache: ogidCache,
	}
	if err := store.init(); err != nil {
		return nil, err
	}
	return store, nil
}

func (m *SqliteStore) Close() error {
	if m.dblock != nil {
		if err := m.dblock.Unlock(); err != nil {
			return fmt.Errorf("unlock sqlite db: %v", err)
		}
	}
	return m.dbpool.Close()
}

func (m *SqliteStore) init() error {
	conn, err := m.dbpool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("init sqlite: %v", err)
	}
	defer m.dbpool.Put(conn)

	if err := applySqliteMigrations(m, conn); err != nil {
		return err
	}

	if err := sqlitex.ExecuteTransient(conn, "SELECT operations.id FROM operations ORDER BY operations.id DESC LIMIT 1", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			m.lastIDVal.Store(stmt.GetInt64("id"))
			return nil
		},
	}); err != nil {
		return fmt.Errorf("init sqlite: %v", err)
	}

	return nil
}

func (m *SqliteStore) Version() (int64, error) {
	conn, err := m.dbpool.Take(context.Background())
	if err != nil {
		return 0, fmt.Errorf("get version: %v", err)
	}
	defer m.dbpool.Put(conn)

	var version int64
	if err := sqlitex.ExecuteTransient(conn, "SELECT version FROM system_info", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			version = stmt.GetInt64("version")
			return nil
		},
	}); err != nil {
		return 0, fmt.Errorf("get version: %v", err)
	}
	return version, nil
}

func (m *SqliteStore) SetVersion(version int64) error {
	conn, err := m.dbpool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("set version: %v", err)
	}
	defer m.dbpool.Put(conn)

	if err := sqlitex.ExecuteTransient(conn, "UPDATE system_info SET version = ?", &sqlitex.ExecOptions{
		Args: []any{version},
	}); err != nil {
		return fmt.Errorf("set version: %v", err)
	}
	return nil
}

func (m *SqliteStore) buildQueryWhereClause(q oplog.Query, includeSelectClauses bool) (string, []any) {
	query := make([]string, 0, 8)
	args := make([]any, 0, 8)

	query = append(query, " 1=1 ")

	if q.PlanID != nil {
		query = append(query, " AND operation_groups.plan_id = ?")
		args = append(args, *q.PlanID)
	}
	if q.RepoGUID != nil {
		query = append(query, " AND operation_groups.repo_guid = ?")
		args = append(args, *q.RepoGUID)
	}
	if q.DeprecatedRepoID != nil {
		query = append(query, " AND operation_groups.repo_id = ?")
		args = append(args, *q.DeprecatedRepoID)
	}
	if q.InstanceID != nil {
		query = append(query, " AND operation_groups.instance_id = ?")
		args = append(args, *q.InstanceID)
	}
	if q.OriginalInstanceKeyid != nil {
		query = append(query, " AND operation_groups.original_instance_keyid = ?")
		args = append(args, *q.OriginalInstanceKeyid)
	}
	if q.SnapshotID != nil {
		query = append(query, " AND operations.snapshot_id = ?")
		args = append(args, *q.SnapshotID)
	}
	if q.FlowID != nil {
		query = append(query, " AND operations.flow_id = ?")
		args = append(args, *q.FlowID)
	}
	if q.OriginalID != nil {
		query = append(query, " AND operations.original_id = ?")
		args = append(args, *q.OriginalID)
	}
	if q.OriginalFlowID != nil {
		query = append(query, " AND operations.original_flow_id = ?")
		args = append(args, *q.OriginalFlowID)
	}
	if q.OpIDs != nil {
		query = append(query, " AND operations.id IN (")
		for i, id := range q.OpIDs {
			if i > 0 {
				query = append(query, ",")
			}
			query = append(query, "?")
			args = append(args, id)
		}
		query = append(query, ")")
	}

	if includeSelectClauses {
		if q.Reversed {
			query = append(query, " ORDER BY operations.start_time_ms DESC, operations.id DESC")
		} else {
			query = append(query, " ORDER BY operations.start_time_ms ASC, operations.id ASC")
		}

		if q.Limit > 0 {
			query = append(query, " LIMIT ?")
			args = append(args, q.Limit)
		} else {
			query = append(query, " LIMIT -1")
		}

		if q.Offset > 0 {
			query = append(query, " OFFSET ?")
			args = append(args, q.Offset)
		}
	}

	return strings.Join(query, "")[1:], args
}

func (m *SqliteStore) Query(q oplog.Query, f func(*v1.Operation) error) error {
	conn, err := m.dbpool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("query: %v", err)
	}
	defer m.dbpool.Put(conn)

	where, args := m.buildQueryWhereClause(q, true)
	if err := sqlitex.ExecuteTransient(conn, "SELECT operations.operation FROM operations JOIN operation_groups ON operations.ogid = operation_groups.ogid WHERE "+where, &sqlitex.ExecOptions{
		Args: args,
		ResultFunc: func(stmt *sqlite.Stmt) error {
			opBytes := make([]byte, stmt.ColumnLen(0))
			n := stmt.ColumnBytes(0, opBytes)
			opBytes = opBytes[:n]

			var op v1.Operation
			if err := proto.Unmarshal(opBytes, &op); err != nil {
				return fmt.Errorf("unmarshal operation bytes: %v", err)
			}
			return f(&op)
		},
	}); err != nil && !errors.Is(err, oplog.ErrStopIteration) {
		return err
	}
	return nil
}

func (m *SqliteStore) QueryMetadata(q oplog.Query, f func(oplog.OpMetadata) error) error {
	conn, err := m.dbpool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("query metadata: %v", err)
	}
	defer m.dbpool.Put(conn)

	where, args := m.buildQueryWhereClause(q, false)
	if err := sqlitex.ExecuteTransient(conn, "SELECT operations.id, operations.modno, operations.original_id, operations.flow_id, operations.original_flow_id, operations.status FROM operations JOIN operation_groups ON operations.ogid = operation_groups.ogid WHERE "+where, &sqlitex.ExecOptions{
		Args: args,
		ResultFunc: func(stmt *sqlite.Stmt) error {
			return f(oplog.OpMetadata{
				ID:             stmt.ColumnInt64(0),
				Modno:          stmt.ColumnInt64(1),
				OriginalID:     stmt.ColumnInt64(2),
				FlowID:         stmt.ColumnInt64(3),
				OriginalFlowID: stmt.ColumnInt64(4),
				Status:         v1.OperationStatus(stmt.ColumnInt64(5)),
			})
		},
	}); err != nil && !errors.Is(err, oplog.ErrStopIteration) {
		return err
	}
	return nil
}

// tidyGroups deletes operation groups that are no longer referenced, it takes an int64 specifying the maximum group ID to consider.
// this allows ignoring newly created groups that may not yet be referenced.
func (m *SqliteStore) tidyGroups(conn *sqlite.Conn, eligibleIDsBelow int64) {
	err := sqlitex.ExecuteTransient(conn, "DELETE FROM operation_groups WHERE ogid NOT IN (SELECT DISTINCT ogid FROM operations WHERE ogid < ?)", &sqlitex.ExecOptions{
		Args: []any{eligibleIDsBelow},
	})
	if err != nil {
		zap.S().Warnf("tidy groups: %v", err)
	}
}

func (m *SqliteStore) findOrCreateGroup(conn *sqlite.Conn, op *v1.Operation) (ogid int64, err error) {
	ogidKey := groupInfoForOp(op)
	if cachedOGID, ok := m.ogidCache.Get(ogidKey); ok {
		return cachedOGID, nil
	}

	var found bool
	if err := sqlitex.Execute(conn, "SELECT ogid FROM operation_groups WHERE instance_id = ? AND original_instance_keyid = ? AND repo_id = ? AND plan_id = ? AND repo_guid = ? LIMIT 1", &sqlitex.ExecOptions{
		Args: []any{op.InstanceId, op.OriginalInstanceKeyid, op.RepoId, op.PlanId, op.RepoGuid},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			ogid = stmt.ColumnInt64(0)
			found = true
			return nil
		},
	}); err != nil {
		return 0, fmt.Errorf("find operation group: %v", err)
	}

	if !found {
		if err := sqlitex.Execute(conn, "INSERT INTO operation_groups (instance_id, original_instance_keyid, repo_id, plan_id, repo_guid) VALUES (?, ?, ?, ?, ?) RETURNING ogid", &sqlitex.ExecOptions{
			Args: []any{op.InstanceId, op.OriginalInstanceKeyid, op.RepoId, op.PlanId, op.RepoGuid},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				ogid = stmt.ColumnInt64(0)
				return nil
			},
		}); err != nil {
			return 0, fmt.Errorf("insert operation group: %v", err)
		}
	}

	m.ogidCache.Add(ogidKey, ogid)
	return ogid, nil
}

func (m *SqliteStore) Transform(q oplog.Query, f func(*v1.Operation) (*v1.Operation, error)) error {
	conn, err := m.dbpool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("transform: %v", err)
	}
	defer m.dbpool.Put(conn)

	where, args := m.buildQueryWhereClause(q, true)
	return withImmediateSqliteTransaction(conn, func() error {
		return sqlitex.ExecuteTransient(conn, "SELECT operations.operation FROM operations JOIN operation_groups ON operations.ogid = operation_groups.ogid WHERE "+where, &sqlitex.ExecOptions{
			Args: args,
			ResultFunc: func(stmt *sqlite.Stmt) error {
				opBytes := make([]byte, stmt.ColumnLen(0))
				n := stmt.ColumnBytes(0, opBytes)
				opBytes = opBytes[:n]

				var op v1.Operation
				if err := proto.Unmarshal(opBytes, &op); err != nil {
					return fmt.Errorf("unmarshal operation bytes: %v", err)
				}

				newOp, err := f(&op)
				if err != nil {
					return err
				} else if newOp == nil {
					return nil
				}

				newOp.Modno = oplog.NewRandomModno(op.Modno)

				return m.updateInternal(conn, newOp)
			},
		})
	})
}

func (m *SqliteStore) addInternal(conn *sqlite.Conn, op ...*v1.Operation) error {
	for _, o := range op {
		ogid, err := m.findOrCreateGroup(conn, o)
		if err != nil {
			return fmt.Errorf("find ogid: %v", err)
		}

		query := `INSERT INTO operations 
			(id, ogid, original_id, original_flow_id, modno, flow_id, start_time_ms, status, snapshot_id, operation)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		bytes, err := proto.Marshal(o)
		if err != nil {
			return fmt.Errorf("marshal operation: %v", err)
		}

		if err := sqlitex.Execute(conn, query, &sqlitex.ExecOptions{
			Args: []any{
				o.Id, ogid, o.OriginalId, o.OriginalFlowId, o.Modno, o.FlowId,
				o.UnixTimeStartMs, int64(o.Status), o.SnapshotId, bytes,
			},
		}); err != nil {
			if sqlite.ErrCode(err) == sqlite.ResultConstraintUnique {
				return fmt.Errorf("operation already exists %v: %w", o.Id, oplog.ErrExist)
			}
			return fmt.Errorf("add operation: %v", err)
		}
	}
	return nil
}

func (m *SqliteStore) Add(op ...*v1.Operation) error {
	conn, err := m.dbpool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("add operation: %v", err)
	}
	defer m.dbpool.Put(conn)

	return withImmediateSqliteTransaction(conn, func() error {
		for _, o := range op {
			o.Id = m.lastIDVal.Add(1)
			if o.FlowId == 0 {
				o.FlowId = o.Id
			}
			if err := protoutil.ValidateOperation(o); err != nil {
				return err
			}
		}

		return m.addInternal(conn, op...)
	})
}

func (m *SqliteStore) Update(op ...*v1.Operation) error {
	conn, err := m.dbpool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("update operation: %v", err)
	}
	defer m.dbpool.Put(conn)

	return withImmediateSqliteTransaction(conn, func() error {
		return m.updateInternal(conn, op...)
	})
}

func (m *SqliteStore) updateInternal(conn *sqlite.Conn, op ...*v1.Operation) error {
	for _, o := range op {
		if err := protoutil.ValidateOperation(o); err != nil {
			return err
		}
		bytes, err := proto.Marshal(o)
		if err != nil {
			return fmt.Errorf("marshal operation: %v", err)
		}

		ogid, err := m.findOrCreateGroup(conn, o)
		if err != nil {
			return fmt.Errorf("find ogid: %v", err)
		}

		if err := sqlitex.Execute(conn, "UPDATE operations SET operation = ?, ogid = ?, start_time_ms = ?, flow_id = ?, snapshot_id = ?, modno = ?, original_id = ?, original_flow_id = ?, status = ? WHERE id = ?", &sqlitex.ExecOptions{
			Args: []any{bytes, ogid, o.UnixTimeStartMs, o.FlowId, o.SnapshotId, o.Modno, o.OriginalId, o.OriginalFlowId, int64(o.Status), o.Id},
		}); err != nil {
			return fmt.Errorf("update operation: %v", err)
		}
		if conn.Changes() == 0 {
			return fmt.Errorf("couldn't update %d: %w", o.Id, oplog.ErrNotExist)
		}
	}
	return nil
}

func (m *SqliteStore) Get(opID int64) (*v1.Operation, error) {
	conn, err := m.dbpool.Take(context.Background())
	if err != nil {
		return nil, fmt.Errorf("get operation: %v", err)
	}
	defer m.dbpool.Put(conn)

	var found bool
	var opBytes []byte
	if err := sqlitex.Execute(conn, "SELECT operation FROM operations WHERE id = ?", &sqlitex.ExecOptions{
		Args: []any{opID},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			found = true
			opBytes = make([]byte, stmt.ColumnLen(0))
			n := stmt.GetBytes("operation", opBytes)
			opBytes = opBytes[:n]
			return nil
		},
	}); err != nil {
		return nil, fmt.Errorf("get operation: %v", err)
	}
	if !found {
		return nil, oplog.ErrNotExist
	}

	var op v1.Operation
	if err := proto.Unmarshal(opBytes, &op); err != nil {
		return nil, fmt.Errorf("unmarshal operation bytes: %v", err)
	}

	return &op, nil
}

func (m *SqliteStore) Delete(opID ...int64) ([]*v1.Operation, error) {
	conn, err := m.dbpool.Take(context.Background())
	if err != nil {
		return nil, fmt.Errorf("delete operation: %v", err)
	}
	defer m.dbpool.Put(conn)

	ops := make([]*v1.Operation, 0, len(opID))
	return ops, withImmediateSqliteTransaction(conn, func() error {
		for _, batch := range ioutil.Batchify(opID, ioutil.DefaultBatchSize) {
			// Optimize for the case of 1 element or batch size elements (which will be common)
			useTransient := len(batch) != ioutil.DefaultBatchSize || len(batch) == 1
			batchOps, err := m.deleteHelper(conn, useTransient, batch...)
			if err != nil {
				return err
			}
			ops = append(ops, batchOps...)
		}
		return nil
	})
}

func (m *SqliteStore) deleteHelper(conn *sqlite.Conn, transient bool, opID ...int64) ([]*v1.Operation, error) {
	// fetch all the operations we're about to delete
	predicate := []string{"operations.id IN ("}
	args := []any{}
	for i, id := range opID {
		if i > 0 {
			predicate = append(predicate, ",")
		}
		predicate = append(predicate, "?")
		args = append(args, id)
	}
	predicate = append(predicate, ")")
	predicateStr := strings.Join(predicate, "")

	var ops []*v1.Operation
	if err := sqlitex.ExecuteTransient(conn, "SELECT operations.operation FROM operations JOIN operation_groups ON operations.ogid = operation_groups.ogid WHERE "+predicateStr, &sqlitex.ExecOptions{
		Args: args,
		ResultFunc: func(stmt *sqlite.Stmt) error {
			opBytes := make([]byte, stmt.ColumnLen(0))
			n := stmt.GetBytes("operation", opBytes)
			opBytes = opBytes[:n]

			var op v1.Operation
			if err := proto.Unmarshal(opBytes, &op); err != nil {
				return fmt.Errorf("unmarshal operation bytes: %v", err)
			}
			ops = append(ops, &op)
			return nil
		},
	}); err != nil {
		return nil, fmt.Errorf("load operations for delete: %v", err)
	}

	if len(ops) != len(opID) {
		return nil, fmt.Errorf("couldn't find all operations to delete: %w", oplog.ErrNotExist)
	}

	// Delete the operations
	execFunc := sqlitex.Execute
	if transient {
		execFunc = sqlitex.ExecuteTransient
	}
	if err := execFunc(conn, "DELETE FROM operations WHERE "+predicateStr, &sqlitex.ExecOptions{
		Args: args,
	}); err != nil {
		return nil, fmt.Errorf("delete operations: %v", err)
	}

	return ops, nil
}

func (m *SqliteStore) ResetForTest(t *testing.T) error {
	conn, err := m.dbpool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("reset for test: %v", err)
	}
	defer m.dbpool.Put(conn)

	if err := sqlitex.Execute(conn, "DELETE FROM operations", &sqlitex.ExecOptions{}); err != nil {
		return fmt.Errorf("reset for test: %v", err)
	}
	m.lastIDVal.Store(0)
	return nil
}

type opGroupInfo struct {
	repo          string
	repoGuid      string
	plan          string
	inst          string
	origInstKeyid string
}

func groupInfoForOp(op *v1.Operation) opGroupInfo {
	return opGroupInfo{
		repo:          op.RepoId,
		repoGuid:      op.RepoGuid,
		plan:          op.PlanId,
		inst:          op.InstanceId,
		origInstKeyid: op.OriginalInstanceKeyid,
	}
}
