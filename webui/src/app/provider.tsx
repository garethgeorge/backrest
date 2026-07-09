import { ChakraProvider, defaultSystem } from "@chakra-ui/react";
import { ThemeProvider } from "next-themes";
import React, { useContext, useEffect, useRef, useState } from "react";
import { useUserPreferences } from "../lib/userPreferences";
import { Config, Repo } from "../../gen/ts/v1/config_pb";
import { backrestService } from "../api/client";

// Config Context Logic
type ConfigCtx = [Config | null, (config: Config) => void];
export const ConfigContext = React.createContext<ConfigCtx>([null, () => {}]);

export const useConfig = (): ConfigCtx => {
  const context = useContext(ConfigContext);
  return context;
};

// Lookup of repos by their stable GUID, derived from the config. Cached on the
// config object's identity so it is built once per config instance and shared
// across all consumers (and garbage-collected when the config is replaced),
// giving an O(1) lookup instead of rescanning config.repos on every render.
const EMPTY_REPOS_BY_GUID: Map<string, Repo> = new Map();
const reposByGuidCache = new WeakMap<Config, Map<string, Repo>>();

export const useReposByGuid = (): Map<string, Repo> => {
  const [config] = useConfig();
  if (!config) return EMPTY_REPOS_BY_GUID;
  let byGuid = reposByGuidCache.get(config);
  if (!byGuid) {
    byGuid = new Map(config.repos.map((r) => [r.guid, r]));
    reposByGuidCache.set(config, byGuid);
  }
  return byGuid;
};

export function AppProvider(props: { children: React.ReactNode }) {
  useUserPreferences(); // Ensure locale is synced on startup
  const [config, setConfig] = useState<Config | null>(null);
  const configRef = useRef(config);
  configRef.current = config;

  // Refresh config from the backend when the window regains focus,
  // so background config changes (e.g. from another tab or peer sync)
  // become visible without a manual reload.
  useEffect(() => {
    const handleFocus = async () => {
      // Skip the initial focus tick before bootstrap has loaded a config.
      if (!configRef.current) return;
      try {
        const fresh = await backrestService.getConfig({});
        setConfig(fresh);
      } catch {
        // Ignore: transient fetch failures shouldn't break the UI.
      }
    };
    window.addEventListener("focus", handleFocus);
    return () => window.removeEventListener("focus", handleFocus);
  }, []);

  return (
    <ChakraProvider value={defaultSystem}>
      <ThemeProvider attribute="class" disableTransitionOnChange>
        <ConfigContext.Provider value={[config, setConfig]}>
          {props.children}
        </ConfigContext.Provider>
      </ThemeProvider>
    </ChakraProvider>
  );
}
