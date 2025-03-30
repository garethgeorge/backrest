package oplog

import (
	"fmt"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var migrations = []func(*OpLog) error{
	migration001FlowID,
	migration002InstanceID,
	migrationNoop,
	migration002InstanceID, // re-run migration002InstanceID to fix improperly set instance IDs
	migration003DeduplicateIndexedSnapshots,
}

var CurrentVersion = int64(len(migrations))

func ApplyMigrations(oplog *OpLog) error {
	startMigration, err := oplog.store.Version()
	if err != nil {
		zap.L().Error("failed to get migration version", zap.Error(err))
		return fmt.Errorf("couldn't get migration version: %w", err)
	}
	if startMigration < 0 {
		startMigration = 0
	} else if startMigration > CurrentVersion {
		zap.S().Warnf("oplog spec %d is greater than the latest known spec %d. Were you previously running a newer version of backrest? Ensure that your install is up to date.", startMigration, CurrentVersion)
		return fmt.Errorf("oplog spec %d is greater than the latest known spec %d", startMigration, CurrentVersion)
	}

	for idx := startMigration; idx < int64(len(migrations)); idx += 1 {
		zap.L().Info("oplog applying data migration", zap.Int64("migration_no", idx))
		if err := migrations[idx](oplog); err != nil {
			zap.L().Error("failed to apply data migration", zap.Int64("migration_no", idx), zap.Error(err))
			return fmt.Errorf("couldn't apply migration %d: %w", idx, err)
		}
		if err := oplog.store.SetVersion(idx + 1); err != nil {
			zap.L().Error("failed to set migration version, database may be corrupt", zap.Int64("migration_no", idx), zap.Error(err))
			return fmt.Errorf("couldn't set migration version %d: %w", idx, err)
		}
	}

	return nil
}

func transformOperations(oplog *OpLog, f func(op *v1.Operation) error) error {
	oplog.store.Transform(SelectAll, func(op *v1.Operation) (*v1.Operation, error) {
		copy := proto.Clone(op).(*v1.Operation)
		err := f(copy)
		if err != nil {
			return nil, err
		}

		if proto.Equal(copy, op) {
			return nil, nil
		}

		return copy, nil
	})

	return nil
}

// migration001FlowID sets the flow ID for operations that are missing it.
// All operations with the same snapshot ID will have the same flow ID.
func migration001FlowID(oplog *OpLog) error {
	snapshotIdToFlow := make(map[string]int64)

	return transformOperations(oplog, func(op *v1.Operation) error {
		if op.FlowId != 0 {
			return nil
		}

		if op.SnapshotId == "" {
			op.FlowId = op.Id
			return nil
		}

		if flowId, ok := snapshotIdToFlow[op.SnapshotId]; ok {
			op.FlowId = flowId
		} else {
			snapshotIdToFlow[op.SnapshotId] = op.Id
			op.FlowId = op.Id
		}

		return nil
	})
}

func migration002InstanceID(oplog *OpLog) error {
	return transformOperations(oplog, func(op *v1.Operation) error {
		if op.InstanceId != "" {
			return nil
		}

		op.InstanceId = "_unassociated_"
		return nil
	})
}

func migration003DeduplicateIndexedSnapshots(oplog *OpLog) error {
	var snapshotIDs = make(map[string]struct{})
	var deleteIDs []int64
	if err := oplog.Query(SelectAll, func(op *v1.Operation) error {
		if _, ok := op.Op.(*v1.Operation_OperationIndexSnapshot); ok {
			if _, ok := snapshotIDs[op.SnapshotId]; ok {
				deleteIDs = append(deleteIDs, op.Id)
			} else {
				snapshotIDs[op.SnapshotId] = struct{}{}
			}
		}
		return nil
	}); err != nil {
		return err
	}

	if len(deleteIDs) == 0 {
		return nil
	}
	_, err := oplog.store.Delete(deleteIDs...)
	return err
}

// migrationNoop is a migration that does nothing; replaces deprecated migrations.
func migrationNoop(oplog *OpLog) error {
	return nil
}
