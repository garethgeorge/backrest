# Introduction & Concepts

[Backrest](https://github.com/garethgeorge/backrest) is a web UI and orchestrator for [restic](https://restic.net), the fast, secure, deduplicating backup tool. Backrest wraps the restic CLI with a browser-based interface for creating repositories, scheduling backups, browsing snapshots, and restoring files. It also runs in the background to schedule backups and repository maintenance.

This page explains the concepts the rest of the documentation builds on. If you'd rather learn by doing, jump straight to [Installation](/introduction/installation) and [Your First Backup](/introduction/first-backup).

## How Backrest Relates to Restic

Backrest executes every backup, prune, and restore by running the restic binary. Backrest downloads and verifies a copy of restic automatically, or you can [configure it to use your own](/docs/configuration).

Because restic performs the actual storage operations, repositories created by Backrest are standard restic repositories. You can browse them with the restic CLI, restore from them on a machine that has never run Backrest, and use Backrest alongside your own restic scripts. Backrest adds orchestration, history tracking, and a UI; the underlying data format is unchanged.

## Core Concepts

### Instance

One installation of Backrest, identified by an **instance ID** you choose at first launch (e.g. `home-server`). Snapshots are tagged with the instance that created them, so multiple machines can safely share one repository. The instance ID cannot be changed later without orphaning the association with existing snapshots.

### Repository

Where your encrypted backup data lives. The term is used at two levels:

- A **restic repository** is the on-disk/remote storage format: a content-addressed, encrypted, deduplicated store that any restic client can read.
- A **Backrest repository** is that restic repository *plus* Backrest's configuration for it: the URI, credentials, environment variables and flags, maintenance policies (prune/check), and repository-level hooks.

One repository can serve many plans, and deduplication works across all of them.

### Plan

A plan defines what to back up (paths and exclude patterns), when to back it up (a schedule), and how long to keep the results (a retention policy). Plans belong to exactly one repository and can carry their own hooks (e.g. notify on failure). A machine typically has a small number of plans, such as one for documents backed up hourly and one for photos backed up daily, writing to one or more repositories.

### Operations

Everything Backrest does to a repository is an **operation**: backup, forget, prune, check, restore, and a few housekeeping tasks. Operations are the unit you see in the UI's history tree, each with a status (pending → in progress → success, warning, or error) and full logs.

The four you'll interact with most:

| Operation | What it does |
| --- | --- |
| **Backup** | Creates a new snapshot of your plan's paths |
| **Forget** | Applies retention policy, *marking* aged-out snapshots (no data deleted yet) |
| **Prune** | Reclaims storage by deleting data no snapshot references |
| **Restore** | Copies files from a snapshot back to disk |

### The Operation Log

Backrest records every operation in a local database (the *oplog*), which powers the history tree, statistics graphs, and multihost monitoring. The operation log is stored separately from the backup data, which lives in the repository.

### The `_system_` Plan

Repository-level maintenance (prune, check, repository-wide forget) is not tied to any of your plans, so it appears in the UI under a synthetic plan named `_system_`. Operations listed there are repository maintenance run by Backrest itself.

## How the Pieces Fit Together

```
 Plan "mydrive-documents"          Plan "mydrive-photos"
   what: /home/me/Documents          what: /home/me/Photos
   when: hourly                      when: daily
   keep: 30 daily, 12 monthly        keep: 12 monthly
        │                                 │
        └────────────┬────────────────────┘
                     ▼
       Repository "mydrive"  (s3:...  + credentials)
         maintenance: prune weekly, check monthly (_system_)
                     ▼
            restic repository (encrypted, deduplicated)
```

Backrest's orchestrator runs one operation at a time per repository, queued by time and priority, so that backups, follow-up forgets, and scheduled maintenance do not conflict over repository locks. The [Operational Model reference](/docs/operations) covers this in detail.

## Where to Go Next

| I want to… | Read |
| --- | --- |
| Install Backrest | [Installation](/introduction/installation) |
| Take my first backup | [Your First Backup](/introduction/first-backup) |
| Get files back | [Restoring Files](/introduction/restore-files) |
| Tune when backups run | [Scheduling Backups](/guides/scheduling) |
| Manage retention and repo health | [Retention & Repo Health](/guides/repo-health) |
| Configure S3/B2/Azure/GCS/rclone | [Storage Backends](/guides/storage-backends) |
| Back up over SSH | [SFTP & SSH Remotes](/guides/sftp) |
| Get notified about failures | [Notifications](/guides/notifications) |
| Secure or expose my instance | [Authentication & Security](/guides/security) |
| Understand the internals | [Operational Model](/docs/operations) |
| Look up paths, env vars, config fields | [Configuration & Paths](/docs/configuration) |

::: warning Protect your credentials
Your repository password is required to decrypt your backups, and restic provides no way to reset it. Store the password (ideally your whole `config.json`) in a password manager or other secure location. See [Authentication & Security](/guides/security).
:::
