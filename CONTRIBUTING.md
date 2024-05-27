# Contributing

## Commits

This repo uses [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/).

## Build Depedencies

- [Node.js](https://nodejs.org/en) for UI development
- [Go](https://go.dev/) 1.21 or greater for server development
- [goreleaser](https://github.com/goreleaser/goreleaser) `go install github.com/goreleaser/goreleaser@latest`

**To Edit Protobuffers**

```sh
apt install -y protobuf-compiler
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/bufbuild/buf/cmd/buf@v1.27.2
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
npm install -g @bufbuild/protoc-gen-es @connectrpc/protoc-gen-connect-es
```

## Building

```sh
(cd webui && npm i && npm run build)
(cd cmd/backrest && go build .)
```
To run the web UI after building, you can execute: `go run .`

## Using VSCode Dev Containers

You can also use VSCode with [Dev Containers](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers) extension to quickly get up and running with a working development and debugging environment.

0. Make sure Docker and VSCode with [Dev Containers](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers) extension is installed
1. Clone this repository
2. Open this folder in VSCode
3. When prompted, click on `Open in Container` button, or run `> Dev Containers: Rebuild and Reopen in Containers` command
4. When container is started, go to `Run and Debug`, choose `Debug Backrest (backend+frontend)` and run it

> [!NOTE]
> Provided launch configuration has hot reload for typescript frontend.
