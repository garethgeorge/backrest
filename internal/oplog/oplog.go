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

func (o *OpLog) QueryMetadata(q Query, f func(OpMetadata) error) error {
	return o.store.QueryMetadata(q, f)
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
		if o.Modno == 0 {
			o.Modno = NewRandomModno(0)
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
		o.Modno = NewRandomModno(o.Modno)
	}

	if err := o.store.Update(ops...); err != nil {
		return err
	}

	o.notify(ops, OPERATION_UPDATED)
	return nil
}

// Set is an alias for Update that does not increment the modno, provided for use by the syncapi.
func (o *OpLog) Set(op *v1.Operation) error {
	var err error
	if op.Id == 0 {
		err = o.store.Add(op)
	} else {
		err = o.store.Update(op)
		if errors.Is(err, ErrNotExist) {
			err = o.store.Add(op)
		}
	}
	if err != nil {
		return err
	}
	o.notify([]*v1.Operation{op}, OPERATION_UPDATED)
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
	// Query returns all operations that match the query.
	Query(q Query, f func(*v1.Operation) error) error
	// QueryMetadata is like Query, but only returns metadata about the operations.
	// this is useful for very high performance scans that don't deserialize the operation itself.
	QueryMetadata(q Query, f func(OpMetadata) error) error
	// Get returns the operation with the given ID.
	Get(opID int64) (*v1.Operation, error)
	// Add adds the given operations to the store.
	Add(op ...*v1.Operation) error
	// Update updates the given operations in the store.
	Update(op ...*v1.Operation) error
	// Delete removes the operations with the given IDs from the store, and returns the removed operations.
	Delete(opID ...int64) ([]*v1.Operation, error)
	// Transform applies the given function to each operation that matches the query.
	Transform(q Query, f func(*v1.Operation) (*v1.Operation, error)) error
	// Version returns the current data version
	Version() (int64, error)
	// SetVersion sets the data version
	SetVersion(version int64) error
}

// OpMetadata is a struct that contains metadata about an operation without fetching the operation itself.
type OpMetadata struct {
	ID             int64
	FlowID         int64
	Modno          int64
	OriginalID     int64
	OriginalFlowID int64
}
