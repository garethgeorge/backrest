package restic

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"slices"
	"time"

	v1 "github.com/garethgeorge/resticui/gen/go/v1"
)

type Snapshot struct {
	Id         string   `json:"id"`
	Time       string   `json:"time"`
	Tree       string   `json:"tree"`
	Paths      []string `json:"paths"`
	Hostname   string   `json:"hostname"`
	Username   string   `json:"username"`
	Tags       []string `json:"tags"`
	Parent     string   `json:"parent"`
	unixTimeMs int64    `json:"-"`
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
	MessageType string `json:"message_type"` // "summary" or "status"

	// Summary fields
	FilesNew            int     `json:"files_new"`
	FilesChanged        int     `json:"files_changed"`
	FilesUnmodified     int     `json:"files_unmodified"`
	DirsNew             int     `json:"dirs_new"`
	DirsChanged         int     `json:"dirs_changed"`
	DirsUnmodified      int     `json:"dirs_unmodified"`
	DataBlobs           int     `json:"data_blobs"`
	TreeBlobs           int     `json:"tree_blobs"`
	DataAdded           int     `json:"data_added"`
	TotalFilesProcessed int     `json:"total_files_processed"`
	TotalBytesProcessed int     `json:"total_bytes_processed"`
	TotalDuration       float64 `json:"total_duration"`
	SnapshotId          string  `json:"snapshot_id"`

	// Status fields
	PercentDone float64 `json:"percent_done"`
	TotalFiles  int     `json:"total_files"`
	FilesDone   int     `json:"files_done"`
	TotalBytes  int     `json:"total_bytes"`
	BytesDone   int     `json:"bytes_done"`
}

func (b *BackupProgressEntry) Validate() error {
	if b.MessageType == "summary" {
		if b.SnapshotId == "" {
			return errors.New("summary message must have snapshot_id")
		}
		if err := ValidateSnapshotId(b.SnapshotId); err != nil {
			return err
		}
	}

	return nil
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
		if err := event.Validate(); err != nil {
			return nil, err
		}
	}

	// remaining events are parsed as JSON
	var summary *BackupProgressEntry

	for scanner.Scan() {
		var event *BackupProgressEntry
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
		if err := event.Validate(); err != nil {
			return nil, err
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

type LsEntry struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Path  string `json:"path"`
	Uid   int    `json:"uid"`
	Gid   int    `json:"gid"`
	Size  int    `json:"size"`
	Mode  int    `json:"mode"`
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

func ValidateSnapshotId(id string) error {
	if len(id) != 64 {
		return fmt.Errorf("restic may be out of date (check with `restic self-upgrade`): snapshot ID must be 64 chars, got %v chars", len(id))
	}
	return nil
}
