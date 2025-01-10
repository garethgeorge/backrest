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
  sudo diskutil unmount /Volumes/RAM_Disk_1GB
  hdiutil detach /Volumes/RAM_Disk_1GB
elif [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then
  sudo umount /mnt/ramdisk
fi

unset TMPDIR
unset XDG_CACHE_HOME
