import React, { useEffect, useState } from "react";
import {
  detailsForOperation,
  displayTypeToString,
  getTypeForDisplay,
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

export const ActivityBar = () => {
  const [activeOperations, setActiveOperations] = useState<Operation[]>([]);
  const setRefresh = useState<number>(0)[1];

  useEffect(() => {
    const callback = (event?: OperationEvent, err?: Error) => {
      if (!event || !event.operation) return;

      const operation = event.operation;

      setActiveOperations((ops) => {
        ops = ops.filter((op) => op.id !== operation.id);
        if (
          event.type !== OperationEventType.EVENT_DELETED &&
          operation.status === OperationStatus.STATUS_INPROGRESS
        ) {
          ops.push(operation);
        }
        ops.sort((a, b) => Number(b.unixTimeStartMs - a.unixTimeStartMs));
        return ops;
      });
    };

    subscribeToOperations(callback);

    setInterval(() => {
      setRefresh((r) => r + 1);
    }, 500);

    return () => {
      unsubscribeFromOperations(callback);
    };
  }, []);

  const details = activeOperations.map((op) => {
    return {
      op: op,
      details: detailsForOperation(op),
      displayName: displayTypeToString(getTypeForDisplay(op)),
    };
  });

  return (
    <span style={{ color: "white" }}>
      {details.map((details, idx) => {
        return (
          <span key={idx}>
            {details.displayName} in progress for plan {details.op.planId} to{" "}
            {details.op.repoId} for {formatDuration(details.details.duration)}
          </span>
        );
      })}
    </span>
  );
};
