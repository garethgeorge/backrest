import React, { useEffect, useMemo, useState } from "react";
import {
  BackupInfo,
  BackupInfoCollector,
  getOperations,
  shouldHideStatus,
  subscribeToOperations,
  unsubscribeFromOperations,
} from "../state/oplog";
import { Col, Divider, Empty, Row, Tree } from "antd";
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
import { OperationEvent, OperationStatus } from "../../gen/ts/v1/operations.pb";
import { useAlertApi } from "./Alerts";
import { OperationList } from "./OperationList";
import { GetOperationsRequest } from "../../gen/ts/v1/service.pb";

type OpTreeNode = DataNode & {
  backup?: BackupInfo;
};

export const OperationTree = ({
  req,
}: React.PropsWithoutRef<{ req: GetOperationsRequest }>) => {
  const alertApi = useAlertApi();
  const [backups, setBackups] = useState<BackupInfo[]>([]);
  const [selectedBackupId, setSelectedBackupId] = useState<string | null>(null);

  // track backups for this operation tree view.
  useEffect(() => {
    console.log("using effect");
    setSelectedBackupId(null);
    const backupCollector = new BackupInfoCollector();
    const lis = (opEvent: OperationEvent) => {
      if (!!req.planId && opEvent.operation!.planId !== req.planId) {
        return;
      }
      if (!!req.repoId && opEvent.operation!.repoId !== req.repoId) {
        return;
      }
      backupCollector.addOperation(opEvent.type!, opEvent.operation!);
    };
    subscribeToOperations(lis);

    backupCollector.subscribe(() => {
      let backups = backupCollector.getAll();
      backups = backups.filter((b) => {
        return !shouldHideStatus(b.status);
      });
      backups.sort((a, b) => {
        return b.startTimeMs - a.startTimeMs;
      });
      setBackups(backups);
    });

    getOperations(req)
      .then((ops) => {
        backupCollector.bulkAddOperations(ops);
      })
      .catch((e) => {
        alertApi!.error("Failed to fetch operations: " + e.message);
      });
    return () => {
      unsubscribeFromOperations(lis);
    };
  }, [JSON.stringify(req)]);

  const treeData = useMemo(() => {
    return buildTreeYear(backups);
  }, [backups]);

  if (backups.length === 0) {
    return (
      <Empty
        description="No backups yet."
        image={Empty.PRESENTED_IMAGE_SIMPLE}
      ></Empty>
    );
  }

  let oplist: React.ReactNode | null = null;
  if (selectedBackupId) {
    const backup = backups.find((b) => b.id === selectedBackupId);
    if (!backup) {
      oplist = <Empty description="Backup not found." />;
    } else {
      oplist = (
        <>
          <h3>Backup at {formatTime(backup.displayTime)}</h3>
          <OperationList key={backup.id} useBackups={[backup]} />
        </>
      );
    }
  }

  return (
    <Row>
      <Col span={12}>
        <h3>Browse Backups</h3>
        <Tree<OpTreeNode>
          treeData={treeData}
          showIcon
          defaultExpandedKeys={[backups[0].id!]}
          onSelect={(keys, info) => {
            if (info.selectedNodes.length === 0) return;
            const backup = info.selectedNodes[0].backup;
            if (!backup) {
              setSelectedBackupId(null);
              return;
            }
            console.log("SELECTED ID: " + backup.id);
            setSelectedBackupId(backup.id!);
          }}
          titleRender={(node: OpTreeNode): React.ReactNode => {
            if (node.title) {
              return <>{node.title}</>;
            }
            if (node.backup) {
              const b = node.backup;
              const details: string[] = [];

              if (b.operations.length === 1) {
                if (b.operations[0].operationForget) {
                  return <>Forget {formatTime(b.displayTime)}</>;
                }
                if (b.operations[0].operationPrune) {
                  return <>Prune {formatTime(b.displayTime)}</>;
                }
              }

              if (b.status === OperationStatus.STATUS_PENDING) {
                details.push("scheduled, waiting");
              } else if (b.status === OperationStatus.STATUS_SYSTEM_CANCELLED) {
                details.push("system cancel");
              } else if (b.status === OperationStatus.STATUS_USER_CANCELLED) {
                details.push("cancelled");
              }

              if (b.backupLastStatus) {
                if (b.backupLastStatus.summary) {
                  const s = b.backupLastStatus.summary;
                  details.push(
                    `${formatBytes(s.totalBytesProcessed)} in ${formatDuration(
                      s.totalDuration! * 1000.0 // convert to ms
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
                  <span className="restora operation-details">
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
      </Col>
      <Col span={12}>{oplist}</Col>
    </Row>
  );
};

const buildTreeYear = (operations: BackupInfo[]): OpTreeNode[] => {
  const grouped = _.groupBy(operations, (op) => {
    return op.displayTime.toISOString().substring(0, 4);
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
    return op.displayTime.toISOString().substring(0, 7);
  });
  const entries: OpTreeNode[] = _.map(grouped, (value, key) => {
    return {
      key: key,
      title: value[0].displayTime.toLocaleString("default", {
        month: "long",
      }),
      children: buildTreeDay(value),
    };
  });
  entries.sort(sortByKey);
  return entries;
};

const buildTreeDay = (operations: BackupInfo[]): OpTreeNode[] => {
  const grouped = _.groupBy(operations, (op) => {
    return op.displayTime.toISOString().substring(0, 10);
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
      case OperationStatus.STATUS_USER_CANCELLED:
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
