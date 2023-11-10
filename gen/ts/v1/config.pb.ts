/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/
export type Config = {
  version?: number
  logDir?: string
  repos?: Repo[]
  plans?: Plan[]
}

export type Repo = {
  id?: string
  uri?: string
  password?: string
  env?: EnvVar[]
}

export type Plan = {
  id?: string
  repo?: string
  repoPath?: string
  paths?: string[]
}

export type EnvVar = {
  name?: string
  value?: string
}