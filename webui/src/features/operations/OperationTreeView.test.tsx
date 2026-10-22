import { screen, waitFor } from "@testing-library/react";
import React, { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { OperationStatus, OperationSchema } from "../../../gen/ts/v1/operations_pb";
import { GetOperationsRequestSchema } from "../../../gen/ts/v1/service_pb";
import { renderWithProviders } from "../../test/render";
import { makeConfig, makeRepo } from "../../test/proto";
import { OperationTreeView } from "./OperationTreeView";

// The splitter is Zag-based and needs DOM primitives jsdom lacks; it is
// unrelated to the tree remount behavior under test.
vi.mock("../../components/ui/splitter", () => ({
  Splitter: ({ children }: any) => <div>{children}</div>,
  SplitterPanel: ({ children }: any) => <div>{children}</div>,
  SplitterResizeTrigger: () => null,
}));

// Drive the real OplogState without a backend: syncStateFromRequest
// immediately "loads" a prepared set of operations (the initial-load path)
// and reports the load as complete.
const hoisted = vi.hoisted(() => ({ ops: [] as any[] }));
vi.mock("../../api/logState", async (importOriginal) => {
  const actual = await importOriginal<any>();
  return {
    ...actual,
    syncStateFromRequest: (
      state: any,
      _req: any,
      _onError?: any,
      onInitialLoad?: any,
    ) => {
      state.add(...hoisted.ops);
      onInitialLoad?.();
      return () => {};
    },
  };
});

const makeOp = (id: number, dayOffset: number) =>
  create(OperationSchema, {
    id: BigInt(id),
    flowId: BigInt(id),
    repoId: "test-repo",
    planId: "test-plan",
    instanceId: "test-instance",
    snapshotId: `snap-${id}`,
    status: OperationStatus.STATUS_SUCCESS,
    // Spread the ops across days/months so they sit under separate tree
    // branches, like a real history.
    unixTimeStartMs: BigInt(1_700_000_000_000 + dayOffset * 86_400_000 + id),
    unixTimeEndMs: BigInt(1_700_000_500_000 + dayOffset * 86_400_000 + id),
    op: { case: "operationBackup", value: {} },
  });

hoisted.ops = [makeOp(1, 0), makeOp(2, 5), makeOp(3, 60)];

// Recreates the operations request on every render (the pattern RepoView
// used before the fix): req identity churn must not remount the tree.
const Harness = () => {
  const [tick, setTick] = useState(0);
  const req = create(GetOperationsRequestSchema, {
    selector: { repoGuid: "test-repo-guid" },
    lastN: 500n,
  });
  return (
    <>
      <button data-testid="tick" onClick={() => setTick(tick + 1)}>
        tick
      </button>
      <OperationTreeView req={req} />
    </>
  );
};

describe("OperationTreeView", () => {
  it("does not remount tree leaves when an unrelated parent re-renders", async () => {
    const config = makeConfig({
      repos: [makeRepo({ id: "test-repo", guid: "test-repo-guid" })],
    });
    const { user } = renderWithProviders(<Harness />, { config });

    const leaves = () => screen.getAllByTestId("tree-node");
    await waitFor(() => expect(leaves().length).toBe(3));
    const before = leaves();

    // Two unrelated parent re-renders, each with a fresh req object.
    await user.click(screen.getByTestId("tick"));
    await user.click(screen.getByTestId("tick"));

    const after = leaves();
    expect(after.length).toBe(3);
    // Same DOM nodes, i.e. the leaves were not remounted.
    for (let i = 0; i < before.length; i++) {
      expect(after[i]).toBe(before[i]);
    }
  });
});
