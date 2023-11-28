#! /bin/sh
set -x

(cd proto && ./update.sh)
(cd webui && npm run build)
rm -f resticui
go build ./cmd/resticui
rice append --exec resticui
