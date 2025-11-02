package protoutil

import (
	"errors"
	"reflect"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
)

func OpSelectorToQuery(sel *v1.OpSelector) (oplog.Query, error) {
	if sel == nil {
		return oplog.Query{}, errors.New("empty selector")
	}

	q := oplog.Query{
		RepoGUID:              sel.RepoGuid,
		PlanID:                sel.PlanId,
		SnapshotID:            sel.SnapshotId,
		FlowID:                sel.FlowId,
		InstanceID:            sel.InstanceId,
		OriginalInstanceKeyid: sel.OriginalInstanceKeyid,
		ModnoGte:              sel.ModnoGte,
	}
	if len(sel.Ids) > 0 && !reflect.DeepEqual(q, oplog.Query{}) {
		return oplog.Query{}, errors.New("cannot specify both query and ids")
	}
	q.OpIDs = sel.Ids
	return q, nil
}
