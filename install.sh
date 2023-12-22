#! /bin/bash

if [ "$EUID" -ne "0" ]; then
  echo "Please run as root e.g. sudo ./install.sh"
  exit
fi

install_unix() {
  echo "Installing backrest to /usr/local/bin"
  mkdir -p /usr/local/bin
  cp $(ls -1 backrest | head -n 1) /usr/local/bin
}

create_systemd_service() {
  if [ ! -d /etc/systemd/system ]; then
    echo "Systemd not found. This script is only for systemd based systems."
    exit 1
  fi

  echo "Creating systemd service at /etc/systemd/system/backrest.service"

  cat > /etc/systemd/system/backrest.service <<- EOM
[Unit]
Description=Backrest Service
After=network.target

[Service]
Type=simple
User=$(whoami)
Group=$(whoami)
ExecStart=/usr/local/bin/backrest

[Install]
WantedBy=multi-user.target
EOM
  
  echo "Reloading systemd daemon"
  systemctl daemon-reload
}

create_launchd_plist() {
  echo "Creating launchd plist at /Library/LaunchAgents/com.backrest.plist"

  cat > /Library/LaunchAgents/com.backrest.plist <<- EOM 
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
	<dict>
		<key>Label</key>
		<string>com.backrest</string>
		<key>Program</key>
		<string>/usr/local/bin/backrest</string>
		<key>RunAtLoad</key>
		<true/>
    <key>StandardOutPath</key>
    <string>/tmp/backrest.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/backrest.log</string>
	</dict>
</plist>
EOM
}

enable_launchd_plist() {
  echo "Enabling launchd plist com.backrest.plist"
  launchctl unload /Library/LaunchAgents/com.backrest.plist || true
  launchctl load -w /Library/LaunchAgents/com.backrest.plist
}

OS=$(uname -s)
if [ "$OS" = "Darwin" ]; then
  echo "Installing on Darwin"
  install_unix
  create_launchd_plist
  enable_launchd_plist
elif [ "$OS" = "Linux" ]; then
  echo "Installing on Linux"
  install_unix
  create_systemd_service
  echo "Enabling systemd service backrest.service"
  systemctl enable backrest
  systemctl start backrest
else
  echo "Unknown OS: $OS. This script only supports Darwin and Linux."
  exit 1
fi
