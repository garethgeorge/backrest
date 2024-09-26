package sqlitestore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/protoutil"
	"google.golang.org/protobuf/proto"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type SqliteStore struct {
	dbpool    *sqlitex.Pool
	nextIDVal atomic.Int64
}

var _ oplog.OpStore = (*SqliteStore)(nil)

func NewSqliteStore(db string) (*SqliteStore, error) {
	dbpool, err := sqlitex.NewPool(db, sqlitex.PoolOptions{
		PoolSize: 16,
		Flags:    sqlite.OpenReadWrite | sqlite.OpenCreate | sqlite.OpenWAL | sqlite.OpenSharedCache,
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite pool: %v", err)
	}
	store := &SqliteStore{dbpool: dbpool}
	return store, store.init()
}

func (m *SqliteStore) Close() error {
	return m.dbpool.Close()
}

func (m *SqliteStore) init() error {
	var script = `
PRAGMA journal_mode=WAL;
PRAGMA page_size=4096;
CREATE TABLE IF NOT EXISTS operations (
	id INTEGER PRIMARY KEY,
	flow_id INTEGER NOT NULL,
	instance_id STRING NOT NULL,
	plan_id STRING NOT NULL,
	repo_id STRING NOT NULL,
	snapshot_id STRING NOT NULL,
	operation BLOB NOT NULL
);
CREATE TABLE IF NOT EXISTS system_info (
	version INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS operations_instance_id ON operations (instance_id);
CREATE INDEX IF NOT EXISTS operations_plan_id ON operations (plan_id);
CREATE INDEX IF NOT EXISTS operations_repo_id ON operations (repo_id);
CREATE INDEX IF NOT EXISTS operations_snapshot_id ON operations (snapshot_id);

INSERT INTO system_info (version)
SELECT 0 WHERE NOT EXISTS (SELECT 1 FROM system_info);
`
	conn, err := m.dbpool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("init sqlite: %v", err)
	}
	defer m.dbpool.Put(conn)
	if err := sqlitex.ExecScript(conn, script); err != nil {
		return fmt.Errorf("init sqlite: %v", err)
	}

	// find the next id value
	if err := sqlitex.ExecuteTransient(conn, "SELECT MAX(id) FROM operations", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			m.nextIDVal.Store(stmt.GetInt64("MAX(id)") + 1)
			return nil
		},
	}); err != nil {
		return fmt.Errorf("get max ID: %v", err)
	}
	if m.nextIDVal.Load() == 0 {
		m.nextIDVal.Store(1)
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

func (m *SqliteStore) buildQuery(q oplog.Query, includeSelectClauses bool) (string, []any) {
	query := []string{`SELECT operation FROM operations WHERE 1=1`}
	args := []any{}

	if q.FlowID != 0 {
		query = append(query, " AND flow_id = ?")
		args = append(args, q.FlowID)
	}
	if q.InstanceID != "" {
		query = append(query, " AND instance_id = ?")
		args = append(args, q.InstanceID)
	}
	if q.PlanID != "" {
		query = append(query, " AND plan_id = ?")
		args = append(args, q.PlanID)
	}
	if q.RepoID != "" {
		query = append(query, " AND repo_id = ?")
		args = append(args, q.RepoID)
	}
	if q.SnapshotID != "" {
		query = append(query, " AND snapshot_id = ?")
		args = append(args, q.SnapshotID)
	}
	if q.OpIDs != nil {
		query = append(query, " AND id IN (")
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
			query = append(query, " ORDER BY id DESC")
		} else {
			query = append(query, " ORDER BY id ASC")
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

	return strings.Join(query, ""), args
}

func (m *SqliteStore) Query(q oplog.Query, f func(*v1.Operation) error) error {
	conn, err := m.dbpool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("query: %v", err)
	}
	defer m.dbpool.Put(conn)

	query, args := m.buildQuery(q, true)

	if err := sqlitex.ExecuteTransient(conn, query, &sqlitex.ExecOptions{
		Args: args,
		ResultFunc: func(stmt *sqlite.Stmt) error {
			opBytes := make([]byte, stmt.ColumnLen(0))
			n := stmt.GetBytes("operation", opBytes)
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

func (m *SqliteStore) Transform(q oplog.Query, f func(*v1.Operation) (*v1.Operation, error)) error {
	conn, err := m.dbpool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("transform: %v", err)
	}
	defer m.dbpool.Put(conn)

	query, args := m.buildQuery(q, false)

	return withSqliteTransaction(conn, func() error {
		return sqlitex.ExecuteTransient(conn, query, &sqlitex.ExecOptions{
			Args: args,
			ResultFunc: func(stmt *sqlite.Stmt) error {
				opBytes := make([]byte, stmt.ColumnLen(0))
				n := stmt.GetBytes("operation", opBytes)
				opBytes = opBytes[:n]

				var op v1.Operation
				if err := proto.Unmarshal(opBytes, &op); err != nil {
					return fmt.Errorf("unmarshal operation bytes: %v", err)
				}

				newOp, err := f(&op)
				if err != nil {
					return err
				}

				newOpBytes, err := proto.Marshal(newOp)
				if err != nil {
					return fmt.Errorf("marshal operation: %v", err)
				}

				if err := sqlitex.Execute(conn, "UPDATE operations SET operation = ? WHERE id = ?", &sqlitex.ExecOptions{
					Args: []any{newOpBytes, stmt.GetInt64("id")},
				}); err != nil {
					return fmt.Errorf("update operation: %v", err)
				}
				return nil
			},
		})
	})
}

func (m *SqliteStore) Add(op ...*v1.Operation) error {
	conn, err := m.dbpool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("add operation: %v", err)
	}
	defer m.dbpool.Put(conn)

	return withSqliteTransaction(conn, func() error {
		for _, o := range op {
			o.Id = m.nextIDVal.Add(1)
			if o.FlowId == 0 {
				o.FlowId = o.Id
			}
			if err := protoutil.ValidateOperation(o); err != nil {
				return err
			}

			query := "INSERT INTO operations (id, flow_id, instance_id, plan_id, repo_id, snapshot_id, operation) VALUES (?, ?, ?, ?, ?, ?, ?)"

			bytes, err := proto.Marshal(o)
			if err != nil {
				return fmt.Errorf("marshal operation: %v", err)
			}

			if err := sqlitex.Execute(conn, query, &sqlitex.ExecOptions{
				Args: []any{o.Id, o.FlowId, o.InstanceId, o.PlanId, o.RepoId, o.SnapshotId, bytes},
			}); err != nil {
				if sqlite.ErrCode(err) == sqlite.ResultConstraintUnique {
					return fmt.Errorf("operation already exists: %w", oplog.ErrExist)
				}
				return fmt.Errorf("add operation: %v", err)
			}

		}
		return nil
	})
}

func (m *SqliteStore) Update(op ...*v1.Operation) error {
	conn, err := m.dbpool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("update operation: %v", err)
	}
	defer m.dbpool.Put(conn)

	return withSqliteTransaction(conn, func() error {
		for _, o := range op {
			if err := protoutil.ValidateOperation(o); err != nil {
				return err
			}
			bytes, err := proto.Marshal(o)
			if err != nil {
				return fmt.Errorf("marshal operation: %v", err)
			}
			if err := sqlitex.Execute(conn, "UPDATE operations SET operation = ?, flow_id = ?, instance_id = ?, plan_id = ?, repo_id = ?, snapshot_id = ? WHERE id = ?", &sqlitex.ExecOptions{
				Args: []any{bytes, o.FlowId, o.InstanceId, o.PlanId, o.RepoId, o.SnapshotId, o.Id},
			}); err != nil {
				return fmt.Errorf("update operation: %v", err)
			}
			if conn.Changes() == 0 {
				return fmt.Errorf("couldn't update %d: %w", o.Id, oplog.ErrNotExist)
			}
		}
		return nil
	})
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
	return ops, withSqliteTransaction(conn, func() error {
		// fetch all the operations we're about to delete
		predicate := []string{"id IN ("}
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

		if err := sqlitex.ExecuteTransient(conn, "SELECT operation FROM operations WHERE "+predicateStr, &sqlitex.ExecOptions{
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
			return fmt.Errorf("load operations for delete: %v", err)
		}

		if len(ops) != len(opID) {
			return fmt.Errorf("couldn't find all operations to delete: %w", oplog.ErrNotExist)
		}

		// delete the operations
		if err := sqlitex.ExecuteTransient(conn, "DELETE FROM operations WHERE "+predicateStr, &sqlitex.ExecOptions{
			Args: args,
		}); err != nil {
			return fmt.Errorf("delete operations: %v", err)
		}
		return nil
	})
}

func withSqliteTransaction(conn *sqlite.Conn, f func() error) error {
	var err error
	endFunc := sqlitex.Transaction(conn)
	err = f()
	endFunc(&err)
	return err
}
