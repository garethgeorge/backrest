import { ChakraProvider, defaultSystem } from "@chakra-ui/react"
import { ThemeProvider } from "next-themes"
import React, { useContext, useState } from "react";
import { Config } from "../../gen/ts/v1/config_pb";

// Config Context Logic
type ConfigCtx = [Config | null, (config: Config) => void];
const ConfigContext = React.createContext<ConfigCtx>([null, () => { }]);

export const useConfig = (): ConfigCtx => {
  const context = useContext(ConfigContext);
  return context;
};

export function AppProvider(props: { children: React.ReactNode }) {
  const [config, setConfig] = useState<Config | null>(null);

  return (
    <ChakraProvider value={defaultSystem}>
      <ThemeProvider attribute="class" disableTransitionOnChange>
        <ConfigContext.Provider value={[config, setConfig]}>
          {props.children}
        </ConfigContext.Provider>
      </ThemeProvider>
    </ChakraProvider>
  )
}
