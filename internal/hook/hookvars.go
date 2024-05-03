package hook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
	"time"

	"github.com/alessio/shellescape"
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
	default:
		return "unknown"
	}
}

func (v HookVars) FormatTime(t time.Time) string {
	return t.Format(time.RFC3339)
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
	size := v.number(val)
	sizes := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	i := 0
	prev := size
	for size > 1000 {
		size /= 1000
		prev = size
		i++
	}
	return fmt.Sprintf("%d.%03d %s", size, prev, sizes[i])
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
	case v1.Hook_CONDITION_SNAPSHOT_END:
		return v.renderTemplate(templateForSnapshotEnd)
	case v1.Hook_CONDITION_ANY_ERROR:
		return v.renderTemplate(templateForError)
	case v1.Hook_CONDITION_SNAPSHOT_ERROR:
		return v.renderTemplate(templateForError)
	case v1.Hook_CONDITION_SNAPSHOT_WARNING:
		return v.renderTemplate(templateForError)
	default:
		return "unknown event", nil
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

var templateForSnapshotEnd = `
Backrest Notification for Snapshot End
Task: "{{ .Task }}" at {{ .FormatTime .CurTime }}
Event: {{ .EventName .Event }}
Repo: {{ .Repo.Id }} 
Plan: {{ .Plan.Id }} 
Snapshot: {{ .SnapshotId }}
{{ if .Error -}}
Failed to create snapshot: {{ .Error }}
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

var templateForError = `
Backrest Notification for Error
Task: "{{ .Task }}" at {{ .FormatTime .CurTime }}
{{ if .Error -}}
Error: {{ .Error }}
{{ end }}
{{ if .Items -}}

`

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
