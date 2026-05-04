import { ChakraProvider, defaultSystem } from "@chakra-ui/react";
import { ThemeProvider } from "next-themes";
import React, { useContext, useEffect, useRef, useState } from "react";
import { useUserPreferences } from "../lib/userPreferences";
import { Config } from "../../gen/ts/v1/config_pb";
import { backrestService } from "../api/client";

// Config Context Logic
type ConfigCtx = [Config | null, (config: Config) => void];
const ConfigContext = React.createContext<ConfigCtx>([null, () => {}]);

export const useConfig = (): ConfigCtx => {
  const context = useContext(ConfigContext);
  return context;
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
