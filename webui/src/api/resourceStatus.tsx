import { useEffect, useState } from "react";
import { Operation, OperationEvent, OperationStatus } from "../../gen/ts/v1/operations_pb";
import { OpSelector } from "../../gen/ts/v1/service_pb";
import { subscribeToOperations, unsubscribeFromOperations } from "./oplog";
import { getStatusForSelector, matchSelector } from "./logState";
import { debounce } from "../lib/util";

// Module-level shared state: all registered selectors and their cached statuses.
const selectors = new Map<string, { selector: OpSelector; refCount: number }>();
const statuses = new Map<string, OperationStatus>();
const listeners = new Set<() => void>();
let subscribed = false;

const notify = () => {
  for (const l of listeners) l();
};

const fetchStatus = async (key: string, selector: OpSelector) => {
  try {
    const status = await getStatusForSelector(selector);
    statuses.set(key, status);
    notify();
  } catch (_) {}
};

// Track pending keys to refresh; the debounced function drains this set.
const pendingKeys = new Set<string>();

const flushPending = debounce(
  () => {
    for (const key of pendingKeys) {
      const entry = selectors.get(key);
      if (entry) {
        fetchStatus(key, entry.selector);
      }
    }
    pendingKeys.clear();
  },
  1000,
  { maxWait: 10000, trailing: true },
);

const refreshAll = () => {
  for (const key of selectors.keys()) {
    pendingKeys.add(key);
  }
  flushPending();
};

const refreshMatching = (ops: Operation[]) => {
  for (const [key, { selector }] of selectors) {
    if (ops.some((op) => matchSelector(selector, op))) {
      pendingKeys.add(key);
    }
  }
  if (pendingKeys.size > 0) {
    flushPending();
  }
};

const handleEvent = (event?: OperationEvent, _err?: Error) => {
  if (!event || !event.event) return;
  switch (event.event.case) {
    case "createdOperations":
    case "updatedOperations":
      refreshMatching(event.event.value.operations);
      break;
    case "deletedOperations":
      refreshAll();
      break;
  }
};

const register = (selector: OpSelector): string => {
  const key = JSON.stringify(selector);
  const existing = selectors.get(key);
  if (existing) {
    existing.refCount++;
    return key;
  }
  selectors.set(key, { selector, refCount: 1 });
  fetchStatus(key, selector);
  if (!subscribed) {
    subscribed = true;
    subscribeToOperations(handleEvent);
  }
  return key;
};

const unregister = (key: string) => {
  const entry = selectors.get(key);
  if (!entry) return;
  entry.refCount--;
  if (entry.refCount <= 0) {
    selectors.delete(key);
    statuses.delete(key);
    notify();
    if (selectors.size === 0 && subscribed) {
      subscribed = false;
      unsubscribeFromOperations(handleEvent);
      flushPending.cancel();
      pendingKeys.clear();
    }
  }
};

/**
 * Hook that returns the OperationStatus for a given selector.
 * Shares a single global operation event subscription across all consumers.
 */
export const useResourceStatus = (selector: OpSelector): OperationStatus => {
  const key = JSON.stringify(selector);
  const [status, setStatus] = useState<OperationStatus>(
    () => statuses.get(key) ?? OperationStatus.STATUS_UNKNOWN,
  );

  useEffect(() => {
    const k = register(selector);
    const listener = () => {
      setStatus(statuses.get(k) ?? OperationStatus.STATUS_UNKNOWN);
    };
    listeners.add(listener);
    // Sync in case status was fetched between render and effect
    listener();
    return () => {
      listeners.delete(listener);
      unregister(k);
    };
  }, [key]);

  return status;
};
