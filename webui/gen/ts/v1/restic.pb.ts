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
export type ResticSnapshot = {
  id?: string
  unixTimeMs?: string
  hostname?: string
  username?: string
  tree?: string
  parent?: string
  paths?: string[]
  tags?: string[]
}


type BaseBackupProgressEntry = {
}

export type BackupProgressEntry = BaseBackupProgressEntry
  & OneOf<{ status: BackupProgressStatusEntry; summary: BackupProgressSummary }>

export type BackupProgressStatusEntry = {
  percentDone?: number
  totalFiles?: string
  totalBytes?: string
  filesDone?: string
  bytesDone?: string
}

export type BackupProgressSummary = {
  filesNew?: string
  filesChanged?: string
  filesUnmodified?: string
  dirsNew?: string
  dirsChanged?: string
  dirsUnmodified?: string
  dataBlobs?: string
  treeBlobs?: string
  dataAdded?: string
  totalFilesProcessed?: string
  totalBytesProcessed?: string
  totalDuration?: string
  snapshotId?: string
}