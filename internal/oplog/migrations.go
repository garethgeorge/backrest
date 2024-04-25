package oplog

import (
	"errors"
	"fmt"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"
)

var migrations = []func(*OpLog, *bbolt.Tx) error{
	migration001FlowID,
}

var CurrentVersion = int32(len(migrations))

func ApplyMigrations(oplog *OpLog, tx *bbolt.Tx, version int64) (int64, error) {
	startMigration := int(version)
	if startMigration < 0 {
		startMigration = 0
	}
	for idx := startMigration; idx < len(migrations); idx += 1 {
		if err := migrations[idx](oplog, tx); err != nil {
			return version, fmt.Errorf("couldn't apply migration %d: %w", idx, err)
		}
	}

	return int64(CurrentVersion), nil
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
