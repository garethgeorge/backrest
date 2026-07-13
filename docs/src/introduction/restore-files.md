# Restoring Files

This guide covers finding the right snapshot, restoring files to disk, downloading files through the browser, and recovering data on a different machine.

::: tip
Perform a small test restore after setting up your first plan. It verifies repository access, the repository password, and data integrity before you need them in a real recovery.
:::

## Finding the Right Snapshot

Snapshots appear in Backrest's tree view, ordered by creation time, grouped under the plan (or repository) that created them. Click a snapshot to open a side panel showing its metadata, the operation history that produced it, and a **Snapshot Browser** for exploring its contents.

<img src="/screenshots/tree-view-for-restore-article.png" alt="Operation tree view with a snapshot selected" style="width: 700px; height: auto;" />

### If Snapshots Are Missing: Index Them

Backrest indexes snapshots automatically when you add a repository and after each backup it runs. If snapshots were created outside Backrest (by the restic CLI, or by another machine writing to the same repository), click **Index Snapshots** in the repository view to import them.

<img src="/screenshots/index-snapshots-btn.png" alt="Index Snapshots button in the repository view" style="width: 700px; height: auto;" />

## Browsing a Snapshot

Open the **Snapshot Browser** from the snapshot's panel and navigate to the files or directories you want back.

<img src="/screenshots/snapshot-browser.png" alt="Snapshot browser expanded showing directories and files inside a snapshot" style="max-width: 100%; border-radius: 8px; margin-bottom: 20px;">

::: warning
With remote storage, browsing can be slow because restic fetches pack files on demand to reconstruct the snapshot's directory structure. Expect a delay on the first expansion of large directories.
:::

## Restoring to a Path

Hover over a file or directory in the browser, click the restore icon, and choose **Restore to path**:

<img src="/screenshots/restore-dialog.png" alt="Restore dialog with target path options" style="width: 700px; height: auto;" />

- **Specific location**: the dialog pre-fills a path based on the folder name plus the first 8 characters of the snapshot ID. You can change it to any path. Restoring to a fresh directory and moving files into place afterwards is safer than restoring over live data.
- **Left empty**: Backrest restores to a timestamped folder in your Downloads directory (e.g. `~/Downloads/restic-restore-2026-07-12T10-30-00`).

The target directory must not already exist; Backrest will not overwrite an existing directory.

Click **Restore**. The restore runs as a tracked operation at the top of the operation tree, with live progress (bytes and files restored):

<img src="/screenshots/restore-progress.png" alt="Restore operation showing progress" style="width: 700px; height: auto;" />

::: info Restoring in Docker
Restore paths are **inside the container**. To get files onto the host, restore to a directory under one of your bind mounts (e.g. `/userdata/restored`), or use the download option below.
:::

## Downloading Files Directly

Files can also be retrieved through the browser instead of being restored to the server's filesystem:

- After a restore completes, the operation offers a **download** link that packages the restored files as a `.tar.gz` archive through your browser.
- Individual files can be downloaded straight from the snapshot browser. Directories arrive as `.tar` archives (Backrest streams them via `restic dump` under the hood).

This is useful when Backrest runs on a NAS or remote server and you need a few files on the machine you are browsing from.

## Restoring on a Different Machine

Backrest repositories are standard restic repositories, so there are two recovery paths that do not depend on the original machine:

1. **Another Backrest instance**: install Backrest anywhere, add the repository with the same URI and password, click **Index Snapshots**, and restore from the UI.
2. **The restic CLI**: point restic at the repository directly, e.g.:

```bash
export RESTIC_REPOSITORY=s3:s3.amazonaws.com/my-bucket/backrest-repo
export RESTIC_PASSWORD='your-repo-password'
restic snapshots
restic restore latest --target /tmp/restored
```

Recovery requires only the repository location, its password, and any storage credentials. All three are recorded in your [Backrest config](/docs/configuration), which is a good reason to keep a copy of it somewhere safe.

## Verifying Restored Data

After a restore, spot-check the results: open a few files, compare directory sizes against the source, and check file counts in the restore operation's summary. For ongoing verification of repository integrity, schedule **check** operations; see [Retention & Repo Health](/guides/repo-health).
