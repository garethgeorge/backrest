---
layout: home

hero:
  name: "Backrest"
  text: "Web UI and orchestrator for Restic backup."
  tagline: "Backrest is a web-accessible backup solution built on top of restic and providing a WebUI which wraps the restic CLI and makes it easy to create repos, browse snapshots, and restore files. Additionally, Backrest can run in the background and take an opinionated approach to scheduling snapshots and orchestrating repo health operations."
  actions:
    - theme: brand
      text: Get started
      link: /introduction/getting-started
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

## Installation

::: code-group
```bash [Linux (Script)]
# Download the latest release from https://github.com/garethgeorge/backrest/releases
curl -sLO https://github.com/garethgeorge/backrest/releases/latest/download/backrest_Linux_x86_64.tar.gz
mkdir backrest && tar -xzvf backrest_Linux_x86_64.tar.gz -C backrest
cd backrest && ./install.sh
```
```bash [Linux (systemd)]
sudo mv backrest /usr/local/bin/backrest
sudo tee /etc/systemd/system/backrest.service > /dev/null <<EOT
[Unit]
Description=Backrest
After=network.target

[Service]
Type=simple
User=$(whoami)
ExecStart=/usr/local/bin/backrest
Environment="BACKREST_PORT=127.0.0.1:9898"

[Install]
WantedBy=multi-user.target
EOT
sudo systemctl enable --now backrest
```
```bash [Arch Linux]
paru -Sy backrest
sudo systemctl enable --now backrest@$USER.service
```
```bash [MacOS (Homebrew)]
brew tap garethgeorge/homebrew-backrest-tap
brew install backrest
brew services start backrest
```
```bash [MacOS (Script)]
# Download the latest release from https://github.com/garethgeorge/backrest/releases
curl -sLO https://github.com/garethgeorge/backrest/releases/latest/download/backrest_Darwin_arm64.tar.gz
mkdir backrest && tar -xzvf backrest_Darwin_arm64.tar.gz -C backrest
cd backrest && ./install.sh
```
```yaml [docker-compose]
version: "3.8"
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
      - ./backrest/rclone:/root/.config/rclone # Mount for rclone config
      - /path/to/backup/data:/userdata  # Mount local paths to backup
      - /path/to/local/repos:/repos     # Mount local repos (optional)
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
:::
