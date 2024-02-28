## Based on microsoft go devcontainer - https://github.com/microsoft/vscode-dev-containers/blob/v0.205.2/containers/go/.devcontainer/Dockerfile
# [Choice] Go version (use -bullseye variants on local arm64/Apple Silicon): 1, 1.16, 1.17, 1-bullseye, 1.16-bullseye, 1.17-bullseye, 1-buster, 1.16-buster, 1.17-buster
ARG VARIANT=1-bullseye
FROM mcr.microsoft.com/vscode/devcontainers/go:${VARIANT}

# [Choice] Node.js version: none, lts/*, 16, 14, 12, 10
ARG NODE_VERSION="none"
RUN if [ "${NODE_VERSION}" != "none" ]; then su vscode -c "umask 0002 && . /usr/local/share/nvm/nvm.sh && nvm install ${NODE_VERSION} 2>&1"; fi

RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends protobuf-compiler

USER vscode

RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest \
    && go install github.com/bufbuild/buf/cmd/buf@v1.27.2 \
    && go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest \
    && go install google.golang.org/protobuf/cmd/protoc-gen-go@latest \
    && go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest \
    && go install github.com/GeertJohan/go.rice/rice@latest \
    && go install github.com/goreleaser/goreleaser@latest \
    && npm install -g @bufbuild/protoc-gen-es @connectrpc/protoc-gen-connect-es
