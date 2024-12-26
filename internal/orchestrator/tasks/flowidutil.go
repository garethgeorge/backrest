package tasks

import (
	"fmt"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog"
)

// FlowIDForSnapshotID returns the flow ID associated with the backup task that created snapshot ID or 0 if not found.
func FlowIDForSnapshotID(log *oplog.OpLog, snapshotID string) (int64, error) {
	var flowID int64
	if err := log.Query(oplog.Query{SnapshotID: &snapshotID}, func(op *v1.Operation) error {
		if _, ok := op.Op.(*v1.Operation_OperationBackup); !ok {
			return nil
		}
		if flowID != 0 {
			return fmt.Errorf("multiple flow IDs found for snapshot %q", snapshotID)
		}
		flowID = op.FlowId
		return nil
	}); err != nil {
		return 0, fmt.Errorf("get flow id for snapshot %q : %w", snapshotID, err)
	}
	return flowID, nil
}
