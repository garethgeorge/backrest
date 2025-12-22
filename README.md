<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./webui/assets/logo.svg" width="400px">
    <source media="(prefers-color-scheme: light)" srcset="./webui/assets/logo-black.svg" width="400px">
    <img src="./webui/assets/logo.svg" width="400px">
  </picture>
</p>

<p align="center">
  <img src="https://github.com/garethgeorge/backrest/actions/workflows/test.yml/badge.svg" />
  <img src="https://img.shields.io/github/downloads/garethgeorge/backrest/total" />
  <img src="https://img.shields.io/docker/pulls/garethgeorge/backrest" />
</p>

---

**Overview**

Backrest is a web-accessible backup solution built on top of [restic](https://restic.net/). Backrest provides a WebUI which wraps the restic CLI and makes it easy to create repos, browse snapshots, and restore files. Additionally, Backrest can run in the background and take an opinionated approach to scheduling snapshots and orchestrating repo health operations.

By building on restic, Backrest leverages its mature, fast, reliable, and secure backup capabilities while adding an intuitive interface.

Built with Go, Backrest is distributed as a standalone, lightweight binary with restic as its sole dependency. It can securely create new repositories or manage existing ones. Once storage is configured, the WebUI handles most operations, while still allowing direct access to the powerful [restic CLI](https://restic.readthedocs.io/en/latest/manual_rest.html) for advanced operations when needed.

## Preview

<p align="center">
   <img src="https://f000.backblazeb2.com/file/gshare/screenshots/2024/Screenshot+from+2024-01-04+18-19-50.png" width="60%" />
   <img src="https://f000.backblazeb2.com/file/gshare/screenshots/2024/Screenshot+from+2024-01-04+18-30-14.png" width="60%" />
</p>

## Key Features

- **Web Interface**: Access locally or remotely (perfect for NAS deployments)
- **Multi-Platform Support**: 
  - Linux
  - macOS
  - Windows
  - FreeBSD
  - [Docker](https://hub.docker.com/r/garethgeorge/backrest)
- **Backup Management**:
  - Import existing restic repositories
  - Cron-scheduled backups and maintenance (e.g. prune, check, forget, etc)
  - Browse and restore files from snapshots
  - Configurable notifications (Discord, Slack, Shoutrrr, Gotify, Healthchecks)
  - Pre/post backup command hooks to execute shell scripts
- **Storage Options**:
  - Compatible with rclone remotes
  - Supports all restic storage backends (S3, B2, Azure, GCS, local, SFTP, and [all rclone remotes](https://rclone.org/))

---

# User Guide

[See the Backrest docs](https://garethgeorge.github.io/backrest/introduction/getting-started).

# Installation

Backrest is packaged as a single executable. It can be run directly on Linux, macOS, and Windows. [restic](https://github.com/restic/restic) will be downloaded and installed on first run.

### Quick Start Options

1. **Pre-built Release**: Download from the [releases page](https://github.com/garethgeorge/backrest/releases)
2. **Docker**: Use `garethgeorge/backrest:latest` ([Docker Hub](https://hub.docker.com/r/garethgeorge/backrest))
   - Includes rclone and common Unix utilities
   - For minimal image, use `garethgeorge/backrest:scratch`
3. **Build from Source**: See [Building](#building) section below

Once installed, access Backrest at `http://localhost:9898` (default port). First-time setup will prompt for username and password creation.

> [!NOTE]
> To change the default port, set the `BACKREST_PORT` environment variable (e.g., `BACKREST_PORT=0.0.0.0:9898` to listen on all interfaces)
> 
> Backrest will use your system's installed version of restic if it's available and compatible. If not, Backrest will download and install a suitable version in its data directory, keeping it updated. To use a specific restic binary, set the `BACKREST_RESTIC_COMMAND` environment variable to the desired path.


### Running with Docker Compose

Docker image: https://hub.docker.com/r/garethgeorge/backrest

Example compose file:

```yaml
version: "3.8"
services:
  backrest:
    image: garethgeorge/backrest:latest
    container_name: backrest
    hostname: backrest
    volumes:
      - ./backrest/data:/data
      - ./backrest/config:/config
      - ./backrest/cache:/cache
      - ./backrest/tmp:/tmp
      - ./backrest/rclone:/root/.config/rclone # Mount for rclone config (needed when using rclone remotes)
      - /path/to/backup/data:/userdata  # Mount local paths to backup
      - /path/to/local/repos:/repos     # Mount local repos (optional for remote storage)
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

## Running on Linux

### Running on Linux

1. **Download the Release**
   - Get the latest release from the [releases page](https://github.com/garethgeorge/backrest/releases)

2. **Installation Options**

   a) Using the Install Script (Recommended)
   ```sh
   mkdir backrest && tar -xzvf backrest_Linux_x86_64.tar.gz -C backrest
   cd backrest && ./install.sh
   ```
   This script will:
   - Move the Backrest binary to `/usr/local/bin`
   - Create and start a systemd service running as the current user (use `sudo ./install.sh` to install as root)

   b) Manual Installation with systemd
   ```sh
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

   c) Using cron (Basic)
   ```sh
   sudo mv backrest /usr/local/bin/backrest
   (crontab -l 2>/dev/null; echo "@reboot /usr/local/bin/backrest") | crontab -
   ```

3. **Verify Installation**
   - Access Backrest at `http://localhost:9898`
   - For the systemd service: `sudo systemctl status backrest`

> [!NOTE]
> Adjust the `User` in the systemd service file if needed. The install script and manual systemd instructions use your current user by default.
>
> By default backrest listens only on localhost, you can open optionally open it up to remote connections by setting the `BACKREST_PORT` environment variable. For systemd installations, run `sudo systemctl edit backrest` and add:
> ```
> [Service]
> Environment="BACKREST_PORT=0.0.0.0:9898"
> ```
> Using `0.0.0.0` allows connections from any interface.

### Arch Linux

> [!Note]
> [Backrest on AUR](https://aur.archlinux.org/packages/backrest) is not maintained by the Backrest official and has made minor adjustments to the recommended services. Please refer to [here](https://aur.archlinux.org/cgit/aur.git/tree/backrest@.service?h=backrest) for details. In [backrest@.service](https://aur.archlinux.org/cgit/aur.git/tree/backrest@.service?h=backrest), use `restic` from the Arch Linux official repository by setting `BACKREST_RESTIC_COMMAND`. And for information on enable/starting/stopping services, please refer to [Systemd#Using_units](https://wiki.archlinux.org/title/Systemd#Using_units).

```shell
## Install Backrest from AUR
paru -Sy backrest  # or: yay -Sy backrest

## Enable Backrest service for current user
sudo systemctl enable --now backrest@$USER.service
```

## Running on macOS

### Using Homebrew (Recommended)

Backrest is available via a [Homebrew tap](https://github.com/garethgeorge/homebrew-backrest-tap):

```sh
brew tap garethgeorge/homebrew-backrest-tap
brew install backrest
brew services start backrest
```

This method uses [Brew Services](https://github.com/Homebrew/homebrew-services) to manage Backrest. It will launch on startup and run on port 127.0.0.1:9898 by default.

> [!NOTE]
> You may need to grant Full Disk Access to Backrest. Go to `System Preferences > Security & Privacy > Privacy > Full Disk Access` and add `/usr/local/bin/backrest`.

### Manual Installation

1. Download the latest Darwin release from the [releases page](https://github.com/garethgeorge/backrest/releases).
2. Extract and install:

```sh
mkdir backrest && tar -xzvf backrest_Darwin_arm64.tar.gz -C backrest
cd backrest && ./install.sh
```

The install script will:
- Move the Backrest binary to `/usr/local/bin`
- Create a launch agent at `~/Library/LaunchAgents/com.backrest.plist`
- Load the launch agent

> [!TIP]
> Review the script before running to ensure you're comfortable with its operations.

## Running on Windows

#### Windows Installer

Download the Windows installer for your architecture from the [releases page](https://github.com/garethgeorge/backrest/releases). The installer, named Backrest-setup-[arch].exe, will place Backrest and a GUI tray application in `%localappdata%\Programs\Backrest\`. The tray application, set to start on login, monitors Backrest.

> [!TIP]
> To override the default port before installation, set a user environment variable named BACKREST_PORT. On Windows 10+, navigate to Settings > About > Advanced system settings > Environment Variables. Under "User variables", create a new variable `BACKREST_PORT` with the value "127.0.0.1:port" (e.g., "127.0.0.1:8080" for port 8080). If changing post-installation, re-run the installer to update shortcuts with the new port.

# Configuration

## Environment Variables (Unix)

| Variable                  | Description                 | Default                                                                                                             |
| ------------------------- | --------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| `BACKREST_PORT`           | Port to bind to             | 127.0.0.1:9898 (or 0.0.0.0:9898 for the docker images)                                                              |
| `BACKREST_CONFIG`         | Path to config file         | `$HOME/.config/backrest/config.json`<br>(or, if `$XDG_CONFIG_HOME` is set, `$XDG_CONFIG_HOME/backrest/config.json`) |
| `BACKREST_DATA`           | Path to the data directory  | `$HOME/.local/share/backrest`<br>(or, if `$XDG_DATA_HOME` is set, `$XDG_DATA_HOME/backrest`)                        |
| `BACKREST_RESTIC_COMMAND` | Path to restic binary       | Defaults to a Backrest managed version of restic at `$XDG_DATA_HOME/backrest/restic-x.x.x`                          |
| `XDG_CACHE_HOME`          | Path to the cache directory |                                                                                                                     |

## Environment Variables (Windows)

| Variable                  | Description                 | Default                                                                                    |
| ------------------------- | --------------------------- | ------------------------------------------------------------------------------------------ |
| `BACKREST_PORT`           | Port to bind to             | 127.0.0.1:9898                                                                             |
| `BACKREST_CONFIG`         | Path to config file         | `%appdata%\backrest\config.json`                                                           |
| `BACKREST_DATA`           | Path to the data directory  | `%appdata%\backrest\data`                                                                  |
| `BACKREST_RESTIC_COMMAND` | Path to restic binary       | Defaults to a Backrest managed version of restic in `C:\Program Files\restic\restic-x.x.x` |
| `XDG_CACHE_HOME`          | Path to the cache directory |                                                                                            |

# Contributing

Contributions are welcome! See the [issues](https://github.com/garethgeorge/backrest/issues) or feel free to open a new issue to discuss a project. Beyond the core codebase, contributions to [documentation](https://garethgeorge.github.io/backrest/introduction/getting-started), [cookbooks](https://garethgeorge.github.io/backrest/cookbooks/command-hook-examples), and testing are always welcome.

## Build Depedencies

- [Node.js](https://nodejs.org/en) for UI development
- [Go](https://go.dev/) 1.21 or greater for server development
- [goreleaser](https://github.com/goreleaser/goreleaser) `go install github.com/goreleaser/goreleaser@latest`

**(Optional) To Edit Protobuffers**

```sh
apt install -y protobuf-compiler
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/bufbuild/buf/cmd/buf@v1.47.2
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
npm install -g @bufbuild/protoc-gen-es
```

## Compiling

```sh
(cd webui && npm i && npm run build)
(cd cmd/backrest && go build .)
```

## Using VSCode Dev Containers

You can also use VSCode with [Dev Containers](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers) extension to quickly get up and running with a working development and debugging environment.

0. Make sure Docker and VSCode with [Dev Containers](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers) extension is installed
1. Clone this repository
2. Open this folder in VSCode
3. When prompted, click on `Open in Container` button, or run `> Dev Containers: Rebuild and Reopen in Containers` command
4. When container is started, go to `Run and Debug`, choose `Debug Backrest (backend+frontend)` and run it

> [!NOTE]
> Provided launch configuration has hot reload for typescript frontend.

## Translations

Translations are stored in [./webui/messages](./webui/messages) and are generated using [inlang](https://inlang.com/). Machine translations can be updated by running `npx @inlang/cli machine translate --project ./project.inlang`. 

Text is translated on a best-effort basis and is not guaranteed to be accurate. If you find any translations that are incorrect, please submit a pull request to fix them. Contributions here are greatly appreciated!