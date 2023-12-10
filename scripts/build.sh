#! /bin/sh
set -x

(cd webui && npm i && npm run build)
rm -f restora
go build .
rice append --exec restora
