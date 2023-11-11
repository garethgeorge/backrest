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
export type Http = {
  rules?: HttpRule[]
  fullyDecodeReservedExpansion?: boolean
}


type BaseHttpRule = {
  selector?: string
  body?: string
  responseBody?: string
  additionalBindings?: HttpRule[]
}

export type HttpRule = BaseHttpRule
  & OneOf<{ get: string; put: string; post: string; delete: string; patch: string; custom: CustomHttpPattern }>

export type CustomHttpPattern = {
  kind?: string
  path?: string
}