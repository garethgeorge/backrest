#! /bin/sh
set -x

(cd webui && npm i && npm run build)
rm -f backrest
go build .
rice append --exec backrest
