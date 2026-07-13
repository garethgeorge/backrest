# Scheduling Backups

Every recurring activity in Backrest (plan backups, and per-repository prune, check, and forget) is driven by the same schedule system. This guide explains the schedule types, the clock options, and how Backrest behaves when a machine is off or asleep.

## Schedule Types

| Type | Meaning | Example |
| --- | --- | --- |
| **Cron** | Fire at wall-clock times matching a cron expression | `0 2 * * *` — daily at 2:00 AM |
| **Interval (hours)** | Fire every N hours (N ≥ 1) | every 4 hours |
| **Interval (days)** | Fire every N days (N ≥ 1) | every 2 days |
| **Disabled** | Never fire | pause a plan without deleting it |

Cron expressions use the standard five fields (minute, hour, day-of-month, month, day-of-week). [crontab.guru](https://crontab.guru/) is useful for validating expressions.

<img src="/screenshots/schedule-form.png" alt="Schedule form showing the schedule type options and reference clock selection" style="max-width: 100%; border-radius: 8px; margin-bottom: 20px;">

```
0 2 * * *      daily at 02:00
*/30 * * * *   every 30 minutes
0 3 * * 1      Mondays at 03:00
0 1 1 * *      first of the month at 01:00
```

## Choosing a Clock

Each schedule also has a **clock**, which determines the reference point for computing the next run:

| Clock | Next run is computed from | Behavior |
| --- | --- | --- |
| **Local** (default) | Current time, local timezone | Cron fires at local wall-clock times. Intervals fire at *now + interval*, re-evaluated after each run. |
| **UTC** | Current time, UTC | Same as Local but cron times are interpreted in UTC. |
| **Last Run Time** | When the operation last ran | Intervals fire at *last run + interval* — a fixed cadence that "catches up" after downtime. |

The clocks differ in how they handle interval schedules:

- With **Local/UTC**, "every 24 hours" means 24 hours after Backrest last *evaluated* the schedule. If the machine was asleep at the scheduled moment, the interval effectively restarts when it wakes.
- With **Last Run Time**, "every 24 hours" means 24 hours after the operation last *completed*. If the machine wakes up late, the run is already overdue and fires promptly.

::: tip Recommendations
- **Always-on machines** (servers, NAS): cron with the **Local** clock — predictable wall-clock timing, easy to place backups in quiet hours.
- **Sometimes-off machines** (laptops, desktops): interval with the **Last Run Time** clock — an overdue backup runs soon after the machine wakes, regardless of when it was last on.
- **Infrequent maintenance** (prune/check, monthly): **Last Run Time**, so a missed window does not delay the operation by a full additional period.
:::

::: info First run
A newly created interval schedule with the Last Run Time clock has no "last run" yet, so Backrest uses the creation time as the reference: the first backup fires one interval after the plan is created. Use **Backup Now** to run one immediately.
:::

## When the Machine Is Off or Asleep

Backrest maintains a queue of upcoming tasks, each with a scheduled time. The following behaviors apply:

- **Missed cron fires are not replayed.** If the machine is off at 2 AM, a `0 2 * * *` backup simply waits for the next 2 AM. For machines with unpredictable uptime, prefer interval + Last Run Time.
- **Clock jumps are handled.** Backrest watches for the system clock drifting from its expected timeline (sleep/hibernate, NTP corrections) and recomputes all scheduled tasks when it detects a jump of more than ~30 seconds.
- **Config changes reschedule everything.** Editing any plan or repository resets the queue and recomputes every schedule.

## Scheduling Health Operations

Prune, check, and repository-level forget schedules live in **repository settings** (not on plans), because they maintain the repository as a whole. Their operations appear in the UI under the synthetic `_system_` plan.

Two orchestration behaviors are relevant here:

- Operations on a repository run one at a time; a long backup and a prune queue behind each other rather than conflicting over repository locks.
- When prune and check are due together, prune runs first, so check verifies the repository's post-cleanup state.

See [Retention & Repo Health](/guides/repo-health) for what these operations do and how to pick their policies.

## Skip If Unchanged

Plans have a **Skip if unchanged** option: when enabled, a scheduled backup that finds no file changes produces no new snapshot. The run is recorded as *skipped* (firing the `CONDITION_SNAPSHOT_SKIPPED` hook rather than success/error hooks), and follow-up forget and indexing work is skipped as well. This prevents frequent schedules on mostly-idle machines from accumulating identical snapshots.

## Recipes

**Nightly backup at 2 AM, server:**
- Schedule: cron `0 2 * * *`, clock **Local**

**Laptop that should back up roughly hourly whenever it's awake:**
- Schedule: interval **1 hour**, clock **Last Run Time**, with **Skip if unchanged** enabled

**Weekend-only backups:**
- Schedule: cron `0 3 * * 6,0` (Sat & Sun at 3 AM), clock **Local**

**Monthly maintenance that never silently skips a month:**
- Prune: interval **30 days**, clock **Last Run Time**
- Check: interval **30 days**, clock **Last Run Time** (queued after prune automatically when both are due)

## See Also

- [Retention & Repo Health](/guides/repo-health) — what forget/prune/check actually do
- [Operational Model](/docs/operations) — the task queue, priorities, and lifecycle behind the schedules
