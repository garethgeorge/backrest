#!/bin/bash
set -euo pipefail

# Detect curl-piped invocation (e.g. `curl ... | bash` or `curl ... | sudo bash`).
# Under that mode $0 is "bash" rather than a path to a real script file.
if [ -f "$0" ]; then
  PIPED_INVOCATION=false
  SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
else
  PIPED_INVOCATION=true
  SCRIPT_DIR=""
fi

# --- Defaults ---
ALLOW_REMOTE_ACCESS=false
INSTALL_MODE=""       # "tray", "service", or "" (auto-detect)
UNINSTALL_ONLY=false
ACQUISITION_MODE=""   # "local", "source", "release", or "" (auto-detect)
RELEASE_BINARY=""     # set by acquire_binary in release mode
ROOT_INSTALL=false    # acknowledged install as root (services run as root)

# --- Parse arguments ---
for arg in "$@"; do
  case $arg in
    --uninstall)
      UNINSTALL_ONLY=true
      ;;
    --allow-remote-access)
      ALLOW_REMOTE_ACCESS=true
      ;;
    --experimental-linux-tray)
      INSTALL_MODE="tray"
      ;;
    --no-tray)
      INSTALL_MODE="service"
      ;;
    --from-local)
      ACQUISITION_MODE="local"
      ;;
    --from-source)
      ACQUISITION_MODE="source"
      ;;
    --from-release)
      ACQUISITION_MODE="release"
      ;;
    --root)
      ROOT_INSTALL=true
      ;;
    *)
      progname="$0"
      [ "$PIPED_INVOCATION" = true ] && progname="install.sh"
      echo "Unknown option: $arg"
      echo "Usage: $progname [OPTIONS]"
      echo ""
      echo "Options:"
      echo "  --uninstall                 Uninstall backrest (remove all artifacts, don't reinstall)"
      echo "  --experimental-linux-tray   Install as desktop tray app (Linux only, XDG autostart; experimental)"
      echo "  --no-tray                   Install as system service (Linux: systemd/openrc; skip tray)"
      echo "  --allow-remote-access       Bind to 0.0.0.0:9898 instead of 127.0.0.1:9898"
      echo "  --from-local                Use ./backrest binary in script directory (default if present)"
      echo "  --from-source               Build from source"
      echo "  --from-release              Download latest release from GitHub"
      echo "  --root                      Acknowledge install as root (services run as root)"
      echo ""
      echo "Curl-piped invocation:"
      echo "  curl -fsSL https://raw.githubusercontent.com/garethgeorge/backrest/main/install.sh | sudo bash"
      echo "  curl -fsSL https://raw.githubusercontent.com/garethgeorge/backrest/main/install.sh | sudo bash -s -- --no-tray"
      exit 1
      ;;
  esac
done

# --- Curl-piped invocation guards ---
if [ "$PIPED_INVOCATION" = true ]; then
  case "$ACQUISITION_MODE" in
    local|source)
      echo "Error: --from-$ACQUISITION_MODE is not supported when running via 'curl | bash';"
      echo "the script has no on-disk location to read from. Either drop the flag"
      echo "(downloads the latest release from GitHub) or download install.sh to disk first."
      exit 1
      ;;
  esac
  # Force release acquisition; SCRIPT_DIR is empty so local detection wouldn't work anyway.
  if [ -z "$ACQUISITION_MODE" ]; then
    ACQUISITION_MODE="release"
  fi
fi

# --- Determine the install user (whose name goes into service units / autostart) ---
# When invoked via sudo, $(whoami) is "root" but $SUDO_USER points at the real user;
# we want services owned by that user, not root. Bare-root installs require --root.
if [ "$(id -u)" -eq 0 ]; then
  if [ "$ROOT_INSTALL" = true ]; then
    INSTALL_USER="root"
    INSTALL_HOME="/root"
  elif [ -n "${SUDO_USER:-}" ] && [ "$SUDO_USER" != "root" ]; then
    INSTALL_USER="$SUDO_USER"
    INSTALL_HOME="$(getent passwd "$SUDO_USER" 2>/dev/null | cut -d: -f6)"
    [ -z "$INSTALL_HOME" ] && INSTALL_HOME="/home/$SUDO_USER"
  else
    echo "Error: this script is being run as root directly."
    echo ""
    echo "Running as root means backrest will run as root and store its config"
    echo "and data under /root, which is rarely what you want. Either:"
    echo "  - Re-run as your normal user (sudo will be invoked when needed), or"
    echo "  - Pass --root to acknowledge that you really want a root install."
    exit 1
  fi
else
  INSTALL_USER="$(whoami)"
  INSTALL_HOME="$HOME"
fi

# Resolve the install user's XDG config dir. Don't honor $XDG_CONFIG_HOME from
# the current env when running via sudo -- that value belongs to root.
if [ "$(id -u)" -eq 0 ] && [ "$INSTALL_USER" != "root" ]; then
  INSTALL_XDG_CONFIG_HOME="$INSTALL_HOME/.config"
else
  INSTALL_XDG_CONFIG_HOME="${XDG_CONFIG_HOME:-$INSTALL_HOME/.config}"
fi

# Run a command as the install user (drop privileges if currently root).
run_as_install_user() {
  if [ "$(id -u)" -eq 0 ] && [ "$INSTALL_USER" != "root" ]; then
    sudo -u "$INSTALL_USER" "$@"
  else
    "$@"
  fi
}

# --- Bind address ---
if [ "$ALLOW_REMOTE_ACCESS" = true ]; then
  BACKREST_PORT="0.0.0.0:9898"
else
  BACKREST_PORT="127.0.0.1:9898"
fi

# --- Pre-authenticate sudo so subsequent calls don't prompt mid-install ---
# When piped from curl, stdin is occupied by the script body, so sudo can't read
# a password from it. We force a TTY prompt up front; if no TTY is available
# (e.g. fully non-interactive shell), fail clearly with the canonical fix.
if [ "$(id -u)" -ne 0 ]; then
  if ! sudo -n true 2>/dev/null; then
    if [ -t 0 ] || [ -t 1 ] || [ -r /dev/tty ]; then
      echo "Caching sudo credentials (you may be prompted for your password)..."
      if [ -r /dev/tty ]; then
        sudo -v </dev/tty
      else
        sudo -v
      fi
    else
      echo "Error: this script needs sudo, but no TTY is available to prompt for a password."
      if [ "$PIPED_INVOCATION" = true ]; then
        echo "Re-run with sudo on the outside, e.g.:"
        echo "  curl -fsSL https://raw.githubusercontent.com/garethgeorge/backrest/main/install.sh | sudo bash"
      else
        echo "Run 'sudo -v' first, or invoke the script with sudo."
      fi
      exit 1
    fi
  fi
fi

# =============================================================================
# Detection helpers
# =============================================================================

detect_previous_version() {
  if [ -x /usr/local/bin/backrest ]; then
    /usr/local/bin/backrest --version 2>/dev/null || echo "unknown"
  else
    echo ""
  fi
}

detect_previous_mode() {
  if [ -f /etc/systemd/system/backrest.service ]; then
    echo "systemd"
  elif [ -f /etc/init.d/backrest ]; then
    echo "openrc"
  elif [ -f /Library/LaunchAgents/com.backrest.plist ]; then
    echo "launchd"
  elif [ -f "$INSTALL_XDG_CONFIG_HOME/autostart/backrest.desktop" ]; then
    echo "tray"
  else
    echo ""
  fi
}

# Returns 0 if a D-Bus session and graphical display are available (tray mode viable)
# Currently unused: tray install is opt-in via --experimental-linux-tray.
# detect_dbus_desktop() {
#   if [ -z "${DISPLAY:-}" ] && [ -z "${WAYLAND_DISPLAY:-}" ]; then
#     return 1
#   fi
#   if [ -n "${DBUS_SESSION_BUS_ADDRESS:-}" ]; then
#     return 0
#   fi
#   if command -v dbus-launch &>/dev/null || command -v dbus-daemon &>/dev/null; then
#     return 0
#   fi
#   return 1
# }

# =============================================================================
# Removal functions (idempotent, shared by uninstall and pre-install cleanup)
# =============================================================================

remove_systemd_service() {
  if [ -f /etc/systemd/system/backrest.service ]; then
    if systemctl is-active --quiet backrest 2>/dev/null; then
      sudo systemctl stop backrest || true
    fi
    sudo systemctl disable backrest 2>/dev/null || true
    sudo rm -f /etc/systemd/system/backrest.service
    sudo systemctl daemon-reload
    echo "Removed systemd service"
  fi
}

remove_openrc_service() {
  if [ -f /etc/init.d/backrest ]; then
    sudo rc-service backrest --ifstarted stop 2>/dev/null || true
    sudo rc-update del backrest default 2>/dev/null || true
    sudo rm -f /etc/init.d/backrest
    echo "Removed openrc service"
  fi
}

remove_launchd_plist() {
  if [ -f /Library/LaunchAgents/com.backrest.plist ]; then
    local uid prefix
    uid="$(id -u "$(logname 2>/dev/null || echo "$USER")")"
    if [ "$(id -u)" -eq 0 ]; then
      prefix="sudo -u #$uid "
    else
      prefix=""
    fi
    echo "Tearing down launchd agent:"
    echo "  ${prefix}launchctl bootout gui/$uid/com.backrest"
    if [ "$(id -u)" -eq 0 ]; then
      sudo -u "#$uid" launchctl bootout "gui/$uid/com.backrest" 2>/dev/null || true
    else
      launchctl bootout "gui/$uid/com.backrest" 2>/dev/null || true
    fi
    launchctl unload /Library/LaunchAgents/com.backrest.plist 2>/dev/null || true
    sudo rm -f /Library/LaunchAgents/com.backrest.plist
    echo "Removed launchd plist"
  fi
}

remove_autostart_desktop() {
  local desktop_file="$INSTALL_XDG_CONFIG_HOME/autostart/backrest.desktop"
  if [ -f "$desktop_file" ]; then
    run_as_install_user rm -f "$desktop_file"
    echo "Removed tray autostart entry"
  fi
}

remove_binary() {
  if [ -f /usr/local/bin/backrest ]; then
    sudo rm -f /usr/local/bin/backrest
    echo "Removed /usr/local/bin/backrest"
  fi
}

# Remove all known install artifacts
remove_all() {
  remove_systemd_service
  remove_openrc_service
  remove_launchd_plist
  remove_autostart_desktop
  run_as_install_user pkill -f "backrest.*--tray" 2>/dev/null || true
  remove_binary
}

# =============================================================================
# Binary acquisition
# =============================================================================

acquire_binary() {
  local mode="$1"

  case "$mode" in
    local)
      if [ ! -f "$SCRIPT_DIR/backrest" ]; then
        echo "Error: no backrest binary found in $SCRIPT_DIR"
        exit 1
      fi
      echo "Using local binary from $SCRIPT_DIR/backrest"
      ;;
    source)
      echo "Building from source..."
      if ! command -v go &>/dev/null; then
        echo "Error: go is not installed or not in PATH."
        exit 1
      fi
      if ! command -v npm &>/dev/null; then
        echo "Error: npm is not installed or not in PATH."
        exit 1
      fi

      echo "Installing webui dependencies..."
      pushd "$SCRIPT_DIR/webui" >/dev/null
      if command -v pnpm &>/dev/null; then
        echo "Using pnpm..."
        pnpm install
      else
        echo "Using npm..."
        npm install
      fi
      popd >/dev/null

      (cd "$SCRIPT_DIR" && go generate ./... && go build -tags tray -o backrest ./cmd/backrest)
      echo "Build complete"
      ;;
    release)
      echo "Downloading latest release from GitHub..."
      local os arch artifact url tmpdir
      os="$(uname -s)"
      arch="$(uname -m)"

      # Map architecture to goreleaser artifact naming
      case "$arch" in
        x86_64)        arch="x86_64" ;;
        aarch64|arm64) arch="arm64" ;;
        armv6l)        arch="armv6" ;;
        armv7l)        arch="armv7" ;;
        *)
          echo "Error: unsupported architecture: $arch"
          exit 1
          ;;
      esac

      artifact="backrest_${os}_${arch}.tar.gz"
      url="https://github.com/garethgeorge/backrest/releases/latest/download/${artifact}"

      tmpdir="$(mktemp -d)"
      # shellcheck disable=SC2064
      trap "rm -rf '$tmpdir'" EXIT

      echo "Downloading $url"
      if command -v curl &>/dev/null; then
        curl -fsSL "$url" -o "$tmpdir/$artifact"
      elif command -v wget &>/dev/null; then
        wget -qO "$tmpdir/$artifact" "$url"
      else
        echo "Error: neither curl nor wget found"
        exit 1
      fi

      tar -xzf "$tmpdir/$artifact" -C "$tmpdir"
      if [ ! -f "$tmpdir/backrest" ]; then
        echo "Error: backrest binary not found in downloaded archive"
        exit 1
      fi

      # Stage in tmpdir; install_binary reads RELEASE_BINARY
      RELEASE_BINARY="$tmpdir/backrest"
      echo "Downloaded $artifact"
      ;;
    *)
      echo "Error: unknown acquisition mode: $mode"
      exit 1
      ;;
  esac
}

install_binary() {
  local src
  if [ -n "${RELEASE_BINARY:-}" ]; then
    src="$RELEASE_BINARY"
  else
    src="$SCRIPT_DIR/backrest"
  fi
  echo "Installing backrest to /usr/local/bin"
  sudo mkdir -p /usr/local/bin
  sudo cp "$src" /usr/local/bin/backrest
  sudo chmod +x /usr/local/bin/backrest
}

# =============================================================================
# Service creation functions
# =============================================================================

create_systemd_service() {
  echo "Creating systemd service at /etc/systemd/system/backrest.service"
  sudo tee /etc/systemd/system/backrest.service >/dev/null <<-EOM
[Unit]
Description=Backrest Service
After=network.target

[Service]
Type=simple
User=${INSTALL_USER}
Group=${INSTALL_USER}
ExecStart=/usr/local/bin/backrest
Restart=on-failure
Environment="BACKREST_PORT=$BACKREST_PORT"

[Install]
WantedBy=multi-user.target
EOM
  sudo systemctl daemon-reload
}

create_openrc_service() {
  echo "Creating openrc service at /etc/init.d/backrest"
  sudo tee /etc/init.d/backrest >/dev/null <<-EOM
#!/sbin/openrc-run
description="Backrest Service"

depend() {
    need loopback
    use net logger
}

export BACKREST_PORT="${BACKREST_PORT}"

command=/usr/local/bin/backrest
command_background=true
pidfile="/run/\${RC_SVCNAME}.pid"
command_user="${INSTALL_USER}:${INSTALL_USER}"
supervisor=supervise-daemon

EOM
  sudo chmod 755 /etc/init.d/backrest
}

create_autostart_desktop() {
  local autostart_dir="$INSTALL_XDG_CONFIG_HOME/autostart"
  local desktop_file="$autostart_dir/backrest.desktop"

  run_as_install_user mkdir -p "$autostart_dir"
  run_as_install_user tee "$desktop_file" >/dev/null <<-EOM
[Desktop Entry]
Name=Backrest
Comment=Backrest backup manager tray applet
Exec=env BACKREST_PORT=$BACKREST_PORT /usr/local/bin/backrest --tray
Icon=backrest
Type=Application
Categories=Utility;
X-GNOME-Autostart-enabled=true
EOM
  echo "Created tray autostart entry at $desktop_file"
}

create_launchd_plist() {
  echo "Creating launchd plist at /Library/LaunchAgents/com.backrest.plist"
  sudo tee /Library/LaunchAgents/com.backrest.plist >/dev/null <<-EOM
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.backrest</string>
    <key>ProgramArguments</key>
    <array>
    <string>/usr/local/bin/backrest</string>
    </array>
    <key>KeepAlive</key>
    <true/>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
        <key>BACKREST_PORT</key>
        <string>$BACKREST_PORT</string>
    </dict>
</dict>
</plist>
EOM
}

enable_launchd_plist() {
  local uid target prefix
  uid="$(id -u "$(logname 2>/dev/null || echo "$USER")")"
  target="gui/$uid"

  if [ "$(id -u)" -eq 0 ]; then
    prefix="sudo -u #$uid "
  else
    prefix=""
  fi

  echo "Bootstrapping launchd agent into $target:"
  echo "  ${prefix}launchctl bootout $target/com.backrest    # tear down any existing instance"
  echo "  ${prefix}launchctl bootstrap $target /Library/LaunchAgents/com.backrest.plist"
  echo "  ${prefix}launchctl kickstart -k $target/com.backrest"

  if [ "$(id -u)" -eq 0 ]; then
    sudo -u "#$uid" launchctl bootout "$target/com.backrest" 2>/dev/null || true
    sudo -u "#$uid" launchctl bootstrap "$target" /Library/LaunchAgents/com.backrest.plist
    sudo -u "#$uid" launchctl kickstart -k "$target/com.backrest"
  else
    launchctl bootout "$target/com.backrest" 2>/dev/null || true
    launchctl bootstrap "$target" /Library/LaunchAgents/com.backrest.plist
    launchctl kickstart -k "$target/com.backrest"
  fi
  echo "Loaded and started launchd agent"
  echo ""
  echo "To manually stop/unload later:"
  echo "  ${prefix}launchctl bootout $target/com.backrest"
  echo "To manually start/load again:"
  echo "  ${prefix}launchctl bootstrap $target /Library/LaunchAgents/com.backrest.plist"
  echo "  ${prefix}launchctl kickstart -k $target/com.backrest"
}

# =============================================================================
# Install orchestration
# =============================================================================

install_linux() {
  local mode="$1"

  case "$mode" in
    tray)
      install_binary
      create_autostart_desktop
      echo ""
      echo "Backrest installed in tray mode."
      echo "It will start automatically when you next log in to your desktop session."
      # Try to start it now in the background so the user doesn't have to log out/in.
      # This only works if we have access to the user's graphical session; if not,
      # the autostart entry will still pick it up on next login.
      if [ "$(id -u)" -eq 0 ] && [ "$INSTALL_USER" != "root" ]; then
        echo "Starting backrest tray as $INSTALL_USER..."
        sudo -u "$INSTALL_USER" \
          env BACKREST_PORT="$BACKREST_PORT" \
          nohup /usr/local/bin/backrest --tray >/dev/null 2>&1 &
        disown 2>/dev/null || true
        echo "(If the tray icon doesn't appear, your graphical session may not be"
        echo " reachable from this shell -- log out/in and the autostart entry will"
        echo " launch it. Or run manually: /usr/local/bin/backrest --tray &)"
      else
        echo "Starting backrest tray..."
        BACKREST_PORT="$BACKREST_PORT" \
          nohup /usr/local/bin/backrest --tray >/dev/null 2>&1 &
        disown 2>/dev/null || true
      fi
      ;;
    service)
      install_binary
      if systemctl --version &>/dev/null; then
        echo "Systemd detected."
        create_systemd_service
        sudo systemctl enable backrest
        sudo systemctl start backrest
        echo "Started backrest systemd service"
      elif rc-status --version &>/dev/null; then
        echo "OpenRC detected."
        create_openrc_service
        sudo rc-update add backrest default
        sudo rc-service backrest start
        echo "Started backrest openrc service"
      else
        echo "Neither systemd nor openrc found. Binary installed without service."
      fi
      ;;
  esac
}

install_darwin() {
  install_binary
  sudo xattr -d com.apple.quarantine /usr/local/bin/backrest 2>/dev/null || true
  create_launchd_plist
  enable_launchd_plist
}

# =============================================================================
# Main
# =============================================================================

OS="$(uname -s)"
if [ "$OS" != "Darwin" ] && [ "$OS" != "Linux" ]; then
  echo "Error: unsupported OS '$OS'. This script supports Linux and macOS (Darwin) only."
  exit 1
fi

# Step 1: Detect previous installation
PREV_VERSION="$(detect_previous_version)"
PREV_MODE="$(detect_previous_mode)"

if [ -n "$PREV_VERSION" ]; then
  echo "Detected previous installation: version=${PREV_VERSION}, mode=${PREV_MODE:-unknown}"
elif [ -n "$PREV_MODE" ]; then
  echo "Detected previous installation: mode=${PREV_MODE} (version unknown)"
fi

# Step 2: Remove all existing artifacts
if [ -n "$PREV_MODE" ] || [ -n "$PREV_VERSION" ]; then
  echo "Removing previous installation..."
  remove_all
  echo ""
fi

# If uninstall-only, we're done
if [ "$UNINSTALL_ONLY" = true ]; then
  if [ -n "$PREV_MODE" ] || [ -n "$PREV_VERSION" ]; then
    echo "Uninstall complete."
  else
    echo "No existing backrest installation found. Nothing to remove."
  fi
  exit 0
fi

# Step 3: Determine acquisition mode
# (Already forced to "release" earlier under PIPED_INVOCATION, so SCRIPT_DIR is
# guaranteed non-empty when we hit the local-detect branch below.)
if [ -z "$ACQUISITION_MODE" ]; then
  if [ -n "$SCRIPT_DIR" ] && [ -f "$SCRIPT_DIR/backrest" ]; then
    ACQUISITION_MODE="local"
  else
    ACQUISITION_MODE="release"
  fi
fi

# Acquire the binary
acquire_binary "$ACQUISITION_MODE"

# Determine install mode
if [ "$OS" = "Darwin" ]; then
  EFFECTIVE_MODE="launchd"
elif [ -z "$INSTALL_MODE" ]; then
  # Auto-detect from previous install; default to service otherwise.
  if [ "$PREV_MODE" = "tray" ]; then
    INSTALL_MODE="tray"
    echo "Re-using previous install mode: tray"
  elif [ "$PREV_MODE" = "systemd" ] || [ "$PREV_MODE" = "openrc" ]; then
    INSTALL_MODE="service"
    echo "Re-using previous install mode: service"
  # elif detect_dbus_desktop; then
  #   INSTALL_MODE="tray"
  #   echo "Desktop session with D-Bus detected -- installing as tray app."
  #   echo "Use --no-tray to install as a system service instead."
  else
    INSTALL_MODE="service"
    echo "Defaulting to system service install."
  fi
  EFFECTIVE_MODE="$INSTALL_MODE"
else
  EFFECTIVE_MODE="$INSTALL_MODE"
fi

# Step 4: Install
echo ""
if [ "$OS" = "Darwin" ]; then
  echo "Installing on macOS (launchd)..."
  install_darwin
elif [ "$OS" = "Linux" ]; then
  echo "Installing on Linux (${EFFECTIVE_MODE})..."
  install_linux "$EFFECTIVE_MODE"
fi

# Step 5: Summary
NEW_VERSION="$(/usr/local/bin/backrest --version 2>/dev/null || echo "unknown")"

echo ""
echo "=== Installation Summary ==="
if [ -n "$PREV_VERSION" ]; then
  echo "  Previous version : $PREV_VERSION"
  echo "  New version      : $NEW_VERSION"
else
  echo "  Version          : $NEW_VERSION"
fi
echo "  Install mode     : $EFFECTIVE_MODE"
echo "  Logs             : ~/.local/share/backrest/processlogs/backrest.log"
if [ "$ALLOW_REMOTE_ACCESS" = true ]; then
  echo "  Access URL       : http://0.0.0.0:9898 (remote access enabled)"
  echo "  Note: Ensure proper firewall configuration or authentication is set up."
else
  echo "  Access URL       : http://localhost:9898"
fi
echo "============================"
