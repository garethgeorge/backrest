import React, { useEffect, useRef, useState } from "react";
import { Col, Empty, Flex, Modal, Row, Splitter, Tooltip, Tree } from "antd";
import _ from "lodash";
import { DataNode } from "antd/es/tree";
import { formatDate, formatTime, localISOTime } from "../lib/formatting";
import { ExclamationOutlined, QuestionOutlined } from "@ant-design/icons";
import {
  OperationEventType,
  OperationStatus,
} from "../../gen/ts/v1/operations_pb";
import { useAlertApi } from "./Alerts";
import { OperationList } from "./OperationList";
import {
  ClearHistoryRequestSchema,
  ForgetRequestSchema,
  GetOperationsRequest,
} from "../../gen/ts/v1/service_pb";
import { isMobile } from "../lib/browserutil";
import { useShowModal } from "./ModalManager";
import { backrestService } from "../api";
import { ConfirmButton } from "./SpinButton";
import { OplogState, syncStateFromRequest } from "../state/logstate";
import {
  FlowDisplayInfo,
  colorForStatus,
  displayInfoForFlow,
  displayTypeToString,
} from "../state/flowdisplayaggregator";
import { OperationIcon } from "./OperationIcon";
import { shouldHideOperation } from "../state/oplog";
import { create } from "@bufbuild/protobuf";

type OpTreeNode = DataNode & {
  backup?: FlowDisplayInfo;
};

export const OperationTree = ({
  req,
  isPlanView,
}: React.PropsWithoutRef<{
  req: GetOperationsRequest;
  isPlanView?: boolean;
}>) => {
  const alertApi = useAlertApi();
  const setScreenWidth = useState(window.innerWidth)[1];
  const [backups, setBackups] = useState<FlowDisplayInfo[]>([]);
  const [treeData, setTreeData] = useState<{
    tree: OpTreeNode[];
    expanded: React.Key[];
  }>({ tree: [], expanded: [] });
  const [selectedBackupId, setSelectedBackupId] = useState<bigint | null>(null);

  // track the screen width so we can switch between mobile and desktop layouts.
  useEffect(() => {
    const handleResize = () => {
      setScreenWidth(window.innerWidth);
    };
    window.addEventListener("resize", handleResize);
    return () => {
      window.removeEventListener("resize", handleResize);
    };
  }, []);

  // track backups for this operation tree view.
  useEffect(() => {
    setSelectedBackupId(null);

    const logState = new OplogState((op) => !shouldHideOperation(op));

    const backupInfoByFlowID = new Map<bigint, FlowDisplayInfo>();

    const refresh = _.debounce(
      () => {
        const flows = Array.from(backupInfoByFlowID.values());
        setTreeData(buildTree(flows, isPlanView || false));
        setBackups(flows);
      },
      100,
      { leading: true, trailing: true }
    );

    logState.subscribe((ids, flowIDs, event) => {
      if (
        event === OperationEventType.EVENT_CREATED ||
        event === OperationEventType.EVENT_UPDATED
      ) {
        for (const flowID of flowIDs) {
          const ops = logState.getByFlowID(flowID);
          if (!ops || ops[0].op.case === "operationRunHook") {
            continue;
          }

          const displayInfo = displayInfoForFlow(ops);
          if (!displayInfo.hidden) {
            backupInfoByFlowID.set(flowID, displayInfo);
          } else {
            backupInfoByFlowID.delete(flowID);
          }
        }
      } else if (event === OperationEventType.EVENT_DELETED) {
        for (const flowID of flowIDs) {
          backupInfoByFlowID.delete(flowID);
        }
      }
      refresh();
    });

    return syncStateFromRequest(logState, req, (err) => {
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
        setSelectedBackupId(backup!.flowID);
      }}
      titleRender={(node: OpTreeNode): React.ReactNode => {
        if (node.title !== undefined) {
          return node.title as React.ReactNode;
        }
        if (node.backup !== undefined) {
          const b = node.backup;

          return (
            <>
              {displayTypeToString(b.type)} {formatTime(b.displayTime)}{" "}
              {b.subtitleComponents && b.subtitleComponents.length > 0 && (
                <span className="backrest operation-details">
                  [{b.subtitleComponents.join(", ")}]
                </span>
              )}
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
    const backup = backups.find((b) => b.flowID === selectedBackupId);
    return (
      <>
        <Modal
          open={!!backup}
          footer={null}
          onCancel={() => {
            setSelectedBackupId(null);
          }}
          width="60vw"
        >
          <BackupView backup={backup} />
        </Modal>
        {backupTree}
      </>
    );
  }

  return (
    <Flex vertical gap="middle">
      <Splitter>
        <Splitter.Panel defaultSize="50%" min="20%" max="70%">
          {backupTree}
        </Splitter.Panel>
        <Splitter.Panel style={{ paddingLeft: "10px" }}>
          <BackupViewContainer>
            {selectedBackupId ? (
              <BackupView
                backup={backups.find((b) => b.flowID === selectedBackupId)}
              />
            ) : null}
          </BackupViewContainer>{" "}
        </Splitter.Panel>
      </Splitter>
    </Flex>
  );
};

const treeLeafCache = new WeakMap<FlowDisplayInfo, OpTreeNode>();
const buildTree = (
  operations: FlowDisplayInfo[],
  isForPlanView: boolean
): { tree: OpTreeNode[]; expanded: React.Key[] } => {
  const buildTreeInstanceID = (operations: FlowDisplayInfo[]): OpTreeNode[] => {
    const grouped = _.groupBy(operations, (op) => {
      return op.instanceID;
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
        key: "i" + value[0].instanceID,
        children: buildTreePlan(value),
      };
    });
    entries.sort(sortByKeyReverse);
    return entries;
  };

  const buildTreePlan = (operations: FlowDisplayInfo[]): OpTreeNode[] => {
    const grouped = _.groupBy(operations, (op) => {
      return op.planID;
    });
    const entries: OpTreeNode[] = _.map(grouped, (value, key) => {
      let title: React.ReactNode = value[0].planID;
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
        key: "p" + value[0].planID + "\x01" + value[0].instanceID, // use \x01 as separator
        title,
        children: buildTreeDay(key, value),
      };
    });
    entries.sort(sortByKeyReverse);
    return entries;
  };

  const buildTreeDay = (
    keyPrefix: string,
    operations: FlowDisplayInfo[]
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

  const buildTreeLeaf = (operations: FlowDisplayInfo[]): OpTreeNode[] => {
    const entries = _.map(operations, (b): OpTreeNode => {
      let cached = treeLeafCache.get(b);
      if (cached) {
        return cached;
      }
      let iconColor = colorForStatus(b.status);
      let icon: React.ReactNode | null = <QuestionOutlined />;

      if (
        b.status === OperationStatus.STATUS_ERROR ||
        b.status === OperationStatus.STATUS_WARNING
      ) {
        icon = <ExclamationOutlined style={{ color: iconColor }} />;
      } else {
        icon = <OperationIcon status={b.status} type={b.type} />;
      }

      let newLeaf = {
        key: b.flowID,
        backup: b,
        icon: icon,
      };
      treeLeafCache.set(b, newLeaf);
      return newLeaf;
    });
    entries.sort((a, b) => {
      return b.backup!.displayTime - a.backup!.displayTime;
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
      if (!ref.current) {
        return;
      }
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

const BackupView = ({ backup }: { backup?: FlowDisplayInfo }) => {
  const alertApi = useAlertApi();
  if (!backup) {
    return <Empty description="Backup not found." />;
  } else {
    const doDeleteSnapshot = async () => {
      try {
        await backrestService.forget(
          create(ForgetRequestSchema, {
            planId: backup.planID!,
            repoId: backup.repoID!,
            snapshotId: backup.snapshotID!,
          })
        );
        alertApi!.success("Snapshot forgotten.");
      } catch (e) {
        alertApi!.error("Failed to forget snapshot: " + e);
      }
    };

    const snapshotInFlow = backup?.operations.find(
      (op) => op.op.case === "operationIndexSnapshot"
    );

    const deleteButton =
      snapshotInFlow && snapshotInFlow.snapshotId ? (
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
              create(ClearHistoryRequestSchema, {
                selector: {
                  flowId: backup.flowID,
                },
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
          <h3>{formatTime(backup.displayTime)}</h3>
          <div style={{ position: "absolute", right: "20px" }}>
            {backup.status !== OperationStatus.STATUS_PENDING &&
            backup.status !== OperationStatus.STATUS_INPROGRESS
              ? deleteButton
              : null}
          </div>
        </div>
        <OperationList key={backup.flowID} useOperations={backup.operations} />
      </div>
    );
  }
};
