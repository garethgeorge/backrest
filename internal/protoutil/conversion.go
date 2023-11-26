package protoutil

import (
	v1 "github.com/garethgeorge/resticui/gen/go/v1"
	"github.com/garethgeorge/resticui/pkg/restic"
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
				},
			},
		}
	default:
		return nil
	}
}
