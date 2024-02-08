<img src="./webui/assets/logo-black.svg" width="400px"/>

[![Build and Test](https://github.com/garethgeorge/backrest/actions/workflows/build-and-test.yml/badge.svg)](https://github.com/garethgeorge/backrest/actions/workflows/build-and-test.yml)


Backrest is a web-accessible backup solution built on top of [restic](https://restic.net/). Backrest provides a WebUI which wraps the restic CLI and makes it easy to create repos, browse snapshots, and restore files. Additionally, Backrest can run in the background and take an opinionated approach to scheduling snapshots and orchestrating repo health operations.

By building on restic Backrest gets the advantages of restic mature feature set: restic provides fast, reliable (used by tens of thousands of individuals and by corporations in production environments), and secure backup operations. Backrest itself is built in Golang (matching restic's implementation) and is shipped as a self-contained and light weight (<20 MB on all platforms) binary with no dependecies other than restic (which backrest can download for you and keep up to date).

This project aims to be the easiest way to setup and get started with backups on any system. You can expect to be able to perform all operations from the web interface but should you ever need more poworeful control, you are free to browse your repo and perform operations using the [restic cli](https://restic.readthedocs.io/en/latest/manual_rest.html). Backrest safely detects and imports external operations (e.g. manual backups).

**Platform Support**

 * [Docker](https://hub.docker.com/r/garethgeorge/backrest)
 * Linux
 * MacOS
 * (experimental, no CI coverage) Windows

**Features**

 * WebUI for restic supports local and remote access (e.g. run on a NAS and access from your desktop)
 * Realtime UI e.g. live progress bars for backup operations and live refreshes of operation history.
 * Snapshot browser
 * Restore interface
 * Configurable backup plans
   * Cronexprs provide flexible scheduling options
   * Configurable retention policies with restic forget (e.g. keep 1 snapshot per day for 30 days, 1 snapshot per week for 6 months, etc)
   * Include lists
   * Exclusion lists
   * Add custom CLI flags for detailed control of restic e.g. for use with rclone
   * Supported destinations are any restic supported repository (e.g. local filesystem, S3, Backblaze, rclone, etc).
 * Automatic repo health operations e.g. forget and prune.
   * Forget runs after every backup.
   * Prune once every 7 days by default.
 * Multiple backup plans can be configured running on different schedules and with different retention policies.
 * Multiple restic repositories can be configured and used in different plans.
 * Event hooks for notifications
   * Lifecycle hooks are triggered with status information from operations backrest runs on your behalf.
   * Supported services: Discord, Gotify, Shell Command
   * Events: Backup Start, Backup Finish, Backup Error, Any Error
 * Multi-user authentication: backrest can be secured with a username and password.

# Preview

Operation History & Snapshot Browser

<img src="https://f000.backblazeb2.com/file/gshare/screenshots/2024/Screenshot+from+2024-01-04+18-30-14.png" width="700px" />

Repo Creation Wizard

<img src="https://f000.backblazeb2.com/file/gshare/screenshots/2024/Screenshot+from+2024-01-04+18-19-50.png" width="700px" />


# User Guide

[See the backrest wiki](https://github.com/garethgeorge/backrest/wiki).

# Installation

Backrest is packaged as a single executable. It can be run directly on Linux, MacOS, and Windows. [restic](https://github.com/restic/restic) will be downloaded and installed in the data directory on first run.

Download options

 * Download and run a release from the [releases page](https://github.com/garethgeorge/backrest/releases).
 * Build from source ([see below](#building)).
 * Run with docker: `garethgeorge/backrest:latest` ([see on dockerhub](https://hub.docker.com/r/garethgeorge/backrest))

Backrest is accessible from a web browser. By default it binds to `0.0.0.0:9898` and can be accessed at `http://localhost:9898`. Change the port with the `BACKREST_PORT` environment variable e.g. `BACKREST_PORT=127.0.0.1 backrest` to listen only on local interfaces. On first startup backrest will prompt you to create a default username and password, this can be changed later in the settings page.

Note: backrest installs a specific restic version to ensure that the version of restic matches the version backrest is tested against. This provides the best guarantees for stability. If you wish to use a different version of restic OR if you would prefer to install restic manually you may do so by setting the `BACKREST_RESTIC_COMMAND` environment variable to the path of the restic binary you wish to use.

## Running with Docker Compose

Docker image: https://hub.docker.com/r/garethgeorge/backrest

Example compose file:

```yaml
version: "3.2"
services:
  backrest:
    image: garethgeorge/backrest
    container_name: backrest 
    volumes:
      - ./backrest/data:/data
      - ./backrest/config:/config
      - ./backrest/cache:/cache
      - /MY-BACKUP-DATA:/userdata # mount your directories to backup somewhere in the filesystem
      - /MY-REPOS:/repos # (optional) mount your restic repositories somewhere in the filesystem.
    environment:
      - BACKREST_DATA=/data # path for backrest data. restic binary and the database are placed here.
      - BACKREST_CONFIG=/config/config.json # path for the backrest config file.
      - XDG_CACHE_HOME=/cache # path for the restic cache which greatly improves performance.
    restart: unless-stopped
    ports:
      - 9898:9898
```

## Running on Linux

Download a release from the [releases page](https://github.com/garethgeorge/backrest/releases)

#### Using systemd with the install script (Recommended)

Extract the release you downloaded and run the install script:

```
# extract the release to a subfolder of the current directory
mkdir backrest && tar -xzvf backrest_Linux_x86_64.tar.gz -C backrest
# run the install script
cd backrest && ./install.sh
```

The install script will:

 * Move the backrest binary to `/usr/local/bin`
 * Create a systemd service file at `/etc/systemd/system/backrest.service`
 * Enable and start the service

Read the script before running it to make sure you are comfortable with these operations.

#### Run on startup with cron (Basic)

Move the backrest binary to `/usr/local/bin`:

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

[Install]
WantedBy=multi-user.target
```

Then run the following commands to enable and start the service:

```sh
sudo systemctl enable backrest
sudo systemctl start backrest
```

Note: you can set the linux user and group to your primary user (e.g. `whoami` when logged in).

## Running on MacOS

Download a Darwin release from the [releases page](https://github.com/garethgeorge/backrest/releases) and install it to `/usr/local/bin`.

#### Using launchd with the install script (Recommended)

Extract the release you downloaded and run the install script:

```
# extract the release to a subfolder of the current directory
mkdir backrest && tar -xzvf backrest_Darwin_arm64.tar.gz -C backrest
# run the install script
cd backrest && ./install.sh
```

The install script will:

 * Move the backrest binary to `/usr/local/bin`
 * Create a launch agent file at `~/Library/LaunchAgents/com.backrest.plist`
 * Load the launch agent

Read the script before running it to make sure you are comfortable with these operations.

#### Manually

If setting up backrest manually it's recommended to install the binary to `/usr/local/bin` and run it manually. You can also create a launch agent to run it on startup or may run it manually when needed.

## Running on Windows

Download a Windows release from the [releases page](https://github.com/garethgeorge/backrest/releases) and install it to `C:\Program Files\Backrest\backrest.exe` (create the path if it does not exist).

To run the binary on login, create a shortcut to the binary and place it in the `shell:startup` folder. See [this windows support article](https://support.microsoft.com/en-us/windows/add-an-app-to-run-automatically-at-startup-in-windows-10-150da165-dcd9-7230-517b-cf3c295d89dd) for more details.

warning: If you get filesystem errors you may need to run Backrest as administrator for full filesystem access.

warning: Backrest is not tested on Windows to the same bar as Linux and MacOS. Some features may not work as expected.

# Configuration

## Environment Variables

 * `BACKREST_PORT` - the port to bind to. Defaults to 9898.
 * `BACKREST_CONFIG` - the path to the config file. Defaults to `$HOME/.config/backrest/config.json` or if `$XDG_CONFIG_HOME` is set, `$XDG_CONFIG_HOME/backrest/config.json`.
 * `BACKREST_DATA` - the path to the data directory. Defaults to `$HOME/.local/share/backrest` or if `$XDG_DATA_HOME` is set, `$XDG_DATA_HOME/backrest`.
 * `BACKREST_RESTIC_COMMAND` - the path to the restic binary. Defaults managed version of restic which will be downloaded and installed in the data directory.
 * `XDG_CACHE_HOME` -- the path to the cache directory. This is propagated to restic. 
