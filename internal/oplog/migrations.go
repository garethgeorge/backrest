package oplog

import (
	"errors"
	"fmt"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/oplog/serializationutil"
	"go.etcd.io/bbolt"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var migrations = []func(*OpLog, *bbolt.Tx) error{
	migration001FlowID,
	migration002InstanceID,
	migration003ResetLastValidated,
	migration002InstanceID, // re-run migration002InstanceID to fix improperly set instance IDs
}

var CurrentVersion = int64(len(migrations))

func ApplyMigrations(oplog *OpLog, tx *bbolt.Tx) error {
	var version int64
	versionBytes := tx.Bucket(SystemBucket).Get([]byte("version"))
	if versionBytes == nil {
		version = 0
	} else {
		v, err := serializationutil.Btoi(versionBytes)
		if err != nil {
			return fmt.Errorf("couldn't parse version: %w", err)
		}
		version = v
	}

	startMigration := int(version)
	if startMigration < 0 {
		startMigration = 0
	}
	for idx := startMigration; idx < len(migrations); idx += 1 {
		zap.L().Info("applying oplog migration", zap.Int("migration_no", idx))
		if err := migrations[idx](oplog, tx); err != nil {
			zap.L().Error("failed to apply migration", zap.Int("migration_no", idx), zap.Error(err))
			return fmt.Errorf("couldn't apply migration %d: %w", idx, err)
		}
	}

	if err := tx.Bucket(SystemBucket).Put([]byte("version"), serializationutil.Itob(CurrentVersion)); err != nil {
		return fmt.Errorf("couldn't update version: %w", err)
	}
	return nil
}

func transformOperations(oplog *OpLog, tx *bbolt.Tx, f func(op *v1.Operation) error) error {
	opLogBucket := tx.Bucket(OpLogBucket)

	if opLogBucket == nil {
		return errors.New("oplog bucket not found")
	}

	c := opLogBucket.Cursor()
	for k, v := c.First(); k != nil; k, v = c.Next() {
		op := &v1.Operation{}
		if err := proto.Unmarshal(v, op); err != nil {
			return fmt.Errorf("unmarshal operation: %w", err)
		}

		copy := proto.Clone(op).(*v1.Operation)
		err := f(copy)
		if err != nil {
			return err
		}

		if proto.Equal(copy, op) {
			continue
		}

		if _, err := oplog.deleteOperationHelper(tx, op.Id); err != nil {
			return fmt.Errorf("delete operation: %w", err)
		}
		if err := oplog.addOperationHelper(tx, copy); err != nil {
			return fmt.Errorf("create operation: %w", err)
		}
	}

	return nil
}

// migration001FlowID sets the flow ID for operations that are missing it.
// All operations with the same snapshot ID will have the same flow ID.
func migration001FlowID(oplog *OpLog, tx *bbolt.Tx) error {
	snapshotIdToFlow := make(map[string]int64)

	return transformOperations(oplog, tx, func(op *v1.Operation) error {
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

func migration002InstanceID(oplog *OpLog, tx *bbolt.Tx) error {
	return transformOperations(oplog, tx, func(op *v1.Operation) error {
		if op.InstanceId != "" {
			return nil
		}

		op.InstanceId = "_unassociated_"
		return nil
	})
}

func migration003ResetLastValidated(oplog *OpLog, tx *bbolt.Tx) error {
	return tx.Bucket(SystemBucket).Delete([]byte("last_validated"))
}
