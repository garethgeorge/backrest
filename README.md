# Restora

[![Build and Test](https://github.com/garethgeorge/restora/actions/workflows/build-and-test.yml/badge.svg)](https://github.com/garethgeorge/restora/actions/workflows/build-and-test.yml)

Restora is a free and open-source web UI wrapper for [restic](https://restic.net/). 

**Project goals:**

 * Full featured web UI for restic: supports all basic operations (e.g. backup, restore, browse snapshots, prune old data, etc).
 * Interactive: UI is fast and responds to operation progress in real time (e.g. backups show live progress bars).
 * Safe: all backups leverage simple [restic](https://restic.net/) features and have test coverage. 
 * Easy to pull back the curtain: all common operations should be possible from the UI, but it should be easy to drop down to the command line and use restic directly if needed.
 * Lightweight: your backup orchestration should blend into the background. The web UI binary is fully self contained as a single executable and the binary is ~25 MB with very light memory overhead at runtime.
 * Runs everywhere: Restora should be able to run on any platform that restic supports and that you need backed up. Restora is originally conceived of as a self-hosted backup tool for NAS devices but runs just as well on an interactive Desktop or Laptop. 

**Platform Support**

 * Linux
 * Docker
 * MacOS (Darwin)
 * (experimental) Windows

**Features**

 * Scheduled (and one off) backup operations
 * Scheduled restic forget and prune operations with configurable retention policy (e.g. keep 1 snapshot per day for 30 days, 1 snapshot per week for 1 year, etc) to manage repo size.
 * Backup to local or remote repositories (e.g. S3, Backblaze, etc)
 * Graphical backup browser and restore interface
 * Real time progress visualization for backup and restore operations.

# Preview

|                                                                                                                |                                                                                                       |
| -------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| <img src="https://f000.backblazeb2.com/file/gshare/screenshots/restora-backup-view.png" width="400px"/>        | <img src="https://f000.backblazeb2.com/file/gshare/screenshots/restora-add-repo.png" width="400px" /> |
| <img src="https://f000.backblazeb2.com/file/gshare/screenshots/restora-realtime-progress.png" width="400px" /> | <img src="https://f000.backblazeb2.com/file/gshare/screenshots/restora-add-plan.png" width="400px" /> |

# Getting Started 

Restora is packaged as a single executable. It can be run directly on Linux, MacOS, and Windows with no dependencies. [restic](https://github.com/restic/restic) will be downloaded and installed automatically on first run.

Download options

 * Download and run a release from the [releases page](https://github.com/garethgeorge/restora/releases).
 * Build from source ([see below](#building)).
 * Run with docker: `garethgeorge/restora:latest` ([see on dockerhub](https://hub.docker.com/repository/docker/garethgeorge/restora/))

Restora is accessible from a web browser. By default it binds to `0.0.0.0:9898` and can be accessed at `http://localhost:9898`. 

## Running with Docker Compose

Docker image: https://hub.docker.com/garethgeorge/restora

<details>
Example compose file:

```yaml
version: "3.2"
services:
  restora:
    image: garethgeorge/restora
    container_name: restora 
    volumes:
      - ./restora/data:/data
      - ./restora/config:/config
      - ./restora/cache:/cache
      - /MY-BACKUP-DATA:/userdata # mount your directories to backup somewhere in the filesystem
    environment:
      - RESTORA_DATA=/data # path for restora data. restic binary and the database are placed here.
      - RESTORA_CONFIG=/config/config.json # path for the restora config file.
      - XDG_CACHE_HOME=/cache # path for the restic cache which greatly improves performance.
    restart: unless-stopped
```
</details>

## Running on Linux

<details>

Download a release from the [releases page](https://github.com/garethgeorge/restora/releases)

#### Run on startup with cron (Basic)

Move the restora binary to `/usr/local/bin`:

```sh
sudo mv restora /usr/local/bin/restora
```

Add the following line to your crontab (e.g. `crontab -e`):

```sh
@reboot /usr/local/bin/restora
```

#### Run on startup with systemd (Recommended)



Download a Linux release from the [releases page](https://github.com/garethgeorge/restora/releases). Move the restora binary to `/usr/local/bin`:

```sh
sudo mv restora /usr/local/bin/restora
```

Create a systemd service file at `/etc/systemd/system/restora.service` with the following contents:

```ini
[Unit]
Description=ResticWeb
After=network.target

[Service]
Type=simple
User=<your linux user>
Group=<your linux group>
ExecStart=/usr/local/bin/restora

[Install]
WantedBy=multi-user.target
```

Then run the following commands to enable and start the service:

```sh
sudo systemctl enable restora
sudo systemctl start restora
```

Note: you can set the linux user and group to your primary user (e.g. `whoami` when logged in).

</details>

## Running on MacOS

<details>

Download a Darwin release from the [releases page](https://github.com/garethgeorge/restora/releases) and install it to `/usr/local/bin`.

At the moment there is no automated way to run Restora on startup on MacOS. You can run it manually or create a launch agent to run it on startup. See [lingon](https://www.peterborgapps.com/lingon/) for a GUI tool to create a launch agent that runs Restora at startup.

</details>

## Running on Windows

<details>

Download a Windows release from the [releases page](https://github.com/garethgeorge/restora/releases) and install it to `C:\Program Files\Restora\restora.exe` (create the path if it does not exist).

To run the binary on login, create a shortcut to the binary and place it in the `shell:startup` folder. See [this windows support article](https://support.microsoft.com/en-us/windows/add-an-app-to-run-automatically-at-startup-in-windows-10-150da165-dcd9-7230-517b-cf3c295d89dd) for more details.

warning: Restora is not tested on Windows to the same bar as Linux and MacOS. Please report any issues you encounter. Some folders may not be accessible to Restora or to restic on Windows due to permissions issues.

</details>

# Configuration

## Environment Variables

 * `RESTORA_PORT` - the port to bind to. Defaults to 9898.
 * `RESTORA_CONFIG` - the path to the config file. Defaults to `$HOME/.config/restora/config.json` or if `$XDG_CONFIG_HOME` is set, `$XDG_CONFIG_HOME/restora/config.json`.
 * `RESTORA_DATA` - the path to the data directory. Defaults to `$HOME/.local/share/restora` or if `$XDG_DATA_HOME` is set, `$XDG_DATA_HOME/restora`.
 * `RESTORA_RESTIC_COMMAND` - the path to the restic binary. Defaults managed version of restic which will be downloaded and installed in the data directory.
 * `XDG_CACHE_HOME` -- the path to the cache directory. This is propagated to restic. 

# Usage 

## Adding a Repository

A restora repository maps to the concept of a restic repository (and is indeed a restic repo under-the-hood). A repository is a location where restora will store your backups.

To add a repository, click the "Add Repository" button on the side nav. You will be prompted to enter a name for the repository and a path to the repository. The path can be a local path or a remote path (e.g. an S3 bucket). See the [restic docs](https://restic.readthedocs.io/en/stable/030_preparing_a_new_repo.html) for more details on the types of repositories that restic supports. Restora allows you to configure environment variables which should be used to pass additional credentials for remote repositories. For example, if you are using an S3 bucket, you can configure the `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables to pass your AWS credentials to restic.

## Adding a Plan

A plan is a new concept introduced by restora. A plan is a set of rules for backing up data. A plan can be configured to backup one or more directories to a single repository. Each plan has it's own schedule and retention policy controlling when backups are run and how long backups are kept.

To add a plan, click the "Add Plan" button on the side nav. You will be prompted to enter a name for the plan and select a repository to backup to. You will then be prompted to select one or more directories to backup. A default retention policy is given but you can also pick between time based retention or keeping a configurable number of snapshots. 

## Running a Backup

Backups are run automatically based on the scheduled specified in your plan. You may additionally click the "Backup Now" button on the plan page to run a backup immediately. You can additionally trigger an immediate "Prune Now" or "Unlock Now" operation from the plan page, these operations are also run automatically in the course of a backup cycle but can be run manually if needed.

## Best Practices

 * Configure a reasonable retention policy for each plan. Restora performs well up to a history of ~1000s of snapshots but too many may eventually slow performance.
 * Backup your configuration (e.g. `$RESTORA_CONFIG` or `$HOME/.config/restora/config.json` by default on Linux/MacOS)
   * Your configuration contains the encryption keys for your repositories. If you loose this file you will not be able to restore your backups.
   * You may alternatively backup your encryption keys individually in which case you will be able to use restic directly to restore your backups.
