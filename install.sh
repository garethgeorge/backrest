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

stop_systemd_service() {
  if systemctl is-active --quiet backrest; then
    sudo systemctl stop backrest
    echo "Paused backrest for update"
  fi
}

stop_openrc_service() {
  sudo rc-service backrest --ifstarted stop
  echo "Paused backrest for update (if started)"
}

install_unix() {
  echo "Installing backrest to /usr/local/bin"
  sudo mkdir -p /usr/local/bin

  sudo cp $(ls -1 backrest | head -n 1) /usr/local/bin
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

  echo "Enabling systemd service backrest.service"
  sudo systemctl enable backrest  
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

command=/usr/local/bin/backrest
command_background=true
pidfile="/run/\${RC_SVCNAME}.pid"
command_user="$(whoami):$(whoami)"
supervisor=supervise-daemon

export BACKREST_PORT=$BACKREST_PORT
EOM

  sudo chmod 755 /etc/init.d/backrest
  echo "Adding backrest to runlevel default"
  sudo rc-update add backrest default
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

  systemctl --version
  systemd_=$?
  rc-status --version
  openrc_=$?

  if [ $systemd_ -eq 0 ]; then
    echo "Systemd found."

    stop_systemd_service
    install_unix
    create_systemd_service
    echo "Reloading systemd service"
    sudo systemctl start backrest
  elif [ $openrc_ -eq 0 ]; then
    echo "Openrc found."

    stop_openrc_service
    install_unix
    create_openrc_service
    echo "Reloading openrc service"
    sudo rc-service backrest start
  else
    echo "neither systemd nor openrc were found"
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
