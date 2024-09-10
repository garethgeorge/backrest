<p align="center"><img src="./webui/assets/logo-black.svg" width="400px"/></p>

<p align="center">
  <img src="https://github.com/garethgeorge/backrest/actions/workflows/test.yml/badge.svg" />
  <img src="https://img.shields.io/github/downloads/garethgeorge/backrest/total" />
  <img src="https://img.shields.io/docker/pulls/garethgeorge/backrest" />
</p>

---

**Overview**

Backrest is a web-accessible backup solution built on top of [restic](https://restic.net/). Backrest provides a WebUI which wraps the restic CLI and makes it easy to create repos, browse snapshots, and restore files. Additionally, Backrest can run in the background and take an opinionated approach to scheduling snapshots and orchestrating repo health operations.

By building on restic, Backrest leverages restic's mature feature set. Restic provides fast, reliable, and secure backup operations.

Backrest itself is built in Golang (matching restic's implementation) and is shipped as a self-contained and light weight binary with no dependecies other than restic. This project aims to be the easiest way to setup and get started with backups on any system. You can expect to be able to perform all operations from the web interface but should you ever need more control, you are free to browse your repo and perform operations using the [restic cli](https://restic.readthedocs.io/en/latest/manual_rest.html). Additionally, Backrest can safely detect and import your existing snapshots (or externally created snapshots on an ongoing basis).

**Preview**

<p align="center">
   <img src="https://f000.backblazeb2.com/file/gshare/screenshots/2024/Screenshot+from+2024-01-04+18-19-50.png" width="60%" />
   <img src="https://f000.backblazeb2.com/file/gshare/screenshots/2024/Screenshot+from+2024-01-04+18-30-14.png" width="60%" />
</p>

**Platform Support**

- [Docker](https://hub.docker.com/r/garethgeorge/backrest)
- Linux
- macOS
- Windows
- FreeBSD

**Features**

- WebUI supports local and remote access (e.g. run on a NAS and access from your desktop)
- Multi-platform support (Linux, macOS, Windows, FreeBSD, [Docker](https://hub.docker.com/r/garethgeorge/backrest))
- Import your existing restic repositories
- Cron scheduled backups and health operations (e.g. prune, check, forget)
- UI for browing and restoring files from snapshots
- Configurable backup notifications (e.g. Discord, Slack, Shoutrrr, Gotify)
- Add shell command hooks to run before and after backup operations.
- Compatible with rclone remotes
- Backup to any restic supported storage (e.g. S3, B2, Azure, GCS, local, SFTP, and all [rclone remotes](https://rclone.org/))

---

# User Guide

[See the Backrest docs](https://garethgeorge.github.io/backrest/introduction/getting-started).

# Installation

Backrest is packaged as a single executable. It can be run directly on Linux, macOS, and Windows. [restic](https://github.com/restic/restic) will be downloaded and installed on first run.

Download options

- Download and run a release from the [releases page](https://github.com/garethgeorge/backrest/releases).
- Build from source ([see below](#building)).
- Run with docker: `garethgeorge/backrest:latest` ([see on dockerhub](https://hub.docker.com/r/garethgeorge/backrest)) for an image that includes rclone and common unix utilities or `garethgeorge/backrest:scratch` for a minimal image.

Backrest is accessible from a web browser. By default it binds to `127.0.0.1:9898` and can be accessed at `http://localhost:9898`. Change the port with the `BACKREST_PORT` environment variable e.g. `BACKREST_PORT=0.0.0.0:9898 backrest` to listen on all network interfaces. On first startup backrest will prompt you to create a default username and password, this can be changed later in the settings page.

> [!Note]
> Backrest installs a specific restic version to ensure that it is compatible. If you wish to use a different version of restic OR if you would prefer to install restic manually, use the `BACKREST_RESTIC_COMMAND` environment variable to specify the path of your restic install.

## Running with Docker Compose

Docker image: https://hub.docker.com/r/garethgeorge/backrest

Example compose file:

```yaml
version: "3.2"
services:
  backrest:
    image: garethgeorge/backrest:latest
    container_name: backrest
    hostname: backrest
    volumes:
      - ./backrest/data:/data
      - ./backrest/config:/config
      - ./backrest/cache:/cache
      - /MY-BACKUP-DATA:/userdata # [optional] mount local paths to backup here.
      - /MY-REPOS:/repos # [optional] mount repos if using local storage, not necessary for remotes e.g. B2, S3, etc.
    environment:
      - BACKREST_DATA=/data # path for backrest data. restic binary and the database are placed here.
      - BACKREST_CONFIG=/config/config.json # path for the backrest config file.
      - XDG_CACHE_HOME=/cache # path for the restic cache which greatly improves performance.
      - TZ=America/Los_Angeles # set the timezone for the container, used as the timezone for cron jobs.
    restart: unless-stopped
    ports:
      - 9898:9898
```

## Running on Linux

### All Linux Platforms

Download a release from the [releases page](https://github.com/garethgeorge/backrest/releases)

#### Using systemd with the install script (Recommended)

Extract the release you downloaded and run the install script:

```
# Extract the release to a subfolder of the current directory
mkdir backrest && tar -xzvf backrest_Linux_x86_64.tar.gz -C backrest
# Run the install script
cd backrest && ./install.sh
```

The install script will:

- Move the Backrest binary to `/usr/local/bin`
- Create a systemd service file at `/etc/systemd/system/backrest.service`
- Enable and start the service

Read the script before running it to make sure you are comfortable with these operations.

#### Run on startup with cron (Basic)

Move the Backrest binary to `/usr/local/bin`:

```sh
sudo mv backrest /usr/local/bin/backrest
```

Add the following line to your crontab (e.g. `crontab -e`):

```sh
@reboot /usr/local/bin/backrest
```

#### Run on startup with systemd manually

```sh
sudo mv backrest /usr/local/bin/backrest
```

Create a systemd service file at `/etc/systemd/system/backrest.service` with the following contents:

```ini
[Unit]
Description=ResticWeb
After=network.target

[Service]
Type=simple
User=<your linux user>
Group=<your linux group>
ExecStart=/usr/local/bin/backrest
Environment="BACKREST_PORT=127.0.0.1:9898"

[Install]
WantedBy=multi-user.target
```

Then run the following commands to enable and start the service:

```sh
sudo systemctl enable backrest
sudo systemctl start backrest
```

> [!NOTE]
> You can set the Linux user and group to your primary user (e.g. `whoami` when logged in).

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

#### Using Homebrew

Backrest is provided as a [homebrew tap](https://github.com/garethgeorge/homebrew-backrest-tap). To install with brew run:

```sh
brew tap garethgeorge/homebrew-backrest-tap
brew install backrest
brew services start backrest
```

This tap uses [Brew services](https://github.com/Homebrew/homebrew-services) to launch and manage Backrest's lifecycle. Backrest will launch on startup and run on port ':9898` by default.

> [!NOTE]
> You may need to enable full disk access on MacOS for backrest to read all files on your computer when running backup operations. Not necessary for browsing.

#### Manually using the install script

Download a Darwin release from the [releases page](https://github.com/garethgeorge/backrest/releases) and install it to `/usr/local/bin`.

Extract the release you downloaded and run the install script:

```
# extract the release to a subfolder of the current directory
mkdir backrest && tar -xzvf backrest_Darwin_arm64.tar.gz -C backrest
# run the install script
cd backrest && ./install.sh
```

The install script will:

- Move the Backrest binary to `/usr/local/bin`
- Create a launch agent file at `~/Library/LaunchAgents/com.backrest.plist`
- Load the launch agent

Read the script before running it to make sure you are comfortable with these operations.

#### Manually

If setting up Backrest manually, it is recommended to install the binary to `/usr/local/bin` and run it manually. You can also create a launch agent to run it on startup or may run it manually when needed.

## Running on Windows

Download a Windows release from the [releases page](https://github.com/garethgeorge/backrest/releases) and install it to `C:\Program Files\Backrest\backrest.exe` (create the path if it does not exist). The binary should be run as administrator on first launch, otherwise the restic installation will fail and the process will terminate.

To run the binary on login, create a shortcut to the binary and place it in the `shell:startup` folder. See [this windows support article](https://support.microsoft.com/en-us/windows/add-an-app-to-run-automatically-at-startup-in-windows-10-150da165-dcd9-7230-517b-cf3c295d89dd) for more details.

> [!WARNING]
> * If you receive filesystem errors, you may need to run Backrest as an administrator for full filesystem access.
> * Backrest is **not** tested on Windows to the same extent as Linux and macOS. Some features may not work as expected.


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
| `BACKREST_CONFIG`         | Path to config file         | `%appdata%\backrest`                                                                       |
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
go install github.com/bufbuild/buf/cmd/buf@v1.27.2
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
npm install -g @bufbuild/protoc-gen-es @connectrpc/protoc-gen-connect-es
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
