package protoutil

import (
	"errors"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/pkg/restic"
)

func SnapshotToProto(s *restic.Snapshot) *v1.ResticSnapshot {
	return &v1.ResticSnapshot{
		Id:         s.Id,
		UnixTimeMs: s.UnixTimeMs(),
		Tree:       s.Tree,
		Paths:      s.Paths,
		Hostname:   s.Hostname,
		Username:   s.Username,
		Tags:       s.Tags,
		Parent:     s.Parent,
	}
}

func LsEntryToProto(e *restic.LsEntry) *v1.LsEntry {
	return &v1.LsEntry{
		Name:  e.Name,
		Type:  e.Type,
		Path:  e.Path,
		Uid:   int64(e.Uid),
		Gid:   int64(e.Gid),
		Size:  int64(e.Size),
		Mode:  int64(e.Mode),
		Mtime: e.Mtime,
		Atime: e.Atime,
		Ctime: e.Ctime,
	}
}

func BackupProgressEntryToProto(b *restic.BackupProgressEntry) *v1.BackupProgressEntry {
	switch b.MessageType {
	case "summary":
		return &v1.BackupProgressEntry{
			Entry: &v1.BackupProgressEntry_Summary{
				Summary: &v1.BackupProgressSummary{
					FilesNew:            int64(b.FilesNew),
					FilesChanged:        int64(b.FilesChanged),
					FilesUnmodified:     int64(b.FilesUnmodified),
					DirsNew:             int64(b.DirsNew),
					DirsChanged:         int64(b.DirsChanged),
					DirsUnmodified:      int64(b.DirsUnmodified),
					DataBlobs:           int64(b.DataBlobs),
					TreeBlobs:           int64(b.TreeBlobs),
					DataAdded:           int64(b.DataAdded),
					TotalFilesProcessed: int64(b.TotalFilesProcessed),
					TotalBytesProcessed: int64(b.TotalBytesProcessed),
					TotalDuration:       float64(b.TotalDuration),
					SnapshotId:          b.SnapshotId,
				},
			},
		}
	case "status":
		return &v1.BackupProgressEntry{
			Entry: &v1.BackupProgressEntry_Status{
				Status: &v1.BackupProgressStatusEntry{
					PercentDone: b.PercentDone,
					TotalFiles:  int64(b.TotalFiles),
					FilesDone:   int64(b.FilesDone),
					TotalBytes:  int64(b.TotalBytes),
					BytesDone:   int64(b.BytesDone),
					CurrentFile: b.CurrentFiles,
				},
			},
		}
	default:
		return nil
	}
}

// BackupProgressEntryToBackupError converts a BackupProgressEntry to a BackupError if it's type is "error"
func BackupProgressEntryToBackupError(b *restic.BackupProgressEntry) (*v1.BackupProgressError, error) {
	if b.MessageType != "error" {
		return nil, errors.New("BackupProgressEntry is not of type error")
	}

	return &v1.BackupProgressError{
		Item:   b.Item,
		During: b.During,
	}, nil
}

func RetentionPolicyFromProto(p *v1.RetentionPolicy) *restic.RetentionPolicy {
	if p.Policy != nil {
		switch p := p.Policy.(type) {
		case *v1.RetentionPolicy_PolicyKeepAll:
			return nil
		case *v1.RetentionPolicy_PolicyTimeBucketed:
			return &restic.RetentionPolicy{
				KeepDaily:   int(p.PolicyTimeBucketed.Daily),
				KeepHourly:  int(p.PolicyTimeBucketed.Hourly),
				KeepWeekly:  int(p.PolicyTimeBucketed.Weekly),
				KeepMonthly: int(p.PolicyTimeBucketed.Monthly),
				KeepYearly:  int(p.PolicyTimeBucketed.Yearly),
			}
		case *v1.RetentionPolicy_PolicyKeepLastN:
			return &restic.RetentionPolicy{
				KeepLastN: int(p.PolicyKeepLastN),
			}
		}
	}

	return &restic.RetentionPolicy{
		KeepLastN:          int(p.KeepLastN),
		KeepHourly:         int(p.KeepHourly),
		KeepDaily:          int(p.KeepDaily),
		KeepWeekly:         int(p.KeepWeekly),
		KeepMonthly:        int(p.KeepMonthly),
		KeepYearly:         int(p.KeepYearly),
		KeepWithinDuration: p.KeepWithinDuration,
	}
}

func RetentionPolicyToProto(p *restic.RetentionPolicy) *v1.RetentionPolicy {
	return &v1.RetentionPolicy{
		KeepLastN:          int32(p.KeepLastN),
		KeepHourly:         int32(p.KeepHourly),
		KeepDaily:          int32(p.KeepDaily),
		KeepWeekly:         int32(p.KeepWeekly),
		KeepMonthly:        int32(p.KeepMonthly),
		KeepYearly:         int32(p.KeepYearly),
		KeepWithinDuration: p.KeepWithinDuration,
	}
}

func RestoreProgressEntryToProto(p *restic.RestoreProgressEntry) *v1.RestoreProgressEntry {
	return &v1.RestoreProgressEntry{
		MessageType:   p.MessageType,
		TotalFiles:    int64(p.TotalFiles),
		FilesRestored: int64(p.FilesRestored),
		TotalBytes:    int64(p.TotalBytes),
		BytesRestored: int64(p.BytesRestored),
		PercentDone:   p.PercentDone,
	}
}

func RepoStatsToProto(s *restic.RepoStats) *v1.RepoStats {
	return &v1.RepoStats{
		TotalSize:             int64(s.TotalSize),
		TotalUncompressedSize: int64(s.TotalUncompressedSize),
		CompressionRatio:      s.CompressionRatio,
		TotalBlobCount:        int64(s.TotalBlobCount),
		SnapshotCount:         int64(s.SnapshotsCount),
	}
}
