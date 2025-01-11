#! /bin/bash 

# Check that the script must be sourced
(
  [[ -n $ZSH_VERSION && $ZSH_EVAL_CONTEXT =~ :file$ ]] || 
  [[ -n $KSH_VERSION && "$(cd -- "$(dirname -- "$0")" && pwd -P)/$(basename -- "$0")" != "$(cd -- "$(dirname -- "${.sh.file}")" && pwd -P)/$(basename -- "${.sh.file}")" ]] || 
  [[ -n $BASH_VERSION ]] && (return 0 2>/dev/null)
) && sourced=1 || sourced=0

if [ $sourced -eq 0 ]; then
  echo "This script should be sourced instead of executed."
  echo "Usage: . $0"
  exit 1
fi

# Check if MacOS 
if [ "$(uname)" = "Darwin" ]; then
  if [ -d "/Volumes/RAM_Disk_1GB" ]; then
    echo "RAM disk /Volumes/RAM_Disk_1GB already exists."
  else 
    sudo diskutil erasevolume HFS+ RAM_Disk_1GB $(hdiutil attach -nomount ram://2048000)
  fi
  export TMPDIR="/Volumes/RAM_Disk_1GB"
  export RESTIC_CACHE_DIR="$TMPDIR/.cache"
  echo "Created 512MB RAM disk at /Volumes/RAM_Disk_1GB"
  echo "TMPDIR=$TMPDIR"
  echo "RESTIC_CACHE_DIR=$RESTIC_CACHE_DIR"
elif [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then
  # Create ramdisk 
  sudo mkdir -p /mnt/ramdisk
  sudo mount -t tmpfs -o size=1024M tmpfs /mnt/ramdisk
  export TMPDIR="/mnt/ramdisk"
  export RESTIC_CACHE_DIR="$TMPDIR/.cache"
fi
