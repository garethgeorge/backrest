import { SyncStateStreamItem } from "../../gen/ts/v1/syncservice_pb";
import { syncStateService } from "../api";

const subscribeToSyncStates = async (
  requestMethod: () => AsyncIterable<SyncStateStreamItem>,
  callback: (syncStates: SyncStateStreamItem[]) => void,
  abortController: AbortController,
): Promise<void> => {
  let updateTimeout: NodeJS.Timeout | null = null;
  const stateMap: { [key: string]: SyncStateStreamItem } = {};

  const streamStates = async () => {
    while (!abortController.signal.aborted) {
      try {
        const generator = requestMethod();
        for await (const state of generator) {
          stateMap[state.peerInstanceId] = state;

          // Debounce updates to avoid excessive re-renders
          if (updateTimeout) {
            clearTimeout(updateTimeout);
          }
          updateTimeout = setTimeout(() => {
            callback(Object.values(stateMap));
          }, 100);
        }
      } catch (error) {
        if (!abortController.signal.aborted) {
          console.error("Error in sync state stream:", error);
        }
      }
    }
    if (updateTimeout) {
      clearTimeout(updateTimeout);
    }
  };

  streamStates();
};

export const subscribeToKnownHostSyncStates = (
  abortController: AbortController,
  callback: (syncStates: SyncStateStreamItem[]) => void,
): void => {
  subscribeToSyncStates(() => {
    return syncStateService.getKnownHostSyncStateStream({ subscribe: true }, {
      signal: abortController.signal,
    });
  }, callback, abortController);
}

export const subscribeToClientSyncStates = (
  abortController: AbortController,
  callback: (syncStates: SyncStateStreamItem[]) => void,
): void => {
  subscribeToSyncStates(() => {
    return syncStateService.getClientSyncStateStream({ subscribe: true }, {
      signal: abortController.signal,
    });
  }, callback, abortController);
}