package restic

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"slices"

	"go.uber.org/zap"
)

type Snapshot struct {
	Time string `json:"time"`
	Tree string `json:"tree"`
	Paths []string `json:"paths"`
	Hostname string `json:"hostname"`
	Username string `json:"username"`
	Id string `json:"id"`
	ShortId string `json:"short_id"`
}

type BackupEvent struct {
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

	// Error fields
	Error string `json:"error"`
}

// readBackupEvents returns the summary event or an error if the command failed.
func readBackupEvents(cmd *exec.Cmd, output io.Reader, callback func(event *BackupEvent)) (*BackupEvent, error) {
	scanner := bufio.NewScanner(output)
	scanner.Split(bufio.ScanLines)

	// first event is handled specially to detect non-JSON output and fast-path out.
	if scanner.Scan() {
		var event BackupEvent

		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			var bytes = slices.Clone(scanner.Bytes())
			for scanner.Scan() {
				bytes = append(bytes, scanner.Bytes()...)
			}

			jsonErr := fmt.Errorf("command output was not JSON: %w", err)

			if err := cmd.Wait(); err != nil {
				return nil, NewCmdError(cmd, bytes, errors.Join(
					fmt.Errorf("command failed: %w", err),
					fmt.Errorf("command output was not JSON: %w", err),
				))
			}

			return nil, NewCmdError(cmd, bytes, jsonErr)
		}
	}

	// remaining events are parsed as JSON
	var summary *BackupEvent

	for scanner.Scan() {
		var event *BackupEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}

		callback(event)

		if event.MessageType == "summary" {
			summary = event
		}
	}

	if err := scanner.Err(); err != nil {
		return summary, fmt.Errorf("scanner encountered error: %w", err)
	}

	zap.L().Debug("finished reading events", zap.String("command", cmd.String()))

	return summary, nil
}
