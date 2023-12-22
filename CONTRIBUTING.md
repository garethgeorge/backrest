# Contributing

## Commits

This repo uses [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/).

## Dev Depedencies

**Build Dependencies**

 * Node.JS for UI development
 * Go 1.21 or greater for server development
 * go.rice `go install github.com/GeertJohan/go.rice@latest` and `go install github.com/GeertJohan/go.rice/rice@latest`

**To Edit Protobuffers**
```sh
apt install -y protobuf-compiler
go install \
    github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest \
    github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
go install github.com/grpc-ecosystem/protoc-gen-grpc-gateway-ts@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/bufbuild/buf/cmd/buf@v1.27.2
```
## Building

```sh
(cd webui && npm i && npm run build)
(cd cmd/backrest && go build .)
```
