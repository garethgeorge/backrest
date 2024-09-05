// @generated by protoc-gen-connect-es v1.3.0 with parameter "target=ts"
// @generated from file v1/service.proto (package v1, syntax proto3)
/* eslint-disable */
// @ts-nocheck

import { Empty, MethodKind } from "@bufbuild/protobuf";
import { Config, Repo } from "./config_pb.js";
import { OperationEvent, OperationList } from "./operations_pb.js";
import { ClearHistoryRequest, DoRepoTaskRequest, ForgetRequest, GetOperationsRequest, ListSnapshotFilesRequest, ListSnapshotFilesResponse, ListSnapshotsRequest, LogDataRequest, RestoreSnapshotRequest, RunCommandRequest } from "./service_pb.js";
import { ResticSnapshotList } from "./restic_pb.js";
import { BytesValue, Int64Value, StringList, StringValue } from "../types/value_pb.js";

/**
 * @generated from service v1.Backrest
 */
export const Backrest = {
  typeName: "v1.Backrest",
  methods: {
    /**
     * @generated from rpc v1.Backrest.GetConfig
     */
    getConfig: {
      name: "GetConfig",
      I: Empty,
      O: Config,
      kind: MethodKind.Unary,
    },
    /**
     * @generated from rpc v1.Backrest.SetConfig
     */
    setConfig: {
      name: "SetConfig",
      I: Config,
      O: Config,
      kind: MethodKind.Unary,
    },
    /**
     * @generated from rpc v1.Backrest.AddRepo
     */
    addRepo: {
      name: "AddRepo",
      I: Repo,
      O: Config,
      kind: MethodKind.Unary,
    },
    /**
     * @generated from rpc v1.Backrest.GetOperationEvents
     */
    getOperationEvents: {
      name: "GetOperationEvents",
      I: Empty,
      O: OperationEvent,
      kind: MethodKind.ServerStreaming,
    },
    /**
     * @generated from rpc v1.Backrest.GetOperations
     */
    getOperations: {
      name: "GetOperations",
      I: GetOperationsRequest,
      O: OperationList,
      kind: MethodKind.Unary,
    },
    /**
     * @generated from rpc v1.Backrest.ListSnapshots
     */
    listSnapshots: {
      name: "ListSnapshots",
      I: ListSnapshotsRequest,
      O: ResticSnapshotList,
      kind: MethodKind.Unary,
    },
    /**
     * @generated from rpc v1.Backrest.ListSnapshotFiles
     */
    listSnapshotFiles: {
      name: "ListSnapshotFiles",
      I: ListSnapshotFilesRequest,
      O: ListSnapshotFilesResponse,
      kind: MethodKind.Unary,
    },
    /**
     * Backup schedules a backup operation. It accepts a plan id and returns empty if the task is enqueued.
     *
     * @generated from rpc v1.Backrest.Backup
     */
    backup: {
      name: "Backup",
      I: StringValue,
      O: Empty,
      kind: MethodKind.Unary,
    },
    /**
     * DoRepoTask schedules a repo task. It accepts a repo id and a task type and returns empty if the task is enqueued.
     *
     * @generated from rpc v1.Backrest.DoRepoTask
     */
    doRepoTask: {
      name: "DoRepoTask",
      I: DoRepoTaskRequest,
      O: Empty,
      kind: MethodKind.Unary,
    },
    /**
     * Forget schedules a forget operation. It accepts a plan id and returns empty if the task is enqueued.
     *
     * @generated from rpc v1.Backrest.Forget
     */
    forget: {
      name: "Forget",
      I: ForgetRequest,
      O: Empty,
      kind: MethodKind.Unary,
    },
    /**
     * Restore schedules a restore operation.
     *
     * @generated from rpc v1.Backrest.Restore
     */
    restore: {
      name: "Restore",
      I: RestoreSnapshotRequest,
      O: Empty,
      kind: MethodKind.Unary,
    },
    /**
     * Cancel attempts to cancel a task with the given operation ID. Not guaranteed to succeed.
     *
     * @generated from rpc v1.Backrest.Cancel
     */
    cancel: {
      name: "Cancel",
      I: Int64Value,
      O: Empty,
      kind: MethodKind.Unary,
    },
    /**
     * GetLogs returns the keyed large data for the given operation.
     *
     * @generated from rpc v1.Backrest.GetLogs
     */
    getLogs: {
      name: "GetLogs",
      I: LogDataRequest,
      O: BytesValue,
      kind: MethodKind.ServerStreaming,
    },
    /**
     * RunCommand executes a generic restic command on the repository.
     *
     * @generated from rpc v1.Backrest.RunCommand
     */
    runCommand: {
      name: "RunCommand",
      I: RunCommandRequest,
      O: BytesValue,
      kind: MethodKind.ServerStreaming,
    },
    /**
     * GetDownloadURL returns a signed download URL given a forget operation ID.
     *
     * @generated from rpc v1.Backrest.GetDownloadURL
     */
    getDownloadURL: {
      name: "GetDownloadURL",
      I: Int64Value,
      O: StringValue,
      kind: MethodKind.Unary,
    },
    /**
     * Clears the history of operations
     *
     * @generated from rpc v1.Backrest.ClearHistory
     */
    clearHistory: {
      name: "ClearHistory",
      I: ClearHistoryRequest,
      O: Empty,
      kind: MethodKind.Unary,
    },
    /**
     * PathAutocomplete provides path autocompletion options for a given filesystem path.
     *
     * @generated from rpc v1.Backrest.PathAutocomplete
     */
    pathAutocomplete: {
      name: "PathAutocomplete",
      I: StringValue,
      O: StringList,
      kind: MethodKind.Unary,
    },
  }
} as const;

