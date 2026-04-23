package tasks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
	"time"

	"al.essio.dev/pkg/shellescape"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/pkg/restic"
)

// HookVars is the set of variables that are available to a hook. Some of these are optional.
// NOTE: names of HookVars may change between versions of backrest. This is not a guaranteed stable API.
// when names change hooks will require updating.
type HookVars struct {
	Task          string                      // the name of the task that triggered the hook.
	Event         v1.Hook_Condition           // the event that triggered the hook.
	Repo          *v1.Repo                    // the v1.Repo that triggered the hook.
	Plan          *v1.Plan                    // the v1.Plan that triggered the hook.
	SnapshotId    string                      // the snapshot ID that triggered the hook.
	SnapshotStats *restic.BackupProgressEntry // the summary of the backup operation.
	CurTime       time.Time                   // the current time as time.Time
	Duration      time.Duration               // the duration of the operation that triggered the hook.
	Error         string                      // the error that caused the hook to run as a string.
}

func (v HookVars) EventName(cond v1.Hook_Condition) string {
	switch cond {
	case v1.Hook_CONDITION_SNAPSHOT_START:
		return "snapshot start"
	case v1.Hook_CONDITION_SNAPSHOT_END:
		return "snapshot end"
	case v1.Hook_CONDITION_ANY_ERROR:
		return "error"
	case v1.Hook_CONDITION_SNAPSHOT_ERROR:
		return "snapshot error"
	case v1.Hook_CONDITION_SNAPSHOT_WARNING:
		return "snapshot warning"
	case v1.Hook_CONDITION_SNAPSHOT_SUCCESS:
		return "snapshot success"
	case v1.Hook_CONDITION_SNAPSHOT_SKIPPED:
		return "snapshot skipped"
	case v1.Hook_CONDITION_CHECK_START:
		return "check start"
	case v1.Hook_CONDITION_CHECK_ERROR:
		return "check error"
	case v1.Hook_CONDITION_CHECK_SUCCESS:
		return "check success"
	case v1.Hook_CONDITION_PRUNE_START:
		return "prune start"
	case v1.Hook_CONDITION_PRUNE_ERROR:
		return "prune error"
	case v1.Hook_CONDITION_PRUNE_SUCCESS:
		return "prune success"
	case v1.Hook_CONDITION_FORGET_START:
		return "forget start"
	case v1.Hook_CONDITION_FORGET_ERROR:
		return "forget error"
	case v1.Hook_CONDITION_FORGET_SUCCESS:
		return "forget success"
	default:
		return "unknown"
	}
}

func (v HookVars) FormatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

func (v HookVars) FormatDuration(d time.Duration) string {
	return d.Truncate(time.Millisecond).String()
}

func (v HookVars) number(n any) int {
	switch n := n.(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	default:
		return 0
	}
}

func (v HookVars) FormatSizeBytes(val any) string {
	size := float64(v.number(val))
	sizes := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	i := 0
	for size > 1000 {
		size /= 1000
		i++
	}
	return fmt.Sprintf("%.3f %s", size, sizes[i])
}

func (v HookVars) IsError(cond v1.Hook_Condition) bool {
	return cond == v1.Hook_CONDITION_ANY_ERROR || cond == v1.Hook_CONDITION_SNAPSHOT_ERROR
}

func (v HookVars) ShellEscape(s string) string {
	return shellescape.Quote(s)
}

func (v HookVars) JsonMarshal(s any) string {
	b, err := json.Marshal(s)
	if err != nil {
		return ""
	}
	return string(b)
}

func (v HookVars) Summary() (string, error) {
	switch v.Event {
	case v1.Hook_CONDITION_SNAPSHOT_START:
		return v.renderTemplate(templateForSnapshotStart)
	case v1.Hook_CONDITION_SNAPSHOT_END, v1.Hook_CONDITION_SNAPSHOT_WARNING, v1.Hook_CONDITION_SNAPSHOT_SUCCESS:
		return v.renderTemplate(templateForSnapshotEnd)
	default:
		return v.renderTemplate(templateDefault)
	}
}

func (v HookVars) renderTemplate(templ string) (string, error) {
	t, err := template.New("t").Parse(templ)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, v)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

var templateDefault = `
{{ if .Error -}}
Backrest Error
Task: {{ .Task }} at {{ .FormatTime .CurTime }}
Event: {{ .EventName .Event }}
Repo: {{ .Repo.Id }}
Error: {{ .Error }}
{{ else -}}
Backrest Notification
Task: {{ .Task }} at {{ .FormatTime .CurTime }}
Event: {{ .EventName .Event }}
{{ end }}
`

var templateForSnapshotEnd = `
Backrest Snapshot Notification
Task: {{ .Task }} at {{ .FormatTime .CurTime }}
Event: {{ .EventName .Event }}
Snapshot: {{ .SnapshotId }}
{{ if .Error -}}
Error: {{ .Error }}
{{ else -}}
{{ if .SnapshotStats -}}

Overview:
- Data added: {{ .FormatSizeBytes .SnapshotStats.DataAdded }}
- Total files processed: {{ .SnapshotStats.TotalFilesProcessed }}
- Total bytes processed: {{ .FormatSizeBytes .SnapshotStats.TotalBytesProcessed }}

Backup Statistics:
- Files new: {{ .SnapshotStats.FilesNew }}
- Files changed: {{ .SnapshotStats.FilesChanged }}
- Files unmodified: {{ .SnapshotStats.FilesUnmodified }}
- Dirs new: {{ .SnapshotStats.DirsNew }}
- Dirs changed: {{ .SnapshotStats.DirsChanged }}
- Dirs unmodified: {{ .SnapshotStats.DirsUnmodified }}
- Data blobs: {{ .SnapshotStats.DataBlobs }}
- Tree blobs: {{ .SnapshotStats.TreeBlobs }}
- Total duration: {{ .SnapshotStats.TotalDuration }}s
{{ end }}
{{ end }}`

var templateForSnapshotStart = `
Backrest Notification for Snapshot Start
Task: "{{ .Task }}" at {{ .FormatTime .CurTime }}
Event: {{ .EventName .Event }}
Repo: {{ .Repo.Id }} 
Plan: {{ .Plan.Id }} 
Paths: 
{{ range .Plan.Paths -}}
 - {{ . }}
{{ end }}`
