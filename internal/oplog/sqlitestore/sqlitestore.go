package sqlitestore

import (
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/config/migrations"
	"github.com/garethgeorge/backrest/internal/ioutil"
	"github.com/garethgeorge/backrest/internal/kvstore"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/protoutil"
	lru "github.com/hashicorp/golang-lru/v2"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/gofrs/flock"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/vfs"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
	_ "github.com/ncruces/go-sqlite3/vfs/memdb"
)

var ErrLocked = errors.New("sqlite db is locked")

const (
	metadataKeyVersion = "version"
)

type SqliteStore struct {
	dbpool *sql.DB
	dblock *flock.Flock

	ogidCache *lru.TwoQueueCache[opGroupInfo, int64]

	kvstore      kvstore.KvStore
	highestModno atomic.Int64
	highestOpID  atomic.Int64
}

var _ oplog.OpStore = (*SqliteStore)(nil)

func NewSqliteStore(db string) (*SqliteStore, error) {
	if err := os.MkdirAll(filepath.Dir(db), 0700); err != nil {
		return nil, fmt.Errorf("create sqlite db directory: %v", err)
	}
	if !vfs.SupportsFileLocking {
		return nil, fmt.Errorf("file locking not supported")
	}
	dbpool, err := sql.Open("sqlite3", db)
	if err != nil {
		return nil, fmt.Errorf("open sqlite pool: %v", err)
	}
	if vfs.SupportsSharedMemory {
		_, err = dbpool.ExecContext(context.Background(), `
			PRAGMA journal_mode = WAL;
			PRAGMA synchronous = NORMAL;
		`)
		if err != nil {
			return nil, fmt.Errorf("run multiline query: %v", err)
		}
	}
	if runtime.GOOS == "darwin" {
		_, err = dbpool.ExecContext(context.Background(), "PRAGMA checkpoint_fullfsync = 1")
		if err != nil {
			return nil, fmt.Errorf("run multiline query: %v", err)
		}
	}

	kvstore, err := kvstore.NewSqliteKVStore(dbpool, "oplog_metadata")
	if err != nil {
		return nil, fmt.Errorf("create kvstore: %v", err)
	}

	ogidCache, _ := lru.New2Q[opGroupInfo, int64](128)
	store := &SqliteStore{
		dbpool:    dbpool,
		dblock:    flock.New(db + ".lock"),
		ogidCache: ogidCache,
		kvstore:   kvstore,
	}
	if locked, err := store.dblock.TryLock(); err != nil {
		return nil, fmt.Errorf("lock sqlite db: %v", err)
	} else if !locked {
		return nil, ErrLocked
	}
	if err := store.backup(db, 3, false); err != nil {
		return nil, fmt.Errorf("backup sqlite db: %v", err)
	}
	if err := store.init(); err != nil {
		return nil, err
	}
	return store, nil
}

func NewMemorySqliteStore(t testing.TB) (*SqliteStore, error) {
	dbpool, err := sql.Open("sqlite3", memdb.TestDB(t))
	if err != nil {
		return nil, fmt.Errorf("open sqlite pool: %v", err)
	}

	kvstore, err := kvstore.NewSqliteKVStore(dbpool, "oplog_metadata")
	if err != nil {
		return nil, fmt.Errorf("create kvstore: %v", err)
	}

	ogidCache, _ := lru.New2Q[opGroupInfo, int64](128)
	store := &SqliteStore{
		dbpool:    dbpool,
		ogidCache: ogidCache,
		kvstore:   kvstore,
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
	if err := applySqliteMigrations(m, m.dbpool); err != nil {
		return err
	}
	// highestOpID from all instances
	highestID, _, err := m.GetHighestOpIDAndModno(oplog.Query{})
	if err != nil {
		return err
	}
	_, highestModno, err := m.GetHighestOpIDAndModno(oplog.Query{}.SetOriginalInstanceKeyid(""))
	if err != nil {
		return err
	}
	m.highestModno.Store(highestModno)
	m.highestOpID.Store(highestID)
	return nil
}

// backup creates a backup of the database using VACUUM INTO.
// keepCount specifies how many old backups to keep (older ones are deleted).
// force skips the time check and creates a backup even if the latest is recent.
func (m *SqliteStore) backup(to string, keepCount int, force bool) error {
	dir := filepath.Dir(to)
	base := filepath.Base(to)
	pattern := fmt.Sprintf("%s-*.backup", base)
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return fmt.Errorf("glob for old backups: %v", err)
	}
	sort.Strings(matches)

	backupSuffix := fmt.Sprintf("s%02dm%02d.backup", sqlSchemaVersion, migrations.CurrentVersion)
	if !force && len(matches) > 0 {
		latestBackup := matches[len(matches)-1]
		info, err := os.Stat(latestBackup)
		if err != nil {
			return fmt.Errorf("stat latest backup %q: %w", latestBackup, err)
		}
		if strings.HasSuffix(latestBackup, backupSuffix) && time.Since(info.ModTime()) < 7*24*time.Hour {
			// Don't create a new backup more than once a week if the last one matches the schema.
			return nil
		}
	}

	// Create the backup using VACUUM INTO
	backupPath := fmt.Sprintf("%s-%s-%s", to, time.Now().Format("20060102.150405.000"), backupSuffix)
	_, err = m.dbpool.ExecContext(context.Background(), "VACUUM INTO ?", backupPath)
	if err != nil {
		return fmt.Errorf("backup sqlite db: %v", err)
	}

	// Delete old backups, keeping only the specified number
	if len(matches) > keepCount-1 {
		toDelete := matches[:len(matches)-keepCount+1]
		for _, f := range toDelete {
			if err := os.Remove(f); err != nil {
				return fmt.Errorf("delete old backup %q: %w", f, err)
			}
		}
	}

	return nil
}

func (m *SqliteStore) GetHighestOpIDAndModno(q oplog.Query) (int64, int64, error) {
	var highestID sql.NullInt64
	var highestModno sql.NullInt64
	where, args := m.buildQueryWhereClause(q, false)
	row := m.dbpool.QueryRowContext(context.Background(), "SELECT MAX(operations.id), MAX(operations.modno) FROM operations JOIN operation_groups ON operations.ogid = operation_groups.ogid WHERE "+where, args...)
	if err := row.Scan(&highestID, &highestModno); err != nil {
		return 0, 0, err
	}
	return highestID.Int64, highestModno.Int64, nil
}

func (m *SqliteStore) Version() (int64, error) {
	versionBytes, err := m.kvstore.Get(metadataKeyVersion)
	if err != nil {
		if errors.Is(err, kvstore.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	if len(versionBytes) != 8 {
		return 0, fmt.Errorf("version bytes length is not 8: %d", len(versionBytes))
	}
	return int64(binary.LittleEndian.Uint64(versionBytes)), nil
}

func (m *SqliteStore) SetVersion(version int64) error {
	return m.kvstore.Set(metadataKeyVersion, binary.LittleEndian.AppendUint64(nil, uint64(version)))
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
	if q.ModnoGte != nil {
		query = append(query, " AND operations.modno >= ?")
		args = append(args, *q.ModnoGte)
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
	where, args := m.buildQueryWhereClause(q, true)
	rows, err := m.dbpool.QueryContext(context.Background(), "SELECT operations.operation FROM operations JOIN operation_groups ON operations.ogid = operation_groups.ogid WHERE "+where, args...)
	if err != nil {
		return fmt.Errorf("query: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var opBytes []byte
		if err := rows.Scan(&opBytes); err != nil {
			return fmt.Errorf("query: scan: %v", err)
		}

		var op v1.Operation
		if err := proto.Unmarshal(opBytes, &op); err != nil {
			return fmt.Errorf("unmarshal operation bytes: %v", err)
		}
		if err := f(&op); err != nil {
			if errors.Is(err, oplog.ErrStopIteration) {
				return nil
			}
			return err
		}
	}

	return rows.Err()
}

func (m *SqliteStore) QueryMetadata(q oplog.Query, f func(oplog.OpMetadata) error) error {
	where, args := m.buildQueryWhereClause(q, false)
	rows, err := m.dbpool.QueryContext(context.Background(), "SELECT operations.id, operations.modno, operations.original_id, operations.flow_id, operations.original_flow_id, operations.status FROM operations JOIN operation_groups ON operations.ogid = operation_groups.ogid WHERE "+where, args...)
	if err != nil {
		return fmt.Errorf("query metadata: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var meta oplog.OpMetadata
		if err := rows.Scan(&meta.ID, &meta.Modno, &meta.OriginalID, &meta.FlowID, &meta.OriginalFlowID, &meta.Status); err != nil {
			return fmt.Errorf("query metadata: scan: %v", err)
		}
		if err := f(meta); err != nil {
			if errors.Is(err, oplog.ErrStopIteration) {
				return nil
			}
			return err
		}
	}

	return rows.Err()
}

// tidyGroups deletes operation groups that are no longer referenced, it takes an int64 specifying the maximum group ID to consider.
// this allows ignoring newly created groups that may not yet be referenced.
func (m *SqliteStore) tidyGroups(conn *sql.DB, eligibleIDsBelow int64) {
	_, err := conn.ExecContext(context.Background(), "DELETE FROM operation_groups WHERE ogid NOT IN (SELECT DISTINCT ogid FROM operations WHERE ogid < ?)", eligibleIDsBelow)
	if err != nil {
		zap.S().Warnf("tidy groups: %v", err)
	}
}

func (m *SqliteStore) findOrCreateGroup(tx *sql.Tx, op *v1.Operation) (ogid int64, err error) {
	ogidKey := groupInfoForOp(op)
	if cachedOGID, ok := m.ogidCache.Get(ogidKey); ok {
		return cachedOGID, nil
	}

	err = tx.QueryRowContext(context.Background(), "SELECT ogid FROM operation_groups WHERE instance_id = ? AND original_instance_keyid = ? AND repo_id = ? AND plan_id = ? AND repo_guid = ? LIMIT 1", op.InstanceId, op.OriginalInstanceKeyid, op.RepoId, op.PlanId, op.RepoGuid).Scan(&ogid)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("find operation group: %v", err)
	}

	if errors.Is(err, sql.ErrNoRows) {
		err = tx.QueryRowContext(context.Background(), "INSERT INTO operation_groups (instance_id, original_instance_keyid, repo_id, plan_id, repo_guid) VALUES (?, ?, ?, ?, ?) RETURNING ogid", op.InstanceId, op.OriginalInstanceKeyid, op.RepoId, op.PlanId, op.RepoGuid).Scan(&ogid)
		if err != nil {
			return 0, fmt.Errorf("insert operation group: %v", err)
		}
	}

	m.ogidCache.Add(ogidKey, ogid)
	return ogid, nil
}

func (m *SqliteStore) Transform(q oplog.Query, f func(*v1.Operation) (*v1.Operation, error)) error {
	tx, err := m.dbpool.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("transform: begin tx: %v", err)
	}
	defer tx.Rollback()

	where, args := m.buildQueryWhereClause(q, true)
	rows, err := tx.QueryContext(context.Background(), "SELECT operations.operation FROM operations JOIN operation_groups ON operations.ogid = operation_groups.ogid WHERE "+where, args...)
	if err != nil {
		return fmt.Errorf("transform: query: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var opBytes []byte
		if err := rows.Scan(&opBytes); err != nil {
			return fmt.Errorf("transform: scan: %v", err)
		}

		var op v1.Operation
		if err := proto.Unmarshal(opBytes, &op); err != nil {
			return fmt.Errorf("unmarshal operation bytes: %v", err)
		}

		newOp, err := f(&op)
		if err != nil {
			return err
		} else if newOp == nil {
			continue
		}

		if err := m.updateInternal(tx, newOp); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("transform: rows err: %v", err)
	}

	return tx.Commit()
}

func (m *SqliteStore) addInternal(tx *sql.Tx, op ...*v1.Operation) error {
	for _, o := range op {
		ogid, err := m.findOrCreateGroup(tx, o)
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

		_, err = tx.ExecContext(context.Background(), query, o.Id, ogid, o.OriginalId, o.OriginalFlowId, o.Modno, o.FlowId, o.UnixTimeStartMs, int64(o.Status), o.SnapshotId, bytes)
		if err != nil {
			// TODO: check for a more specific error
			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				return fmt.Errorf("operation already exists %v: %w", o.Id, oplog.ErrExist)
			}
			return fmt.Errorf("add operation: %v", err)
		}
	}
	return nil
}

func (m *SqliteStore) Add(op ...*v1.Operation) error {
	tx, err := m.dbpool.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("add operation: begin tx: %v", err)
	}
	defer tx.Rollback()

	for _, o := range op {
		o.Id = m.highestOpID.Add(1)
		o.Modno = m.highestModno.Add(1)
		if o.FlowId == 0 {
			o.FlowId = o.Id
		}
		if err := protoutil.ValidateOperation(o); err != nil {
			return err
		}
	}

	if err := m.addInternal(tx, op...); err != nil {
		return err
	}

	return tx.Commit()
}

func (m *SqliteStore) Update(op ...*v1.Operation) error {
	tx, err := m.dbpool.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("update operation: begin tx: %v", err)
	}
	defer tx.Rollback()

	if err := m.updateInternal(tx, op...); err != nil {
		return err
	}

	return tx.Commit()
}

func (m *SqliteStore) updateInternal(tx *sql.Tx, op ...*v1.Operation) error {
	for _, o := range op {
		o.Modno = m.highestModno.Add(1)
		if err := protoutil.ValidateOperation(o); err != nil {
			return err
		}
		bytes, err := proto.Marshal(o)
		if err != nil {
			return fmt.Errorf("marshal operation: %v", err)
		}

		ogid, err := m.findOrCreateGroup(tx, o)
		if err != nil {
			return fmt.Errorf("find ogid: %v", err)
		}

		res, err := tx.ExecContext(context.Background(), "UPDATE operations SET operation = ?, ogid = ?, start_time_ms = ?, flow_id = ?, snapshot_id = ?, modno = ?, original_id = ?, original_flow_id = ?, status = ? WHERE id = ?", bytes, ogid, o.UnixTimeStartMs, o.FlowId, o.SnapshotId, o.Modno, o.OriginalId, o.OriginalFlowId, int64(o.Status), o.Id)
		if err != nil {
			return fmt.Errorf("update operation: %v", err)
		}
		rowsAffected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("update operation: get rows affected: %v", err)
		}
		if rowsAffected == 0 {
			return fmt.Errorf("couldn't update %d: %w", o.Id, oplog.ErrNotExist)
		}
	}
	return nil
}

func (m *SqliteStore) Get(opID int64) (*v1.Operation, error) {
	var opBytes []byte
	err := m.dbpool.QueryRowContext(context.Background(), "SELECT operation FROM operations WHERE id = ?", opID).Scan(&opBytes)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, oplog.ErrNotExist
		}
		return nil, fmt.Errorf("get operation: %v", err)
	}

	var op v1.Operation
	if err := proto.Unmarshal(opBytes, &op); err != nil {
		return nil, fmt.Errorf("unmarshal operation bytes: %v", err)
	}

	return &op, nil
}

func (m *SqliteStore) Delete(opID ...int64) ([]*v1.Operation, error) {
	tx, err := m.dbpool.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, fmt.Errorf("delete operation: begin tx: %v", err)
	}
	defer tx.Rollback()

	ops := make([]*v1.Operation, 0, len(opID))
	for _, batch := range ioutil.Batchify(opID, ioutil.DefaultBatchSize) {
		batchOps, err := m.deleteHelper(tx, batch...)
		if err != nil {
			return nil, err
		}
		ops = append(ops, batchOps...)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("delete operation: commit: %v", err)
	}
	return ops, nil
}

func (m *SqliteStore) deleteHelper(tx *sql.Tx, opID ...int64) ([]*v1.Operation, error) {
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

	rows, err := tx.QueryContext(context.Background(), "SELECT operations.operation FROM operations JOIN operation_groups ON operations.ogid = operation_groups.ogid WHERE "+predicateStr, args...)
	if err != nil {
		return nil, fmt.Errorf("load operations for delete: %v", err)
	}
	defer rows.Close()

	var ops []*v1.Operation
	for rows.Next() {
		var opBytes []byte
		if err := rows.Scan(&opBytes); err != nil {
			return nil, fmt.Errorf("load operations for delete: scan: %v", err)
		}

		var op v1.Operation
		if err := proto.Unmarshal(opBytes, &op); err != nil {
			return nil, fmt.Errorf("unmarshal operation bytes: %v", err)
		}
		ops = append(ops, &op)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load operations for delete: rows err: %v", err)
	}

	if len(ops) != len(opID) {
		return nil, fmt.Errorf("couldn't find all operations to delete: %w", oplog.ErrNotExist)
	}

	// Delete the operations
	res, err := tx.ExecContext(context.Background(), "DELETE FROM operations WHERE "+predicateStr, args...)
	if err != nil {
		return nil, fmt.Errorf("delete operations: %v", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("delete operations: get rows affected: %v", err)
	}
	if int(rowsAffected) != len(opID) {
		return nil, fmt.Errorf("expected to delete %d operations, but deleted %d", len(opID), rowsAffected)
	}

	return ops, nil
}

func (m *SqliteStore) ResetForTest(t *testing.T) error {
	_, err := m.dbpool.ExecContext(context.Background(), "DELETE FROM operations")
	if err != nil {
		return fmt.Errorf("reset for test: %v", err)
	}
	m.highestOpID.Store(0)
	m.highestModno.Store(0)
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
