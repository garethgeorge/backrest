package bboltstore

import (
	"errors"
	"fmt"
	"os"
	"path"
	"slices"
	"testing"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/oplog/bboltstore/indexutil"
	"github.com/garethgeorge/backrest/internal/oplog/bboltstore/serializationutil"
	"github.com/garethgeorge/backrest/internal/protoutil"
	"go.etcd.io/bbolt"
	bolt "go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"
)

type EventType int

const (
	EventTypeUnknown   = EventType(iota)
	EventTypeOpCreated = EventType(iota)
	EventTypeOpUpdated = EventType(iota)
)

var (
	SystemBucket        = []byte("oplog.system")       // system stores metadata
	OpLogBucket         = []byte("oplog.log")          // oplog stores existant operations.
	RepoIndexBucket     = []byte("oplog.repo_idx")     // repo_index tracks IDs of operations affecting a given repo
	PlanIndexBucket     = []byte("oplog.plan_idx")     // plan_index tracks IDs of operations affecting a given plan
	FlowIdIndexBucket   = []byte("oplog.flow_id_idx")  // flow_id_index tracks IDs of operations affecting a given flow
	InstanceIndexBucket = []byte("oplog.instance_idx") // instance_id_index tracks IDs of operations affecting a given instance
	SnapshotIndexBucket = []byte("oplog.snapshot_idx") // snapshot_index tracks IDs of operations affecting a given snapshot
)

// OpLog represents a log of operations performed.
// Operations are indexed by repo and plan.
type BboltStore struct {
	db *bolt.DB
}

var _ oplog.OpStore = &BboltStore{}

func NewBboltStore(databasePath string) (*BboltStore, error) {
	if err := os.MkdirAll(path.Dir(databasePath), 0700); err != nil {
		return nil, fmt.Errorf("error creating database directory: %s", err)
	}

	db, err := bolt.Open(databasePath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("error opening database: %s", err)
	}

	o := &BboltStore{
		db: db,
	}

	if err := db.Update(func(tx *bolt.Tx) error {
		// Create the buckets if they don't exist
		for _, bucket := range [][]byte{
			SystemBucket, OpLogBucket, RepoIndexBucket, PlanIndexBucket, SnapshotIndexBucket, FlowIdIndexBucket, InstanceIndexBucket,
		} {
			if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
				return fmt.Errorf("creating bucket %s: %s", string(bucket), err)
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return o, nil
}

func (o *BboltStore) Close() error {
	return o.db.Close()
}

func (o *BboltStore) Version() (int64, error) {
	var version int64
	o.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(SystemBucket)
		if b == nil {
			return nil
		}
		var err error
		version, err = serializationutil.Btoi(b.Get([]byte("version")))
		return err
	})
	return version, nil
}

func (o *BboltStore) SetVersion(version int64) error {
	return o.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(SystemBucket)
		if err != nil {
			return fmt.Errorf("creating system bucket: %w", err)
		}
		return b.Put([]byte("version"), serializationutil.Itob(version))
	})
}

// Add adds a generic operation to the operation log.
func (o *BboltStore) Add(ops ...*v1.Operation) error {

	return o.db.Update(func(tx *bolt.Tx) error {
		for _, op := range ops {
			err := o.addOperationHelper(tx, op)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (o *BboltStore) Update(ops ...*v1.Operation) error {
	for _, op := range ops {
		if op.Id == 0 {
			return errors.New("operation does not have an ID, OpLog.Update expects operation with an ID")
		}
	}
	return o.db.Update(func(tx *bolt.Tx) error {
		var err error
		for _, op := range ops {
			_, err = o.deleteOperationHelper(tx, op.Id)
			if err != nil {
				return fmt.Errorf("deleting existing value prior to update: %w", err)
			}
			if err := o.addOperationHelper(tx, op); err != nil {
				return fmt.Errorf("adding updated value: %w", err)
			}
		}
		return nil
	})
}

func (o *BboltStore) Delete(ids ...int64) ([]*v1.Operation, error) {
	removedOps := make([]*v1.Operation, 0, len(ids))
	err := o.db.Update(func(tx *bolt.Tx) error {
		for _, id := range ids {
			removed, err := o.deleteOperationHelper(tx, id)
			if err != nil {
				return fmt.Errorf("deleting operation %v: %w", id, err)
			}
			removedOps = append(removedOps, removed)
		}
		return nil
	})
	return removedOps, err
}

func (o *BboltStore) getOperationHelper(b *bolt.Bucket, id int64) (*v1.Operation, error) {
	bytes := b.Get(serializationutil.Itob(id))
	if bytes == nil {
		return nil, fmt.Errorf("opid %v: %w", id, oplog.ErrNotExist)
	}

	var op v1.Operation
	if err := proto.Unmarshal(bytes, &op); err != nil {
		return nil, fmt.Errorf("error unmarshalling operation: %w", err)
	}

	return &op, nil
}

func (o *BboltStore) nextID(b *bolt.Bucket, unixTimeMs int64) (int64, error) {
	seq, err := b.NextSequence()
	if err != nil {
		return 0, fmt.Errorf("next sequence: %w", err)
	}
	return int64(unixTimeMs<<20) | int64(seq&((1<<20)-1)), nil
}

func (o *BboltStore) addOperationHelper(tx *bolt.Tx, op *v1.Operation) error {
	b := tx.Bucket(OpLogBucket)
	if op.Id == 0 {
		var err error
		op.Id, err = o.nextID(b, time.Now().UnixMilli())
		if err != nil {
			return fmt.Errorf("create next operation ID: %w", err)
		}
	}

	if op.FlowId == 0 {
		op.FlowId = op.Id
	}

	if err := protoutil.ValidateOperation(op); err != nil {
		return fmt.Errorf("validating operation: %w", err)
	}

	bytes, err := proto.Marshal(op)
	if err != nil {
		return fmt.Errorf("error marshalling operation: %w", err)
	}

	if err := b.Put(serializationutil.Itob(op.Id), bytes); err != nil {
		return fmt.Errorf("error putting operation into bucket: %w", err)
	}

	// Update always universal indices
	if op.RepoId != "" {
		if err := indexutil.IndexByteValue(tx.Bucket(RepoIndexBucket), []byte(op.RepoId), op.Id); err != nil {
			return fmt.Errorf("error adding operation to repo index: %w", err)
		}
	}

	if op.PlanId != "" {
		if err := indexutil.IndexByteValue(tx.Bucket(PlanIndexBucket), []byte(op.PlanId), op.Id); err != nil {
			return fmt.Errorf("error adding operation to repo index: %w", err)
		}
	}

	if op.SnapshotId != "" {
		if err := indexutil.IndexByteValue(tx.Bucket(SnapshotIndexBucket), []byte(op.SnapshotId), op.Id); err != nil {
			return fmt.Errorf("error adding operation to snapshot index: %w", err)
		}
	}

	if op.FlowId != 0 {
		if err := indexutil.IndexByteValue(tx.Bucket(FlowIdIndexBucket), serializationutil.Itob(op.FlowId), op.Id); err != nil {
			return fmt.Errorf("error adding operation to flow index: %w", err)
		}
	}

	if op.InstanceId != "" {
		if err := indexutil.IndexByteValue(tx.Bucket(InstanceIndexBucket), []byte(op.InstanceId), op.Id); err != nil {
			return fmt.Errorf("error adding operation to instance index: %w", err)
		}
	}

	return nil
}

func (o *BboltStore) deleteOperationHelper(tx *bolt.Tx, id int64) (*v1.Operation, error) {
	b := tx.Bucket(OpLogBucket)

	prevValue, err := o.getOperationHelper(b, id)
	if err != nil {
		return nil, fmt.Errorf("getting operation %v: %w", id, err)
	}

	if prevValue.PlanId != "" {
		if err := indexutil.IndexRemoveByteValue(tx.Bucket(PlanIndexBucket), []byte(prevValue.PlanId), id); err != nil {
			return nil, fmt.Errorf("removing operation %v from plan index: %w", id, err)
		}
	}

	if prevValue.RepoId != "" {
		if err := indexutil.IndexRemoveByteValue(tx.Bucket(RepoIndexBucket), []byte(prevValue.RepoId), id); err != nil {
			return nil, fmt.Errorf("removing operation %v from repo index: %w", id, err)
		}
	}

	if prevValue.SnapshotId != "" {
		if err := indexutil.IndexRemoveByteValue(tx.Bucket(SnapshotIndexBucket), []byte(prevValue.SnapshotId), id); err != nil {
			return nil, fmt.Errorf("removing operation %v from snapshot index: %w", id, err)
		}
	}

	if prevValue.FlowId != 0 {
		if err := indexutil.IndexRemoveByteValue(tx.Bucket(FlowIdIndexBucket), serializationutil.Itob(prevValue.FlowId), id); err != nil {
			return nil, fmt.Errorf("removing operation %v from flow index: %w", id, err)
		}
	}

	if prevValue.InstanceId != "" {
		if err := indexutil.IndexRemoveByteValue(tx.Bucket(InstanceIndexBucket), []byte(prevValue.InstanceId), id); err != nil {
			return nil, fmt.Errorf("removing operation %v from instance index: %w", id, err)
		}
	}

	if err := b.Delete(serializationutil.Itob(id)); err != nil {
		return nil, fmt.Errorf("deleting operation %v from bucket: %w", id, err)
	}

	return prevValue, nil
}

func (o *BboltStore) Get(id int64) (*v1.Operation, error) {
	var op *v1.Operation
	if err := o.db.View(func(tx *bolt.Tx) error {
		var err error
		op, err = o.getOperationHelper(tx.Bucket(OpLogBucket), id)
		return err
	}); err != nil {
		return nil, err
	}
	return op, nil
}

// Query represents a query to the operation log.
type Query struct {
	RepoId     string
	PlanId     string
	SnapshotId string
	FlowId     int64
	InstanceId string
	Ids        []int64
}

func (o *BboltStore) Query(q oplog.Query, f func(*v1.Operation) error) error {
	return o.queryHelper(q, func(tx *bbolt.Tx, op *v1.Operation) error {
		return f(op)
	}, true)
}

func (o *BboltStore) Transform(q oplog.Query, f func(*v1.Operation) (*v1.Operation, error)) error {
	return o.queryHelper(q, func(tx *bbolt.Tx, op *v1.Operation) error {
		origId := op.Id
		transformed, err := f(op)
		if err != nil {
			return err
		}
		if transformed == nil {
			return nil
		}
		if _, err := o.deleteOperationHelper(tx, origId); err != nil {
			return fmt.Errorf("deleting old operation: %w", err)
		}
		if err := o.addOperationHelper(tx, transformed); err != nil {
			return fmt.Errorf("adding updated operation: %w", err)
		}
		return nil
	}, false)
}

func (o *BboltStore) queryHelper(query oplog.Query, do func(tx *bbolt.Tx, op *v1.Operation) error, isReadOnly bool) error {
	helper := func(tx *bolt.Tx) error {
		iterators := make([]indexutil.IndexIterator, 0, 5)
		if query.RepoID != "" {
			iterators = append(iterators, indexutil.IndexSearchByteValue(tx.Bucket(RepoIndexBucket), []byte(query.RepoID)))
		}
		if query.PlanID != "" {
			iterators = append(iterators, indexutil.IndexSearchByteValue(tx.Bucket(PlanIndexBucket), []byte(query.PlanID)))
		}
		if query.SnapshotID != "" {
			iterators = append(iterators, indexutil.IndexSearchByteValue(tx.Bucket(SnapshotIndexBucket), []byte(query.SnapshotID)))
		}
		if query.FlowID != 0 {
			iterators = append(iterators, indexutil.IndexSearchByteValue(tx.Bucket(FlowIdIndexBucket), serializationutil.Itob(query.FlowID)))
		}
		if query.InstanceID != "" {
			iterators = append(iterators, indexutil.IndexSearchByteValue(tx.Bucket(InstanceIndexBucket), []byte(query.InstanceID)))
		}

		var ids []int64
		if len(iterators) == 0 && len(query.OpIDs) == 0 {
			if query.Limit == 0 && query.Offset == 0 && !query.Reversed {
				return o.forAll(tx, func(op *v1.Operation) error { return do(tx, op) })
			} else {
				b := tx.Bucket(OpLogBucket)
				c := b.Cursor()
				for k, _ := c.First(); k != nil; k, _ = c.Next() {
					if id, err := serializationutil.Btoi(k); err != nil {
						continue // skip corrupt keys
					} else {
						ids = append(ids, id)
					}
				}
			}
		} else if len(iterators) > 0 {
			ids = indexutil.CollectAll()(indexutil.NewJoinIterator(iterators...))
		}
		ids = append(ids, query.OpIDs...)

		if query.Reversed {
			slices.Reverse(ids)
		}
		if query.Offset > 0 {
			if len(ids) <= query.Offset {
				return nil
			}
			ids = ids[query.Offset:]
		}
		if query.Limit > 0 && len(ids) > query.Limit {
			ids = ids[:query.Limit]
		}

		return o.forOpsByIds(tx, ids, func(op *v1.Operation) error {
			return do(tx, op)
		})
	}
	if isReadOnly {
		return o.db.View(helper)
	} else {
		return o.db.Update(helper)
	}
}

func (o *BboltStore) forOpsByIds(tx *bolt.Tx, ids []int64, do func(*v1.Operation) error) error {
	b := tx.Bucket(OpLogBucket)
	for _, id := range ids {
		op, err := o.getOperationHelper(b, id)
		if err != nil {
			return err
		}
		if err := do(op); err != nil {
			if err == oplog.ErrStopIteration {
				break
			}
			return err
		}
	}
	return nil
}

func (o *BboltStore) forAll(tx *bolt.Tx, do func(*v1.Operation) error) error {
	b := tx.Bucket(OpLogBucket)
	c := b.Cursor()
	for k, v := c.First(); k != nil; k, v = c.Next() {
		var op v1.Operation
		if err := proto.Unmarshal(v, &op); err != nil {
			return fmt.Errorf("error unmarshalling operation: %w", err)
		}
		if err := do(&op); err != nil {
			if err == oplog.ErrStopIteration {
				break
			}
			return err
		}
	}
	return nil
}

func (o *BboltStore) ResetForTest(t *testing.T) {
	if err := o.db.Update(func(tx *bolt.Tx) error {
		for _, bucket := range [][]byte{
			SystemBucket, OpLogBucket, RepoIndexBucket, PlanIndexBucket, SnapshotIndexBucket, FlowIdIndexBucket, InstanceIndexBucket,
		} {
			if err := tx.DeleteBucket(bucket); err != nil {
				return fmt.Errorf("deleting bucket %s: %w", string(bucket), err)
			}
			if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
				return fmt.Errorf("creating bucket %s: %w", string(bucket), err)
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("error resetting database: %s", err)
	}
}
