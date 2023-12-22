#! /bin/bash

if [ "$EUID" -ne "0" ]; then
  echo "Please run as root e.g. sudo ./install.sh"
  exit
fi

uninstall_unix() {
  echo "Uninstalling backrest from /usr/local/bin/backrest"
  rm -f /usr/local/bin/backrest
}

remove_systemd_service() {
  if [ ! -d /etc/systemd/system ]; then
    echo "Systemd not found. This script is only for systemd based systems."
    exit 1
  fi

  echo "Removing systemd service at /etc/systemd/system/backrest.service"
  systemctl stop backrest
  systemctl disable backrest
  rm -f /etc/systemd/system/backrest.service

  echo "Reloading systemd daemon"
  systemctl daemon-reload
}

remove_launchd_plist() {
  echo "Removing launchd plist at /Library/LaunchAgents/com.backrest.plist"

  launchctl unload /Library/LaunchAgents/com.backrest.plist || true
  rm /Library/LaunchAgents/com.backrest.plist
}

OS=$(uname -s)
if [ "$OS" = "Darwin" ]; then
  echo "Uninstalling on Darwin"
  uninstall_unix
  remove_launchd_plist

  echo "Done -- run `launchctl list | grep backrest` to check the service installation."
elif [ "$OS" = "Linux" ]; then
  echo "Unnstalling on Linux"
  uninstall_unix
  remove_systemd_service

  echo "Done -- run `systemctl status backrest` to check the status of the service."
else
  echo "Unknown OS: $OS. This script only supports Darwin and Linux."
  exit 1
fi
