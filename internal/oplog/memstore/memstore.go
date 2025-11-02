package memstore

import (
	"slices"
	"sync"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/protoutil"
	"google.golang.org/protobuf/proto"
)

type MemStore struct {
	mu         sync.Mutex
	operations map[int64]*v1.Operation
	nextID     int64
	nextModno  int64
}

var _ oplog.OpStore = &MemStore{}

func NewMemStore() *MemStore {
	return &MemStore{
		operations: make(map[int64]*v1.Operation),
	}
}

func (m *MemStore) Version() (int64, error) {
	return 0, nil
}

func (m *MemStore) SetVersion(version int64) error {
	return nil
}

func (m *MemStore) idsForQuery(q oplog.Query) []int64 {
	ids := make([]int64, 0, len(m.operations))
	for id := range m.operations {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, func(i, j int64) int { return int(i - j) })
	ids = slices.DeleteFunc(ids, func(id int64) bool {
		op := m.operations[id]
		return !q.Match(op)
	})

	if q.Offset > 0 {
		if int(q.Offset) >= len(ids) {
			ids = nil
		} else {
			ids = ids[q.Offset:]
		}
	}

	if q.Limit > 0 && len(ids) > q.Limit {
		ids = ids[:q.Limit]
	}

	if q.Reversed {
		slices.Reverse(ids)
	}

	return ids
}

func (m *MemStore) Query(q oplog.Query, f func(*v1.Operation) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ids := m.idsForQuery(q)

	for _, id := range ids {
		if err := f(proto.Clone(m.operations[id]).(*v1.Operation)); err != nil {
			if err == oplog.ErrStopIteration {
				break
			}
			return err
		}
	}

	return nil
}

func (m *MemStore) QueryMetadata(q oplog.Query, f func(meta oplog.OpMetadata) error) error {
	for _, id := range m.idsForQuery(q) {
		op := m.operations[id]
		if err := f(oplog.OpMetadata{
			ID:             op.Id,
			Modno:          op.Modno,
			FlowID:         op.FlowId,
			OriginalID:     op.OriginalId,
			OriginalFlowID: op.OriginalFlowId,
			Status:         op.Status,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (m *MemStore) Transform(q oplog.Query, f func(*v1.Operation) (*v1.Operation, error)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ids := m.idsForQuery(q)

	changes := make(map[int64]*v1.Operation)

	for _, id := range ids {
		if op, err := f(proto.Clone(m.operations[id]).(*v1.Operation)); err != nil {
			if err == oplog.ErrStopIteration {
				break
			}
			return err
		} else if op != nil {
			m.nextModno++
			op.Modno = m.nextModno
			changes[id] = op
		}
	}

	// Apply changes after the loop to avoid modifying the map until the transaction is complete.
	for id, op := range changes {
		m.operations[id] = op
	}

	return nil
}

func (m *MemStore) Add(op ...*v1.Operation) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, o := range op {
		m.nextID++
		o.Id = m.nextID
		m.nextModno++
		o.Modno = m.nextModno
		if o.FlowId == 0 {
			o.FlowId = o.Id
		}
		if err := protoutil.ValidateOperation(o); err != nil {
			return err
		}
	}

	for _, o := range op {
		m.operations[o.Id] = o
	}
	return nil
}

func (m *MemStore) Get(opID int64) (*v1.Operation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	op, ok := m.operations[opID]
	if !ok {
		return nil, oplog.ErrNotExist
	}
	return op, nil
}

func (m *MemStore) Delete(opID ...int64) ([]*v1.Operation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ops := make([]*v1.Operation, 0, len(opID))
	for _, id := range opID {
		ops = append(ops, m.operations[id])
		delete(m.operations, id)
	}
	return ops, nil
}

func (m *MemStore) Update(op ...*v1.Operation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, o := range op {
		m.nextModno++
		o.Modno = m.nextModno
		if err := protoutil.ValidateOperation(o); err != nil {
			return err
		}
		if _, ok := m.operations[o.Id]; !ok {
			return oplog.ErrNotExist
		}
		m.operations[o.Id] = o
	}
	return nil
}

func (m *MemStore) GetHighestOpIDAndModno(q oplog.Query) (int64, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var highestID int64
	var highestModno int64
	for id, op := range m.operations {
		if !q.Match(op) {
			continue
		}
		if id > highestID {
			highestID = id
		}
		if op.Modno > highestModno {
			highestModno = op.Modno
		}
	}
	return highestID, highestModno, nil
}
