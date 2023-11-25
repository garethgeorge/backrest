import React, { useEffect, useState } from "react";
import {
  BackupInfo,
  BackupInfoCollector,
  EOperation,
  getOperations,
  subscribeToOperations,
  toEop,
  unsubscribeFromOperations,
} from "../state/oplog";
import { Tree } from "antd";
import _ from "lodash";
import { DataNode } from "antd/es/tree";
import {
  formatBytes,
  formatDate,
  formatDuration,
  formatTime,
  normalizeSnapshotId,
} from "../lib/formatting";
import {
  ExclamationOutlined,
  QuestionOutlined,
  SaveOutlined,
} from "@ant-design/icons";
import {
  OperationEvent,
  OperationEventType,
  OperationStatus,
} from "../../gen/ts/v1/operations.pb";
import { useAlertApi } from "./Alerts";
import { MAX_OPERATION_HISTORY } from "../constants";
import { OperationList } from "./OperationList";

type OpTreeNode = DataNode & {
  backup?: BackupInfo;
};

export const OperationTree = ({
  planId,
  repoId,
}: React.PropsWithoutRef<{ planId?: string; repoId?: string }>) => {
  const alertApi = useAlertApi();
  const [backups, setBackups] = useState<BackupInfo[]>([]);

  // track backups for this operation tree view.
  useEffect(() => {
    const backupCollector = new BackupInfoCollector();
    const lis = (opEvent: OperationEvent) => {
      backupCollector.addOperation(opEvent.type!, opEvent.operation!);
    };
    subscribeToOperations(lis);

    backupCollector.subscribe(() => {
      const backups = backupCollector.getAll();
      backups.sort((a, b) => {
        return b.startTimeMs - a.startTimeMs;
      });
      setBackups(backups);
    });

    getOperations({
      planId,
      repoId,
      snapshotId,
      lastN: "" + MAX_OPERATION_HISTORY,
    })
      .then((ops) => {
        backupCollector.bulkAddOperations(ops);
      })
      .catch((e) => {
        alertApi!.error("Failed to fetch operations: " + e.message);
      });
    return () => {
      unsubscribeFromOperations(lis);
    };
  }, [planId, repoId]);

  if (backups.length === 0) {
    return (
      <div>
        <QuestionOutlined /> No operations yet.
      </div>
    );
  }

  const treeData = buildTreeYear(backups);

  return (
    <Tree<OpTreeNode>
      treeData={treeData}
      showIcon
      defaultExpandedKeys={[backups[0].id!]}
      titleRender={(node: OpTreeNode): React.ReactNode => {
        if (node.title) {
          return <>{node.title}</>;
        }
        if (node.backup) {
          const b = node.backup;
          const details: string[] = [];

          if (b.backupLastStatus) {
            if (b.backupLastStatus.summary) {
              const s = b.backupLastStatus.summary;
              details.push(
                `${formatBytes(s.totalBytesProcessed)} in ${formatDuration(
                  s.totalDuration!
                )}`
              );
            } else if (b.backupLastStatus.status) {
              const s = b.backupLastStatus.status;
              const bytesDone = parseInt(s.bytesDone!);
              const bytesTotal = parseInt(s.totalBytes!);
              const percent = Math.floor((bytesDone / bytesTotal) * 100);
              details.push(
                `${percent}% processed ${formatBytes(
                  bytesDone
                )} / ${formatBytes(bytesTotal)}`
              );
            }
          }
          if (b.snapshotInfo) {
            details.push(`ID: ${normalizeSnapshotId(b.snapshotInfo.id!)}`);
          }

          let detailsElem: React.ReactNode | null = null;
          if (details.length > 0) {
            detailsElem = (
              <span className="resticui backup-details">
                [{details.join(", ")}]
              </span>
            );
          }

          return (
            <>
              Backup {formatTime(b.displayTime)} {detailsElem}
            </>
          );
        }
        return (
          <span>ERROR: this element should not appear, this is a bug.</span>
        );
      }}
    ></Tree>
  );
};

const BackupInfoPanel = ({
  backup,
}: React.PropsWithoutRef<{ backup: BackupInfo }>) => {
  return (
    <>
      <OperationList operations={backup.operations.map(toEop)} />
    </>
  );
};

const buildTreeYear = (operations: BackupInfo[]): OpTreeNode[] => {
  const grouped = _.groupBy(operations, (op) => {
    return op.displayTime.getFullYear();
  });

  const entries: OpTreeNode[] = _.map(grouped, (value, key) => {
    return {
      key: "y" + key,
      title: "" + key,
      children: buildTreeMonth(value),
    };
  });
  entries.sort(sortByKey);
  return entries;
};

const buildTreeMonth = (operations: BackupInfo[]): OpTreeNode[] => {
  const grouped = _.groupBy(operations, (op) => {
    return `y${op.displayTime.getFullYear()}m${op.displayTime.getMonth()}`;
  });
  const entries: OpTreeNode[] = _.map(grouped, (value, key) => {
    return {
      key: key,
      title: value[0].displayTime.toLocaleString("default", {
        month: "long",
        year: "numeric",
      }),
      children: buildTreeDay(value),
    };
  });
  entries.sort(sortByKey);
  return entries;
};

const buildTreeDay = (operations: BackupInfo[]): OpTreeNode[] => {
  const grouped = _.groupBy(operations, (op) => {
    return `y${op.displayTime.getFullYear()}m${op.displayTime.getMonth()}d${op.displayTime.getDate()}`;
  });

  const entries = _.map(grouped, (value, key) => {
    return {
      key: "d" + key,
      title: formatDate(value[0].displayTime),
      children: buildTreeLeaf(value),
    };
  });
  entries.sort(sortByKey);
  return entries;
};

const buildTreeLeaf = (operations: BackupInfo[]): OpTreeNode[] => {
  const entries = _.map(operations, (b): OpTreeNode => {
    let iconColor = "grey";
    let icon: React.ReactNode | null = <QuestionOutlined />;

    switch (b.status) {
      case OperationStatus.STATUS_PENDING:
        iconColor = "grey";
        break;
      case OperationStatus.STATUS_SUCCESS:
        iconColor = "green";
        break;
      case OperationStatus.STATUS_ERROR:
        iconColor = "red";
        break;
      case OperationStatus.STATUS_INPROGRESS:
        iconColor = "blue";
        break;
      case OperationStatus.STATUS_CANCELLED:
        iconColor = "orange";
        break;
    }

    if (b.status === OperationStatus.STATUS_ERROR) {
      icon = <ExclamationOutlined style={{ color: iconColor }} />;
    } else {
      icon = <SaveOutlined style={{ color: iconColor }} />;
    }

    return {
      key: b.id!,
      backup: b,
      icon: icon,
    };
  });
  entries.sort(sortByKey);
  return entries;
};

const sortByKey = (a: OpTreeNode, b: OpTreeNode) => {
  if (a.key < b.key) {
    return 1;
  } else if (a.key > b.key) {
    return -1;
  }
  return 0;
};
