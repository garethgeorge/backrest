package protoutil

import (
	"errors"
	"fmt"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/pkg/restic"
	"go.uber.org/zap"
)

// ValidateOperation verifies critical properties of the operation proto.
func ValidateOperation(op *v1.Operation) error {
	if op.Id == 0 {
		return errors.New("operation.id is required")
	}
	if op.FlowId == 0 {
		return errors.New("operation.flow_id is required")
	}
	if op.RepoId == "" {
		return errors.New("operation.repo_id is required")
	}
	if op.PlanId == "" {
		return errors.New("operation.plan_id is required")
	}
	if op.InstanceId == "" {
		zap.L().Warn("operation.instance_id should typically be set")
	}
	if op.SnapshotId != "" {
		if err := restic.ValidateSnapshotId(op.SnapshotId); err != nil {
			return fmt.Errorf("operation.snapshot_id is invalid: %w", err)
		}
	}
	return nil
}

// ValidateSnapshot verifies critical properties of the snapshot proto representation.
func ValidateSnapshot(s *v1.ResticSnapshot) error {
	if s.Id == "" {
		return errors.New("snapshot.id is required")
	}
	if s.UnixTimeMs == 0 {
		return errors.New("snapshot.unix_time_ms must be non-zero")
	}
	if err := restic.ValidateSnapshotId(s.Id); err != nil {
		return err
	}
	return nil
}
