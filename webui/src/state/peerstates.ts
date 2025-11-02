import { useEffect, useState } from "react";
import { PeerState } from "../../gen/ts/v1sync/syncservice_pb";
import { syncStateService } from "../api";

// Type intersection to combine properties from Repo and RepoMetadata
export interface RepoProps {
  id: string;
  guid: string;
}

const subscribeToSyncStates = async (
  requestMethod: () => AsyncIterable<PeerState>,
  callback: (syncStates: PeerState[]) => void,
  abortController: AbortController,
): Promise<void> => {
  let updateTimeout: NodeJS.Timeout | null = null;
  const stateMap: { [key: string]: PeerState } = {};

  const streamStates = async () => {
    while (!abortController.signal.aborted) {
      let nextConnWaitUntil = new Date().getTime() + 5000;
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
          console.warn("Error in sync state stream:", error);
        }
      }
      await new Promise(resolve => setTimeout(resolve, nextConnWaitUntil - new Date().getTime()));
    }
    if (updateTimeout) {
      clearTimeout(updateTimeout);
    }
  };

  streamStates();
};

let peerStates: Map<string, PeerState> = new Map();
const subscribers: Set<(peerStates: PeerState[]) => void> = new Set();

const subscribeToPeerStates = (
  callback: (peerStates: PeerState[]) => void,
): void => {
  subscribers.add(callback);
  callback(Array.from(peerStates.values()));
};

const unsubscribeFromPeerStates = (
  callback: (peerStates: PeerState[]) => void,
): void => {
  subscribers.delete(callback);
};


(async () => {
  const abortController = new AbortController(); // never aborts at the moment.
  subscribeToSyncStates(() => {
    return syncStateService.getPeerSyncStatesStream({ subscribe: true }, {
      signal: abortController.signal,
    });
  }, (updatedStates) => {
    console.log("Received updated states for peers: ", updatedStates);
    for (const state of updatedStates) {
      peerStates.set(state.peerKeyid, state);
    }
    const curStates = Array.from(peerStates.values());
    for (const subscriber of subscribers) {
      subscriber(curStates);
    }
  }, abortController);
})();

export const useSyncStates = (): PeerState[] => {
  const [syncStates, setSyncStates] = useState<PeerState[]>(() =>
    Array.from(peerStates.values())
  );

  useEffect(() => {
    const handleStateUpdate = (states: PeerState[]) => {
      setSyncStates(states);
    };

    subscribeToPeerStates(handleStateUpdate);

    return () => {
      unsubscribeFromPeerStates(handleStateUpdate);
    };
  }, []);

  return syncStates;
};
