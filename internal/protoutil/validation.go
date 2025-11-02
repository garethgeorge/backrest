package protoutil

import (
	"errors"
	"fmt"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/pkg/restic"
)

var (
	errIDRequired              = errors.New("id is required")
	errModnoRequired           = errors.New("modno is required")
	errFlowIDRequired          = errors.New("flow_id is required")
	errRepoIDRequired          = errors.New("repo_id is required")
	errRepoGUIDRequired        = errors.New("repo_guid is required")
	errPlanIDRequired          = errors.New("plan_id is required")
	errInstanceIDRequired      = errors.New("instance_id is required")
	errUnixTimeStartMsRequired = errors.New("unix_time_start_ms must be non-zero")
)

// ValidateOperation verifies critical properties of the operation proto.
func ValidateOperation(op *v1.Operation) error {
	if op.Id == 0 {
		return errIDRequired
	}
	if op.Modno == 0 {
		return errModnoRequired
	}
	if op.RepoGuid == "" {
		return errRepoGUIDRequired
	}
	if op.FlowId == 0 {
		return errFlowIDRequired
	}
	if op.RepoId == "" {
		return errRepoIDRequired
	}
	if op.PlanId == "" {
		return errPlanIDRequired
	}
	if op.InstanceId == "" {
		return errInstanceIDRequired
	}
	if op.UnixTimeStartMs == 0 {
		return errUnixTimeStartMsRequired
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
