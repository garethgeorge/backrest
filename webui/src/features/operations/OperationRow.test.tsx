import { create } from "@bufbuild/protobuf";
import { ChakraProvider, defaultSystem } from "@chakra-ui/react";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { OperationStatus } from "../../../gen/ts/v1/operations_pb";
import { BackupProgressEntrySchema } from "../../../gen/ts/v1/restic_pb";
import { BackupOperationStatus } from "./OperationRow";

describe("backup remaining time in operation details", () => {
  it.each([
    OperationStatus.STATUS_ERROR,
    OperationStatus.STATUS_USER_CANCELLED,
    OperationStatus.STATUS_SYSTEM_CANCELLED,
  ])(
    "removes the estimate after transitioning to status %s",
    (terminalStatus) => {
      const status = create(BackupProgressEntrySchema, {
        entry: {
          case: "status",
          value: {
            secondsElapsed: 60n,
            secondsRemaining: 200n,
          bytesDone: 1024n,
          totalBytes: 2048n,
          },
        },
      });
      const { rerender } = render(
        <BackupOperationStatus
          status={status}
          operationStatus={OperationStatus.STATUS_INPROGRESS}
        />,
        {
          wrapper: ({ children }) => (
            <ChakraProvider value={defaultSystem}>{children}</ChakraProvider>
          ),
        },
      );
      expect(screen.getByText("1m / ~3m20s")).toBeInTheDocument();

      rerender(
        <BackupOperationStatus
          status={status}
          operationStatus={terminalStatus}
        />,
      );
      expect(screen.queryByText(/~3m20s/)).not.toBeInTheDocument();
      expect(screen.getByText("1m")).toBeInTheDocument();
    },
  );
});
