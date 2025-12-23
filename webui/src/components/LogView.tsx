import React, { useEffect, useState } from "react";
import {
  LogDataRequest,
  LogDataRequestSchema,
} from "../../gen/ts/v1/service_pb";
import { backrestService } from "../api";
import { Button } from "./ui/button";
import { Box, Spinner, Text, Center } from "@chakra-ui/react";
import { create } from "@bufbuild/protobuf";

// TODO: refactor this to use the provider pattern
export const LogView = ({ logref }: { logref: string }) => {
  const [lines, setLines] = useState<string[]>([]);
  const [limit, setLimit] = useState(100);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!logref) {
      setLoading(false);
      return;
    }

    const controller = new AbortController();
    setLines([]);
    setLoading(true);
    setError(null);

    (async () => {
      try {
        for await (const log of backrestService.getLogs(
          create(LogDataRequestSchema, {
            ref: logref,
          }),
          { signal: controller.signal }
        )) {
          // If we receive data, we are no longer "loading" in the sense of waiting for connection
          setLoading(false);
          const text = new TextDecoder("utf-8").decode(log.value);
          const lines = text.split("\n");
          setLines((prev) => {
            const copy = [...prev];
            if (copy.length === 0) {
              copy.push("");
            }
            copy[copy.length - 1] += lines[0];
            copy.push(...lines.slice(1));
            return copy;
          });
        }
      } catch (e: any) {
        if (e.name !== 'AbortError' && !e.message?.includes("signal is aborted without reason")) {
          setError(e.message || "Failed to fetch logs");
        }
      } finally {
        setLoading(false);
      }
    })();

    return () => {
      setLines([]);
      controller.abort();
    };
  }, [logref]);

  let displayLines = lines;
  if (lines.length > limit) {
    displayLines = lines.slice(0, limit);
  }

  return (
    <Box
      overflowX="scroll"
      width="100%"
      bg="bg.muted"
      p={2}
      borderRadius="md"
      fontFamily="mono"
      fontSize="xs"
      minH="100px" // Ensure some height for spinner
    >
      {error && (
        <Text color="fg.error" mb={2} fontWeight="bold">
          Error: {error}
        </Text>
      )}

      {displayLines.map((line, i) => (
        <pre
          style={{ margin: "0px", whiteSpace: "pre", overflow: "visible" }}
          key={i}
        >
          {line}
        </pre>
      ))}

      {loading && lines.length === 0 && (
        <Center py={4}>
          <Spinner size="sm" />
        </Center>
      )}

      {lines.length > limit ? (
        <>
          <Button
            variant="ghost"
            size="xs"
            onClick={() => setLimit(limit * 10)}
            mt={2}
          >
            Show {Math.min(limit * 9, lines.length - limit)} more lines out of{" "}
            {lines.length} available...
          </Button>
        </>
      ) : null}
    </Box>
  );
};
