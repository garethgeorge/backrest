/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../fetch.pb"
import * as GoogleProtobufEmpty from "../google/protobuf/empty.pb"
import * as TypesValue from "../types/value.pb"
import * as V1Config from "./config.pb"
import * as V1Operations from "./operations.pb"
import * as V1Restic from "./restic.pb"
export type ClearHistoryRequest = {
  repoId?: string
  planId?: string
  onlyFailed?: boolean
}

export type ListSnapshotsRequest = {
  repoId?: string
  planId?: string
}

export type GetOperationsRequest = {
  repoId?: string
  planId?: string
  snapshotId?: string
  ids?: string[]
  lastN?: string
}

export type RestoreSnapshotRequest = {
  repoId?: string
  planId?: string
  snapshotId?: string
  path?: string
  target?: string
}

export type ListSnapshotFilesRequest = {
  repoId?: string
  snapshotId?: string
  path?: string
}

export type ListSnapshotFilesResponse = {
  path?: string
  entries?: LsEntry[]
}

export type LsEntry = {
  name?: string
  type?: string
  path?: string
  uid?: string
  gid?: string
  size?: string
  mode?: string
  mtime?: string
  atime?: string
  ctime?: string
}

export class Restora {
  static GetConfig(req: GoogleProtobufEmpty.Empty, initReq?: fm.InitReq): Promise<V1Config.Config> {
    return fm.fetchReq<GoogleProtobufEmpty.Empty, V1Config.Config>(`/v1/config?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
  static SetConfig(req: V1Config.Config, initReq?: fm.InitReq): Promise<V1Config.Config> {
    return fm.fetchReq<V1Config.Config, V1Config.Config>(`/v1/config`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static AddRepo(req: V1Config.Repo, initReq?: fm.InitReq): Promise<V1Config.Config> {
    return fm.fetchReq<V1Config.Repo, V1Config.Config>(`/v1/config/repo`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static GetOperationEvents(req: GoogleProtobufEmpty.Empty, entityNotifier?: fm.NotifyStreamEntityArrival<V1Operations.OperationEvent>, initReq?: fm.InitReq): Promise<void> {
    return fm.fetchStreamingRequest<GoogleProtobufEmpty.Empty, V1Operations.OperationEvent>(`/v1/events/operations?${fm.renderURLSearchParams(req, [])}`, entityNotifier, {...initReq, method: "GET"})
  }
  static GetOperations(req: GetOperationsRequest, initReq?: fm.InitReq): Promise<V1Operations.OperationList> {
    return fm.fetchReq<GetOperationsRequest, V1Operations.OperationList>(`/v1/operations`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static ListSnapshots(req: ListSnapshotsRequest, initReq?: fm.InitReq): Promise<V1Restic.ResticSnapshotList> {
    return fm.fetchReq<ListSnapshotsRequest, V1Restic.ResticSnapshotList>(`/v1/snapshots`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static ListSnapshotFiles(req: ListSnapshotFilesRequest, initReq?: fm.InitReq): Promise<ListSnapshotFilesResponse> {
    return fm.fetchReq<ListSnapshotFilesRequest, ListSnapshotFilesResponse>(`/v1/snapshots/files`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static Backup(req: TypesValue.StringValue, initReq?: fm.InitReq): Promise<GoogleProtobufEmpty.Empty> {
    return fm.fetchReq<TypesValue.StringValue, GoogleProtobufEmpty.Empty>(`/v1/cmd/backup`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static Prune(req: TypesValue.StringValue, initReq?: fm.InitReq): Promise<GoogleProtobufEmpty.Empty> {
    return fm.fetchReq<TypesValue.StringValue, GoogleProtobufEmpty.Empty>(`/v1/cmd/prune`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static Forget(req: TypesValue.StringValue, initReq?: fm.InitReq): Promise<GoogleProtobufEmpty.Empty> {
    return fm.fetchReq<TypesValue.StringValue, GoogleProtobufEmpty.Empty>(`/v1/cmd/forget`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static Restore(req: RestoreSnapshotRequest, initReq?: fm.InitReq): Promise<GoogleProtobufEmpty.Empty> {
    return fm.fetchReq<RestoreSnapshotRequest, GoogleProtobufEmpty.Empty>(`/v1/cmd/restore`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static Unlock(req: TypesValue.StringValue, initReq?: fm.InitReq): Promise<GoogleProtobufEmpty.Empty> {
    return fm.fetchReq<TypesValue.StringValue, GoogleProtobufEmpty.Empty>(`/v1/cmd/unlock`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static Cancel(req: TypesValue.Int64Value, initReq?: fm.InitReq): Promise<GoogleProtobufEmpty.Empty> {
    return fm.fetchReq<TypesValue.Int64Value, GoogleProtobufEmpty.Empty>(`/v1/cmd/cancel`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static ClearHistory(req: ClearHistoryRequest, initReq?: fm.InitReq): Promise<GoogleProtobufEmpty.Empty> {
    return fm.fetchReq<ClearHistoryRequest, GoogleProtobufEmpty.Empty>(`/v1/cmd/clear_history`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static PathAutocomplete(req: TypesValue.StringValue, initReq?: fm.InitReq): Promise<TypesValue.StringList> {
    return fm.fetchReq<TypesValue.StringValue, TypesValue.StringList>(`/v1/autocomplete/path`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
}