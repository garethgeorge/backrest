/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

type Absent<T, K extends keyof T> = { [k in Exclude<keyof T, K>]?: undefined };
type OneOf<T> =
  | { [k in keyof T]?: undefined }
  | (
    keyof T extends infer K ?
      (K extends string & keyof T ? { [k in K]: T[K] } & Absent<T, K>
        : never)
    : never);

export enum Status {
  UNKNOWN = "UNKNOWN",
  IN_PROGRESS = "IN_PROGRESS",
  SUCCESS = "SUCCESS",
  FAILED = "FAILED",
}


type BaseEvent = {
  timestamp?: string
}

export type Event = BaseEvent
  & OneOf<{ log: LogEvent; backupStatusChange: BackupStatusEvent }>

export type LogEvent = {
  message?: string
}

export type BackupStatusEvent = {
  plan?: string
  status?: Status
  percent?: number
}