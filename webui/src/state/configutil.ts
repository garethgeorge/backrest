import { Config } from "../../gen/ts/v1/config_pb";

export const shouldShowSettings = (config: Config) => {
  return !config.instance || !config.auth || (!config.auth.disabled && config.auth.users.length === 0);
}