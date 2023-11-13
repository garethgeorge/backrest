# ResticUI

WIP project to build a UI for restic.

Project goals

 * Single binary for easy and _very lightweight_ deployment with or without containerization.
 * WebUI supporting
   * Backup plan creation and configuration
   * Backup status
   * Snapshot browsing and restore

# High Level Goals

 * Restic repository configuration and initialization
 * Multiple backup plans with independent file sets, schedules, etc can be configured and store data to a single repository definition.
 * Backup status info at a glance
   * Shows health of last backup operation attempted for a given repository.
   * Shows health of last backup operation attempted for a given plan.
 * Support for browsing snapshots belonging to a given repository.
 * Support for browsing snapshots belonging to a given plan.

# Roadmap 

 - [x] Restic repository configuration and initialization
 - [x] Backup plan configuration
 - [ ] Health checks in backend/GUI
   - [ ] Backend state store
   - [ ] Health status view for repos
   - [ ] Health status view for plans
 - [ ] Backup operation
   - [X] Backend implementatio
   - [ ] Backup command in UI
 - [ ] Snapshot browser
   - [X] Backend snapshot listing
   - [ ] Snapshot browser GUI on repo view 
   - [ ] Snapshot browser GUI on backup plan view
 - [ ] Operation history
   - [ ] Backend operation tracking in state store e.g. log of backup operations, cleanup operations, etc.
     - [ ] Scheduler log stored on plan?
     - [ ] Operation result log stored on repo?
   - [ ] Operation history GUI on repo view. 
   - [ ] Operation history GUI on plan view.

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
