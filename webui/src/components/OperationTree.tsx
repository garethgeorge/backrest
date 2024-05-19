import React, { useEffect, useMemo, useState } from "react";
import {
  BackupInfo,
  BackupInfoCollector,
  colorForStatus,
  displayTypeToString,
  getOperations,
  getTypeForDisplay,
  matchSelector,
  shouldHideOperation,
  subscribeToOperations,
  unsubscribeFromOperations,
} from "../state/oplog";
import { Button, Col, Divider, Empty, Modal, Row, Tooltip, Tree } from "antd";
import _ from "lodash";
import { DataNode } from "antd/es/tree";
import {
  formatBytes,
  formatDate,
  formatDuration,
  formatTime,
  localISOTime,
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
} from "../../gen/ts/v1/operations_pb";
import { useAlertApi } from "./Alerts";
import { OperationList } from "./OperationList";
import {
  ClearHistoryRequest,
  ForgetRequest,
  GetOperationsRequest,
  OpSelector,
} from "../../gen/ts/v1/service_pb";
import { isMobile } from "../lib/browserutil";
import { useShowModal } from "./ModalManager";
import { backrestService } from "../api";
import { ConfirmButton } from "./SpinButton";

type OpTreeNode = DataNode & {
  backup?: BackupInfo;
};

export const OperationTree = ({
  req,
}: React.PropsWithoutRef<{ req: GetOperationsRequest }>) => {
  const alertApi = useAlertApi();
  const showModal = useShowModal();
  const [loading, setLoading] = useState(true);
  const [backups, setBackups] = useState<BackupInfo[]>([]);
  const [selectedBackupId, setSelectedBackupId] = useState<string | null>(null);

  // track backups for this operation tree view.
  useEffect(() => {
    setSelectedBackupId(null);
    const backupCollector = new BackupInfoCollector();
    const lis = (opEvent: OperationEvent) => {
      if (
        !req.selector ||
        !opEvent.operation ||
        !matchSelector(req.selector, opEvent.operation)
      ) {
        return;
      }
      if (opEvent.type !== OperationEventType.EVENT_DELETED) {
        backupCollector.addOperation(opEvent.type!, opEvent.operation!);
      } else {
        backupCollector.removeOperation(opEvent.operation!);
      }
    };
    subscribeToOperations(lis);

    backupCollector.subscribe(
      _.debounce(() => {
        let backups = backupCollector.getAll();
        backups.sort((a, b) => {
          return b.startTimeMs - a.startTimeMs;
        });
        setBackups(backups);
      }, 50)
    );

    getOperations(req)
      .then((ops) => {
        backupCollector.bulkAddOperations(ops);
      })
      .catch((e) => {
        alertApi!.error("Failed to fetch operations: " + e.messag);
      })
      .finally(() => {
        setLoading(false);
      });
    return () => {
      unsubscribeFromOperations(lis);
    };
  }, [JSON.stringify(req)]);

  const treeData = useMemo(() => {
    return buildTreePlan(backups);
  }, [backups]);

  if (backups.length === 0) {
    return (
      <Empty
        description={loading ? "Loading..." : "No backups yet."}
        image={Empty.PRESENTED_IMAGE_SIMPLE}
      ></Empty>
    );
  }

  const useMobileLayout = isMobile();

  const backupTree = (
    <Tree<OpTreeNode>
      treeData={treeData}
      showIcon
      defaultExpandedKeys={backups
        .slice(0, Math.min(5, backups.length))
        .map((b) => b.id!)}
      onSelect={(keys, info) => {
        if (info.selectedNodes.length === 0) return;
        const backup = info.selectedNodes[0].backup;
        if (!backup) {
          setSelectedBackupId(null);
          return;
        }
        setSelectedBackupId(backup.id!);

        if (useMobileLayout) {
          showModal(
            <Modal
              visible={true}
              footer={null}
              onCancel={() => {
                showModal(null);
              }}
            >
              <BackupView backup={backup} />
            </Modal>
          );
        }
      }}
      titleRender={(node: OpTreeNode): React.ReactNode => {
        if (node.title) {
          return <>{node.title}</>;
        }
        if (node.backup) {
          const b = node.backup;
          const details: string[] = [];

          if (b.operations.length === 0) {
            // this happens when all operations in a backup are deleted; it should be hidden until the deletion propagates to a refresh of the tree layout.
            return null;
          }

          if (b.status === OperationStatus.STATUS_PENDING) {
            details.push("scheduled, waiting");
          } else if (b.status === OperationStatus.STATUS_SYSTEM_CANCELLED) {
            details.push("system cancel");
          } else if (b.status === OperationStatus.STATUS_USER_CANCELLED) {
            details.push("cancelled");
          }

          if (b.backupLastStatus) {
            if (b.backupLastStatus.entry.case === "summary") {
              const s = b.backupLastStatus.entry.value;
              details.push(
                `${formatBytes(
                  Number(s.totalBytesProcessed)
                )} in ${formatDuration(
                  s.totalDuration! * 1000.0 // convert to ms
                )}`
              );
            } else if (b.backupLastStatus.entry.case === "status") {
              const s = b.backupLastStatus.entry.value;
              const percent = Math.floor(
                (Number(s.bytesDone) / Number(s.totalBytes)) * 100
              );
              details.push(
                `${percent}% processed ${formatBytes(
                  Number(s.bytesDone)
                )} / ${formatBytes(Number(s.totalBytes))}`
              );
            }
          }
          if (b.snapshotInfo) {
            details.push(`ID: ${normalizeSnapshotId(b.snapshotInfo.id)}`);
          }

          let detailsElem: React.ReactNode | null = null;
          if (details.length > 0) {
            detailsElem = (
              <span className="backrest operation-details">
                [{details.join(", ")}]
              </span>
            );
          }

          const type = getTypeForDisplay(b.operations[0]);
          return (
            <>
              {displayTypeToString(type)} {formatTime(b.displayTime)}{" "}
              {detailsElem}
            </>
          );
        }
        return (
          <span>ERROR: this element should not appear, this is a bug.</span>
        );
      }}
    />
  );

  if (useMobileLayout) {
    return backupTree;
  }

  return (
    <Row>
      <Col span={12}>{backupTree}</Col>
      <Col span={12}>
        {selectedBackupId ? (
          <BackupView backup={backups.find((b) => b.id === selectedBackupId)} />
        ) : null}
      </Col>
    </Row>
  );
};

const buildTreePlan = (operations: BackupInfo[]): OpTreeNode[] => {
  const grouped = _.groupBy(operations, (op) => {
    return op.operations[0].planId!;
  });

  const entries: OpTreeNode[] = _.map(grouped, (value, key) => {
    return {
      key: key,
      title: key,
      children: buildTreeDay(key, value),
    };
  });
  if (entries.length === 1) {
    return entries[0].children!;
  }
  entries.sort(sortByKey);
  return entries;
};

const buildTreeDay = (
  keyPrefix: string,
  operations: BackupInfo[]
): OpTreeNode[] => {
  const grouped = _.groupBy(operations, (op) => {
    return localISOTime(op.displayTime).substring(0, 10);
  });

  const entries = _.map(grouped, (value, key) => {
    return {
      key: keyPrefix + key,
      title: formatDate(value[0].displayTime),
      children: buildTreeLeaf(value),
    };
  });
  entries.sort(sortByKey);
  return entries;
};

const buildTreeLeaf = (operations: BackupInfo[]): OpTreeNode[] => {
  const entries = _.map(operations, (b): OpTreeNode => {
    let iconColor = colorForStatus(b.status);
    let icon: React.ReactNode | null = <QuestionOutlined />;

    if (b.status === OperationStatus.STATUS_ERROR) {
      icon = <ExclamationOutlined style={{ color: iconColor }} />;
    } else {
      icon = <SaveOutlined style={{ color: iconColor }} />;
    }

    return {
      key: b.id,
      backup: b,
      icon: icon,
    };
  });
  entries.sort((a, b) => {
    return b.backup!.startTimeMs - a.backup!.startTimeMs;
  });
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

const BackupView = ({ backup }: { backup?: BackupInfo }) => {
  const alertApi = useAlertApi();
  if (!backup) {
    return <Empty description="Backup not found." />;
  } else {
    const doDeleteSnapshot = async () => {
      try {
        await backrestService.forget(
          new ForgetRequest({
            planId: backup.planId!,
            repoId: backup.repoId!,
            snapshotId: backup.snapshotId!,
          })
        );
        alertApi!.success("Snapshot forgotten.");
      } catch (e) {
        alertApi!.error("Failed to forget snapshot: " + e);
      }
    };

    const deleteButton = backup.snapshotId ? (
      <Tooltip title="This will remove the snapshot from the repository. This is irreversible.">
        <ConfirmButton
          type="text"
          confirmTitle="Confirm forget?"
          confirmTimeout={2000}
          onClickAsync={doDeleteSnapshot}
        >
          Forget (Destructive)
        </ConfirmButton>
      </Tooltip>
    ) : (
      <ConfirmButton
        type="text"
        confirmTitle="Confirm clear?"
        onClickAsync={async () => {
          backrestService.clearHistory(
            new ClearHistoryRequest({
              selector: new OpSelector({
                ids: backup.operations.map((op) => op.id),
              }),
            })
          );
        }}
      >
        Delete Event
      </ConfirmButton>
    );

    return (
      <div style={{ width: "100%" }}>
        <div
          style={{
            alignItems: "center",
            display: "flex",
            flexDirection: "row",
            width: "100%",
            height: "60px",
          }}
        >
          <h3>Backup on {formatTime(backup.displayTime)}</h3>
          <div style={{ position: "absolute", right: "20px" }}>
            {backup.status !== OperationStatus.STATUS_PENDING &&
            backup.status != OperationStatus.STATUS_INPROGRESS
              ? deleteButton
              : null}
          </div>
        </div>
        <OperationList key={backup.id} useBackups={[backup]} />
      </div>
    );
  }
};
