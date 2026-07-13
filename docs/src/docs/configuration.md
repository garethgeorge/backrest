# Configuration & Paths

A reference for where Backrest keeps its files, the environment variables and flags it recognizes, and the shape of its configuration file.

## File Locations

| What | Linux / macOS | Windows | Docker (default) |
| --- | --- | --- | --- |
| Config file | `$XDG_CONFIG_HOME/backrest/config.json`, falling back to `~/.config/backrest/config.json` | `%APPDATA%\backrest\config.json` | `/config/config.json` |
| Data directory | `$XDG_DATA_HOME/backrest`, falling back to `~/.local/share/backrest` | `%APPDATA%\backrest\data` | `/data` |
| restic cache | `$XDG_CACHE_HOME` (restic's default rules otherwise) | restic default | `/cache` |
| SSH keys for SFTP | `<config dir>/.backrest-ssh` | — | `/config/.backrest-ssh` |

The **data directory** holds Backrest's operational state: the operation log (`oplog.sqlite`), task logs (`tasklogs`), Backrest's own process logs (`processlogs`), the JWT signing secret (`jwt-secret`), and, if no system restic is used, the managed restic binary (`restic-x.x.x`).

::: tip What to back up
Rebuilding a Backrest install from scratch requires only the config file, which holds repository definitions, credentials, and plans; keep a copy somewhere safe, such as a password manager. The data directory holds operational history: losing it removes the UI history and statistics but does not affect backup data.
:::

## Environment Variables and Flags

Each setting can be provided as an environment variable or a command-line flag; **flags take precedence over environment variables**, which take precedence over defaults.

| Environment variable | Flag | Default | Purpose |
| --- | --- | --- | --- |
| `BACKREST_PORT` | `--bind-address` | `127.0.0.1:9898` (Docker images: `0.0.0.0:9898`) | Address/port to serve on. A bare number like `9898` is treated as `:9898`. |
| `BACKREST_CONFIG` | `--config-file` | see table above | Path to `config.json` |
| `BACKREST_DATA` | `--data-dir` | see table above | Path to the data directory |
| `BACKREST_RESTIC_COMMAND` | `--restic-cmd` | Backrest-managed restic | Use a specific restic binary |
| `BACKREST_MULTIHOST_HEARTBEAT_INTERVAL` | `--multihost-heartbeat-interval` | `600s` | Keepalive interval for [multihost sync](/docs/multihost) connections (lower it below your reverse proxy's idle timeout if sync connections drop) |
| `XDG_CONFIG_HOME` / `XDG_DATA_HOME` / `XDG_CACHE_HOME` | — | platform defaults | Standard XDG overrides for the paths above |
| `TMPDIR` | — | system default | Temp space (the Docker images set `/tmp`) |
| `TZ` | — | system default | Timezone used by cron schedules with the Local clock — set this in Docker |

Standalone flags: `--version` prints version and commit; `--install-deps-only` downloads restic and exits (used by the Docker image build); tray-enabled builds add `--tray` (macOS/Linux) and `--windows-tray`.

Backrest also passes its process environment through to restic, so restic's own variables (`RESTIC_PASSWORD`, `AWS_ACCESS_KEY_ID`, `RCLONE_*`, etc.) work when set globally, though per-repository env vars in the config are usually the better place for credentials. See [Storage Backends](/guides/storage-backends) for precedence details.

## The Config File

Backrest stores all of its configuration (instance identity, repositories, plans, users, and sync settings) in a single JSON file. The UI is the intended way to edit it, but understanding its structure is useful for backups, audits, and occasional manual fixes.

An annotated example (not every field shown; omitted fields take defaults):

```json
{
  "modno": 42,            // internal change counter, managed by Backrest
  "version": 4,           // config format version, managed by Backrest
  "instance": "home-server",
  "repos": [
    {
      "id": "mydrive",
      "uri": "s3:s3.amazonaws.com/my-bucket/backrest",
      "guid": "...",       // derived from the restic repo's identity, managed by Backrest
      "password": "...",   // repository encryption password (plaintext -- protect this file)
      "env": ["AWS_ACCESS_KEY_ID=...", "AWS_SECRET_ACCESS_KEY=..."],
      "flags": ["--limit-upload", "4000"],
      "prunePolicy": {
        "schedule": { "maxFrequencyDays": 30, "clock": "CLOCK_LAST_RUN_TIME" },
        "maxUnusedPercent": 25
      },
      "checkPolicy": {
        "schedule": { "maxFrequencyDays": 30, "clock": "CLOCK_LAST_RUN_TIME" },
        "readDataSubsetPercent": 0
      },
      "autoUnlock": false,     // remove stale repo locks automatically before operations
      "autoInitialize": false, // initialize the repo if it does not exist yet
      "commandPrefix": {       // resource limits applied to restic (Unix)
        "ioNice": "IO_BEST_EFFORT_LOW",
        "cpuNice": "CPU_LOW"
      },
      "hooks": []              // repo-level hooks, see the Hooks reference
    }
  ],
  "plans": [
    {
      "id": "mydrive-documents",
      "repo": "mydrive",
      "paths": ["/home/me/Documents"],
      "excludes": ["*node_modules*"],
      "iexcludes": [".cache"],          // case-insensitive excludes
      "schedule": { "cron": "0 2 * * *", "clock": "CLOCK_LOCAL" },
      "retention": {
        "policyTimeBucketed": { "daily": 7, "weekly": 4, "monthly": 12 }
        // or: "policyKeepLastN": 30
        // or: "policyKeepAll": true
      },
      "backup_flags": ["--one-file-system"],
      "skipIfUnchanged": true,
      "hooks": []
    }
  ],
  "auth": {
    "disabled": false,
    "users": [{ "name": "me", "passwordBcrypt": "..." }]
  },
  "sync": {}                  // multihost identity, peers, and permissions
}
```

Notes on specific fields:

- **`schedule`** appears on plans and on prune/check/forget policies; it holds exactly one of `cron`, `maxFrequencyHours`, `maxFrequencyDays`, or `disabled: true`, plus a `clock` (`CLOCK_LOCAL`, `CLOCK_UTC`, or `CLOCK_LAST_RUN_TIME`). Semantics are covered in the [Scheduling guide](/guides/scheduling).
- **`retention`** holds exactly one policy variant. A repo may additionally define a scheduled `forgetPolicy` (`schedule` + `retention`), which **replaces all per-plan retention** for that repo — see [Retention & Repo Health](/guides/repo-health).
- **`env` and `flags`** support `${VAR}` expansion from Backrest's process environment, which is useful for keeping secrets out of the file itself.
- **`shared` / `originInstanceId`** on repos are managed by [multihost sync](/docs/multihost); repos pushed from another instance are not locally editable.

## Editing the Config Directly

Backrest owns this file while running: it validates the file on load and rewrites it on every change made in the UI. If you need to edit it by hand:

1. Stop Backrest first, make your edits, then start it again. Edits made while Backrest is running can be overwritten.
2. Keep the JSON valid, and do not modify the `modno`, `version`, or `guid` fields, which are managed by Backrest.
3. If the config fails validation, Backrest reports the error at startup; check the process logs.

The most common manual edit is resetting a lost password: delete the `"users"` key under `"auth"` and restart, and first-launch user creation will run again. See [Authentication & Security](/guides/security) for details.

::: warning
`config.json` contains repository passwords and storage credentials in plaintext (Backrest writes it with owner-only permissions). Treat backups of this file with the same care as the passwords themselves.
:::
