#! /bin/bash

latest_restic_version=$(./scripts/latest-restic-version.sh)

if [ -z "$latest_restic_version" ]; then
  echo "Failed to get latest restic version"
  exit 1
fi

echo "Latest restic version: $latest_restic_version"

sed -i -E "s/^.*RequiredResticVersion\ =\ .*$/	RequiredResticVersion\ =\ \"$latest_restic_version\"/g" internal/resticinstaller/resticinstaller.go