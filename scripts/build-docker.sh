#! /bin/sh
set -x

./scripts/build-all.sh
cp ./resticui-linux-amd64 ./docker
docker build -t garethgeorge/resticui:latest ./docker
# docker push garethgeorge/resticui:latest