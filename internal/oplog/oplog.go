package oplog

import (
	"errors"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	storm "github.com/asdine/storm/v3"
	"github.com/asdine/storm/v3/codec/protobuf"
	"github.com/asdine/storm/v3/index"
	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"
)

type EventType int

const (
	EventTypeUnknown   = EventType(iota)
	EventTypeOpCreated = EventType(iota)
	EventTypeOpUpdated = EventType(iota)
)

const (
	OpLogSchemaVersion = 1
)

var (
	OpLogSysBucket   = "oplog.sys"
	OpLogBucket      = "oplog.log"
	schemaVersionKey = "schema_version"
)

// OpLog represents a log of operations performed.
// Operations are indexed by repo and plan.
type OpLog struct {
	db  *bolt.DB
	sdb *storm.DB

	subscribersMu sync.RWMutex
	subscribers   []*func(EventType, *v1.Operation)
}

func NewOpLog(databasePath string) (*OpLog, error) {
	if err := os.MkdirAll(path.Dir(databasePath), 0700); err != nil {
		return nil, fmt.Errorf("creating database directory: %s", err)
	}

	db, err := bolt.Open(databasePath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("opening database: %s", err)
	}

	sdb, err := storm.Open("", storm.UseDB(db), storm.Codec(protobuf.Codec))
	if err != nil {
		return nil, fmt.Errorf("opening storm: %s", err)
	}

	var schemaVersion int
	if err := sdb.Get(OpLogSysBucket, schemaVersionKey, &schemaVersion); err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			schemaVersion = -1
		} else {
			return nil, fmt.Errorf("getting schema version: %s", err)
		}
	}
	if schemaVersion != OpLogSchemaVersion {
		zap.L().Info("oplog schema version mismatch, reindexing")
		sdb.From(OpLogBucket).ReIndex(&v1.Operation{})
		if err := sdb.Set(OpLogSysBucket, schemaVersionKey, OpLogSchemaVersion); err != nil {
			return nil, fmt.Errorf("setting schema version: %s", err)
		}
	}

	return &OpLog{
		db:  db,
		sdb: sdb,
	}, nil
}

func (o *OpLog) Close() error {
	return o.db.Close()
}

// Add adds a generic operation to the operation log.
func (o *OpLog) Add(op *v1.Operation) error {
	if op.Id != 0 {
		return errors.New("operation already has an ID, OpLog.Add is expected to set the ID")
	}
	op.SnapshotId = NormalizeSnapshotId(op.SnapshotId)

	if err := o.sdb.From(OpLogBucket).Save(op); err != nil {
		return fmt.Errorf("saving operation: %w", err)
	}

	o.notifyHelper(EventTypeOpCreated, op)
	return nil
}

func (o *OpLog) BulkAdd(ops []*v1.Operation) error {
	tx, err := o.sdb.From(OpLogBucket).Begin(true)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	for _, op := range ops {
		if op.Id != 0 {
			return errors.New("operation already has an ID, OpLog.BulkAdd is expected to set the ID")
		}

		if err := tx.Save(op); err != nil {
			return fmt.Errorf("saving operation: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	for _, op := range ops {
		o.notifyHelper(EventTypeOpCreated, op)
	}

	return nil
}

func (o *OpLog) Update(op *v1.Operation) error {
	if err := o.sdb.From(OpLogBucket).Update(op); err != nil {
		return fmt.Errorf("updating operation %v: %w", op.Id, err)
	}
	o.notifyHelper(EventTypeOpUpdated, op)
	return nil
}

func (o *OpLog) notifyHelper(eventType EventType, op *v1.Operation) {
	o.subscribersMu.RLock()
	defer o.subscribersMu.RUnlock()
	for _, sub := range o.subscribers {
		(*sub)(eventType, op)
	}
}

func (o *OpLog) Get(id int64) (*v1.Operation, error) {
	var op v1.Operation
	if err := o.sdb.From(OpLogBucket).One("Id", id, &op); err != nil {
		return nil, fmt.Errorf("getting operation: %w", err)
	}
	return &op, nil
}

func (o *OpLog) GetByRepo(repoId string, filter Filter) ([]*v1.Operation, error) {
	var ops []v1.Operation
	if err := o.sdb.From(OpLogBucket).Find("RepoId", repoId, &ops); err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting operations by repo %q: %w", repoId, err)
	}
	return opsToRefs(ops), nil
}

func (o *OpLog) GetByPlan(planId string, filter Filter) ([]*v1.Operation, error) {
	var ops []v1.Operation
	if err := o.sdb.From(OpLogBucket).Find("PlanId", planId, &ops, filter); err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting operations by plan %q: %w", planId, err)
	}
	return opsToRefs(ops), nil
}

func (o *OpLog) GetBySnapshotId(snapshotId string) ([]*v1.Operation, error) {
	snapshotId = NormalizeSnapshotId(snapshotId)
	var ops []v1.Operation
	if err := o.sdb.From(OpLogBucket).Find("SnapshotId", snapshotId, &ops); err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting operations by snapshot %q: %w", snapshotId, err)
	}
	return opsToRefs(ops), nil
}
func (o *OpLog) GetAll(filter Filter) ([]*v1.Operation, error) {
	ops := []v1.Operation{}
	if err := o.sdb.From(OpLogBucket).All(ops); err != nil {
		return nil, fmt.Errorf("getting all operations: %w", err)
	}
	return opsToRefs(ops), nil
}

func (o *OpLog) Subscribe(callback *func(EventType, *v1.Operation)) {
	o.subscribersMu.Lock()
	defer o.subscribersMu.Unlock()
	o.subscribers = append(o.subscribers, callback)
}

func (o *OpLog) Unsubscribe(callback *func(EventType, *v1.Operation)) {
	o.subscribersMu.Lock()
	defer o.subscribersMu.Unlock()
	subs := o.subscribers
	for i, c := range subs {
		if c == callback {
			subs[i] = subs[len(subs)-1]
			o.subscribers = subs[:len(o.subscribers)-1]
		}
	}
}

type Filter func(*index.Options)

func FilterKeepAll() Filter {
	return func(*index.Options) {}
}

func FilterLastN(n int64) Filter {
	return storm.Limit(int(n))
}

func FilterLimitOffset(limit, offset int64) Filter {
	return func(opts *index.Options) {
		storm.Skip(int(offset))(opts)
		storm.Limit(int(limit))(opts)
	}
}

func NormalizeSnapshotId(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

func opsToRefs(ops []v1.Operation) []*v1.Operation {
	refs := make([]*v1.Operation, len(ops))
	for i := range ops {
		refs[i] = &ops[i]
	}
	return refs
}
