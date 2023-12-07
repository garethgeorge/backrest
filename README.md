# ResticUI

[![Build and Test](https://github.com/garethgeorge/resticui/actions/workflows/build-and-test.yml/badge.svg)](https://github.com/garethgeorge/resticui/actions/workflows/build-and-test.yml)

ResticUI is a WebUI wrapper for [restic](https://restic.net/). It is intended to be used as a self-hosted application for managing backups of your data.

The goals of this project are:

 * Full featured web UI for restic: supports all basic operations (e.g. backup, restore, browse snapshots, prune old data, etc).
 * Interactive: UI is fast and responds to operation progress in real time (e.g. backups show live progress bars).
 * Safe: all backups leverage simple [restic](https://restic.net/) features and have test coverage. 
 * Easy to pull back the curtain: all common operations should be possible from the UI, but it should be easy to drop down to the command line and use restic directly if needed.
 * Lightweight: your backup orchestration should blend into the background. The web UI binary is fully self contained as a single executable and the binary is <20 MB with very light memory overhead at runtime.

OS Support

 * Linux 
 * MacOS (Darwin)
 * Windows (note: must be run as administrator on first execution to install the restic binary in Program Files).

# Getting Started 

## Running 

Installation options

 * Download and run a release from the [releases page](https://github.com/garethgeorge/resticui/releases).
 * Build from source ([see below](#building)).
 * Run with docker: `garethgeorge/resticweb:latest` ([see on dockerhub](https://hub.docker.com/repository/docker/garethgeorge/resticweb/))

ResticUI is accessible from a web browser. By default it binds to `0.0.0.0:9898` and can be accessed at `http://localhost:9898`. 


# Configuration

## Environment Variables

 * `RESTICUI_PORT` - the port to bind to. Defaults to 9898.
 * `RESTICUI_CONFIG_PATH` - the path to the config file. Defaults to `$HOME/.config/resticui/config.json` or if `$XDG_CONFIG_HOME` is set, `$XDG_CONFIG_HOME/resticui/config.json`.
 * `RESTICUI_DATA_DIR` - the path to the data directory. Defaults to `$HOME/.local/share/resticui` or if `$XDG_DATA_HOME` is set, `$XDG_DATA_HOME/resticui`.
 * `RESTICUI_RESTIC_BIN_PATH` - the path to the restic binary. Defaults managed version of restic which will be downloaded and installed in the data directory.
 * `XDG_CACHE_HOME` -- the path to the cache directory. This is propagated to restic. 


## Configuring ResticWeb at startup

ResticWeb is shipped today as a standalone executable, in future releases we'll provide system service installation for common operating systems.

### Linux

<details>

#### Cron (Basic)

Move the resticweb binary to `/usr/local/bin`:

```sh
sudo mv resticui /usr/local/bin/resticui
```

Add the following line to your crontab (e.g. `crontab -e`):

```sh
@reboot /usr/local/bin/resticui
```

#### Systemd (Recommended)

Move the resticweb binary to `/usr/local/bin`:

```sh
sudo mv resticui /usr/local/bin/resticui
```

Create a systemd service file at `/etc/systemd/system/resticweb.service` with the following contents:

```ini
[Unit]
Description=ResticWeb
After=network.target

[Service]
Type=simple
User=<your linux user>
Group=<your linux group>
ExecStart=/usr/local/bin/resticweb

[Install]
WantedBy=multi-user.target
```

Then run the following commands to enable and start the service:

```sh
sudo systemctl enable resticweb
sudo systemctl start resticweb
```

Note: you can set the linux user and group to your primary user (e.g. `whoami` when logged in).

</details>

# Developer Setup

## Dev Depedencies

**Build Dependencies**

 * Node.JS for UI development
 * Go 1.21 or greater for server development
 * go.rice `go install github.com/GeertJohan/go.rice@latest` and `go install github.com/GeertJohan/go.rice/rice@latest`

**To Edit Protobuffers**
```sh
apt install -y protobuf-compiler
go install \
    github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest \
    github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
go install github.com/grpc-ecosystem/protoc-gen-grpc-gateway-ts@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/bufbuild/buf/cmd/buf@v1.27.2
```
## Building

```sh
(cd webui && npm i && npm run build)
(cd cmd/resticui && go build .)
```
