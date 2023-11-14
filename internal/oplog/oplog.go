package oplog

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"time"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	bolt "go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"
)


type EventType int

const (
	EventTypeUnknown = EventType(iota)
	EventTypeOpCreated = EventType(iota)
	EventTypeOpUpdated = EventType(iota)
)

var (
	SystemBucket = []byte("oplog.system") // system stores metadata
	OpLogBucket  = []byte("oplog.log") // oplog stores the operations themselves
	RepoIndexBucket = []byte("oplog.repo_idx") // repo_index tracks IDs of operations affecting a given repo
	PlanIndexBucket = []byte("oplog.plan_idx") // plan_index tracks IDs of operations affecting a given plan
)


// OpLog represents a log of operations performed.
// TODO: implement trim support for old operations.
type OpLog struct {
	db *bolt.DB

	subscribersMu sync.RWMutex
	subscribers []*func(EventType, *v1.Operation)
}

func NewOpLog(databaseDir string) (*OpLog, error) {
	db, err := bolt.Open(databaseDir, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("error opening database: %s", err)
	}

	// Create the buckets if they don't exist
	if err := db.Update(func(tx *bolt.Tx) error {
		for _, bucket := range [][]byte{SystemBucket, OpLogBucket, RepoIndexBucket, PlanIndexBucket} {
			if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
				return fmt.Errorf("error creating bucket %s: %s", string(bucket), err)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &OpLog{db: db}, nil
}

func (o *OpLog) Close() error {
	return o.db.Close()
}

func (o *OpLog) Add(op *v1.Operation) error {
	if op.Id != 0 {
		return errors.New("operation already has an ID, OpLog.Add is expected to set the ID")
	}

	err := o.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(OpLogBucket)

		id, err := b.NextSequence()
		if err != nil {
			return fmt.Errorf("error getting next sequence: %w", err)
		}

		op.Id = int64(id)

		bytes, err := proto.Marshal(op)
		if err != nil {
			return fmt.Errorf("error marshalling operation: %w", err)
		}

		if err := b.Put(itob(op.Id), bytes); err != nil {
			return fmt.Errorf("error putting operation into bucket: %w", err)
		}

		if op.RepoId != "" {
			if err := o.addOpToIndexBucket(tx, RepoIndexBucket, op.RepoId, op.Id); err != nil {
				return fmt.Errorf("error adding operation to repo index: %w", err)
			}
		}

		if op.PlanId != "" {
			if err := o.addOpToIndexBucket(tx, PlanIndexBucket, op.PlanId, op.Id); err != nil {
				return fmt.Errorf("error adding operation to plan index: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		o.subscribersMu.RLock()
		defer o.subscribersMu.RUnlock()
		for _, sub := range o.subscribers {
			(*sub)(EventTypeOpCreated, op)
		}
	}
	return err
}

func (o *OpLog) Update(op *v1.Operation) error {
	if op.Id == 0 {
		return errors.New("operation does not have an ID, OpLog.Update expects operation with an ID")
	}

	err := o.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(OpLogBucket)

		if b.Get(itob(op.Id)) == nil {
			return fmt.Errorf("operation with ID %d does not exist", op.Id)
		}

		bytes, err := proto.Marshal(op)
		if err != nil {
			return fmt.Errorf("error marshalling operation: %w", err)
		}


		if err := b.Put(itob(op.Id), bytes); err != nil {
			return fmt.Errorf("error putting operation into bucket: %w", err)
		}

		return nil
	})
	if err != nil {
		o.subscribersMu.RLock()
		defer o.subscribersMu.RUnlock()
		for _, sub := range o.subscribers {
			(*sub)(EventTypeOpUpdated, op)
		}
	}
	return err
}

func (o *OpLog) getHelper(b *bolt.Bucket, id int64) (*v1.Operation, error) {
	bytes := b.Get(itob(id))
	if bytes == nil {
		return nil, fmt.Errorf("operation with ID %d does not exist", id)
	}

	var op v1.Operation
	if err := proto.Unmarshal(bytes, &op); err != nil {
		return nil, fmt.Errorf("error unmarshalling operation: %w", err)
	}

	return &op, nil
}

func (o *OpLog) Get(id int64) (*v1.Operation, error) {
	var op *v1.Operation
	if err := o.db.View(func(tx *bolt.Tx) error {
		var err error
		op, err = o.getHelper(tx.Bucket(OpLogBucket), id)
		return err 
	}); err != nil {
		return nil, err
	}
	return op, nil
}

func (o *OpLog) GetByRepo(repoId string, filter Filter) ([]*v1.Operation, error) {
	var ops []*v1.Operation
	if err := o.db.View(func(tx *bolt.Tx) error {
		ids, err := o.readOpsFromIndexBucket(tx, RepoIndexBucket, repoId)
		if err != nil {
			return err
		}

		b := tx.Bucket(OpLogBucket)
		for _, id := range ids {
			op, err := o.getHelper(b, id)
			if err != nil {
				return err
			}
			ops = append(ops, op)
		}

		return nil
	}); err != nil {
		return nil, err
	}
	return ops, nil
}

func (o *OpLog) GetByPlan(planId string, filter Filter) ([]*v1.Operation, error) {
	var ops []*v1.Operation
	if err := o.db.View(func(tx *bolt.Tx) error {
		ids, err := o.readOpsFromIndexBucket(tx, PlanIndexBucket, planId)
		if err != nil {
			return err
		}

		b := tx.Bucket(OpLogBucket)
		for _, id := range ids {
			op, err := o.getHelper(b, id)
			if err != nil {
				return err
			}
			ops = append(ops, op)
		}

		return nil
	}); err != nil {
		return nil, err
	}
	return ops, nil
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
			subs[i] = subs[len(subs) - 1]
			o.subscribers = subs[:len(o.subscribers) - 1]
		}
	}
}

// addOpToIndexBucket adds the given operation ID to the given index bucket for the given key ID.
func (o *OpLog) addOpToIndexBucket(tx *bolt.Tx, bucket []byte, indexId string, opId int64) error {
	b := tx.Bucket(bucket)

	var key []byte 
	key = append(key, itob(int64(len(indexId)))...)
	key = append(key, []byte(indexId)...)
	key = append(key, itob(opId)...)
	if err := b.Put(key, []byte{}); err != nil {
		return fmt.Errorf("error adding operation to repo index: %w", err)
	}
	return nil
}

// readOpsFromIndexBucket reads all operations from the given index bucket for the given key ID.
func (o *OpLog) readOpsFromIndexBucket(tx *bolt.Tx, bucket []byte, indexId string) ([]int64, error) {
	b := tx.Bucket(bucket)

	var ops []int64
	c := b.Cursor()
	var prefix []byte
	prefix = append(prefix, itob(int64(len(indexId)))...)
	prefix = append(prefix, []byte(indexId)...)
	for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
		ops = append(ops, btoi(k[len(prefix):]))
	}

	return ops, nil
}

type Filter func([]int64)[]int64

func FilterKeepAll() Filter {
	return func(ids []int64) []int64 {
		return ids
	}
}

func FilterLastN(n int64) Filter {
	return func(ids []int64) []int64 {
		if len(ids) > int(n) {
			ids = ids[len(ids)-int(n):]
		}
		return ids
	}
}

func FilterLimitOffset(limit, offset int64) Filter {
	return func(ids []int64) []int64 {
		if len(ids) > int(offset) {
			ids = ids[offset:]
		}
		if len(ids) > int(limit) {
			ids = ids[:limit]
		}
		return ids
	}
}
