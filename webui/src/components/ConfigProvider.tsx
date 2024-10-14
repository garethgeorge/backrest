import React, { useContext, useEffect, useState } from "react";
import { Config, Repo } from "../../gen/ts/v1/config_pb";

type ConfigCtx = [Config | null, (config: Config) => void];

const ConfigContext = React.createContext<ConfigCtx>([null, () => {}]);

export const ConfigContextProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  const [config, setConfig] = useState<Config | null>(null);
  return (
    <>
      <ConfigContext.Provider value={[config, setConfig]}>
        {children}
      </ConfigContext.Provider>
    </>
  );
};

export const useConfig = (): ConfigCtx => {
  const context = useContext(ConfigContext);
  return context;
};
