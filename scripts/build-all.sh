#! /bin/sh

(cd webui && npm i && npm run build)

for bin in resticui-*; do
    rm -f $bin 
done

find webui/dist -name '*.map' -exec rm ./{} \;

GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o resticui-linux-amd64
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o resticui-linux-arm64
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o resticui-darwin-amd64 
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o resticui-darwin-arm64

for bin in resticui-*; do
    rice append --exec $bin
done
