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
	migrationNoop,          // migration003Reset	Validated,
	migration002InstanceID, // re-run migration002InstanceID to fix improperly set instance IDs
}

var CurrentVersion = int64(len(migrations))

func ApplyMigrations(oplog *OpLog) error {
	startMigration := oplog.store.Version()
	if startMigration < 0 {
		startMigration = 0
	}

	for idx := startMigration; idx < len(migrations); idx += 1 {
		zap.L().Info("applying oplog migration", zap.Int("migration_no", idx))
		if err := migrations[idx](oplog); err != nil {
			zap.L().Error("failed to apply migration", zap.Int("migration_no", idx), zap.Error(err))
			return fmt.Errorf("couldn't apply migration %d: %w", idx, err)
		}
		if err := oplog.store.SetVersion(idx + 1); err != nil {
			zap.L().Error("failed to set migration version, database may be corrupt", zap.Int("migration_no", idx), zap.Error(err))
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

// migrationNoop is a migration that does nothing; replaces deprecated migrations.
func migrationNoop(oplog *OpLog) error {
	return nil
}
