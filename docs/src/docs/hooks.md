# Hooks

Hooks in Backrest allow you to respond to various operation lifecycle events, enabling automation and monitoring of your backup operations. This page is the reference for hook conditions, actions, error policies, and templates. For a task-oriented walkthrough of setting up notifications, see the [Notifications guide](/guides/notifications).

Hooks can be attached to a **repository** (fires for all activity in that repo, including `_system_` maintenance) or to a **plan** (fires only for that plan's operations). For each event, repository hooks run before plan hooks, and each hook fires at most once per event — its first matching condition wins.

## Event Types

Hooks can be triggered by the following events:

### Snapshot Events
- `CONDITION_SNAPSHOT_START`: Triggered when a backup operation begins and will complete before the snapshot starts. The [Error Handling](#error-handling) configuration can be used to stop the backup if the command isn't successful.
- `CONDITION_SNAPSHOT_END`: Triggered when a backup operation completes (regardless of success/failure)
- `CONDITION_SNAPSHOT_SUCCESS`: Triggered when a backup operation completes successfully
- `CONDITION_SNAPSHOT_ERROR`: Triggered when a backup operation fails
- `CONDITION_SNAPSHOT_WARNING`: Triggered when a backup operation encounters non-fatal issues (e.g. some files could not be read; a snapshot is still created)
- `CONDITION_SNAPSHOT_SKIPPED`: Triggered when a backup is skipped because nothing changed (only with the plan's *skip if unchanged* option enabled)

### Prune Events
- `CONDITION_PRUNE_START`: Triggered when a prune operation begins
- `CONDITION_PRUNE_SUCCESS`: Triggered when a prune operation completes successfully
- `CONDITION_PRUNE_ERROR`: Triggered when a prune operation fails

### Check Events
- `CONDITION_CHECK_START`: Triggered when a check operation begins
- `CONDITION_CHECK_SUCCESS`: Triggered when a check operation completes successfully
- `CONDITION_CHECK_ERROR`: Triggered when a check operation fails

### Forget Events
- `CONDITION_FORGET_START`: Triggered when a forget operation begins
- `CONDITION_FORGET_SUCCESS`: Triggered when a forget operation completes successfully
- `CONDITION_FORGET_ERROR`: Triggered when a forget operation fails

### General Events
- `CONDITION_ANY_ERROR`: Triggered when any operation fails

## Notification Services

Backrest supports multiple notification services for hook delivery:

| Service  | Description                            | Documentation                                                                                       |
| -------- | -------------------------------------- | --------------------------------------------------------------------------------------------------- |
| Discord  | Send notifications to Discord channels | [Discord Webhooks Guide](https://support.discord.com/hc/en-us/articles/228383668-Intro-to-Webhooks) |
| Slack    | Send notifications to Slack channels   | [Slack Webhooks Guide](https://api.slack.com/messaging/webhooks)                                    |
| Gotify   | Send notifications via Gotify server   | [Gotify Documentation](https://github.com/gotify/server)                                            |
| Telegram | Send messages via a Telegram bot (bot token + chat ID) | [Telegram Bot API](https://core.telegram.org/bots)                                   |
| Shoutrrr | Multi-provider notification service (ntfy, Pushover, email, and more) | [Shoutrrr Documentation](https://containrrr.dev/shoutrrr/v0.8/)      |
| Healthchecks | Ping Healthchecks.io monitoring URLs | [Healthchecks API](https://healthchecks.io/docs/http_api/)                                          |
| Webhook  | Send the rendered template to any HTTP endpoint (GET or POST) | —                                                                            |
| Command  | Execute custom shell commands          | See [command cookbook](/cookbooks/command-hook-examples)                                            |

::: warning Command hooks and the scratch Docker image
Command hooks run through a shell, which the minimal `ghcr.io/garethgeorge/backrest:scratch` image does not include — use the default (alpine-based) `latest` image if you rely on command hooks.
:::

### Healthchecks.io Integration

The Healthchecks hook type is specifically designed to integrate with [Healthchecks.io](https://healthchecks.io/) or compatible self-hosted instances. 

When configured, Backrest automatically appends the correct status endpoint to your webhook URL based on the event type:
- **Start events** (e.g., `CONDITION_SNAPSHOT_START`): Appends `/start` to the URL.
- **Error events** (e.g., `CONDITION_SNAPSHOT_ERROR`): Appends `/fail` to the URL.
- **Log events**: Appends `/log` to the URL.
- **Success & Other events**: Pings the base URL directly.

It also sends the formatted template summary as the HTTP POST body in plain text, which Healthchecks.io captures as the "ping payload". This is particularly useful for reading error messages or backup statistics directly from the Healthchecks.io dashboard.

## Error Handling

Every hook has an **error behavior** that determines how Backrest responds if the hook itself fails (a non-zero exit code for command hooks, or a delivery failure for notifications):

- `ON_ERROR_IGNORE`: Continue execution despite the hook failure
- `ON_ERROR_CANCEL`: Stop the operation, marking it cancelled. Error-condition hooks are *not* triggered, which makes this suitable for pre-backup checks that should skip a run without raising an error (only meaningful on `*_START` conditions).
- `ON_ERROR_FATAL`: Stop the operation, marking it failed. Error-condition hooks *are* triggered (only meaningful on `*_START` conditions).
- `ON_ERROR_RETRY_1MINUTE`: Retry the hook every minute until it succeeds
- `ON_ERROR_RETRY_10MINUTES`: Retry every 10 minutes
- `ON_ERROR_RETRY_EXPONENTIAL_BACKOFF`: Retry with doubling delays (10s, 20s, 40s, …) capped at 1 hour

While a hook is retrying, the operation that triggered it stays pending; retry policies are a good fit for notification hooks that should survive transient network failures.

## Template System

Hooks use Go templates for formatting notifications and scripts. The following variables and functions are available:

### Available Variables

| Variable        | Type                         | Description                 | Example Usage                     |
| --------------- | ---------------------------- | --------------------------- | --------------------------------- |
| `Event`         | `v1.Hook_Condition`          | The triggering event        | <code v-pre>{{ .Event }}</code>                    |
| `Task`          | `string`                     | Task name                   | <code v-pre>{{ .Task }}</code>                     |
| `Repo`          | `v1.Repo`                    | Repository information      | <code v-pre>{{ .Repo.Id }}</code>                  |
| `Plan`          | `v1.Plan`                    | Plan information            | <code v-pre>{{ .Plan.Id }}</code>                  |
| `SnapshotId`    | `string`                     | ID of associated snapshot   | <code v-pre>{{ .SnapshotId }}</code>               |
| `SnapshotStats` | `restic.BackupProgressEntry` | Backup operation statistics | See example below                 |
| `CurTime`       | `time.Time`                  | Current timestamp           | <code v-pre>{{ .FormatTime .CurTime }}</code>      |
| `Duration`      | `time.Duration`              | Operation duration          | <code v-pre>{{ .FormatDuration .Duration }}</code> |
| `Error`         | `string`                     | Error message if applicable | <code v-pre>{{ .Error }}</code>                    |

### Helper Functions

| Function           | Description                     | Example                             |
| ------------------ | ------------------------------- | ----------------------------------- |
| `.Summary`         | Generates default event summary | <code v-pre>{{ .Summary }}</code>                    |
| `.FormatTime`      | Formats timestamp               | <code v-pre>{{ .FormatTime .CurTime }}</code>        |
| `.FormatDuration`  | Formats time duration           | <code v-pre>{{ .FormatDuration .Duration }}</code>   |
| `.FormatSizeBytes` | Formats byte sizes              | <code v-pre>{{ .FormatSizeBytes 1048576 }}</code>    |
| `.ShellEscape`     | Escapes strings for shell usage | <code v-pre>{{ .ShellEscape "my string" }}</code>    |
| `.JsonMarshal`     | Converts value to JSON          | <code v-pre>{{ .JsonMarshal .SnapshotStats }}</code> |

## Default Summary Template

Below is the implementation of the `.Summary` function, which you can use as a reference for creating custom templates:

<div v-pre>

```text
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
{{ end }}
```
</div>

## See Also

- [Notifications guide](/guides/notifications) — step-by-step setup for the services above
- [Command hook examples](/cookbooks/command-hook-examples) — pre-backup checks, filesystem snapshots, desktop notifications
- [Slack Block Kit examples](/cookbooks/slack-hook-build-kit-examples) — richly formatted Slack messages
