package tasks

import (
	"fmt"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
	"github.com/garethgeorge/backrest/internal/oplog/indexutil"
)

// FlowIDForSnapshotID returns the flow ID associated with the backup task that created snapshot ID or 0 if not found.
func FlowIDForSnapshotID(log *oplog.OpLog, snapshotID string) (int64, error) {
	var flowID int64
	if err := log.ForEach(oplog.Query{SnapshotId: snapshotID}, indexutil.CollectAll(), func(op *v1.Operation) error {
		if _, ok := op.Op.(*v1.Operation_OperationBackup); !ok {
			return nil
		}
		flowID = op.FlowId
		return nil
	}); err != nil {
		return 0, fmt.Errorf("get flow id for snapshot %q : %w", snapshotID, err)
	}
	return flowID, nil
}
