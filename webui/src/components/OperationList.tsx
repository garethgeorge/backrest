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
  showPlan,
  filter,
}: React.PropsWithoutRef<{
  req?: GetOperationsRequest;
  useBackups?: BackupInfo[];
  showPlan?: boolean;
  filter?: (op: Operation) => boolean; // if provided, only operations that pass this filter will be displayed.
}>) => {
  const alertApi = useAlertApi();

  let backups: BackupInfo[] = [];
  if (req) {
    const [backupState, setBackups] = useState<BackupInfo[]>(useBackups || []);
    backups = backupState;

    // track backups for this operation tree view.
    useEffect(() => {
      const backupCollector = new BackupInfoCollector();
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

  if (backups.length === 0) {
    return (
      <Empty
        description="No operations yet."
        image={Empty.PRESENTED_IMAGE_SIMPLE}
      ></Empty>
    );
  }

  let operations = backups.flatMap((b) => b.operations);
  operations.sort((a, b) => {
    return Number(b.unixTimeStartMs - a.unixTimeStartMs);
  });
  return (
    <List
      itemLayout="horizontal"
      size="small"
      dataSource={operations}
      renderItem={(op, index) => {
        return (
          <OperationRow
            alertApi={alertApi!}
            key={op.id}
            operation={op}
            showPlan={showPlan || false}
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
