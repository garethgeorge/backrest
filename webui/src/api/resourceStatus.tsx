import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useSyncExternalStore,
} from "react";
import { OperationEvent, OperationStatus } from "../../gen/ts/v1/operations_pb";
import { OpSelector } from "../../gen/ts/v1/service_pb";
import { subscribeToOperations, unsubscribeFromOperations } from "./oplog";
import { getStatusForSelector, matchSelector } from "./logState";
import { debounce } from "../lib/util";

// Serialize an OpSelector into a stable string key for use as a map key.
const selectorKey = (sel: OpSelector): string => {
  const parts: string[] = [];
  if (sel.planId) parts.push(`p:${sel.planId}`);
  if (sel.repoGuid) parts.push(`r:${sel.repoGuid}`);
  if (sel.instanceId) parts.push(`i:${sel.instanceId}`);
  if (sel.originalInstanceKeyid) parts.push(`o:${sel.originalInstanceKeyid}`);
  if (sel.snapshotId) parts.push(`s:${sel.snapshotId}`);
  if (sel.flowId) parts.push(`f:${sel.flowId}`);
  return parts.join("|");
};

type StatusMap = Map<string, OperationStatus>;

/**
 * ResourceStatusStore manages status for all registered selectors via a single
 * operation event subscription. Components register/unregister their selectors
 * and get updates through useSyncExternalStore.
 */
class ResourceStatusStore {
  private statuses: StatusMap = new Map();
  private selectors: Map<string, { selector: OpSelector; refCount: number }> =
    new Map();
  private listeners: Set<() => void> = new Set();
  private subscribed = false;

  private notify() {
    for (const listener of this.listeners) {
      listener();
    }
  }

  subscribe = (listener: () => void) => {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  };

  getSnapshot = (): StatusMap => {
    return this.statuses;
  };

  getStatus(key: string): OperationStatus {
    return this.statuses.get(key) ?? OperationStatus.STATUS_UNKNOWN;
  }

  /** Register a selector. Fetches its initial status and subscribes to updates. */
  register(selector: OpSelector): string {
    const key = selectorKey(selector);
    const existing = this.selectors.get(key);
    if (existing) {
      existing.refCount++;
      return key;
    }

    this.selectors.set(key, { selector, refCount: 1 });
    this.fetchStatus(key, selector);
    this.ensureSubscribed();
    return key;
  }

  /** Unregister a selector. Cleans up when refCount hits 0. */
  unregister(key: string) {
    const entry = this.selectors.get(key);
    if (!entry) return;
    entry.refCount--;
    if (entry.refCount <= 0) {
      this.selectors.delete(key);
      this.statuses.delete(key);
      this.notify();
      if (this.selectors.size === 0) {
        this.teardownSubscription();
      }
    }
  }

  private async fetchStatus(key: string, selector: OpSelector) {
    try {
      const status = await getStatusForSelector(selector);
      this.statuses = new Map(this.statuses);
      this.statuses.set(key, status);
      this.notify();
    } catch (e) {
      // Silently ignore fetch errors — icon will stay in UNKNOWN state
    }
  }

  private refreshDebounced = debounce(
    () => {
      for (const [key, { selector }] of this.selectors) {
        this.fetchStatus(key, selector);
      }
    },
    1000,
    { maxWait: 10000, trailing: true },
  );

  private handleOperationEvent = (
    event?: OperationEvent,
    _err?: Error,
  ) => {
    if (!event || !event.event) return;

    switch (event.event.case) {
      case "createdOperations":
      case "updatedOperations": {
        const ops = event.event.value.operations;
        let needsRefresh = false;
        for (const [, { selector }] of this.selectors) {
          if (ops.find((op) => matchSelector(selector, op))) {
            needsRefresh = true;
            break;
          }
        }
        if (needsRefresh) {
          this.refreshDebounced();
        }
        break;
      }
      case "deletedOperations":
        this.refreshDebounced();
        break;
    }
  };

  private ensureSubscribed() {
    if (this.subscribed) return;
    this.subscribed = true;
    subscribeToOperations(this.handleOperationEvent);
  }

  private teardownSubscription() {
    if (!this.subscribed) return;
    this.subscribed = false;
    unsubscribeFromOperations(this.handleOperationEvent);
    this.refreshDebounced.cancel();
  }
}

const storeInstance = new ResourceStatusStore();

const ResourceStatusContext = createContext<ResourceStatusStore>(storeInstance);

export const ResourceStatusProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  return (
    <ResourceStatusContext.Provider value={storeInstance}>
      {children}
    </ResourceStatusContext.Provider>
  );
};

/**
 * Hook that returns the OperationStatus for a given selector.
 * Registers/unregisters the selector with the shared store automatically.
 */
export const useResourceStatus = (selector: OpSelector): OperationStatus => {
  const store = useContext(ResourceStatusContext);
  const keyRef = useRef<string | null>(null);

  // Register on mount, unregister on unmount or selector change
  useEffect(() => {
    const key = store.register(selector);
    keyRef.current = key;
    return () => {
      store.unregister(key);
      keyRef.current = null;
    };
  }, [selectorKey(selector)]);

  const statusMap = useSyncExternalStore(store.subscribe, store.getSnapshot);
  return keyRef.current
    ? (statusMap.get(keyRef.current) ?? OperationStatus.STATUS_UNKNOWN)
    : OperationStatus.STATUS_UNKNOWN;
};
