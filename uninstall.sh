#! /bin/bash

cd "$(dirname "$0")" # cd to the directory of this script

uninstall_unix() {
  echo "Uninstalling backrest from /usr/local/bin/backrest"
  sudo rm -f /usr/local/bin/backrest
}

remove_systemd_service() {
  echo "Removing systemd service at /etc/systemd/system/backrest.service"
  sudo systemctl stop backrest
  sudo systemctl disable backrest
  sudo rm -f /etc/systemd/system/backrest.service

  echo "Reloading systemd daemon"
  sudo systemctl daemon-reload
}

remove_openrc_service() {
  echo "Removing openrc service at /etc/init.d/backrest"
  sudo rc-service backrest --ifstarted stop
  sudo rc-update del backrest default
  sudo rm -f /etc/init.d/backrest
}

remove_launchd_plist() {
  echo "Removing launchd plist at /Library/LaunchAgents/com.backrest.plist"

  launchctl unload /Library/LaunchAgents/com.backrest.plist || true
  sudo rm /Library/LaunchAgents/com.backrest.plist
}

OS=$(uname -s)
if [ "$OS" = "Darwin" ]; then
  echo "Uninstalling on Darwin"
  uninstall_unix
  remove_launchd_plist

  echo "Done -- run 'launchctl list | grep backrest' to check the service installation."
elif [ "$OS" = "Linux" ]; then
  echo "Unnstalling on Linux"
  uninstall_unix

  systemctl --version
  systemd_=$?
  rc-status --version
  openrc_=$?

  if [ $systemd_ -eq 0 ]; then
    remove_systemd_service
    echo "Done -- run 'systemctl status backrest' to check the status of the service."
  elif [ $openrc_ -eq 0 ]; then
    remove_openrc_service
    echo "Done -- run 'rc-service backrest status' to check the status of the service."
  fi

else
  echo "Unknown OS: $OS. This script only supports Darwin and Linux."
  exit 1
fi
