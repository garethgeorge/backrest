import { AutoComplete } from "antd";
import React, { useEffect, useState } from "react";
import { ResticUI } from "../../gen/ts/v1/service.pb";
import { StringList } from "../../gen/ts/types/value.pb";

let timeout: NodeJS.Timeout | undefined = undefined;

export const URIAutocomplete = (props: React.PropsWithChildren) => {
  const [value, setValue] = useState("");
  const [options, setOptions] = useState<{ value: string }[]>([]);
  const [showOptions, setShowOptions] = useState<{ value: string }[]>([]);

  useEffect(() => {
    setShowOptions(options.filter((o) => o.value.indexOf(value) !== -1));
  }, [options]);

  const onChange = (value: string) => {
    setValue(value);

    const lastSlash = value.lastIndexOf("/");
    if (lastSlash !== -1) {
      value = value.substring(0, lastSlash);
    }

    if (timeout) {
      clearTimeout(timeout);
    }

    timeout = setTimeout(() => {
      ResticUI.PathAutocomplete({ value: value + "/" }, { pathPrefix: "/api" })
        .then((res: StringList) => {
          if (!res.values) {
            return;
          }
          const vals = res.values.map((v) => {
            return {
              value: value + "/" + v,
            };
          });
          setOptions(vals);
        })
        .catch((e) => {
          console.log("Path autocomplete error: ", e);
        });
    }, 100);
  };

  return <AutoComplete options={showOptions} onSearch={onChange} {...props} />;
};
