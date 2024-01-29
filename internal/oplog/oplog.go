package oplog

import (
	"errors"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog/indexutil"
	"github.com/garethgeorge/backrest/internal/oplog/serializationutil"
	"github.com/garethgeorge/backrest/internal/protoutil"
	"github.com/garethgeorge/backrest/pkg/restic"
	"github.com/hashicorp/go-multierror"
	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type EventType int

const (
	EventTypeUnknown   = EventType(iota)
	EventTypeOpCreated = EventType(iota)
	EventTypeOpUpdated = EventType(iota)
)

var ErrNotExist = errors.New("operation does not exist")

var (
	SystemBucket        = []byte("oplog.system")       // system stores metadata
	OpLogBucket         = []byte("oplog.log")          // oplog stores existant operations.
	RepoIndexBucket     = []byte("oplog.repo_idx")     // repo_index tracks IDs of operations affecting a given repo
	PlanIndexBucket     = []byte("oplog.plan_idx")     // plan_index tracks IDs of operations affecting a given plan
	SnapshotIndexBucket = []byte("oplog.snapshot_idx") // snapshot_index tracks IDs of operations affecting a given snapshot
)

// OpLog represents a log of operations performed.
// Operations are indexed by repo and plan.
type OpLog struct {
	db *bolt.DB
	*BigOpDataStore

	subscribersMu sync.RWMutex
	subscribers   []*func(*v1.Operation, *v1.Operation)
}

func NewOpLog(databasePath string) (*OpLog, error) {
	if err := os.MkdirAll(path.Dir(databasePath), 0700); err != nil {
		return nil, fmt.Errorf("error creating database directory: %s", err)
	}

	db, err := bolt.Open(databasePath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("error opening database: %s", err)
	}

	o := &OpLog{
		db: db,
		BigOpDataStore: &BigOpDataStore{
			path: path.Dir(databasePath) + "/opdatav1",
		},
	}

	if err := db.Update(func(tx *bolt.Tx) error {
		// Create the buckets if they don't exist
		for _, bucket := range [][]byte{
			SystemBucket, OpLogBucket, RepoIndexBucket, PlanIndexBucket, SnapshotIndexBucket,
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

// Scan checks the log for incomplete operations. Should only be called at startup.
func (o *OpLog) Scan(onIncomplete func(op *v1.Operation)) error {
	removeIds := make([]int64, 0)

	err := o.db.Update(func(tx *bolt.Tx) error {
		sysBucket := tx.Bucket(SystemBucket)
		opLogBucket := tx.Bucket(OpLogBucket)
		c := opLogBucket.Cursor()
		if lastValidated := sysBucket.Get([]byte("last_validated")); lastValidated != nil {
			c.Seek(lastValidated)
		}
		for k, v := c.First(); k != nil; k, v = c.Next() {
			op := &v1.Operation{}
			if err := proto.Unmarshal(v, op); err != nil {
				zap.L().Error("error unmarshalling operation, there may be corruption in the oplog", zap.Error(err))
				continue
			}

			if op.Status == v1.OperationStatus_STATUS_PENDING || op.Status == v1.OperationStatus_STATUS_SYSTEM_CANCELLED || op.Status == v1.OperationStatus_STATUS_USER_CANCELLED {
				// remove pending or user cancelled operations.
				removeIds = append(removeIds, op.Id)
				continue
			} else if op.Status == v1.OperationStatus_STATUS_INPROGRESS {
				onIncomplete(op)
				removeIds = append(removeIds, op.Id)
			}

			if err := o.addOperationHelper(tx, op); err != nil {
				zap.L().Error("error re-adding operation, there may be corruption in the oplog", zap.Error(err))
				continue
			}
		}
		if lastValidated, _ := c.Last(); lastValidated != nil {
			if err := sysBucket.Put([]byte("last_validated"), lastValidated); err != nil {
				return fmt.Errorf("checkpointing last_validated key: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("scanning log: %v", err)
	}

	if len(removeIds) > 0 {
		if err := o.Delete(removeIds...); err != nil {
			return fmt.Errorf("removing incomplete operations: %w", err)
		}
	}
	return nil
}

func (o *OpLog) Close() error {
	return o.db.Close()
}

// Add adds a generic operation to the operation log.
func (o *OpLog) Add(op *v1.Operation) error {
	if op.Id != 0 {
		return errors.New("operation already has an ID, OpLog.Add is expected to set the ID")
	}

	err := o.db.Update(func(tx *bolt.Tx) error {
		err := o.addOperationHelper(tx, op)
		if err != nil {
			return err
		}
		return nil
	})
	if err == nil {
		o.notifyHelper(nil, op)
	}
	return err
}

func (o *OpLog) BulkAdd(ops []*v1.Operation) error {
	err := o.db.Update(func(tx *bolt.Tx) error {
		for _, op := range ops {
			if op.Id != 0 {
				return errors.New("operation already has an ID, OpLog.BulkAdd is expected to set the ID")
			}
			if err := o.addOperationHelper(tx, op); err != nil {
				return err
			}
		}
		return nil
	})
	if err == nil {
		for _, op := range ops {
			o.notifyHelper(nil, op)
		}
	}
	return err
}

func (o *OpLog) Update(op *v1.Operation) error {
	if op.Id == 0 {
		return errors.New("operation does not have an ID, OpLog.Update expects operation with an ID")
	}
	var oldOp *v1.Operation
	err := o.db.Update(func(tx *bolt.Tx) error {
		var err error
		oldOp, err = o.deleteOperationHelper(tx, op.Id)
		if err != nil {
			return fmt.Errorf("deleting existing value prior to update: %w", err)
		}
		if err := o.addOperationHelper(tx, op); err != nil {
			return fmt.Errorf("adding updated value: %w", err)
		}
		return nil
	})
	if err == nil {
		o.notifyHelper(oldOp, op)
	}
	return err
}

func (o *OpLog) Delete(ids ...int64) error {
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
	if err == nil {
		for _, op := range removedOps {
			o.notifyHelper(op, nil)
		}
		for _, id := range ids {
			if e := o.DeleteOperationData(id); e != nil {
				err = multierror.Append(err, fmt.Errorf("deleting big data for operation %v: %w", id, e))
			}
		}
	}
	return err
}

func (o *OpLog) notifyHelper(old *v1.Operation, new *v1.Operation) {
	o.subscribersMu.RLock()
	defer o.subscribersMu.RUnlock()
	for _, sub := range o.subscribers {
		(*sub)(old, new)
	}
}

func (o *OpLog) getOperationHelper(b *bolt.Bucket, id int64) (*v1.Operation, error) {
	bytes := b.Get(serializationutil.Itob(id))
	if bytes == nil {
		return nil, fmt.Errorf("opid %v: %w", id, ErrNotExist)
	}

	var op v1.Operation
	if err := proto.Unmarshal(bytes, &op); err != nil {
		return nil, fmt.Errorf("error unmarshalling operation: %w", err)
	}

	return &op, nil
}

func (o *OpLog) nextOperationId(b *bolt.Bucket, unixTimeMs int64) (int64, error) {
	seq, err := b.NextSequence()
	if err != nil {
		return 0, fmt.Errorf("next sequence: %w", err)
	}
	return int64(unixTimeMs<<20) | int64(seq&((1<<20)-1)), nil
}

func (o *OpLog) addOperationHelper(tx *bolt.Tx, op *v1.Operation) error {
	b := tx.Bucket(OpLogBucket)
	if op.Id == 0 {
		var err error
		op.Id, err = o.nextOperationId(b, time.Now().UnixMilli())
		if err != nil {
			return fmt.Errorf("create next operation ID: %w", err)
		}
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

	return nil
}

func (o *OpLog) deleteOperationHelper(tx *bolt.Tx, id int64) (*v1.Operation, error) {
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

	if err := b.Delete(serializationutil.Itob(id)); err != nil {
		return nil, fmt.Errorf("deleting operation %v from bucket: %w", id, err)
	}

	return prevValue, nil
}

func (o *OpLog) Get(id int64) (*v1.Operation, error) {
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

func (o *OpLog) ForEachByRepo(repoId string, collector indexutil.Collector, do func(op *v1.Operation) error) error {
	return o.db.View(func(tx *bolt.Tx) error {
		ids := collector(indexutil.IndexSearchByteValue(tx.Bucket(RepoIndexBucket), []byte(repoId)))
		return o.forOpsByIds(tx, ids, do)
	})
}

func (o *OpLog) ForEachByPlan(planId string, collector indexutil.Collector, do func(op *v1.Operation) error) error {
	return o.db.View(func(tx *bolt.Tx) error {
		ids := collector(indexutil.IndexSearchByteValue(tx.Bucket(PlanIndexBucket), []byte(planId)))
		return o.forOpsByIds(tx, ids, do)
	})
}

func (o *OpLog) ForEachBySnapshotId(snapshotId string, collector indexutil.Collector, do func(op *v1.Operation) error) error {
	if err := restic.ValidateSnapshotId(snapshotId); err != nil {
		return nil
	}
	return o.db.View(func(tx *bolt.Tx) error {
		ids := collector(indexutil.IndexSearchByteValue(tx.Bucket(SnapshotIndexBucket), []byte(snapshotId)))
		return o.forOpsByIds(tx, ids, do)
	})
}

func (o *OpLog) forOpsByIds(tx *bolt.Tx, ids []int64, do func(*v1.Operation) error) error {
	b := tx.Bucket(OpLogBucket)
	for _, id := range ids {
		op, err := o.getOperationHelper(b, id)
		if err != nil {
			return err
		}
		if err := do(op); err != nil {
			return err
		}
	}
	return nil
}

func (o *OpLog) ForAll(do func(op *v1.Operation) error) error {
	if err := o.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket(OpLogBucket).Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			op := &v1.Operation{}
			if err := proto.Unmarshal(v, op); err != nil {
				return fmt.Errorf("error unmarshalling operation: %w", err)
			}
			if err := do(op); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil
	}
	return nil
}

func (o *OpLog) Subscribe(callback *func(*v1.Operation, *v1.Operation)) {
	o.subscribersMu.Lock()
	defer o.subscribersMu.Unlock()
	o.subscribers = append(o.subscribers, callback)
}

func (o *OpLog) Unsubscribe(callback *func(*v1.Operation, *v1.Operation)) {
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
