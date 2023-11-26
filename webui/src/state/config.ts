import { atom, useSetRecoilState } from "recoil";
import { Config, Repo } from "../../gen/ts/v1/config.pb";
import { ResticUI } from "../../gen/ts/v1/service.pb";
import { API_PREFIX } from "../constants";

export const configState = atom<Config>({
  key: "config",
  default: {},
});

export const fetchConfig = async (): Promise<Config> => {
  return await ResticUI.GetConfig({}, { pathPrefix: API_PREFIX });
};

export const addRepo = async (repo: Repo): Promise<Config> => {
  return await ResticUI.AddRepo(repo, {
    pathPrefix: API_PREFIX,
  });
};

export const updateConfig = async (config: Config): Promise<Config> => {
  return await ResticUI.SetConfig(config, {
    pathPrefix: API_PREFIX,
  });
};
