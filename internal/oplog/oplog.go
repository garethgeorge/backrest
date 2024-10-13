package oplog

import (
	"errors"
	"slices"
	"sync"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

type OperationEvent int

const (
	OPERATION_ADDED OperationEvent = iota
	OPERATION_UPDATED
	OPERATION_DELETED
)

var (
	ErrStopIteration = errors.New("stop iteration")
	ErrNotExist      = errors.New("operation does not exist")
	ErrExist         = errors.New("operation already exists")

	NullOPID = int64(0)
)

type Subscription = func(ops []*v1.Operation, event OperationEvent)

type subAndQuery struct {
	f *Subscription
	q Query
}

type OpLog struct {
	store OpStore

	subscribersMu sync.Mutex
	subscribers   []subAndQuery
}

func NewOpLog(store OpStore) (*OpLog, error) {
	o := &OpLog{
		store: store,
	}

	if err := ApplyMigrations(o); err != nil {
		return nil, err
	}

	return o, nil
}

func (o *OpLog) curSubscribers() []subAndQuery {
	o.subscribersMu.Lock()
	defer o.subscribersMu.Unlock()
	return slices.Clone(o.subscribers)
}

func (o *OpLog) notify(ops []*v1.Operation, event OperationEvent) {
	for _, sub := range o.curSubscribers() {
		notifyOps := make([]*v1.Operation, 0, len(ops))
		for _, op := range ops {
			if sub.q.Match(op) {
				notifyOps = append(notifyOps, op)
			}
		}
		if len(notifyOps) > 0 {
			(*sub.f)(notifyOps, event)
		}
	}
}

func (o *OpLog) Query(q Query, f func(*v1.Operation) error) error {
	return o.store.Query(q, f)
}

func (o *OpLog) Subscribe(q Query, f *Subscription) {
	o.subscribers = append(o.subscribers, subAndQuery{f: f, q: q})
}

func (o *OpLog) Unsubscribe(f *Subscription) error {
	for i, sub := range o.subscribers {
		if sub.f == f {
			o.subscribers = append(o.subscribers[:i], o.subscribers[i+1:]...)
			return nil
		}
	}
	return errors.New("subscription not found")
}

func (o *OpLog) Get(opID int64) (*v1.Operation, error) {
	return o.store.Get(opID)
}

func (o *OpLog) Add(ops ...*v1.Operation) error {
	for _, o := range ops {
		if o.Id != 0 {
			return errors.New("operation already has an ID, OpLog.Add is expected to set the ID")
		}
	}

	if err := o.store.Add(ops...); err != nil {
		return err
	}

	o.notify(ops, OPERATION_ADDED)
	return nil
}

func (o *OpLog) Update(ops ...*v1.Operation) error {
	for _, o := range ops {
		if o.Id == 0 {
			return errors.New("operation does not have an ID, OpLog.Update is expected to have an ID")
		}
	}

	if err := o.store.Update(ops...); err != nil {
		return err
	}

	o.notify(ops, OPERATION_UPDATED)
	return nil
}

func (o *OpLog) Delete(opID ...int64) error {
	removedOps, err := o.store.Delete(opID...)
	if err != nil {
		return err
	}

	o.notify(removedOps, OPERATION_DELETED)
	return nil
}

func (o *OpLog) Transform(q Query, f func(*v1.Operation) (*v1.Operation, error)) error {
	return o.store.Transform(q, f)
}

type OpStore interface {
	Query(q Query, f func(*v1.Operation) error) error
	Get(opID int64) (*v1.Operation, error)
	Add(op ...*v1.Operation) error
	Update(op ...*v1.Operation) error              // returns the previous values of the updated operations OR an error
	Delete(opID ...int64) ([]*v1.Operation, error) // returns the deleted operations OR an error
	Transform(q Query, f func(*v1.Operation) (*v1.Operation, error)) error
	Version() (int64, error)
	SetVersion(version int64) error
}

type Query struct {
	// Filter by fields
	OpIDs      []int64
	PlanID     string
	RepoID     string
	SnapshotID string
	FlowID     int64
	InstanceID string

	// Pagination
	Limit    int
	Offset   int
	Reversed bool

	opIDmap map[int64]struct{}
}

var SelectAll = Query{}

func (q *Query) buildOpIDMap() {
	if len(q.OpIDs) != len(q.opIDmap) {
		q.opIDmap = make(map[int64]struct{}, len(q.OpIDs))
		for _, opID := range q.OpIDs {
			q.opIDmap[opID] = struct{}{}
		}
	}
}

func (q *Query) Match(op *v1.Operation) bool {
	if len(q.OpIDs) > 0 {
		q.buildOpIDMap()
		if _, ok := q.opIDmap[op.Id]; !ok {
			return false
		}
	}

	if q.InstanceID != "" && op.InstanceId != q.InstanceID {
		return false
	}

	if q.PlanID != "" && op.PlanId != q.PlanID {
		return false
	}

	if q.RepoID != "" && op.RepoId != q.RepoID {
		return false
	}

	if q.SnapshotID != "" && op.SnapshotId != q.SnapshotID {
		return false
	}

	if q.FlowID != 0 && op.FlowId != q.FlowID {
		return false
	}

	return true
}
