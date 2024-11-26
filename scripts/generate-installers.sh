#! /bin/bash

outdir=$(realpath $1) # output directory for the installer binaries 
srcdir=$(realpath $(dirname $0)/..) # source directory

function windows_installer {
  # create the installer for the given architecture
  goreleaser_arch=$1
  nsis_arch=$2
  cd $(mktemp -d)
  unzip $srcdir/dist/backrest_Windows_${goreleaser_arch}.zip
  cp -rl $srcdir/build/windows/* .

  docker run --rm -e TARGET_ARCH=$nsis_arch -v $(pwd):/build binfalse/nsis install.nsi

  cp Backrest-setup.exe $outdir/Backrest-setup-${goreleaser_arch}.exe
}

function macos_installer {
  goreleaser_arch=$1
  cd $(mktemp -d)

  cp -rl $srcdir/build/macos/* .
  unzip $srcdir/dist/backrest_Darwin_${goreleaser_arch}.zip -d Backrest.app/Contents/MacOS
}

windows_installer x86_64 amd64
windows_installer arm64 arm64
macos_installer arm64 backrestmondarwin_darwin_arm64/backrest-macos-tray 