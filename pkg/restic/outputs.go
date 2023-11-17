package restic

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"slices"
	"time"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
)

type LsEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Path string `json:"path"`
	Uid int `json:"uid"`
	Gid int `json:"gid"`
	Size int `json:"size"`
	Mode int `json:"mode"`
	Mtime string `json:"mtime"`
	Atime string `json:"atime"`
	Ctime string `json:"ctime"`
}

func (e *LsEntry) ToProto() *v1.LsEntry {
	return &v1.LsEntry{
		Name: e.Name,
		Type: e.Type,
		Path: e.Path,
		Uid: int64(e.Uid),
		Gid: int64(e.Gid),
		Size: int64(e.Size),
		Mode: int64(e.Mode),
		Mtime: e.Mtime,
		Atime: e.Atime,
		Ctime: e.Ctime,
	}
}

type Snapshot struct {
	Id string `json:"id"`
	Time string `json:"time"`
	Tree string `json:"tree"`
	Paths []string `json:"paths"`
	Hostname string `json:"hostname"`
	Username string `json:"username"`
	Tags []string `json:"tags"`
	Parent string `json:"parent"`
	unixTimeMs int64 `json:"-"`
}

func (s *Snapshot) ToProto() *v1.ResticSnapshot {
	
	return &v1.ResticSnapshot{
		Id: s.Id,
		UnixTimeMs: s.UnixTimeMs(),
		Tree: s.Tree,
		Paths: s.Paths,
		Hostname: s.Hostname,
		Username: s.Username,
		Tags: s.Tags,
		Parent: s.Parent,
	}
}

func (s *Snapshot) UnixTimeMs() int64 {
	if s.unixTimeMs != 0 {
		return s.unixTimeMs
	}
	t, err := time.Parse(time.RFC3339Nano, s.Time)
	if err != nil {
		t = time.Unix(0, 0)
	}
	s.unixTimeMs = t.UnixMilli()
	return s.unixTimeMs
}

type BackupProgressEntry struct {
	// Common fields
	MessageType string `json:"message_type"` // "summary" or "status"

	// Summary fields
	FilesNew int `json:"files_new"`
	FilesChanged int `json:"files_changed"`
	FilesUnmodified int `json:"files_unmodified"`
	DirsNew int `json:"dirs_new"`
	DirsChanged int `json:"dirs_changed"`
	DirsUnmodified int `json:"dirs_unmodified"`
	DataBlobs int `json:"data_blobs"`
	TreeBlobs int `json:"tree_blobs"`
	DataAdded int `json:"data_added"`
	TotalFilesProcessed int `json:"total_files_processed"`
	TotalBytesProcessed int `json:"total_bytes_processed"`
	TotalDuration float64 `json:"total_duration"`
	SnapshotId string `json:"snapshot_id"`

	// Status fields
	PercentDone float64 `json:"percent_done"`
	TotalFiles int `json:"total_files"`
	FilesDone int `json:"files_done"`
	TotalBytes int `json:"total_bytes"`
	BytesDone int `json:"bytes_done"`
}

func (b *BackupProgressEntry) ToProto() *v1.BackupProgressEntry {
	switch b.MessageType {
	case "summary":
		return &v1.BackupProgressEntry{
			Entry: &v1.BackupProgressEntry_Summary{
				Summary: &v1.BackupProgressSummary{
					FilesNew: int64(b.FilesNew),
					FilesChanged: int64(b.FilesChanged),
					FilesUnmodified: int64(b.FilesUnmodified),
					DirsNew: int64(b.DirsNew),
					DirsChanged: int64(b.DirsChanged),
					DirsUnmodified: int64(b.DirsUnmodified),
					DataBlobs: int64(b.DataBlobs),
					TreeBlobs: int64(b.TreeBlobs),
					DataAdded: int64(b.DataAdded),
					TotalFilesProcessed: int64(b.TotalFilesProcessed),
					TotalBytesProcessed: int64(b.TotalBytesProcessed),
					TotalDuration: float64(b.TotalDuration),
					SnapshotId: b.SnapshotId,
				},
			},
		}
	case "status":
		return &v1.BackupProgressEntry{
			Entry: &v1.BackupProgressEntry_Status{
				Status: &v1.BackupProgressStatusEntry{
					PercentDone: b.PercentDone,
					TotalFiles: int64(b.TotalFiles),
					FilesDone: int64(b.FilesDone),
					TotalBytes: int64(b.TotalBytes),
					BytesDone: int64(b.BytesDone),
				},
			},
		}
	default:
		log.Fatalf("unknown message type: %s", b.MessageType)
		return nil 
	}
}

// readBackupProgressEntrys returns the summary event or an error if the command failed.
func readBackupProgressEntries(cmd *exec.Cmd, output io.Reader, callback func(event *BackupProgressEntry)) (*BackupProgressEntry, error) {
	scanner := bufio.NewScanner(output)
	scanner.Split(bufio.ScanLines)

	// first event is handled specially to detect non-JSON output and fast-path out.
	if scanner.Scan() {
		var event BackupProgressEntry

		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			var bytes = slices.Clone(scanner.Bytes())
			for scanner.Scan() {
				bytes = append(bytes, scanner.Bytes()...)
			}

			return nil, NewCmdError(cmd, bytes, fmt.Errorf("command output was not JSON: %w", err))
		}
	}

	// remaining events are parsed as JSON
	var summary *BackupProgressEntry

	for scanner.Scan() {
		var event *BackupProgressEntry
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}

		if callback != nil {
			callback(event)
		}

		if event.MessageType == "summary" {
			summary = event
		}
	}

	if err := scanner.Err(); err != nil {
		return summary, fmt.Errorf("scanner encountered error: %w", err)
	}

	return summary, nil
}

func readLs(output io.Reader) (*Snapshot, []*LsEntry, error) {
	scanner := bufio.NewScanner(output)
	scanner.Split(bufio.ScanLines)

	if !scanner.Scan() {
		return nil, nil, fmt.Errorf("failed to read first line, expected snapshot info")
	}

	var snapshot *Snapshot
	if err := json.Unmarshal(scanner.Bytes(), &snapshot); err != nil {
		return nil, nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	var entries []*LsEntry
	for scanner.Scan() {
		var entry *LsEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
		entries = append(entries, entry)
	}
	return snapshot, entries, nil
}