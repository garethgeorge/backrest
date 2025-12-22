import { AutoComplete } from "antd";
import React, { useEffect, useState } from "react";
import { StringList } from "../../gen/ts/types/value_pb";
import { isWindows } from "../state/buildcfg";
import { backrestService } from "../api";
import { debounce } from "../lib/util";

const sep = isWindows ? "\\" : "/";

export const URIAutocomplete = (props: React.PropsWithChildren<any>) => {
  const [value, setValue] = useState("");
  const [options, setOptions] = useState<{ value: string }[]>([]);
  const [showOptions, setShowOptions] = useState<{ value: string }[]>([]);

  useEffect(() => {
    setShowOptions(options.filter((o) => o.value.indexOf(value) !== -1));
  }, [options]);

  // eslint-disable-next-line react-hooks/exhaustive-deps
  const onChange = React.useMemo(
    () =>
      debounce((value: string) => {
        setValue(value);

        const lastSlash = value.lastIndexOf(sep);
        if (lastSlash !== -1) {
          value = value.substring(0, lastSlash);
        }

        backrestService
          .pathAutocomplete({ value: value + sep })
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
      }, 200),
    []
  );

  return (
    (<AutoComplete
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
      />)
  );
};
