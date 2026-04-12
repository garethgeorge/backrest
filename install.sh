#! /bin/bash

cd "$(dirname "$0")" # cd to the directory of this script

# Parse command line arguments
ALLOW_REMOTE_ACCESS=false
FROM_SOURCE=false
INSTALL_MODE=""  # "tray", "service", or "" (auto-detect)
for arg in "$@"; do
  case $arg in
    --allow-remote-access)
      ALLOW_REMOTE_ACCESS=true
      shift
      ;;
    --from-source)
      FROM_SOURCE=true
      shift
      ;;
    --tray)
      INSTALL_MODE="tray"
      shift
      ;;
    --no-tray)
      INSTALL_MODE="service"
      shift
      ;;
    *)
      echo "Unknown option: $arg"
      echo "Usage: $0 [--allow-remote-access] [--from-source] [--tray] [--no-tray]"
      echo "  --allow-remote-access: Allow remote access by binding to 0.0.0.0:9898 instead of 127.0.0.1:9898"
      echo "  --from-source: Build binary from source"
      echo "  --tray: Install as a desktop tray app (XDG autostart) instead of a system service"
      echo "  --no-tray: Install as a system service even if a desktop session is detected"
      exit 1
      ;;
  esac
done

# Set the appropriate port based on the flag
if [ "$ALLOW_REMOTE_ACCESS" = true ]; then
  BACKREST_PORT="0.0.0.0:9898"
  echo "Remote access enabled: Backrest will bind to 0.0.0.0:9898"
else
  BACKREST_PORT="127.0.0.1:9898"
  echo "Local access only: Backrest will bind to 127.0.0.1:9898, run ./install.sh --allow-remote-access to enable remote access"
fi

if [ "$FROM_SOURCE" = true ]; then
  if ! command -v go &> /dev/null; then
      echo "Error: go is not installed or not in PATH."
      exit 1
  fi

  if ! command -v npm &> /dev/null; then
      echo "Error: npm is not installed or not in PATH."
      exit 1
  fi

  echo "Building from source..."

  echo "Installing webui dependencies..."
  pushd webui > /dev/null || exit 1
  if command -v pnpm &> /dev/null; then
      echo "Using pnpm..."
      pnpm install
  else
      echo "Using npm..."
      npm install
  fi
  popd > /dev/null || exit 1

  go generate ./...
  go build -tags tray -o backrest ./cmd/backrest
fi

# Returns 0 if a D-Bus session and graphical display are available (tray mode is viable)
detect_dbus_desktop() {
  # Check for a graphical display
  if [ -z "$DISPLAY" ] && [ -z "$WAYLAND_DISPLAY" ]; then
    return 1
  fi
  # Check for a running D-Bus session daemon
  if [ -n "$DBUS_SESSION_BUS_ADDRESS" ]; then
    return 0
  fi
  if command -v dbus-launch &> /dev/null || command -v dbus-daemon &> /dev/null; then
    return 0
  fi
  return 1
}

stop_systemd_service() {
  if systemctl is-active --quiet backrest 2>/dev/null; then
    sudo systemctl stop backrest
    echo "Stopped backrest service"
  fi
}

stop_openrc_service() {
  sudo rc-service backrest --ifstarted stop 2>/dev/null
  echo "Stopped backrest service (if running)"
}

remove_systemd_service() {
  if [ -f /etc/systemd/system/backrest.service ]; then
    stop_systemd_service
    sudo systemctl disable backrest 2>/dev/null || true
    sudo rm -f /etc/systemd/system/backrest.service
    sudo systemctl daemon-reload
    echo "Removed systemd service"
  fi
}

remove_openrc_service() {
  if [ -f /etc/init.d/backrest ]; then
    stop_openrc_service
    sudo rc-update del backrest default 2>/dev/null || true
    sudo rm -f /etc/init.d/backrest
    echo "Removed openrc service"
  fi
}

remove_autostart_desktop() {
  local desktop_file="${XDG_CONFIG_HOME:-$HOME/.config}/autostart/backrest.desktop"
  if [ -f "$desktop_file" ]; then
    rm -f "$desktop_file"
    echo "Removed tray autostart entry"
  fi
}

install_unix() {
  echo "Installing backrest to /usr/local/bin"
  sudo mkdir -p /usr/local/bin
  sudo cp $(ls -1 backrest | head -n 1) /usr/local/bin
}

create_autostart_desktop() {
  local autostart_dir="${XDG_CONFIG_HOME:-$HOME/.config}/autostart"
  local desktop_file="$autostart_dir/backrest.desktop"

  mkdir -p "$autostart_dir"

  cat > "$desktop_file" <<- EOM
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

create_systemd_service() {
  if [ -f /etc/systemd/system/backrest.service ]; then
    echo "Systemd unit already exists. Skipping creation."
    return 0
  fi

  echo "Creating systemd service at /etc/systemd/system/backrest.service"

  sudo tee /etc/systemd/system/backrest.service > /dev/null <<- EOM
[Unit]
Description=Backrest Service
After=network.target

[Service]
Type=simple
User=$(whoami)
Group=$(whoami)
ExecStart=/usr/local/bin/backrest
Restart=on-failure
Environment="BACKREST_PORT=$BACKREST_PORT"

[Install]
WantedBy=multi-user.target
EOM

  echo "Reloading systemd daemon"
  sudo systemctl daemon-reload
}

create_openrc_service() {
  if [ -f /etc/init.d/backrest ]; then
    echo "Openrc service already exists. Skipping creation."
    return 0
  fi

  echo "Creating openrc service at /etc/init.d/backrest"

  sudo tee /etc/init.d/backrest > /dev/null <<- EOM
#!/sbin/openrc-run
description="Backrest Service"

depend() {
    need loopback
    use net logger
}

: \${BACKREST_PORT:=${BACKREST_PORT}}

command=/usr/local/bin/backrest
command_background=true
command_args="-bind-address \${BACKREST_PORT}"
pidfile="/run/\${RC_SVCNAME}.pid"
command_user="$(whoami):$(whoami)"
supervisor=supervise-daemon

EOM

  sudo chmod 755 /etc/init.d/backrest
}

create_launchd_plist() {
  echo "Creating launchd plist at /Library/LaunchAgents/com.backrest.plist"

  sudo tee /Library/LaunchAgents/com.backrest.plist > /dev/null <<- EOM
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
  echo "Trying to unload any previous version of com.backrest.plist"
  launchctl unload /Library/LaunchAgents/com.backrest.plist || true
  echo "Loading com.backrest.plist"
  launchctl load -w /Library/LaunchAgents/com.backrest.plist
}

install_linux_tray() {
  # Migrate away from any existing service installation
  remove_systemd_service
  remove_openrc_service

  install_unix
  create_autostart_desktop

  echo ""
  echo "Backrest installed in tray mode."
  echo "It will start automatically when you next log in to your desktop session."
  echo "To start it now, run: /usr/local/bin/backrest --tray &"
}

install_linux_service() {
  # Migrate away from any existing tray installation
  remove_autostart_desktop
  # Kill any running tray process so the new service can bind the port
  pkill -f "backrest.*--tray" 2>/dev/null || true

  systemctl --version &>/dev/null
  systemd_=$?
  rc-status --version &>/dev/null
  openrc_=$?

  if [ $systemd_ -eq 0 ]; then
    echo "Systemd found."
    stop_systemd_service
    install_unix
    create_systemd_service
    echo "Enabling systemd service backrest.service"
    sudo systemctl enable backrest
    echo "Starting systemd service"
    sudo systemctl start backrest
  elif [ $openrc_ -eq 0 ]; then
    echo "Openrc found."
    stop_openrc_service
    install_unix
    create_openrc_service
    echo "Adding backrest to runlevel default"
    sudo rc-update add backrest default
    echo "Starting openrc service"
    sudo rc-service backrest start
  else
    echo "Neither systemd nor openrc were found. Installing binary only."
    install_unix
  fi
}

OS=$(uname -s)
if [ "$OS" = "Darwin" ]; then
  echo "Installing on Darwin"
  install_unix
  create_launchd_plist
  enable_launchd_plist
  sudo xattr -d com.apple.quarantine /usr/local/bin/backrest # remove quarantine flag
elif [ "$OS" = "Linux" ]; then
  echo "Installing on Linux"

  # Auto-detect install mode if not specified
  if [ -z "$INSTALL_MODE" ]; then
    if detect_dbus_desktop; then
      echo "Desktop session with D-Bus detected — installing as tray app."
      echo "Use --no-tray to install as a system service instead."
      INSTALL_MODE="tray"
    else
      echo "No desktop session detected — installing as system service."
      echo "Use --tray to install as a tray app instead."
      INSTALL_MODE="service"
    fi
  fi

  if [ "$INSTALL_MODE" = "tray" ]; then
    install_linux_tray
  else
    install_linux_service
  fi
else
  echo "Unknown OS: $OS. This script only supports Darwin and Linux."
  exit 1
fi

echo "Logs are available at ~/.local/share/backrest/processlogs/backrest.log"
if [ "$ALLOW_REMOTE_ACCESS" = true ]; then
  echo "Access backrest WebUI at http://0.0.0.0:9898 (remote access enabled)"
  echo "Note: Remote access allows connections from any IP address. Ensure proper firewall configuration or that authentication is set up."
else
  echo "Access backrest WebUI at http://localhost:9898"
fi
