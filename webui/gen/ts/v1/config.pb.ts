/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/
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

export type PrunePolicy = {
  maxFrequencyDays?: number
  maxUnusedPercent?: number
  maxUnusedBytes?: number
}