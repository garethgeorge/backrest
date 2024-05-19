import { useMemo } from "react";
import { createConnectTransport } from "@connectrpc/connect-web";
import { createPromiseClient } from "@connectrpc/connect";
import { Backrest } from "../gen/ts/v1/service_connect";
import { Authentication } from "../gen/ts/v1/authentication_connect";

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
  baseUrl: "./",
  useBinaryFormat: false,
  fetch: fetch as typeof globalThis.fetch,
});

export const authenticationService = createPromiseClient(
  Authentication,
  transport,
);
export const backrestService = createPromiseClient(Backrest, transport);
