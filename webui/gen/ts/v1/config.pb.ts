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

export type User = {
  name?: string
  password?: string
}

export type Repo = {
  id?: string
  uri?: string
  password?: string
  env?: string[]
  flags?: string[]
}

export type Plan = {
  id?: string
  repo?: string
  paths?: string[]
  excludes?: string[]
}