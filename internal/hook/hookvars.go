package hook

import (
	"bytes"
	"encoding/json"
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

func (v HookVars) IsError(cond v1.Hook_Condition) bool {
	return cond == v1.Hook_CONDITION_ANY_ERROR || cond == v1.Hook_CONDITION_SNAPSHOT_ERROR
}

func (v HookVars) ShellEscape(s string) string {
	return shellescape.Quote(s)
}

func (v HookVars) JSONEscape(s string) string {
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

var templateForSnapshotEnd = `Task: "{{ .Task }}" at {{ .FormatTime .CurTime }}
Repo: {{ .Repo.Id }} Plan: {{ .Plan.Id }} Snapshot: {{ .SnapshotId }}
{{ if .Error -}}
Failed to create snapshot: {{ .Error }}
{{ else -}}
{{ if .SnapshotStats -}}
Stats:
 - Files new: {{ .SnapshotStats.FilesNew }}
 - Files changed: {{ .SnapshotStats.FilesChanged }}
 - Files unmodified: {{ .SnapshotStats.FilesUnmodified }}
 - Dirs new: {{ .SnapshotStats.DirsNew }}
 - Dirs changed: {{ .SnapshotStats.DirsChanged }}
 - Dirs unmodified: {{ .SnapshotStats.DirsUnmodified }}
 - Data blobs: {{ .SnapshotStats.DataBlobs }}
 - Tree blobs: {{ .SnapshotStats.TreeBlobs }}
 - Data added: {{ .SnapshotStats.DataAdded }} bytes
 - Total files processed: {{ .SnapshotStats.TotalFilesProcessed }}
 - Total bytes processed: {{ .SnapshotStats.TotalBytesProcessed }} bytes
 - Total duration: {{ .SnapshotStats.TotalDuration }}s
{{ end }}
{{ end }}`

var templateForError = `Task: "{{ .Task }}" at {{ .FormatTime .CurTime }}
{{ if .Error }}
Error: {{ .Error }}
{{ end }}`

var templateForSnapshotStart = `Task: "{{ .Task }}" at {{ .FormatTime .CurTime }}
Repo: {{ .Repo.Id }} Plan: {{ .Plan.Id }}
Paths: {{ range .Plan.Paths }}
 - {{ . }}
{{ end }}`
