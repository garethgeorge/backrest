# Retention & Repo Health

Restic repositories need periodic maintenance: removing snapshots that are no longer wanted, reclaiming unused storage, and verifying data integrity. Backrest automates the three restic operations responsible for this. This guide explains what each one does and, in particular, how they interact.

## The Three-Stage Pipeline

| Stage | Operation | What it does | What it does *not* do |
| --- | --- | --- | --- |
| 1 | **Forget** | Applies your retention policy: removes aged-out *snapshot records* | Doesn't delete file data or free space |
| 2 | **Prune** | Deletes data chunks no remaining snapshot references, repacking as needed | Doesn't decide *which* snapshots to keep |
| 3 | **Check** | Verifies repository structure (and optionally re-reads data) | Doesn't fix or clean anything |

Note that storage usage does not decrease when forget runs. Space is only reclaimed when prune runs.

## Retention Policies

Each plan has a retention policy, applied automatically by a **forget** operation after every successful backup that produces a snapshot:

- **Keep last N** — the N most recent snapshots are kept.
- **Time-bucketed** — keep the latest snapshot in each of the last N hourly, daily, weekly, monthly, and yearly buckets (any subset; maps directly to restic's `--keep-hourly`, `--keep-daily`, etc., and can be combined with a keep-last-N floor). This keeps frequent recent snapshots and progressively fewer older ones.
- **Keep all** — no forget is ever scheduled; you manage snapshot lifecycle manually (or via the restic CLI).

For example, a time-bucketed policy of *7 daily + 4 weekly + 12 monthly* lets you restore yesterday's version of a file, any day from the past week, any week from the past month, and any month from the past year. Because restic deduplicates data across snapshots, storage growth under a policy like this levels off over time.

<img src="/screenshots/retention-policy.png" alt="Retention policy form with count-based, time-bucketed, and none options" style="max-width: 100%; border-radius: 8px; margin-bottom: 20px;">

### Retention Is Scoped Per Plan

Backrest tags every snapshot with `plan:<plan-id>` and `created-by:<instance-id>`, and each plan's forget only considers snapshots carrying its own tags. Multiple plans, and multiple machines, can therefore share one repository without their retention policies interfering with each other's snapshots, or with snapshots you create manually with the restic CLI.

### Repository-Level Forget (Override)

Repositories can optionally define their own **forget policy with a schedule**. This is for setups where you want retention applied uniformly across everything in the repository on a fixed cadence, rather than plan-by-plan after each backup.

::: warning A repo-level forget schedule replaces per-plan retention
If a repository has a scheduled forget policy, Backrest stops running per-plan forgets for that repository entirely; the repo-level policy is applied to all snapshots (grouped by tags) on its schedule instead. The two modes are mutually exclusive.
:::

Repo-level forget runs appear under the `_system_` plan and skip themselves (recorded as cancelled) when no new backups have happened since the last run.

## Prune

Prune scans the repository for data no snapshot references and deletes it, repacking partially-used pack files where worthwhile. It's configured in **repository settings**:

<img src="/screenshots/repo-policies.png" alt="Repository scheduling section with prune and check policy configuration" style="max-width: 100%; border-radius: 8px; margin-bottom: 20px;">

- **Schedule** — monthly is typical. Prune is I/O-intensive and, on cloud storage, bandwidth-intensive, so running it more frequently provides little benefit.
- **Max unused percent** (default 25%) — how much unreferenced data restic may leave behind to avoid expensive repacking. Higher values make prunes faster and cheaper but leave more unused storage; lower values reclaim storage more tightly at the cost of more repacking (and, on cloud providers, more download/upload). A max-unused-bytes form is also available if you prefer an absolute cap.

::: tip Cost tuning for cloud storage
Repacking downloads data before re-uploading it. If your provider bills for egress, a higher max-unused setting (10–25%) usually costs less overall than aggressive pruning.
:::

## Check

Check verifies repository integrity so that corruption is detected before a restore is needed. It is configured in **repository settings**:

- **Structure only** (default): verifies the repository's internal consistency (indexes, pack metadata) without downloading file data. This mode is inexpensive and suitable for monthly runs on any repository.
- **Read data subset (%)**: additionally downloads and cryptographically verifies that percentage of the actual data, sampling different packs each run. Over many runs, a small percentage accumulates into broad coverage.

::: warning
Setting read data to 100% downloads the entire repository on every check. On egress-billed cloud storage this can be expensive, and on large repositories it is slow. For most setups, 0% (structure only) or a low single-digit percentage is sufficient.
:::

When prune and check are both due, Backrest orders them so that check runs after prune and verifies the repository state that prune leaves behind.

## Stats and Internal Housekeeping

Two more operations appear in the history that require no configuration:

- **Stats** — records repository size and snapshot counts (at most once per day, plus after prune and repo-level forget). This feeds the storage graphs in the repository view's Stats tab.
- **Collect garbage** — trims Backrest's own *operation history* (the oplog) so the UI stays fast: old operation records age out on type-specific retention rules, and records for forgotten snapshots are cleaned up. This never touches restic data; it is internal bookkeeping only.

## A Recommended Baseline

A reasonable starting point for a personal server or homelab:

| Setting | Value |
| --- | --- |
| Plan retention | Time-bucketed: 7 daily, 4 weekly, 12 monthly |
| Prune schedule | Every 30 days, Last Run Time clock |
| Prune max unused | 10–25% (higher if egress is billed) |
| Check schedule | Every 30 days, Last Run Time clock |
| Check read data | 0% (structure only); a few % if you want deep verification |

Consider also adding a [notification hook](/guides/notifications) on error conditions so that maintenance failures do not go unnoticed.

## See Also

- [Scheduling Backups](/guides/scheduling) — schedule types and clock semantics used by all of the above
- [Operational Model](/docs/operations) — how these tasks queue, prioritize, and record their results
- Restic's own docs on [forget & prune](https://restic.readthedocs.io/en/latest/060_forget.html) and [check](https://restic.readthedocs.io/en/latest/080_check.html)
