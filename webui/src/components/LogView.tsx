import React, { useEffect, useState } from "react";
import { LogDataRequest } from "../../gen/ts/v1/service_pb";
import { backrestService } from "../api";

import AutoSizer from "react-virtualized/dist/commonjs/AutoSizer";
import List from "react-virtualized/dist/commonjs/List";
import { set } from "lodash";

// TODO: refactor this to use the provider pattern
export const LogView = ({ logref }: { logref: string }) => {
  const [lines, setLines] = useState<string[]>([""]);

  console.log("LogView", logref);

  useEffect(() => {
    if (!logref) {
      return;
    }

    const controller = new AbortController();

    (async () => {
      try {
        for await (const log of backrestService.getLogs(
          new LogDataRequest({
            ref: logref,
          }),
          { signal: controller.signal }
        )) {
          const text = new TextDecoder("utf-8").decode(log.value);
          const lines = text.split("\n");
          setLines((prev) => {
            const copy = [...prev];
            copy[copy.length - 1] += lines[0];
            copy.push(...lines.slice(1));
            return copy;
          });
        }
      } catch (e) {
        setLines((prev) => [...prev, `Fetch log error: ${e}`]);
      }
    })();

    return () => {
      setLines([]);
      controller.abort();
    };
  }, [logref]);

  console.log("LogView", lines);

  return (
    <div
      style={{
        overflowX: "scroll",
        width: "100%",
      }}
    >
      {lines.map((line, i) => (
        <pre style={{ margin: "0px" }} key={i}>
          {line}
        </pre>
      ))}
    </div>
  );
};
