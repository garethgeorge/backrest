# Operational Model

This page is a reference for how Backrest works internally: the orchestrator, its task queue, the operation lifecycle, and restic integration. This information is not required for everyday use, but it is helpful for interpreting the operation history and understanding unexpected behavior.

For task-oriented guidance, see [Scheduling Backups](/guides/scheduling) and [Retention & Repo Health](/guides/repo-health).

## The Orchestrator

At Backrest's core is an orchestrator that executes tasks from a **time-ordered priority queue, one at a time**. Serial execution is deliberate: restic repositories use locking, and running operations sequentially avoids lock contention and keeps resource usage predictable.

Each task carries a scheduled run time; the queue dequeues whatever is due next. When multiple tasks are due at the same moment, priority breaks the tie (highest first):

| Priority | Task | Consequence |
| --- | --- | --- |
| Interactive | Anything you trigger from the UI | UI-triggered tasks run ahead of scheduled work |
| Prune | Scheduled prune | Runs before check when both are due |
| Check | Scheduled check | Verifies the post-prune state |
| Index snapshots | Snapshot indexing | |
| Forget | Post-backup retention | |
| Default | Backups and most tasks | |
| Stats | Repository statistics | Runs when nothing else is due |

After a recurring task finishes, the orchestrator computes its next run time from its [schedule](/guides/scheduling) and re-enqueues it. One-off tasks (a manual backup, a restore) run once.

On startup and after **every configuration change**, the queue is rebuilt from scratch: one backup task per plan, plus prune, check, and repo-level forget tasks per repository, plus a garbage-collection task. A watchdog also monitors for system clock jumps (sleep, hibernation, NTP corrections) and rebuilds schedules if the clock drifts more than ~30 seconds from expectations.

## Task Types

| Task | Trigger | What it runs |
| --- | --- | --- |
| `backup` | Plan schedule, or **Backup Now** | `restic backup` with the plan's paths, excludes, and flags |
| `forget` | Automatically after each successful backup (per-plan retention) | `restic forget` scoped to the plan's snapshots |
| `scheduled_forget` | Repo-level forget policy schedule | `restic forget` across the repository, grouped by tags |
| `prune` | Repo prune policy schedule, or UI button | `restic prune` |
| `check` | Repo check policy schedule, or UI button | `restic check` |
| `index_snapshots` | After backups; repo added; UI button | `restic snapshots` — syncs Backrest's view of the repo |
| `restore` | UI restore action | `restic restore` |
| `stats` | At most every 24h; after prune and scheduled forget | `restic stats` — feeds the storage graphs |
| `run_command` | UI **Run Command** | An arbitrary restic command you type |
| `hook` | Fired by conditions on other operations | Your hook's action (command, notification, …) |
| `collect_garbage` | Startup, then every 24h | Internal cleanup of Backrest's operation log (never restic data) |

Repo-level tasks (`prune`, `check`, `scheduled_forget`, and friends) are displayed under the synthetic **`_system_`** plan, since they belong to the repository rather than to any plan you created.

## Operation Lifecycle

Every task run is recorded as an **operation** in Backrest's operation log (the *oplog*), which is what the UI's history tree renders. Operations move through:

```
PENDING ──► INPROGRESS ──► SUCCESS
                       ├──► WARNING          (completed with caveats)
                       ├──► ERROR
                       └──► USER_CANCELLED / SYSTEM_CANCELLED
```

Notes on specific statuses:

- **WARNING on backups indicates a partial backup**: the snapshot was created, but some files could not be read (typically due to permissions or file locks). Follow-up tasks (forget, indexing) still run. The operation log lists the affected files.
- **After a crash or hard reboot**, any operation that was INPROGRESS is marked ERROR at startup, since Backrest cannot determine how far it progressed. Stale PENDING entries are cleared and rescheduled. An error block that appears immediately after a reboot usually reflects this recovery behavior rather than a new failure.
- **Cancellation**: cancelling a queued operation removes it; cancelling a running one interrupts the underlying restic process. Repo-level scheduled forgets also cancel *themselves* (SYSTEM_CANCELLED) when there's nothing new to do.
- **Logs**: each operation stores a summary of restic's output inline (truncated for very large outputs) and full logs are viewable from the operation's detail view while retained.

### Operation Log Retention

The `collect_garbage` task keeps the oplog bounded so the UI stays fast. Old operation records age out on type-specific rules (long-lived history for prune/check/stats, shorter for routine forgets), and records tied to snapshots that have been forgotten are cleaned up. This is Backrest-side bookkeeping only; it never deletes backup data.

## How Backups Are Tagged

Every snapshot Backrest creates carries two restic tags:

- `plan:<plan-id>` — which plan created it
- `created-by:<instance-id>` — which Backrest installation created it

These tags are how retention stays scoped: a plan's forget only touches snapshots with its own plan/instance tags, so multiple plans, multiple machines, and even manual restic CLI snapshots coexist safely in one repository. Backups also run with `--exclude-caches` (skipping directories marked with `CACHEDIR.TAG`) and pass the previous snapshot as `--parent` for faster change detection.

## Hook Execution

Operations fire [hook](/docs/hooks) conditions at lifecycle points (start, success, error, …). For each event, Backrest evaluates repository hooks first, then plan hooks; each hook runs at most once per event (its first matching condition wins) and executes as its own `hook` operation, visible in the history under the operation that triggered it.

A failing hook applies its **error policy**: ignore, cancel the parent operation, mark it fatal, or retry (fixed 1‑minute/10‑minute delays, or exponential backoff capped at an hour). Retrying hooks keep the parent operation pending until they resolve.

## Restic Integration

### Binary Management

Backrest pins a specific restic version per release (currently 0.19.x) and resolves the binary in this order:

1. The `--restic-cmd` flag, if set.
2. The `BACKREST_RESTIC_COMMAND` environment variable, if set.
3. A `restic` on the system `$PATH`, if its version is compatible.
4. Otherwise, Backrest downloads its pinned version from restic's GitHub releases into the data directory, verifying the SHA256 checksum against the restic maintainers' GPG-signed manifest, and keeps it updated across Backrest upgrades.

### Command Execution

For every restic invocation, Backrest injects the repository's environment variables (credentials, etc.), appends the repository's extra flags, and applies the repository's command prefix settings (CPU/IO niceness on Unix systems, useful for keeping background backups from competing with foreground work). `${VAR}` references in the URI, env vars, and flags are expanded from Backrest's own process environment.

If an operation fails, click its block in the history tree to view the full diagnostic log.

## See Also

- [Scheduling Backups](/guides/scheduling) — schedule types and clock semantics
- [Retention & Repo Health](/guides/repo-health) — forget/prune/check policies and interactions
- [Configuration & Paths](/docs/configuration) — where the oplog, logs, and config live on disk
- [Hooks](/docs/hooks) — conditions, actions, and error policies in full
