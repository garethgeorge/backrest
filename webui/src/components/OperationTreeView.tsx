import React, { useEffect, useRef, useState } from "react";
import {
  Col,
  Empty,
  Flex,
  Modal,
  Row,
  Splitter,
  Tooltip,
  Tree,
  Typography,
} from "antd";
import { groupBy } from "../lib/util";
import { DataNode } from "antd/es/tree";
import {
  formatDate,
  formatMonth,
  formatTime,
  localISOTime,
} from "../lib/formatting";
import { ExclamationOutlined, QuestionOutlined } from "@ant-design/icons";
import {
  OperationEventType,
  OperationStatus,
} from "../../gen/ts/v1/operations_pb";
import { useAlertApi } from "./Alerts";
import { OperationListView } from "./OperationListView";
import {
  ClearHistoryRequestSchema,
  ForgetRequestSchema,
  GetOperationsRequestSchema,
  type GetOperationsRequest,
} from "../../gen/ts/v1/service_pb";
import { isMobile } from "../lib/browserutil";
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
import { create, toJsonString } from "@bufbuild/protobuf";
import { useConfig } from "./ConfigProvider";

type OpTreeNode = DataNode & {
  backup?: FlowDisplayInfo;
};

export const OperationTreeView = ({
  req,
  isPlanView,
}: React.PropsWithoutRef<{
  req: GetOperationsRequest;
  isPlanView?: boolean;
}>) => {
  const config = useConfig()[0];
  const alertApi = useAlertApi();
  const setScreenWidth = useState(window.innerWidth)[1];
  const [backups, setBackups] = useState<FlowDisplayInfo[]>([]);
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
    logState.subscribe((ids, flowIDs, event) => {
      if (
        event === OperationEventType.EVENT_CREATED ||
        event === OperationEventType.EVENT_UPDATED
      ) {
        for (const flowID of flowIDs) {
          const ops = logState.getByFlowID(flowID);
          if (!ops || ops[0].op.case === "operationRunHook") {
            // sometimes hook operations become awkwardly orphaned. These are ignored.
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

      setBackups([...backupInfoByFlowID.values()]);
    });

    return syncStateFromRequest(logState, req, (err) => {
      alertApi!.error("API error: " + err.message);
    });
  }, [toJsonString(GetOperationsRequestSchema, req)]);

  if (backups.length === 0) {
    return (
      <Empty description={""} image={Empty.PRESENTED_IMAGE_SIMPLE}></Empty>
    );
  }

  const useMobileLayout = isMobile();

  const backupsByInstance = groupBy(backups, (b) => {
    return b.instanceID;
  });

  let primaryTree: React.ReactNode | null = null;
  const allTrees: React.ReactNode[] = [];

  for (const instance of Object.keys(backupsByInstance)) {
    const instanceBackups = backupsByInstance[instance];
    const instTree = (
      <DisplayOperationTree
        operations={instanceBackups}
        isPlanView={isPlanView}
        onSelect={(flow) => {
          setSelectedBackupId(flow ? flow.flowID : null);
        }}
      />
    );

    if (instance === config!.instance) {
      primaryTree = instTree;
    } else {
      allTrees.push(
        <div key={instance} style={{ marginTop: "20px" }}>
          <Typography.Title level={4}>{instance}</Typography.Title>
          {instTree}
        </div>
      );
    }
  }

  if (primaryTree) {
    allTrees.unshift(
      <div key={config!.instance} style={{ marginTop: "20px" }}>
        {allTrees.length > 0 ? (
          <Typography.Title level={4}>{config!.instance}</Typography.Title>
        ) : null}
        {primaryTree}
      </div>
    );
  }

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
        {allTrees}
      </>
    );
  }

  return (
    <Flex vertical gap="middle">
      <Splitter>
        <Splitter.Panel defaultSize="50%" min="20%" max="70%">
          {allTrees}
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
const DisplayOperationTree = ({
  operations,
  isPlanView,
  onSelect,
}: {
  operations: FlowDisplayInfo[];
  isPlanView?: boolean;
  onSelect?: (flow: FlowDisplayInfo | null) => any;
}) => {
  const [treeData, setTreeData] = useState<OpTreeNode[]>([]);
  const [expandedKeys, setExpandedKeys] = useState<Set<React.Key>>(new Set());
  const [defaultExpandedKeys, setDefaultExpandedKeys] = useState<
    Set<React.Key>
  >(new Set());

  {
    const expandFirstN = (
      n: number,
      nodes: OpTreeNode[],
      expandedKeys: Set<React.Key>
    ) => {
      let added = 0;
      for (let i = 0; i < n && i < nodes.length; i++) {
        expandedKeys.add(nodes[i].key);
        if (nodes[i].isLeaf) {
          added++;
        }
        if (nodes[i].children) {
          added += expandFirstN(n - added, nodes[i].children!, expandedKeys);
        }
        if (added >= n) {
          break;
        }
      }
      return added;
    };

    const createTreeLevel = (
      groupingFn: (op: FlowDisplayInfo) => string,
      nodeFn: (groupKey: string, ops: FlowDisplayInfo[]) => OpTreeNode,
      sortFn: (op1: FlowDisplayInfo, op2: FlowDisplayInfo) => boolean,
      operations: FlowDisplayInfo[],
      expandedKeys: Set<React.Key>,
      keyPrefix: string = ""
    ) => {
      const groups = groupBy(operations, groupingFn);
      const treeData: OpTreeNode[] = [];

      const sortedGroupKeys = Object.keys(groups).sort((a, b) => {
        const opsA = groups[a];
        const opsB = groups[b];
        return sortFn(opsA[0], opsB[0]) ? -1 : 1;
      });

      sortedGroupKeys.forEach((key) => {
        const ops = groups[key];
        const groupKey = keyPrefix + "\0" + key;
        const node = nodeFn(groupKey, ops);
        treeData.push(node);
      });

      return treeData;
    };

    interface treeSpec {
      levels: {
        groupingFn: (op: FlowDisplayInfo) => string;
        titleFn: (exemplar: FlowDisplayInfo) => React.ReactNode;
        sortFn: (op1: FlowDisplayInfo, op2: FlowDisplayInfo) => boolean;
      }[];
      leafGroupFn: (op: FlowDisplayInfo) => string;
      leafFn: (groupKey: string, ops: FlowDisplayInfo[]) => OpTreeNode;
      leafSortFn: (op1: FlowDisplayInfo, op2: FlowDisplayInfo) => boolean;
    }

    const createTree = (
      operations: FlowDisplayInfo[],
      spec: treeSpec,
      expandedKeys: Set<React.Key>
    ) => {
      let levelFn = createTreeLevel.bind(
        null,
        spec.leafGroupFn,
        (groupKey: string, ops: FlowDisplayInfo[]) =>
          spec.leafFn(groupKey, ops),
        spec.leafSortFn
      );
      const [finalLevelFn, foo] = spec.levels.reduceRight(
        ([fn, childGroupFn], level) => {
          return [
            createTreeLevel.bind(
              null,
              level.groupingFn,
              (groupKey: string, ops: FlowDisplayInfo[]) => {
                const exemplar = ops[0];
                const children = ops.length;
                const expanded = expandedKeys.has(groupKey);
                return {
                  key: groupKey,
                  title: (
                    <>
                      <Typography.Text>
                        {level.titleFn(exemplar)}
                      </Typography.Text>
                      {!expanded && (
                        <Typography.Text
                          type="secondary"
                          style={{ fontSize: "12px", marginLeft: "8px" }}
                        >
                          {children === 1 ? "1 item" : `${children} items`}
                        </Typography.Text>
                      )}
                    </>
                  ),
                  children: expanded
                    ? fn(ops, expandedKeys, groupKey)
                    : [{ key: groupKey + "_loading", title: "Loading..." }],
                };
              },
              level.sortFn
            ),
            level.groupingFn,
          ];
        },
        [levelFn, spec.leafGroupFn]
      );
      return finalLevelFn(operations, expandedKeys);
    };

    const planLayer = {
      groupingFn: (op: FlowDisplayInfo) => op.planID,
      titleFn: (exemplar: FlowDisplayInfo) => exemplar.planID,
      sortFn: (op1: FlowDisplayInfo, op2: FlowDisplayInfo) =>
        op1.planID < op2.planID,
    };

    const monthLayer = {
      groupingFn: (op: FlowDisplayInfo) =>
        localISOTime(op.displayTime).slice(0, 7),
      titleFn: (exemplar: FlowDisplayInfo) => formatMonth(exemplar.displayTime),
      sortFn: (op1: FlowDisplayInfo, op2: FlowDisplayInfo) =>
        op1.displayTime > op2.displayTime,
    };

    const dayLayer = {
      groupingFn: (op: FlowDisplayInfo) =>
        localISOTime(op.displayTime).slice(0, 10),
      titleFn: (exemplar: FlowDisplayInfo) => formatDate(exemplar.displayTime),
      sortFn: (op1: FlowDisplayInfo, op2: FlowDisplayInfo) =>
        op1.displayTime > op2.displayTime,
    };

    const leafGroupFn = (op: FlowDisplayInfo) => op.flowID.toString(16);
    const leafFn = (groupKey: string, ops: FlowDisplayInfo[]) => {
      const b = ops[0];
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

      let newLeaf: OpTreeNode = {
        key: groupKey,
        backup: b,
        icon: icon,
        isLeaf: true,
      };
      treeLeafCache.set(b, newLeaf);
      return newLeaf;
    };
    const leafSortFn = (op1: FlowDisplayInfo, op2: FlowDisplayInfo) =>
      op1.displayTime > op2.displayTime;

    const planTreeSpec: treeSpec = {
      levels: [planLayer, monthLayer, dayLayer],
      leafGroupFn,
      leafFn,
      leafSortFn,
    };

    const dayTreeSpec: treeSpec = {
      levels: [monthLayer, dayLayer],
      leafGroupFn,
      leafFn,
      leafSortFn,
    };

    const expandOperation = (
      treeSpec: treeSpec,
      op: FlowDisplayInfo,
      expandedKeys: Set<React.Key>
    ) => {
      let finalPrefix = treeSpec.levels.reduce((keyPrefix, level) => {
        let groupKey = level.groupingFn(op);
        let newKey = keyPrefix + "\0" + groupKey;
        expandedKeys.add(newKey);
        return newKey;
      }, "");
      let groupKey = treeSpec.leafGroupFn(op);
      let newKey = finalPrefix + "\0" + groupKey;
      expandedKeys.add(newKey);
    };

    useEffect(() => {
      const timeoutId = setTimeout(() => {
        let spec = isPlanView ? dayTreeSpec : planTreeSpec;
        let expandedKeysCopy = new Set(expandedKeys);

        // Do expansion passes, the algorithm is multipass since each pass may add new nodes eligible for expansion
        // Bounded at 10 passes which should be deep enough for any tree layout backrest uses.
        if (expandedKeys.size === 0) {
          let prevExpanded = new Set<React.Key>();
          const target = 5;
          for (let i = 0; i < 10; i++) {
            let newExpanded = new Set<React.Key>();
            const added = expandFirstN(
              target,
              createTree(operations, spec, prevExpanded),
              newExpanded
            );
            prevExpanded = newExpanded;
            if (added >= target) {
              break;
            }
          }
          expandedKeysCopy = prevExpanded;
          setExpandedKeys(expandedKeysCopy);
        }

        // Expand in-progress or pending operations.
        for (let op of operations) {
          if (
            op.status === OperationStatus.STATUS_INPROGRESS ||
            op.status === OperationStatus.STATUS_PENDING
          ) {
            expandOperation(planTreeSpec, op, expandedKeysCopy);
          }
        }
        if (expandedKeysCopy.size > expandedKeys.size) {
          setExpandedKeys(expandedKeysCopy);
        }

        setTreeData(createTree(operations, spec, expandedKeysCopy));
      }, 10);
      return () => clearTimeout(timeoutId);
    }, [operations, expandedKeys]);
  }

  if (treeData.length === 0) {
    return <></>;
  }

  return (
    <Tree<OpTreeNode>
      treeData={treeData}
      showIcon
      defaultExpandedKeys={Array.from(expandedKeys)}
      onExpand={(expandedKeys) => {
        setExpandedKeys(new Set(expandedKeys));
      }}
      expandedKeys={Array.from(expandedKeys)}
      onSelect={(keys, info) => {
        if (info.selectedNodes.length === 0) return;
        const backup = info.selectedNodes[0].backup;
        onSelect && onSelect(backup || null);
      }}
      titleRender={(node: OpTreeNode): React.ReactNode => {
        if (node.title !== undefined) {
          return node.title as React.ReactNode;
        }
        if (node.backup !== undefined) {
          const b = node.backup;

          return (
            <>
              <Typography.Text style={{ margin: 0, display: "inline" }}>
                {displayTypeToString(b.type)} {formatTime(b.displayTime)}{" "}
                {b.subtitleComponents && b.subtitleComponents.length > 0 && (
                  <Typography.Text
                    type="secondary"
                    style={{
                      fontFamily: "monospace",
                    }}
                  >
                    [{b.subtitleComponents.join(", ")}]
                  </Typography.Text>
                )}
              </Typography.Text>
            </>
          );
        }
        return (
          <span>ERROR: this element should not appear, this is a bug.</span>
        );
      }}
    />
  );
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
        <OperationListView
          key={backup.flowID}
          useOperations={backup.operations}
        />
      </div>
    );
  }
};
