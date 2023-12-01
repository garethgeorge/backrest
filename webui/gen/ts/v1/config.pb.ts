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
export type Config = {
  modno?: number
  host?: string
  repos?: Repo[]
  plans?: Plan[]
}

export type Repo = {
  id?: string
  uri?: string
  password?: string
  env?: string[]
  flags?: string[]
  prunePolicy?: PrunePolicy
}

export type Plan = {
  id?: string
  repo?: string
  paths?: string[]
  excludes?: string[]
  cron?: string
  retention?: RetentionPolicy
}

export type RetentionPolicy = {
  maxUnusedLimit?: string
  keepLastN?: number
  keepHourly?: number
  keepDaily?: number
  keepWeekly?: number
  keepMonthly?: number
  keepYearly?: number
  keepWithinDuration?: string
}


type BasePrunePolicy = {
  maxFrequencyDays?: number
}

export type PrunePolicy = BasePrunePolicy
  & OneOf<{ maxUnusedPercent: number; maxUnusedBytes: number }>