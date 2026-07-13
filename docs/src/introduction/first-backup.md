# Your First Backup

This tutorial walks you from a fresh Backrest install to a verified first snapshot. Along the way you'll set up the three building blocks of every Backrest deployment: an **instance ID**, a **repository**, and a **plan**. If any of those terms are unfamiliar, skim [Introduction & Concepts](/introduction/getting-started) first.

## Before You Begin

You should have:

- Backrest [installed](/introduction/installation) and reachable at `http://localhost:9898` (or your configured address).
- A user account created (Backrest prompts for this on first launch).
- A location to store backups. For this tutorial a local folder or mounted drive is easiest; cloud storage works the same way, and provider-specific setup is covered in [Storage Backends](/guides/storage-backends).

## Step 1: Set Your Instance ID

On first launch Backrest asks for an **instance ID** — a short unique name for this machine or installation (e.g. `home-server`, `alices-laptop`).

<img src="/screenshots/settings-view.png" alt="Settings view showing instance configuration" style="max-width: 100%; border-radius: 8px; margin-bottom: 20px;">

Every snapshot Backrest creates is tagged `created-by:<your-id>`, which is how Backrest tells its snapshots apart from those created by other machines sharing the same repository.

::: warning
The instance ID cannot be changed from the UI later, because renaming it would break the association with existing snapshots. Choose a value you do not expect to change.
:::

## Step 2: Add a Repository

A repository is where restic stores your encrypted backup data. Click **Add Repo**.

<img src="/screenshots/add-repo-view.png" alt="Add repository view" style="max-width: 100%; border-radius: 8px; margin-bottom: 20px;">

For a first backup to local storage, you only need three fields:

1. **Repository name** — a human-readable ID, e.g. `mydrive`. Immutable after creation.
2. **Repository URI** — where the data lives. For local storage this is a filesystem path, e.g. `/mnt/backupdisk/backrest-repo` (in Docker, a path inside the container such as `/repos/backrest-repo`). Cloud URIs like `s3:...` or `b2:...` are also accepted; each provider's URI format and credentials are covered in [Storage Backends](/guides/storage-backends).
3. **Password** — the encryption key for the repository. Backrest can generate a strong one for you.

::: danger Save your password
The repository password encrypts all backup data. If you lose it, your backups cannot be recovered; there is no reset mechanism. Store it in a password manager along with a copy of your Backrest config file.
:::

The remaining settings (environment variables, flags, prune/check policies, hooks) have reasonable defaults and can be changed later.

Click **Submit**. If the URI points at an empty location, Backrest initializes a brand-new restic repository there.

::: info Importing an existing restic repository
The same flow works for existing repositories: enter the URI and password, then open the repository view and click **Index Snapshots** to import the snapshot history. Backrest can be used alongside the restic CLI; repositories remain standard restic repositories.
:::

## Step 3: Create a Backup Plan

A plan defines *what* to back up, *when*, and *how long to keep it*. Click **Add Plan**.

<img src="/screenshots/add-plan-view.png" alt="Add plan view" style="max-width: 100%; border-radius: 8px; margin-bottom: 20px;">

1. **Plan name** — descriptive and immutable, e.g. `mydrive-documents` (a `[storage]-[content]` naming convention works well).
2. **Repository** — select the repo you just created. Immutable after creation.
3. **Paths** — the directories or files to back up.
4. **Excludes** (optional) — glob patterns to skip, e.g. `*node_modules*` or `.cache`. `iexcludes` are the same but case-insensitive.
5. **Schedule** — when backups run. The default is fine for now; the [Scheduling guide](/guides/scheduling) explains cron expressions, intervals, and clocks in depth.
6. **Retention policy** — how long snapshots are kept. The default time-bucketed policy (a mix of daily/weekly/monthly snapshots) suits most users; see [Retention & Repo Health](/guides/repo-health) for how retention actually works.

Click **Submit**. The plan now appears in the sidebar and will run on its schedule.

## Step 4: Run a Backup Now

You do not need to wait for the schedule. Open your plan and click **Backup Now**.

The operation appears in the plan's history: it starts **pending** (queued), moves to **in progress** with live progress and log output, and finishes as **success**. Click the operation row to see details: files added, bytes processed, and the new snapshot ID. The dashboard reflects the result too:

<img src="/screenshots/summary-dashboard.png" alt="Summary dashboard after a successful backup" style="max-width: 100%; border-radius: 8px; margin-bottom: 20px;">

If the backup finishes with a **warning** status instead, the snapshot was still created. A warning means some files could not be read; locked or permission-denied files are common on first runs. The operation log lists which paths failed.

## Step 5: Verify Your Snapshot

After the backup completes, confirm the snapshot contains what you expect:

1. In the plan or repository view, find your new snapshot in the tree.
2. Expand it to browse the files it contains, and confirm the paths you configured are present.

<img src="/screenshots/snapshot-browser.png" alt="Snapshot operation expanded, showing details and the snapshot browser with backed-up files" style="max-width: 100%; border-radius: 8px; margin-bottom: 20px;">

For a complete end-to-end test, restore a file or two by following the [Restoring Files](/introduction/restore-files) guide. A test restore is the most reliable way to confirm a backup is usable.

## What Happens After a Backup

Additional operations appear in the history after a successful backup. These are follow-up tasks that Backrest schedules automatically:

- A **forget** operation applies your plan's retention policy, removing snapshots that have aged out of it. Forget does not delete the underlying data; the repository's scheduled **prune** operation reclaims that space later.
- An **index snapshots** operation keeps Backrest's local view of the repository in sync.

The [Retention & Repo Health guide](/guides/repo-health) explains this pipeline, and the [Operational Model reference](/docs/operations) covers how the orchestrator schedules and prioritizes everything.

## Next Steps

- [Tune your schedule](/guides/scheduling) — cron vs. intervals, and behavior when a machine is asleep at the scheduled time.
- [Set up retention, prune, and check](/guides/repo-health) — keep the repository healthy and storage bounded.
- [Set up notifications](/guides/notifications) — Discord, Slack, Gotify, Telegram, Healthchecks, and more.
- [Restore files](/introduction/restore-files) — walk through a test restore.

::: warning Back up your Backrest config
Your config file (typically `~/.config/backrest/config.json`) contains your repository definitions, credentials, and plans. Keep a copy somewhere safe; see [Configuration & Paths](/docs/configuration).
:::
