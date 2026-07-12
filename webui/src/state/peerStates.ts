import { useEffect, useState } from "react";
import { PeerState, PeerStateSchema } from "../../gen/ts/v1sync/syncservice_pb";
import { Config } from "../../gen/ts/v1/config_pb";
import { fromBinary, toBinary } from "@bufbuild/protobuf";
import { syncStateService } from "../api/client";
import { createSharedStream } from "../api/streams/sharedStream";
import { useConfig } from "../app/provider";

// Type intersection to combine properties from Repo and RepoMetadata
export interface RepoProps {
  id: string;
  guid: string;
}

// Accumulated peer states, keyed by peer key id.
const peerStates = new Map<string, PeerState>();
const subscribers = new Set<(peerStates: PeerState[]) => void>();

// Debounce to coalesce bursts (snapshot replay, reconnect).
let notifyTimeout: ReturnType<typeof setTimeout> | null = null;
const notifySubscribers = () => {
  if (notifyTimeout) clearTimeout(notifyTimeout);
  notifyTimeout = setTimeout(() => {
    notifyTimeout = null;
    const states = Array.from(peerStates.values());
    for (const subscriber of subscribers) subscriber(states);
  }, 100);
};

// Peer-sync stream, shared across tabs (see sharedStream). Started only while a
// consumer wants it and, via the config gate in useSyncStates, never opens for
// the common no-peer user.
const peerStatesStream = createSharedStream<PeerState>({
  name: "backrest:peer-states",
  connect: (signal) =>
    syncStateService.getPeerSyncStatesStream({ subscribe: true }, { signal }),
  encode: (state) => toBinary(PeerStateSchema, state),
  decode: (bytes) => fromBinary(PeerStateSchema, bytes),
});

// Reload the full current peer-state set from the API. A non-subscribing stream
// sends every current state then closes, so it stands in for a unary snapshot
// fetch without holding a long-lived connection. Replaces local state wholesale
// (this is also how peers that have gone away get pruned).
const reloadPeerStates = async () => {
  const next = new Map<string, PeerState>();
  try {
    for await (const state of syncStateService.getPeerSyncStatesStream({
      subscribe: false,
    })) {
      next.set(state.peerKeyid, state);
    }
  } catch (e) {
    console.warn("failed to reload peer states", e);
    return;
  }
  peerStates.clear();
  for (const [key, value] of next) peerStates.set(key, value);
  notifySubscribers();
};

// One stream subscription feeds the map, ref-counted across consumers.
let streamRefCount = 0;
let streamUnsubscribe: (() => void) | null = null;

const acquireStream = () => {
  if (++streamRefCount === 1) {
    streamUnsubscribe = peerStatesStream.subscribe({
      onMessage: (state) => {
        peerStates.set(state.peerKeyid, state);
        notifySubscribers();
      },
      // On (re)connect, rebuild from the API; streamed deltas keep it current.
      onConnectOrResync: () => void reloadPeerStates(),
    });
  }
};

const releaseStream = () => {
  if (--streamRefCount === 0 && streamUnsubscribe) {
    streamUnsubscribe();
    streamUnsubscribe = null;
  }
};

const configReferencesPeers = (config: Config | null): boolean =>
  !!(
    config?.multihost?.knownHosts?.length ||
    config?.multihost?.authorizedClients?.length
  );

export const useSyncStates = (): PeerState[] => {
  const [config] = useConfig();
  const enabled = configReferencesPeers(config);

  const [syncStates, setSyncStates] = useState<PeerState[]>(() =>
    Array.from(peerStates.values()),
  );

  useEffect(() => {
    if (!enabled) return; // don't open the stream without peers

    const handleStateUpdate = (states: PeerState[]) => setSyncStates(states);
    subscribers.add(handleStateUpdate);
    handleStateUpdate(Array.from(peerStates.values())); // seed
    acquireStream();

    return () => {
      subscribers.delete(handleStateUpdate);
      releaseStream();
    };
  }, [enabled]);

  return syncStates;
};
