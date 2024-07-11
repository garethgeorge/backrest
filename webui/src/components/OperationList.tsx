import React, { useEffect, useState } from "react";
import {
  Operation,
  OperationEvent,
  OperationEventType,
} from "../../gen/ts/v1/operations_pb";
import { Empty, List } from "antd";
import {
  BackupInfo,
  BackupInfoCollector,
  getOperations,
  matchSelector,
  shouldHideStatus,
  subscribeToOperations,
  unsubscribeFromOperations,
} from "../state/oplog";
import _ from "lodash";
import { GetOperationsRequest } from "../../gen/ts/v1/service_pb";
import { useAlertApi } from "./Alerts";
import { OperationRow } from "./OperationRow";

// OperationList displays a list of operations that are either fetched based on 'req' or passed in via 'useBackups'.
// If showPlan is provided the planId will be displayed next to each operation in the operation list.
export const OperationList = ({
  req,
  useBackups,
  useOperations,
  showPlan,
}: React.PropsWithoutRef<{
  req?: GetOperationsRequest;
  useBackups?: BackupInfo[]; // a backup to display; some operations will be filtered out e.g. hook executions.
  useOperations?: Operation[]; // exact set of operations to display; no filtering will be applied.
  showPlan?: boolean;
}>) => {
  const alertApi = useAlertApi();

  let backups: BackupInfo[] = [];
  if (req) {
    const [backupState, setBackups] = useState<BackupInfo[]>(useBackups || []);
    backups = backupState;

    // track backups for this operation tree view.
    useEffect(() => {
      const backupCollector = new BackupInfoCollector(
        (op) => !shouldHideStatus(op.status)
      );
      backupCollector.subscribe(
        _.debounce(
          () => {
            let backups = backupCollector.getAll();
            backups.sort((a, b) => {
              return b.startTimeMs - a.startTimeMs;
            });
            setBackups(backups);
          },
          100,
          { leading: true, trailing: true }
        )
      );

      return backupCollector.collectFromRequest(req, (err) => {
        alertApi!.error("API error: " + err.message);
      });
    }, [JSON.stringify(req)]);
  } else {
    backups = [...(useBackups || [])];
  }

  if (backups.length === 0 && !useOperations) {
    return (
      <Empty
        description="No operations yet."
        image={Empty.PRESENTED_IMAGE_SIMPLE}
      ></Empty>
    );
  }

  const hookExecutionsForOperation: Map<BigInt, Operation[]> = new Map();
  let operations: Operation[] = [];
  if (useOperations) {
    operations = useOperations;
  } else {
    operations = backups
      .flatMap((b) => b.operations)
      .filter((op) => {
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
  }
  operations.sort((a, b) => {
    return Number(b.unixTimeStartMs - a.unixTimeStartMs);
  });
  return (
    <List
      itemLayout="horizontal"
      size="small"
      dataSource={operations}
      renderItem={(op) => {
        return (
          <OperationRow
            alertApi={alertApi!}
            key={op.id}
            operation={op}
            showPlan={showPlan || false}
            hookOperations={hookExecutionsForOperation.get(op.id)}
          />
        );
      }}
      pagination={
        operations.length > 25
          ? { position: "both", align: "center", defaultPageSize: 25 }
          : undefined
      }
    />
  );
};
