# Operations Guide

This guide details the core operations available in Backrest and how to configure them effectively. "Operations" in Backrest refer to any task that interacts with your repository, such as creating backups, managing retention, verifying integrity, or restoring files.

## Restic Integration

Backrest executes operations through the [restic](https://restic.net) backup tool. Each operation maps to specific restic commands with additional functionality provided by Backrest.

### Binary Management
- **Location**: Backrest searches for the restic binary in the following order:
  1. Data directory (typically `~/.local/share/backrest`)
  2. `/bin/` directory
  3. The system `$PATH`
- **Version Requirement**: Backrest is only tested against the latest version of restic. It will selectively reject outdated versions.
- **Auto-download**: If no valid binary is found, Backrest downloads a verified version from [GitHub releases](https://github.com/restic/restic/releases).
- **Verification**: Downloads are verified using SHA256 checksums signed by restic maintainers.
- **Override**: Set `BACKREST_RESTIC_COMMAND` environment variable to use a custom restic binary.

### Command Execution
- **Environment**: Repository-specific environment variables are injected
- **Flags**: Repository-configured flags are appended to commands
- **Logging**: 
  - Error logs: Last ~500 bytes (split between start/end if longer)
  - Full logs: Available via **[View Logs]** in the UI, truncated to 32KB (split if longer)

::: info
If an operation fails, you can always find the full diagnostic logs in the Backrest UI by clicking on the specific operation block in the history tree. 
:::

## Scheduling System

Backrest provides flexible scheduling options for all operations through policies and clocks.

### Schedule Policies

| Policy         | Description                     | Use Case                                                  |
| -------------- | ------------------------------- | --------------------------------------------------------- |
| Disabled       | Operation will not run          | Temporarily disable operations                            |
| Cron           | Standard cron expression timing | Precise scheduling (e.g., `0 0 * * *` for daily midnight) |
| Interval Days  | Run every N days                | Regular daily+ intervals                                  |
| Interval Hours | Run every N hours               | Regular sub-daily intervals                               |

### Schedule Clocks

| Clock         | Description                    | Best For                                |
| ------------- | ------------------------------ | --------------------------------------- |
| Local         | Local timezone wall-clock      | Frequent operations (hourly+)           |
| UTC           | UTC timezone wall-clock        | Cross-timezone coordination             |
| Last Run Time | Relative to previous execution | Infrequent operations, preventing skips |

::: info
**Scheduling Best Practices**
- **Backup Operations** (Plan Settings):
  - Hourly or more frequent: Use "Local" clock
  - Daily or less frequent: Use "Last Run Time" clock
- **Prune/Check Operations** (Repo Settings):
  - Run infrequently (e.g., monthly)
  - Use "Last Run Time" clock to prevent skips
:::

## Operation Types

### 💾 Backup
[Restic Documentation](https://restic.readthedocs.io/en/latest/040_backup.html)

Creates snapshots of your data using `restic backup`.

**Process Flow:**
1. **Start**
   - Triggers `CONDITION_SNAPSHOT_START` hooks
   - Applies hook failure policies if needed
2. **Execution**
   - Runs `restic backup`
   - Tags snapshot with `plan:{PLAN_ID}` and `created-by:{INSTANCE_ID}`
3. **Completion**
   - Records operation metadata (files, bytes, snapshot ID)
   - Triggers appropriate hooks:
     - Error: `CONDITION_SNAPSHOT_ERROR`
     - Success: `CONDITION_SNAPSHOT_SUCCESS`
     - In both cases: `CONDITION_SNAPSHOT_END`
4. **Post-processing**
   - Runs forget operation if retention policy exists

**Snapshot Tags:**
- `plan:{PLAN_ID}`: Groups snapshots by backup plan
- `created-by:{INSTANCE_ID}`: Identifies creating Backrest instance

### 🕰️ Forget
[Restic Documentation](https://restic.readthedocs.io/en/latest/060_forget.html)

Manages snapshot retention using `restic forget --tag plan:{PLAN_ID}`.

**Retention Policies:**
- **By Count**: `--keep-last {COUNT}`
- **By Time Period**: `--keep-{hourly,daily,weekly,monthly,yearly} {COUNT}`

### ✂️ Prune
[Restic Documentation](https://restic.readthedocs.io/en/latest/060_forget.html#removing-unreferenced-data)

Removes unreferenced data using `restic prune`. Like Backup, Prune operations trigger their respective lifecycle hooks (e.g., `CONDITION_PRUNE_START`).

**Configuration:**
- Scheduled in repo settings
- Appears under `_system_` plan
- **Parameters:**
  - Schedule timing
  - Max unused percent (controls repacking threshold)

::: info
**Optimization Tips:**
- Run infrequently (monthly recommended)
- Use higher max unused percent (5-10%) to reduce repacking
- Consider storage costs vs. cleanup frequency
:::

### 🔍 Check
[Restic Documentation](https://restic.readthedocs.io/en/latest/080_check.html)

Verifies repository integrity using `restic check`.

**Configuration:**
- Scheduled in repo settings
- Appears under `_system_` plan
- **Parameters:**
  - Schedule timing
  - Read data percentage

::: warning
A value of 100% for *read data%* will read/download every pack file in your repository. This can be very slow and, if your provider bills for egress bandwidth, can be expensive. It is recommended to set this to 0% or a low value (e.g. 10%) for most use cases.
:::