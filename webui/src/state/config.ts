import { atom, useSetRecoilState } from "recoil";
import { Config } from "../../gen/ts/v1/config.pb";
import { ResticUI } from "../../gen/ts/v1/service.pb";

export const configState = atom({
  key: "config",
  default: null as Config | null,
});

export const fetchConfig = async (): Promise<Config> => {
  return await ResticUI.GetConfig(
    {},
    {
      pathPrefix: "/api/",
    }
  );
};
