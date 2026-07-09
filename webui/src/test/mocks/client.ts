import { vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { OperationListSchema } from "../../../gen/ts/v1/operations_pb";
import type { OperationEvent } from "../../../gen/ts/v1/operations_pb";

// Mock stand-in for src/api/client.ts, installed globally by src/test/setup.tsx
// via vi.mock("@/api/client", ...). It must mirror the real module's exports.
//
// The streaming methods (getOperationEvents, getPeerSyncStatesStream) default
// to a generator that never yields: src/api/oplog.ts and src/state/peerStates.ts
// start infinite reconnect loops at import time, and parking their `for await`
// on a never-resolving promise keeps them inert without mocking those modules
// away — so their subscription logic stays testable via makeEventStream().

const neverStream = async function* (): AsyncGenerator<never> {
  await new Promise(() => {});
};

/** A controllable server-stream for suites that push operation events. */
export const makeEventStream = <T = OperationEvent>() => {
  let pushWaiting: ((value: T | null) => void) | null = null;
  const queue: (T | null)[] = [];
  const stream = (async function* () {
    while (true) {
      const next =
        queue.length > 0
          ? queue.shift()!
          : await new Promise<T | null>((resolve) => (pushWaiting = resolve));
      if (next === null) return;
      yield next;
    }
  })();
  const push = (value: T | null) => {
    if (pushWaiting) {
      const resolve = pushWaiting;
      pushWaiting = null;
      resolve(value);
    } else {
      queue.push(value);
    }
  };
  return {
    stream,
    emit: (event: T) => push(event),
    end: () => push(null),
  };
};

export const backrestService = {
  getConfig: vi.fn(),
  setConfig: vi.fn(),
  setupSftp: vi.fn(),
  checkRepoExists: vi.fn(),
  addRepo: vi.fn(),
  removeRepo: vi.fn(),
  getOperationEvents: vi.fn(neverStream),
  getOperations: vi.fn(async () => create(OperationListSchema, {})),
  listSnapshots: vi.fn(),
  listSnapshotFiles: vi.fn(),
  backup: vi.fn(),
  doRepoTask: vi.fn(),
  forget: vi.fn(),
  restore: vi.fn(),
  cancel: vi.fn(),
  getLogs: vi.fn(neverStream),
  runCommand: vi.fn(),
  getDownloadURL: vi.fn(),
  clearHistory: vi.fn(),
  pathAutocomplete: vi.fn(),
  getSummaryDashboard: vi.fn(),
  generatePairingToken: vi.fn(),
};

export const authenticationService = {
  login: vi.fn(),
  hashPassword: vi.fn(),
};

export const syncStateService = {
  getPeerSyncStatesStream: vi.fn(neverStream),
  setRemoteClientConfig: vi.fn(),
};

// Keep the real module's observable behavior.
export const setAuthToken = (token: string) => {
  localStorage.setItem("backrest-ui-authToken", token);
};

/**
 * Reinstalls the default implementations. Called from setup.tsx afterEach —
 * after vi.clearAllMocks() this restores the parked streams and the empty
 * getOperations default. Suites must never call vi.resetAllMocks(), which
 * would strip these implementations mid-file.
 */
export const resetClientMocks = () => {
  backrestService.getOperationEvents.mockImplementation(neverStream);
  backrestService.getLogs.mockImplementation(neverStream);
  backrestService.getOperations.mockImplementation(async () =>
    create(OperationListSchema, {}),
  );
  syncStateService.getPeerSyncStatesStream.mockImplementation(neverStream);
};
