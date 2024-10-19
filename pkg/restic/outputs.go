package restic

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

type Snapshot struct {
	Id              string          `json:"id"`
	Time            string          `json:"time"`
	Tree            string          `json:"tree"`
	Paths           []string        `json:"paths"`
	Hostname        string          `json:"hostname"`
	Username        string          `json:"username"`
	Tags            []string        `json:"tags"`
	Parent          string          `json:"parent"`
	SnapshotSummary SnapshotSummary `json:"summary"`
	unixTimeMs      int64           `json:"-"`
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

type SnapshotSummary struct {
	BackupStart         string `json:"backup_start"`
	BackupEnd           string `json:"backup_end"`
	FilesNew            int64  `json:"files_new"`
	FilesChanged        int64  `json:"files_changed"`
	FilesUnmodified     int64  `json:"files_unmodified"`
	DirsNew             int64  `json:"dirs_new"`
	DirsChanged         int64  `json:"dirs_changed"`
	DirsUnmodified      int64  `json:"dirs_unmodified"`
	DataBlobs           int64  `json:"data_blobs"`
	TreeBlobs           int64  `json:"tree_blobs"`
	DataAdded           int64  `json:"data_added"`
	DataAddedPacked     int64  `json:"data_added_packed"`
	TotalFilesProcessed int64  `json:"total_files_processed"`
	TotalBytesProcessed int64  `json:"total_bytes_processed"`
	unixDurationMs      int64  `json:"-"`
}

// Duration returns the duration of the snapshot in milliseconds.
func (s *SnapshotSummary) DurationMs() int64 {
	if s.unixDurationMs != 0 {
		return s.unixDurationMs
	}
	start, err := time.Parse(time.RFC3339Nano, s.BackupStart)
	if err != nil {
		return 0
	}
	end, err := time.Parse(time.RFC3339Nano, s.BackupEnd)
	if err != nil {
		return 0
	}
	s.unixDurationMs = end.Sub(start).Milliseconds()
	return s.unixDurationMs
}

func (s *Snapshot) Validate() error {
	if err := ValidateSnapshotId(s.Id); err != nil {
		return fmt.Errorf("snapshot.id invalid: %v", err)
	}
	if s.Time == "" || s.UnixTimeMs() == 0 {
		return fmt.Errorf("snapshot.time invalid: %v", s.Time)
	}
	return nil
}

type BackupProgressEntry struct {
	// Common fields
	MessageType string `json:"message_type"` // "summary" or "status" or "error"

	// Error fields
	Error  any    `json:"error"`
	During string `json:"during"`
	Item   string `json:"item"`

	// Summary fields
	FilesNew            int64   `json:"files_new"`
	FilesChanged        int64   `json:"files_changed"`
	FilesUnmodified     int64   `json:"files_unmodified"`
	DirsNew             int64   `json:"dirs_new"`
	DirsChanged         int64   `json:"dirs_changed"`
	DirsUnmodified      int64   `json:"dirs_unmodified"`
	DataBlobs           int64   `json:"data_blobs"`
	TreeBlobs           int64   `json:"tree_blobs"`
	DataAdded           int64   `json:"data_added"`
	TotalFilesProcessed int64   `json:"total_files_processed"`
	TotalBytesProcessed int64   `json:"total_bytes_processed"`
	TotalDuration       float64 `json:"total_duration"`
	SnapshotId          string  `json:"snapshot_id"`

	// Status fields
	PercentDone  float64  `json:"percent_done"`
	TotalFiles   int64    `json:"total_files"`
	FilesDone    int64    `json:"files_done"`
	TotalBytes   int64    `json:"total_bytes"`
	BytesDone    int64    `json:"bytes_done"`
	CurrentFiles []string `json:"current_files"`
}

func (b *BackupProgressEntry) Validate() error {
	if b.MessageType == "summary" && b.SnapshotId != "" {
		if err := ValidateSnapshotId(b.SnapshotId); err != nil {
			return err
		}
	}

	return nil
}

// readBackupProgressEntries returns the summary event or an error if the command failed.
func readBackupProgressEntries(output io.Reader, callback func(event *BackupProgressEntry)) (*BackupProgressEntry, error) {
	scanner := bufio.NewScanner(output)
	scanner.Split(bufio.ScanLines)

	nonJSONOutput := bytes.NewBuffer(nil)

	var summary *BackupProgressEntry

	// remaining events are parsed as JSON
	for scanner.Scan() {
		var event BackupProgressEntry
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			nonJSONOutput.Write(scanner.Bytes())
			continue
		}
		if err := event.Validate(); err != nil {
			nonJSONOutput.Write(scanner.Bytes())
			continue
		}
		if callback != nil {
			callback(&event)
		}
		if event.MessageType == "summary" {
			summary = &event
		}
	}
	if err := scanner.Err(); err != nil {
		return summary, newErrorWithOutput(err, nonJSONOutput.String())
	}
	if summary == nil {
		return nil, newErrorWithOutput(errors.New("no summary event found"), nonJSONOutput.String())
	}
	return summary, nil
}

type LsEntry struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Path  string `json:"path"`
	Uid   int64  `json:"uid"`
	Gid   int64  `json:"gid"`
	Size  int64  `json:"size"`
	Mode  int64  `json:"mode"`
	Mtime string `json:"mtime"`
	Atime string `json:"atime"`
	Ctime string `json:"ctime"`
}

func (e *LsEntry) ToProto() *v1.LsEntry {
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

type ForgetResult struct {
	Keep   []Snapshot `json:"keep"`
	Remove []Snapshot `json:"remove"`
}

func (r *ForgetResult) Validate() error {
	for _, s := range r.Keep {
		if err := ValidateSnapshotId(s.Id); err != nil {
			return err
		}
	}
	for _, s := range r.Remove {
		if err := ValidateSnapshotId(s.Id); err != nil {
			return err
		}
	}
	return nil
}

type RestoreProgressEntry struct {
	MessageType    string  `json:"message_type"` // "summary" or "status"
	SecondsElapsed float64 `json:"seconds_elapsed"`
	TotalBytes     int64   `json:"total_bytes"`
	BytesRestored  int64   `json:"bytes_restored"`
	TotalFiles     int64   `json:"total_files"`
	FilesRestored  int64   `json:"files_restored"`
	PercentDone    float64 `json:"percent_done"`
}

func (e *RestoreProgressEntry) Validate() error {
	if e.MessageType != "summary" && e.MessageType != "status" {
		return fmt.Errorf("message_type must be 'summary' or 'status', got %v", e.MessageType)
	}
	return nil
}

// readRestoreProgressEntries returns the summary event or an error if the command failed.
func readRestoreProgressEntries(output io.Reader, callback func(event *RestoreProgressEntry)) (*RestoreProgressEntry, error) {
	scanner := bufio.NewScanner(output)
	scanner.Split(bufio.ScanLines)

	nonJSONOutput := bytes.NewBuffer(nil)

	var summary *RestoreProgressEntry

	// remaining events are parsed as JSON
	for scanner.Scan() {
		var event RestoreProgressEntry
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			nonJSONOutput.Write(scanner.Bytes())
			continue
		}
		if err := event.Validate(); err != nil {
			// skip it. Best effort parsing, restic will return with a non-zero exit code if it fails.
			nonJSONOutput.Write(scanner.Bytes())
			continue
		}

		if callback != nil {
			callback(&event)
		}
		if event.MessageType == "summary" {
			summary = &event
		}
	}

	if err := scanner.Err(); err != nil {
		return summary, newErrorWithOutput(err, nonJSONOutput.String())
	}

	if summary == nil {
		return nil, newErrorWithOutput(errors.New("no summary event found"), nonJSONOutput.String())
	}

	return summary, nil
}

func ValidateSnapshotId(id string) error {
	if len(id) != 64 {
		return fmt.Errorf("restic may be out of date (check with `restic self-upgrade`): snapshot ID must be 64 chars, got %v chars", len(id))
	}
	return nil
}

type RepoStats struct {
	TotalSize              int64   `json:"total_size"`
	TotalUncompressedSize  int64   `json:"total_uncompressed_size"`
	CompressionRatio       float64 `json:"compression_ratio"`
	CompressionProgress    float64 `json:"compression_progress"`
	CompressionSpaceSaving float64 `json:"compression_space_saving"`
	TotalBlobCount         int64   `json:"total_blob_count"`
	SnapshotsCount         int64   `json:"snapshots_count"`
}
