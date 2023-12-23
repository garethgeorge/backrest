import { atom, useSetRecoilState } from "recoil";
import { Config, Repo } from "../../gen/ts/v1/config_pb";
import { backrestService } from "../api";

export const configState = atom<Config>({
  key: "config",
  default: new Config(),
});

export const fetchConfig = async (): Promise<Config> => {
  return await backrestService.getConfig({});
};

export const addRepo = async (repo: Repo): Promise<Config> => {
  return await backrestService.addRepo(repo);
};

export const updateConfig = async (config: Config): Promise<Config> => {
  return await backrestService.setConfig(config);
};
