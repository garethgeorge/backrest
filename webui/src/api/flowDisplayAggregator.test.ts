import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import {
  OperationSchema,
  OperationStatus,
} from "../../gen/ts/v1/operations_pb";
import { displayInfoForFlow } from "./flowDisplayAggregator";

describe("backup remaining time in flow subtitles", () => {
  const backup = (status: OperationStatus) =>
    create(OperationSchema, {
      id: 1n,
      status,
      op: {
        case: "operationBackup",
        value: {
          lastStatus: {
            entry: {
              case: "status",
              value: {
                secondsElapsed: 60n,
                secondsRemaining: 200n,
                bytesDone: 1024n,
              },
            },
          },
        },
      },
    });

  it("shows restic's estimate while the backup is running", () => {
    expect(
      displayInfoForFlow([
        backup(OperationStatus.STATUS_INPROGRESS),
      ]).subtitleComponents.join(", "),
    ).toContain("~3m20s left");
  });

  it.each([
    OperationStatus.STATUS_ERROR,
    OperationStatus.STATUS_USER_CANCELLED,
    OperationStatus.STATUS_SYSTEM_CANCELLED,
    OperationStatus.STATUS_PENDING,
    OperationStatus.STATUS_SUCCESS,
  ])("hides the retained estimate for status %s", (status) => {
    const op = backup(OperationStatus.STATUS_INPROGRESS);
    expect(displayInfoForFlow([op]).subtitleComponents.join()).toContain(
      "left",
    );
    op.status = status;
    expect(displayInfoForFlow([op]).subtitleComponents.join()).not.toContain(
      "left",
    );
  });

  it("uses the backup's status even when a later hook is running", () => {
    const hook = create(OperationSchema, {
      id: 2n,
      status: OperationStatus.STATUS_INPROGRESS,
      op: { case: "operationRunHook", value: {} },
    });
    const info = displayInfoForFlow([
      backup(OperationStatus.STATUS_ERROR),
      hook,
    ]);
    expect(info.status).toBe(OperationStatus.STATUS_INPROGRESS);
    expect(info.subtitleComponents.join()).not.toContain("left");
  });
});
