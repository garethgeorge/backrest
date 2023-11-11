/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as fm from "../fetch.pb"
import * as GoogleProtobufEmpty from "../google/protobuf/empty.pb"
import * as V1Config from "./config.pb"
import * as V1Events from "./events.pb"
export class ResticUI {
  static GetConfig(req: GoogleProtobufEmpty.Empty, initReq?: fm.InitReq): Promise<V1Config.Config> {
    return fm.fetchReq<GoogleProtobufEmpty.Empty, V1Config.Config>(`/v1/config?${fm.renderURLSearchParams(req, [])}`, {...initReq, method: "GET"})
  }
  static SetConfig(req: V1Config.Config, initReq?: fm.InitReq): Promise<V1Config.Config> {
    return fm.fetchReq<V1Config.Config, V1Config.Config>(`/v1/config`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static AddRepo(req: V1Config.Repo, initReq?: fm.InitReq): Promise<V1Config.Config> {
    return fm.fetchReq<V1Config.Repo, V1Config.Config>(`/v1/config/repo`, {...initReq, method: "POST", body: JSON.stringify(req, fm.replacer)})
  }
  static GetEvents(req: GoogleProtobufEmpty.Empty, entityNotifier?: fm.NotifyStreamEntityArrival<V1Events.Event>, initReq?: fm.InitReq): Promise<void> {
    return fm.fetchStreamingRequest<GoogleProtobufEmpty.Empty, V1Events.Event>(`/v1/events?${fm.renderURLSearchParams(req, [])}`, entityNotifier, {...initReq, method: "GET"})
  }
}