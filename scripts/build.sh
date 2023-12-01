#! /bin/sh
set -x

(cd webui && npm i && npm run build)
rm -f resticui
go build .
rice append --exec resticui
