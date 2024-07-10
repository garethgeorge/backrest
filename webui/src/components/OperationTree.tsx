import React, { useEffect, useRef, useState } from "react";
import {
  BackupInfo,
  BackupInfoCollector,
  colorForStatus,
  detailsForOperation,
  displayTypeToString,
  getTypeForDisplay,
} from "../state/oplog";
import { Col, Empty, Modal, Row, Tooltip, Tree } from "antd";
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
import { OperationStatus } from "../../gen/ts/v1/operations_pb";
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
  isPlanView,
}: React.PropsWithoutRef<{
  req: GetOperationsRequest;
  isPlanView?: boolean;
}>) => {
  const alertApi = useAlertApi();
  const showModal = useShowModal();
  const [backups, setBackups] = useState<BackupInfo[]>([]);
  const [treeData, setTreeData] = useState<{
    tree: OpTreeNode[];
    expanded: React.Key[];
  }>({ tree: [], expanded: [] });
  const [selectedBackupId, setSelectedBackupId] = useState<string | null>(null);

  // track backups for this operation tree view.
  useEffect(() => {
    setSelectedBackupId(null);

    const backupCollector = new BackupInfoCollector();
    backupCollector.subscribe(
      _.debounce(
        () => {
          let backups = backupCollector.getAll();
          backups.sort((a, b) => {
            return b.startTimeMs - a.startTimeMs;
          });
          setBackups(backups);
          setTreeData(() => buildTree(backups, isPlanView || false));
        },
        100,
        { leading: true, trailing: true }
      )
    );

    return backupCollector.collectFromRequest(req, (err) => {
      alertApi!.error("API error: " + err.message);
    });
  }, [JSON.stringify(req)]);

  if (treeData.tree.length === 0) {
    return (
      <Empty description={""} image={Empty.PRESENTED_IMAGE_SIMPLE}></Empty>
    );
  }

  const useMobileLayout = isMobile();

  const backupTree = (
    <Tree<OpTreeNode>
      treeData={treeData.tree}
      showIcon
      defaultExpandedKeys={treeData.expanded}
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
        if (node.title !== undefined) {
          return <>{node.title}</>;
        }
        if (node.backup !== undefined) {
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
              const percent = Number(s.bytesDone / s.totalBytes) * 100;
              details.push(
                `${percent.toFixed(1)}% processed ${formatBytes(
                  Number(s.bytesDone)
                )} / ${formatBytes(Number(s.totalBytes))}`
              );
            }
          } else if (b.operations.length === 1) {
            const op = b.operations[0];
            const opDetails = detailsForOperation(op);
            if (
              opDetails.percentage &&
              opDetails.percentage > 0.1 &&
              opDetails.percentage < 99.9
            ) {
              details.push(opDetails.displayState);
            }
            if (op.snapshotId) {
              details.push(`ID: ${normalizeSnapshotId(op.snapshotId)}`);
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
        <BackupViewContainer>
          {selectedBackupId ? (
            <BackupView
              backup={backups.find((b) => b.id === selectedBackupId)}
            />
          ) : null}
        </BackupViewContainer>
      </Col>
    </Row>
  );
};

const treeLeafCache = new WeakMap<BackupInfo, OpTreeNode>();
const buildTree = (
  operations: BackupInfo[],
  isForPlanView: boolean
): { tree: OpTreeNode[]; expanded: React.Key[] } => {
  const buildTreeInstanceID = (operations: BackupInfo[]): OpTreeNode[] => {
    const grouped = _.groupBy(operations, (op) => {
      return op.operations[0].instanceId!;
    });

    const entries: OpTreeNode[] = _.map(grouped, (value, key) => {
      let title: React.ReactNode = key;
      if (title === "_unassociated_") {
        title = (
          <Tooltip title="_unassociated_ instance ID collects operations that do not specify a `created-by:` tag denoting the backrest install that created them.">
            _unassociated_
          </Tooltip>
        );
      }

      return {
        title,
        key: "i-" + key,
        children: buildTreePlan(value),
      };
    });
    entries.sort(sortByKeyReverse);
    return entries;
  };

  const buildTreePlan = (operations: BackupInfo[]): OpTreeNode[] => {
    const grouped = _.groupBy(operations, (op) => {
      return op.operations[0].planId!;
    });
    const entries: OpTreeNode[] = _.map(grouped, (value, key) => {
      let title: React.ReactNode = key;
      if (title === "_unassociated_") {
        title = (
          <Tooltip title="_unassociated_ plan ID collects operations that do not specify a `plan:` tag denoting the backup plan that created them.">
            _unassociated_
          </Tooltip>
        );
      } else if (title === "_system_") {
        title = (
          <Tooltip title="_system_ plan ID collects health operations not associated with any single plan e.g. repo level check or prune runs.">
            _system_
          </Tooltip>
        );
      }
      return {
        key: "p-" + key,
        title,
        children: buildTreeDay(key, value),
      };
    });
    entries.sort(sortByKeyReverse);
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
      const children = buildTreeLeaf(value);
      return {
        key: keyPrefix + key,
        title: formatDate(value[0].displayTime),
        children: children,
      };
    });
    entries.sort(sortByKey);
    return entries;
  };

  const buildTreeLeaf = (operations: BackupInfo[]): OpTreeNode[] => {
    const entries = _.map(operations, (b): OpTreeNode => {
      let cached = treeLeafCache.get(b);
      if (cached) {
        return cached;
      }
      let iconColor = colorForStatus(b.status);
      let icon: React.ReactNode | null = <QuestionOutlined />;

      if (b.status === OperationStatus.STATUS_ERROR) {
        icon = <ExclamationOutlined style={{ color: iconColor }} />;
      } else {
        icon = <SaveOutlined style={{ color: iconColor }} />;
      }

      let newLeaf = {
        key: b.id,
        backup: b,
        icon: icon,
      };
      treeLeafCache.set(b, newLeaf);
      return newLeaf;
    });
    entries.sort((a, b) => {
      return b.backup!.startTimeMs - a.backup!.startTimeMs;
    });
    return entries;
  };

  const expandTree = (
    entries: OpTreeNode[],
    budget: number,
    d1: number,
    d2: number
  ) => {
    let expanded: React.Key[] = [];
    const h2 = (
      entries: OpTreeNode[],
      curDepth: number,
      budget: number
    ): number => {
      if (curDepth >= d2) {
        for (const entry of entries) {
          expanded.push(entry.key);
          budget--;
          if (budget <= 0) {
            break;
          }
        }
        return budget;
      }
      for (const entry of entries) {
        if (!entry.children) continue;
        budget = h2(entry.children, curDepth + 1, budget);
        if (budget <= 0) {
          break;
        }
      }
      return budget;
    };
    const h1 = (entries: OpTreeNode[], curDepth: number) => {
      if (curDepth >= d1) {
        h2(entries, curDepth + 1, budget);
        return;
      }

      for (const entry of entries) {
        if (!entry.children) continue;
        h1(entry.children, curDepth + 1);
      }
    };
    h1(entries, 0);
    return expanded;
  };

  let tree: OpTreeNode[];
  let expanded: React.Key[];
  if (isForPlanView) {
    tree = buildTreeDay("", operations);
    expanded = expandTree(tree, 5, 0, 2);
  } else {
    tree = buildTreeInstanceID(operations);
    expanded = expandTree(tree, 5, 2, 4);
  }
  return { tree, expanded };
};

const sortByKey = (a: OpTreeNode, b: OpTreeNode) => {
  if (a.key < b.key) {
    return 1;
  } else if (a.key > b.key) {
    return -1;
  }
  return 0;
};

const sortByKeyReverse = (a: OpTreeNode, b: OpTreeNode) => {
  return -sortByKey(a, b);
};

const BackupViewContainer = ({ children }: { children: React.ReactNode }) => {
  const ref = useRef<HTMLDivElement>(null);
  const innerRef = useRef<HTMLDivElement>(null);
  const refresh = useState(0)[1];
  const [topY, setTopY] = useState(0);
  const [bottomY, setBottomY] = useState(0);

  useEffect(() => {
    if (!ref.current || !innerRef.current) {
      return;
    }

    let offset = 0;

    // handle scroll events to keep the fixed container in view.
    const handleScroll = () => {
      const refRect = ref.current!.getBoundingClientRect();
      let wiggle = Math.max(refRect.height - window.innerHeight, 0);
      let topY = Math.max(ref.current!.getBoundingClientRect().top, 0);
      let bottomY = topY;
      if (topY == 0) {
        // wiggle only if the top is actually the top edge of the screen.
        topY -= wiggle;
        bottomY += wiggle;
      }

      setTopY(topY);
      setBottomY(bottomY);

      refresh(Math.random());
    };

    window.addEventListener("scroll", handleScroll);

    // attach resize observer to ref to update the width of the fixed container.
    const resizeObserver = new ResizeObserver(() => {
      handleScroll();
    });
    if (ref.current) {
      resizeObserver.observe(ref.current);
      resizeObserver.observe(innerRef.current!);
    }
    return () => {
      window.removeEventListener("scroll", handleScroll);
      resizeObserver.disconnect();
    };
  }, [ref.current, innerRef.current]);

  const rect = ref.current?.getBoundingClientRect();

  return (
    <div
      ref={ref}
      style={{
        width: "100%",
        height: innerRef.current?.clientHeight,
      }}
    >
      <div
        ref={innerRef}
        style={{
          position: "fixed",
          top: Math.max(Math.min(rect?.top || 0, bottomY), topY),
          left: rect?.left,
          width: ref.current?.clientWidth,
        }}
      >
        {children}
      </div>
    </div>
  );
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
