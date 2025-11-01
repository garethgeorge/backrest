package oplog

import v1 "github.com/garethgeorge/backrest/gen/go/v1"

type Query struct {
	// Filter by fields
	OpIDs                 []int64
	PlanID                *string
	RepoGUID              *string
	DeprecatedRepoID      *string // Deprecated: use RepoGUID instead
	SnapshotID            *string
	FlowID                *int64
	InstanceID            *string
	OriginalInstanceKeyid *string
	OriginalID            *int64
	OriginalFlowID        *int64
	ModnoGte              *int64

	// Pagination
	Limit    int
	Offset   int
	Reversed bool

	opIDmap map[int64]struct{}
}

func (q Query) SetOpIDs(opIDs []int64) Query {
	q.OpIDs = opIDs
	return q
}

func (q Query) SetPlanID(planID string) Query {
	q.PlanID = &planID
	return q
}

func (q Query) SetRepoGUID(repoGUID string) Query {
	q.RepoGUID = &repoGUID
	return q
}

func (q Query) SetSnapshotID(snapshotID string) Query {
	q.SnapshotID = &snapshotID
	return q
}

func (q Query) SetFlowID(flowID int64) Query {
	q.FlowID = &flowID
	return q
}

func (q Query) SetInstanceID(instanceID string) Query {
	q.InstanceID = &instanceID
	return q
}

func (q Query) SetOriginalInstanceKeyid(instanceKeyid string) Query {
	q.OriginalInstanceKeyid = &instanceKeyid
	return q
}

func (q Query) SetOriginalID(originalID int64) Query {
	q.OriginalID = &originalID
	return q
}

func (q Query) SetOriginalFlowID(originalFlowID int64) Query {
	q.OriginalFlowID = &originalFlowID
	return q
}

func (q Query) SetModnoGte(modnoGte int64) Query {
	q.ModnoGte = &modnoGte
	return q
}

func (q Query) SetLimit(limit int) Query {
	q.Limit = limit
	return q
}

func (q Query) SetOffset(offset int) Query {
	q.Offset = offset
	return q
}

func (q Query) SetReversed(reversed bool) Query {
	q.Reversed = reversed
	return q
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

	if q.InstanceID != nil && op.InstanceId != *q.InstanceID {
		return false
	}

	if q.PlanID != nil && op.PlanId != *q.PlanID {
		return false
	}

	if q.RepoGUID != nil && op.RepoGuid != *q.RepoGUID {
		return false
	}

	if q.SnapshotID != nil && op.SnapshotId != *q.SnapshotID {
		return false
	}

	if q.FlowID != nil && op.FlowId != *q.FlowID {
		return false
	}

	if q.OriginalID != nil && op.OriginalId != *q.OriginalID {
		return false
	}

	if q.OriginalFlowID != nil && op.OriginalFlowId != *q.OriginalFlowID {
		return false
	}

	if q.ModnoGte != nil && op.Modno < *q.ModnoGte {
		return false
	}

	return true
}
