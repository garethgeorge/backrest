import { useMemo } from "react";
import { createConnectTransport } from "@connectrpc/connect-web";
import { createClient } from "@connectrpc/connect";
import { Authentication } from "../gen/ts/v1/authentication_pb";
import { Backrest } from "../gen/ts/v1/service_pb";
import { BackrestSyncStateService } from "../gen/ts/v1sync/syncservice_pb";
import { backendUrl } from "./state/buildcfg";

const tokenKey = "backrest-ui-authToken";

export const setAuthToken = (token: string) => {
  localStorage.setItem(tokenKey, token);
};

const fetch = (
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<Response> => {
  const headers = new Headers(init?.headers);
  let token = localStorage.getItem(tokenKey);
  if (token && token !== "") {
    headers.set("Authorization", "Bearer " + token);
  }
  init = { ...init, headers };
  return window.fetch(input, init);
};

const transport = createConnectTransport({
  baseUrl: backendUrl,
  useBinaryFormat: true,
  fetch: fetch as typeof globalThis.fetch,
});

export const authenticationService = createClient(
  Authentication,
  transport,
);

export const backrestService = createClient(Backrest, transport);

export const syncStateService = createClient(BackrestSyncStateService, transport);
