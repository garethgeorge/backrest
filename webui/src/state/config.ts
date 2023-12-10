import { atom, useSetRecoilState } from "recoil";
import { Config, Repo } from "../../gen/ts/v1/config.pb";
import { Restora } from "../../gen/ts/v1/service.pb";
import { API_PREFIX } from "../constants";

export const configState = atom<Config>({
  key: "config",
  default: {},
});

export const fetchConfig = async (): Promise<Config> => {
  return await Restora.GetConfig({}, { pathPrefix: API_PREFIX });
};

export const addRepo = async (repo: Repo): Promise<Config> => {
  return await Restora.AddRepo(repo, {
    pathPrefix: API_PREFIX,
  });
};

export const updateConfig = async (config: Config): Promise<Config> => {
  return await Restora.SetConfig(config, {
    pathPrefix: API_PREFIX,
  });
};
