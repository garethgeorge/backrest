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

export enum OperationStatus {
  STATUS_UNKNOWN = "STATUS_UNKNOWN",
  STATUS_PENDING = "STATUS_PENDING",
  STATUS_INPROGRESS = "STATUS_INPROGRESS",
  STATUS_SUCCESS = "STATUS_SUCCESS",
}


type BaseOperation = {
  status?: OperationStatus
  unixTimeStartMs?: string
  unixTimeEndMs?: string
}

export type Operation = BaseOperation
  & OneOf<{ backup: OperationBackup }>

export type OperationBackup = {
  repoId?: string
  planId?: string
  lastStatus?: V1Restic.BackupProgressEntry
}