#! /bin/bash

cd "$(dirname "$0")" # cd to the directory of this script

# Parse command line arguments
ALLOW_REMOTE_ACCESS=false
for arg in "$@"; do
  case $arg in
    --allow-remote-access)
      ALLOW_REMOTE_ACCESS=true
      shift
      ;;
    *)
      echo "Unknown option: $arg"
      echo "Usage: $0 [--allow-remote-access]"
      echo "  --allow-remote-access: Allow remote access by binding to 0.0.0.0:9898 instead of 127.0.0.1:9898"
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

install_or_update_unix() {
  if systemctl is-active --quiet backrest; then
    sudo systemctl stop backrest
    echo "Paused backrest for update"
  fi
  install_unix
}

install_unix() {
  echo "Installing backrest to /usr/local/bin"
  sudo mkdir -p /usr/local/bin

  sudo cp $(ls -1 backrest | head -n 1) /usr/local/bin
}

create_systemd_service() {
  if [ ! -d /etc/systemd/system ]; then
    echo "Systemd not found. This script is only for systemd based systems."
    exit 1
  fi

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
Environment="BACKREST_PORT=$BACKREST_PORT"

[Install]
WantedBy=multi-user.target
EOM

  echo "Reloading systemd daemon"
  sudo systemctl daemon-reload
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

OS=$(uname -s)
if [ "$OS" = "Darwin" ]; then
  echo "Installing on Darwin"
  install_unix
  create_launchd_plist
  enable_launchd_plist
  sudo xattr -d com.apple.quarantine /usr/local/bin/backrest # remove quarantine flag
elif [ "$OS" = "Linux" ]; then
  echo "Installing on Linux"
  install_or_update_unix
  create_systemd_service
  echo "Enabling systemd service backrest.service"
  sudo systemctl enable backrest
  sudo systemctl start backrest
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
