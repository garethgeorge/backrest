# Contributing

## Commits

This repo uses [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/).

## Dev Depedencies

**Build Dependencies**

 * Node.JS for UI development
 * Go 1.21 or greater for server development
 * go.rice `go install github.com/GeertJohan/go.rice@latest` and `go install github.com/GeertJohan/go.rice/rice@latest`
 * goreleaser `go install github.com/goreleaser/goreleaser@latest`

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
