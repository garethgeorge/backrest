---
layout: home

hero:
  name: "Backrest"
  text: "Web UI and orchestrator for Restic backup."
  tagline: "Backrest is a web-accessible backup solution built on top of restic. It wraps the restic CLI in a WebUI that makes it easy to create repos, browse snapshots, and restore files — and it runs in the background to schedule backups and keep your repositories healthy."
  actions:
    - theme: brand
      text: Get Started
      link: /introduction/getting-started
    - theme: alt
      text: Install
      link: /introduction/installation
    - theme: alt
      text: Open on GitHub
      link: https://github.com/garethgeorge/backrest

features:
  - title: Existing Repositories
    details: Import your existing restic repositories.
  - title: Cron Scheduling
    details: Cron scheduled backups and health operations (e.g. prune and forget).
  - title: Browse & Restore
    details: UI for browsing and restoring files from snapshots.
  - title: Notifications
    details: Configurable backup notifications (e.g. Discord, Slack, Shoutrrr, Gotify).
  - title: Command Hooks
    details: Add shell command hooks to run before and after backup operations.
  - title: Storage Support
    details: Backup to any restic supported storage (e.g. S3, B2, Azure, GCS, local, SFTP, and all rclone remotes). Cross-platform support.
---

## Quick Install

Run Backrest with Docker Compose or install it natively with one command — see the [full installation guide](/introduction/installation) for Windows, Homebrew, service configuration, and image variants.

::: code-group

```yaml [docker-compose]
services:
  backrest:
    image: ghcr.io/garethgeorge/backrest:latest
    container_name: backrest
    hostname: backrest
    volumes:
      - ./backrest/data:/data
      - ./backrest/config:/config
      - ./backrest/cache:/cache
      - ./backrest/tmp:/tmp
      - /path/to/backup/data:/userdata  # mount local paths to back up
      - /path/to/local/repos:/repos     # mount local repo storage (optional)
    environment:
      - BACKREST_DATA=/data
      - BACKREST_CONFIG=/config/config.json
      - XDG_CACHE_HOME=/cache
      - TMPDIR=/tmp
      - TZ=America/Los_Angeles
    ports:
      - "9898:9898"
    restart: unless-stopped
```

```bash [Linux & macOS]
curl -fsSL https://raw.githubusercontent.com/garethgeorge/backrest/main/install.sh | bash
```

:::

Then open `http://localhost:9898` and follow [Your First Backup](/introduction/first-backup).
