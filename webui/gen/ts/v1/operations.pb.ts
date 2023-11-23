/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as V1Restic from "./restic.pb"

type Absent<T, K extends keyof T> = { [k in Exclude<keyof T, K>]?: undefined };
type OneOf<T> =
  | { [k in keyof T]?: undefined }
  | (
    keyof T extends infer K ?
      (K extends string & keyof T ? { [k in K]: T[K] } & Absent<T, K>
        : never)
    : never);

export enum OperationEventType {
  EVENT_UNKNOWN = "EVENT_UNKNOWN",
  EVENT_CREATED = "EVENT_CREATED",
  EVENT_UPDATED = "EVENT_UPDATED",
}

export enum OperationStatus {
  STATUS_UNKNOWN = "STATUS_UNKNOWN",
  STATUS_PENDING = "STATUS_PENDING",
  STATUS_INPROGRESS = "STATUS_INPROGRESS",
  STATUS_SUCCESS = "STATUS_SUCCESS",
  STATUS_ERROR = "STATUS_ERROR",
}

export type OperationList = {
  operations?: Operation[]
}


type BaseOperation = {
  id?: string
  repoId?: string
  planId?: string
  snapshotId?: string
  status?: OperationStatus
  unixTimeStartMs?: string
  unixTimeEndMs?: string
  displayMessage?: string
}

export type Operation = BaseOperation
  & OneOf<{ operationBackup: OperationBackup; operationIndexSnapshot: OperationIndexSnapshot }>

export type OperationEvent = {
  type?: OperationEventType
  operation?: Operation
}

export type OperationBackup = {
  lastStatus?: V1Restic.BackupProgressEntry
}

export type OperationIndexSnapshot = {
  snapshot?: V1Restic.ResticSnapshot
}