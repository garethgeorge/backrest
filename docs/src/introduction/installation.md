# Installation

Backrest ships as a single executable for Linux, macOS, and Windows, plus official Docker images. On first run it automatically downloads a verified copy of [restic](https://restic.net) if a compatible version is not already installed on your system.

Once installed, Backrest is available at `http://localhost:9898`. On first launch it will prompt you to create a username and password.

## Choosing an Install Method

| Platform | Recommended method | Alternatives |
| --- | --- | --- |
| Linux server | [Install script](#linux-and-macos-install-script) (systemd/OpenRC) | [Docker](#docker), [AUR](#arch-linux-aur) |
| NAS / homelab with containers | [Docker Compose](#docker) | Install script |
| macOS | [Homebrew](#macos-homebrew) | Install script (launchd) |
| Windows | [Installer](#windows) | — |

## Linux and macOS (Install Script)

The install script downloads the latest release, installs the binary to `/usr/local/bin`, and sets up auto-start using your platform's service manager (systemd or OpenRC on Linux, launchd on macOS):

```bash
curl -fsSL https://raw.githubusercontent.com/garethgeorge/backrest/main/install.sh | bash
```

Flags go after `--`:

```bash
# Bind to all interfaces instead of the default 127.0.0.1:9898
curl -fsSL https://raw.githubusercontent.com/garethgeorge/backrest/main/install.sh | bash -s -- --allow-remote-access

# Uninstall (removes the service, autostart entry, and /usr/local/bin/backrest)
curl -fsSL https://raw.githubusercontent.com/garethgeorge/backrest/main/install.sh | bash -s -- --uninstall
```

The service runs as your user by default, so configuration and data live under your `$HOME` (see [Configuration & Paths](/docs/configuration)). To install as `root` instead, pass `--root`; backups will then run with root privileges and can read files your user cannot.

::: tip
Review [install.sh](https://github.com/garethgeorge/backrest/blob/main/install.sh) before piping it into a shell. You can also clone the repository and run `./install.sh` locally; it accepts the same flags.
:::

::: warning Binding to all interfaces
Only use `--allow-remote-access` on trusted networks, and make sure authentication is enabled. See [Authentication & Security](/guides/security) for guidance on exposing Backrest safely.
:::

### macOS (Homebrew)

Install from the [Homebrew tap](https://github.com/garethgeorge/homebrew-backrest-tap):

```bash
brew tap garethgeorge/homebrew-backrest-tap
brew install backrest
brew services start backrest
```

::: info Full Disk Access
macOS restricts access to many directories by default. If backups fail with permission errors, grant Full Disk Access to Backrest under `System Preferences > Security & Privacy > Privacy > Full Disk Access` by adding `/usr/local/bin/backrest`.
:::

### Arch Linux (AUR)

The [AUR package](https://aur.archlinux.org/packages/backrest) is third-party (not maintained by the Backrest project) and uses its own systemd unit:

```bash
paru -Sy backrest  # or: yay -Sy backrest
sudo systemctl enable --now backrest@$USER.service
```

## Docker

The canonical image is `ghcr.io/garethgeorge/backrest` (also mirrored to [Docker Hub](https://hub.docker.com/r/garethgeorge/backrest)). Two variants are published:

| Tag | Base | Contents |
| --- | --- | --- |
| `latest` | Alpine | restic, rclone, openssh, bash, curl, docker CLI, timezone data |
| `scratch` | scratch | restic and Backrest only — no shell or extra tools |

::: warning Choosing scratch
The `scratch` image does not include a shell, ssh, or rclone. Command hooks, the guided SFTP setup, and rclone remotes will not work in it. Use `latest` unless you are sure you do not need those features.
:::

### Docker Compose

```yaml
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
      - ./backrest/rclone:/root/.config/rclone # rclone config (only needed for rclone remotes)
      - /path/to/backup/data:/userdata  # mount the local paths you want to back up
      - /path/to/local/repos:/repos     # mount local repo storage (optional if using remote storage)
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

A few things to know about the container:

- Inside the container Backrest binds `0.0.0.0:9898` and defaults its paths to `/config/config.json`, `/data`, and `/cache`. The compose file above mounts each of these so your configuration and operation history survive container recreation.
- Set `hostname:` (or the instance ID at first launch) to a stable value, since it identifies this instance's snapshots.
- Set `TZ` so cron schedules run in your local timezone rather than UTC.
- Backrest can only back up paths that are mounted into the container, and restores also write to container paths, so plan your mounts accordingly.

## Windows

Download `Backrest-setup-[arch].exe` from the [releases page](https://github.com/garethgeorge/backrest/releases). The installer places Backrest and a tray application in `%localappdata%\Programs\Backrest\`. The tray app starts on login, runs Backrest in the background, and shows its status.

::: tip Changing the port on Windows
Set a user environment variable named `BACKREST_PORT` (Settings > About > Advanced system settings > Environment Variables) with a value like `127.0.0.1:8080`. If you change it after installation, re-run the installer so shortcuts pick up the new port.
:::

## Verifying the Install

Open `http://localhost:9898` in your browser. You should see the Backrest UI and a prompt to create your first user. From there, continue to [Your First Backup](/introduction/first-backup).

If nothing loads:

- **Script/service installs**: check the service status (`systemctl status backrest`, `rc-service backrest status`, or `brew services info backrest`).
- **Docker**: check `docker logs backrest`.
- Confirm nothing else is bound to port 9898, or change the port via `BACKREST_PORT`.

## Upgrading

Backrest is safe to upgrade in place. Configuration and operation history are stored separately from the binary (see [Configuration & Paths](/docs/configuration)) and are migrated automatically when the format changes.

- **Install script**: re-run the script; it replaces the binary with the latest release and restarts the service.
- **Homebrew**: `brew upgrade backrest && brew services restart backrest`.
- **Docker**: pull the new image and recreate the container. Your state lives in the mounted `/config` and `/data` volumes.
- **Windows**: run the new installer over the existing installation.

## Uninstalling

- **Install script**: run it with `--uninstall` (see above). This removes the service and binary but leaves your configuration (`~/.config/backrest`) and data (`~/.local/share/backrest`) in place; delete those directories manually to remove all Backrest state.
- **Docker**: remove the container and delete the mounted config/data directories.
- **Windows**: uninstall via Settings > Apps, then optionally delete `%appdata%\backrest`.

::: warning
Uninstalling Backrest does not touch your restic repositories; your backup data remains wherever it is stored. Keep a copy of your repository passwords, since without them the backups cannot be decrypted.
:::
