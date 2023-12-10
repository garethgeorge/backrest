import { AutoComplete } from "antd";
import React, { useEffect, useState } from "react";
import { Restora } from "../../gen/ts/v1/service.pb";
import { StringList } from "../../gen/ts/types/value.pb";
import { isWindows } from "../state/buildcfg";

let timeout: NodeJS.Timeout | undefined = undefined;
const sep = isWindows ? "\\" : "/";

export const URIAutocomplete = (props: React.PropsWithChildren<any>) => {
  const [value, setValue] = useState("");
  const [options, setOptions] = useState<{ value: string }[]>([]);
  const [showOptions, setShowOptions] = useState<{ value: string }[]>([]);

  useEffect(() => {
    setShowOptions(options.filter((o) => o.value.indexOf(value) !== -1));
  }, [options]);

  const onChange = (value: string) => {
    setValue(value);

    const lastSlash = value.lastIndexOf(sep);
    if (lastSlash !== -1) {
      value = value.substring(0, lastSlash);
    }

    if (timeout) {
      clearTimeout(timeout);
    }

    timeout = setTimeout(() => {
      Restora.PathAutocomplete({ value: value + sep }, { pathPrefix: "/api" })
        .then((res: StringList) => {
          if (!res.values) {
            return;
          }
          const vals = res.values.map((v) => {
            return {
              value: value + sep + v,
            };
          });
          setOptions(vals);
        })
        .catch((e) => {
          console.log("Path autocomplete error: ", e);
        });
    }, 100);
  };

  return (
    <AutoComplete
      options={showOptions}
      onSearch={onChange}
      rules={[
        {
          validator: async (_: any, value: string) => {
            if (props.globAllowed) {
              return Promise.resolve();
            }
            if (isWindows) {
              if (value.match(/^[a-zA-Z]:\\$/)) {
                return Promise.reject(
                  new Error("Path must start with a drive letter e.g. C:\\")
                );
              } else if (value.includes("/")) {
                return Promise.reject(
                  new Error(
                    "Path must use backslashes e.g. C:\\Users\\MyUsers\\Documents"
                  )
                );
              }
            }
            return Promise.resolve();
          },
        },
      ]}
      {...props}
    />
  );
};
