import React, { useEffect, useState } from "react";
import {
  subscribeToOperations,
  unsubscribeFromOperations,
} from "../state/oplog";
import { formatDuration } from "../lib/formatting";
import {
  Operation,
  OperationEvent,
  OperationEventType,
  OperationStatus,
} from "../../gen/ts/v1/operations_pb";
import {
  displayTypeToString,
  getTypeForDisplay,
} from "../state/flowdisplayaggregator";

export const ActivityBar = () => {
  const [activeOperations, setActiveOperations] = useState<Operation[]>([]);
  const setRefresh = useState<number>(0)[1];

  useEffect(() => {
    const callback = (event?: OperationEvent, err?: Error) => {
      if (!event || !event.event) {
        return;
      }

      switch (event.event.case) {
        case "createdOperations":
        case "updatedOperations":
          const ops = event.event.value.operations;
          setActiveOperations((oldOps) => {
            oldOps = oldOps.filter(
              (op) => !ops.find((newOp) => newOp.id === op.id)
            );
            const newOps = ops.filter(
              (newOp) => newOp.status === OperationStatus.STATUS_INPROGRESS
            );
            return [...oldOps, ...newOps];
          });
          break;
        case "deletedOperations":
          const opIDs = event.event.value.values;
          setActiveOperations((ops) =>
            ops.filter((op) => !opIDs.includes(op.id))
          );
          break;
      }
    };

    subscribeToOperations(callback);

    setInterval(() => {
      setRefresh((r) => r + 1);
    }, 500);

    return () => {
      unsubscribeFromOperations(callback);
    };
  }, []);

  return (
    <span style={{ color: "white" }}>
      {activeOperations.map((op, idx) => {
        const displayName = displayTypeToString(getTypeForDisplay(op));

        return (
          <span key={idx}>
            {displayName} in progress for plan {op.planId} to {op.repoId} for{" "}
            {formatDuration(Date.now() - Number(op.unixTimeStartMs))}
          </span>
        );
      })}
    </span>
  );
};
