# ResticUI

WIP project to build a UI for restic.

Project goals

 * Single binary for easy and _very lightweight_ deployment with or without containerization.
 * WebUI supporting
   * Backup plan creation and configuration
   * Backup status
   * Snapshot browsing and restore

# Dependencies

## Dev 

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
