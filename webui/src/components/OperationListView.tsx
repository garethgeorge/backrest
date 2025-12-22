import React, { useEffect, useState } from "react";
import { Operation } from "../../gen/ts/v1/operations_pb";
import { Empty, List } from "antd";
import {
  GetOperationsRequestSchema,
  type GetOperationsRequest,
} from "../../gen/ts/v1/service_pb";
import { useAlertApi } from "./Alerts";
import { OperationRow } from "./OperationRow";
import { OplogState, syncStateFromRequest } from "../state/logstate";
import { shouldHideStatus } from "../state/oplog";
import { toJsonString } from "@bufbuild/protobuf";

// OperationList displays a list of operations that are either fetched based on 'req' or passed in via 'useBackups'.
// If showPlan is provided the planId will be displayed next to each operation in the operation list.
export const OperationListView = ({
  req,
  useOperations,
  showPlan,
  displayHooksInline,
  filter,
  showDelete,
}: React.PropsWithoutRef<{
  req?: GetOperationsRequest;
  useOperations?: Operation[]; // exact set of operations to display; no filtering will be applied.
  showPlan?: boolean;
  displayHooksInline?: boolean;
  filter?: (op: Operation) => boolean;
  showDelete?: boolean; // allows deleting individual operation rows, useful for the list view in the plan / repo panels.
}>) => {
  const alertApi = useAlertApi();

  let [operations, setOperations] = useState<Operation[]>([]);

  if (req) {
    useEffect(() => {
      const logState = new OplogState(
        (op) => !shouldHideStatus(op.status) && (!filter || filter(op))
      );

      logState.subscribe((ids, flowIDs, event) => {
        setOperations(logState.getAll());
      });

      return syncStateFromRequest(logState, req, (e) => {
        alertApi!.error("Failed to fetch operations: " + e.message);
      });
    }, [toJsonString(GetOperationsRequestSchema, req)]);
  }
  if (!operations) {
    return (
      <Empty
        description="No operations yet."
        image={Empty.PRESENTED_IMAGE_SIMPLE}
      ></Empty>
    );
  }

  const hookExecutionsForOperation: Map<BigInt, Operation[]> = new Map();
  let operationsForDisplay: Operation[] = [];
  if (useOperations) {
    operations = [...useOperations];
  }
  if (!displayHooksInline) {
    operationsForDisplay = operations.filter((op) => {
      if (op.op.case === "operationRunHook") {
        const parentOp = op.op.value.parentOp;
        if (!hookExecutionsForOperation.has(parentOp)) {
          hookExecutionsForOperation.set(parentOp, []);
        }
        hookExecutionsForOperation.get(parentOp)!.push(op);
        return false;
      }
      return true;
    });
  } else {
    operationsForDisplay = operations;
  }
  operationsForDisplay.sort((a, b) => {
    return Number(b.unixTimeStartMs - a.unixTimeStartMs);
  });
  return (
    <List
      itemLayout="horizontal"
      size="small"
      dataSource={operationsForDisplay}
      renderItem={(op) => {
        return (
          <OperationRow
            alertApi={alertApi!}
            key={op.id}
            operation={op}
            showPlan={showPlan || false}
            hookOperations={hookExecutionsForOperation.get(op.id)}
            showDelete={showDelete}
          />
        );
      }}
      pagination={
        operationsForDisplay.length > 25
          ? { position: "both", align: "center", defaultPageSize: 25 }
          : undefined
      }
    />
  );
};
