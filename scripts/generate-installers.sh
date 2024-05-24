#! /bin/bash

outdir=$(realpath $1)
srcdir=$(realpath $(dirname $0)/..)

# for each supported windows architecture
for arch in x86_64 arm64; do
  cd $(mktemp -d)
  wget https://github.com/garethgeorge/backrest/releases/latest/download/backrest_Windows_${arch}.zip
  unzip backrest_Windows_${arch}.zip

  cp -rl $srcdir/build/windows/* .

  if [ "$arch" == "x86_64" ]; then
    docker run --rm -v $(pwd):/build binfalse/nsis install.nsi
  else
    docker run --rm -e TARGET_ARCH=arm64 -v $(pwd):/build binfalse/nsis  install.nsi
  fi

  cp Backrest-setup.exe $outdir/Backrest-setup-${arch}.exe
done
