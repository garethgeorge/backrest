import { useMemo } from "react";
import { createConnectTransport } from "@connectrpc/connect-web";
import { createPromiseClient, PromiseClient } from "@connectrpc/connect";
import { Backrest } from "../gen/ts/v1/service_connect";

const transport = createConnectTransport({
  baseUrl: "/",
});

export const backrestService = createPromiseClient(Backrest, transport);
