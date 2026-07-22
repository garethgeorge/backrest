import { useEffect, useState } from "react";
import { operationsStream } from "../../api/oplog";
import { formatDuration } from "../../lib/formatting";
import { Operation, OperationStatus } from "../../../gen/ts/v1/operations_pb";
import * as m from "../../paraglide/messages";
import {
  displayTypeToString,
  getTypeForDisplay,
} from "../../api/flowDisplayAggregator";

export const ActivityBar = () => {
  const [activeOperations, setActiveOperations] = useState<Operation[]>([]);
  const setRefresh = useState<number>(0)[1];

  useEffect(() => {
    const unsubscribe = operationsStream.subscribe({
      onMessage: (event) => {
        switch (event.event.case) {
          case "createdOperations":
          case "updatedOperations":
            const ops = event.event.value.operations;
            setActiveOperations((oldOps) => {
              oldOps = oldOps.filter(
                (op) => !ops.find((newOp) => newOp.id === op.id),
              );
              const newOps = ops.filter(
                (newOp) => newOp.status === OperationStatus.STATUS_INPROGRESS,
              );
              return [...oldOps, ...newOps];
            });
            break;
          case "deletedOperations":
            const opIDs = event.event.value.values;
            setActiveOperations((ops) =>
              ops.filter((op) => !opIDs.includes(op.id)),
            );
            break;
        }
      },
      // Drop the delta-accumulated list; in-progress updates repopulate it.
      onConnectOrResync: () => setActiveOperations([]),
    });

    const interval = setInterval(() => {
      setRefresh((r) => r + 1);
    }, 500);

    return () => {
      unsubscribe();
      clearInterval(interval);
    };
  }, []);

  return (
    <span style={{ color: "white" }}>
      {activeOperations.map((op, idx) => {
        const displayName = displayTypeToString(getTypeForDisplay(op));

        return (
          <span key={idx} style={{ marginRight: "2em" }}>
            {displayName} {m.activity_bar_in_progress_for_plan()} {op.planId} to {op.repoId} for{" "}
            {formatDuration(Date.now() - Number(op.unixTimeStartMs))}
          </span>
        );
      })}
    </span>
  );
};
