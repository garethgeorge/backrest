import React, { useEffect, useMemo, useState } from "react";
import {
  Box,
  Flex,
  Heading,
  Text,
  createTreeCollection,
  EmptyState,
  VStack,
  TreeCollection,
  Center,
  Spinner,
} from "@chakra-ui/react";
import {
  TreeViewRoot,
  TreeViewTree,
  TreeViewNode,
  TreeViewItem,
  TreeViewBranchControl,
  TreeViewBranchText,
  TreeViewItemText,
  TreeViewBranchIndentGuide,
  TreeViewBranchContent,
  TreeViewBranchTrigger,
} from "../../components/ui/tree-view";
import {
  DialogRoot,
  DialogContent,
  DialogBody,
  DialogCloseTrigger,
} from "../../components/ui/dialog";
import {
  Splitter,
  SplitterPanel,
  SplitterResizeTrigger,
} from "../../components/ui/splitter";
import { groupBy } from "../../lib/util";
import {
  formatDate,
  formatMonth,
  formatTime,
  localISOTime,
} from "../../lib/formatting";
import {
  LuFileQuestion,
  LuInfo,
  LuFolder,
  LuFile,
  LuChevronRight,
} from "react-icons/lu";
import {
  OperationEventType,
  OperationStatus,
} from "../../../gen/ts/v1/operations_pb";
import { alerts } from "../../components/common/Alerts";
import { OperationListView } from "./OperationListView";
import {
  ClearHistoryRequestSchema,
  ForgetRequestSchema,
  GetOperationsRequestSchema,
  type GetOperationsRequest,
} from "../../../gen/ts/v1/service_pb";
import { isMobile } from "../../lib/browserUtil";
import { backrestService } from "../../api/client";
import { ConfirmButton } from "../../components/common/SpinButton";
import { OplogState, syncStateFromRequest } from "../../api/logState";
import {
  FlowDisplayInfo,
  colorForStatus,
  displayInfoForFlow,
  displayTypeToString,
} from "../../api/flowDisplayAggregator";
import { OperationIcon } from "./OperationIcon";
import { shouldHideOperation } from "../../api/oplog";
import { create, toJsonString } from "@bufbuild/protobuf";
import { useConfig } from "../../app/provider";

interface OpTreeNode {
  id: string;
  label: string;
  children?: OpTreeNode[];
  backup?: FlowDisplayInfo;
  icon?: React.ReactNode;
}

export const OperationTreeView = ({
  req,
  isPlanView,
}: React.PropsWithoutRef<{
  req: GetOperationsRequest;
  isPlanView?: boolean;
}>) => {
  const config = useConfig()[0];
  const setScreenWidth = useState(window.innerWidth)[1];
  const [backups, setBackups] = useState<FlowDisplayInfo[]>([]);
  const [selectedBackupId, setSelectedBackupId] = useState<bigint | null>(null);
  const [loading, setLoading] = useState(true);

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
      setLoading(false);
    });

    return syncStateFromRequest(logState, req, (err) => {
      alerts.error("API error: " + err.message);
      setLoading(false);
    });
  }, [toJsonString(GetOperationsRequestSchema, req)]);

  if (loading && backups.length === 0) {
    return (
      <Center height="100%">
        <Spinner size="lg" />
      </Center>
    );
  }

  if (backups.length === 0) {
    return (
      <EmptyState.Root>
        <EmptyState.Content>
          <EmptyState.Indicator>
            <LuInfo />
          </EmptyState.Indicator>
          <VStack textAlign="center">
            <EmptyState.Title>No operations found</EmptyState.Title>
            <EmptyState.Description>
              There are no operations to display.
            </EmptyState.Description>
          </VStack>
        </EmptyState.Content>
      </EmptyState.Root>
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
        <Box key={instance} marginTop="4">
          <Heading size="md" marginBottom="2">
            {instance}
          </Heading>
          {instTree}
        </Box>,
      );
    }
  }

  if (primaryTree) {
    allTrees.unshift(
      <Box key={config!.instance} marginTop="4">
        {allTrees.length > 0 ? (
          <Heading size="md" marginBottom="2">
            {config!.instance}
          </Heading>
        ) : null}
        {primaryTree}
      </Box>,
    );
  }

  if (useMobileLayout) {
    const backup = backups.find((b) => b.flowID === selectedBackupId);
    return (
      <>
        <DialogRoot
          open={!!backup}
          onOpenChange={(e: { open: boolean }) => {
            if (!e.open) setSelectedBackupId(null);
          }}
          size="lg"
        >
          <DialogContent>
            <DialogCloseTrigger />
            <DialogBody>
              <BackupView backup={backup} />
            </DialogBody>
          </DialogContent>
        </DialogRoot>

        {allTrees}
      </>
    );
  }

  return (
    <Flex direction="column" gap="4" height="100%">
      <Splitter
        panels={[{ id: "tree", minSize: 20, maxSize: 70 }, { id: "view" }]}
      >
        <SplitterPanel id="tree">
          <Box overflowY="auto" height="100%">
            {allTrees}
          </Box>
        </SplitterPanel>
        <SplitterResizeTrigger id="tree:view" />
        <SplitterPanel id="view">
          <Box paddingLeft="2" height="100%" overflowY="auto">
            <BackupViewContainer>
              {selectedBackupId ? (
                <BackupView
                  backup={backups.find((b) => b.flowID === selectedBackupId)}
                />
              ) : null}
            </BackupViewContainer>
          </Box>
        </SplitterPanel>
      </Splitter>
    </Flex>
  );
};

const DisplayOperationTree = ({
  operations,
  isPlanView,
  onSelect,
}: {
  operations: FlowDisplayInfo[];
  isPlanView?: boolean;
  onSelect?: (flow: FlowDisplayInfo | null) => any;
}) => {
  const [treeCollection, setTreeCollection] =
    useState<TreeCollection<OpTreeNode> | null>(null);
  const [expandedValue, setExpandedValue] = useState<string[]>([]);
  const [selectedValue, setSelectedValue] = useState<string[]>([]);

  const buildTreeData = () => {
    const leafGroupFn = (op: FlowDisplayInfo) => op.flowID.toString(16);
    const leafFn = (groupKey: string, ops: FlowDisplayInfo[]) => {
      const b = ops[0];
      let iconColor = colorForStatus(b.status);
      let icon: React.ReactNode | null = <LuFileQuestion />;
      if (
        b.status === OperationStatus.STATUS_ERROR ||
        b.status === OperationStatus.STATUS_WARNING
      ) {
        icon = <Box color={iconColor}>!</Box>;
      } else {
        icon = <OperationIcon status={b.status} type={b.type} />;
      }

      return {
        id: groupKey,
        label: `${displayTypeToString(b.type)} ${formatTime(b.displayTime)}`,
        backup: b,
        icon,
      };
    };
    const leafSortFn = (op1: FlowDisplayInfo, op2: FlowDisplayInfo) =>
      op1.displayTime > op2.displayTime;

    const buildLevel = (
      ops: FlowDisplayInfo[],
      levels: any[],
      prefix: string,
    ): OpTreeNode[] => {
      if (levels.length === 0) {
        const groups = groupBy(ops, leafGroupFn);
        const sortedKeys = Object.keys(groups).sort((a, b) =>
          leafSortFn(groups[a][0], groups[b][0]) ? -1 : 1,
        );
        return sortedKeys.map((key) =>
          leafFn(prefix + "\0" + key, groups[key]),
        );
      }

      const currentLevel = levels[0];
      const nextLevels = levels.slice(1);
      const groups = groupBy(ops, currentLevel.groupingFn);
      const sortedKeys = Object.keys(groups).sort((a, b) =>
        currentLevel.sortFn(groups[a][0], groups[b][0]) ? -1 : 1,
      );

      return sortedKeys.map((key) => {
        const groupOps = groups[key];
        const groupKey = prefix + "\0" + key;
        const exemplar = groupOps[0];
        const children = buildLevel(groupOps, nextLevels, groupKey);
        return {
          id: groupKey,
          label:
            currentLevel.titleFn(exemplar) +
            (groupOps.length > 1 ? ` (${groupOps.length})` : ""),
          children,
        };
      });
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

    const spec = isPlanView
      ? { levels: [monthLayer, dayLayer] }
      : { levels: [planLayer, monthLayer, dayLayer] };
    return buildLevel(operations, spec.levels, "");
  };

  useEffect(() => {
    const nodes = buildTreeData();
    const collection = createTreeCollection<OpTreeNode>({
      nodeToValue: (node: OpTreeNode) => node.id,
      nodeToString: (node: OpTreeNode) => node.label,
      rootNode: {
        id: "ROOT",
        label: "Root",
        children: nodes,
      },
    });
    setTreeCollection(collection);

    // Calculate initial expansion.
    const toExpand = new Set<string>();
    const findRecentOrInProgress = (nodes: OpTreeNode[]) => {
      let foundInProgress = false;
      for (const node of nodes) {
        let shouldExpandThis = false;
        if (node.backup) {
          if (
            node.backup.status === OperationStatus.STATUS_INPROGRESS ||
            node.backup.status === OperationStatus.STATUS_PENDING
          ) {
            shouldExpandThis = true;
            foundInProgress = true;
          }
        }

        if (node.children) {
          const childFoundInProgress = findRecentOrInProgress(node.children);
          if (childFoundInProgress) {
            shouldExpandThis = true;
            foundInProgress = true;
          }
        }

        if (shouldExpandThis) {
          toExpand.add(node.id);
        }
      }
      return foundInProgress;
    };

    findRecentOrInProgress(nodes);

    // Also expand the very first branch (most recent) if nothing else is expanded.
    if (toExpand.size === 0 && nodes.length > 0) {
      const expandFirst = (node: OpTreeNode) => {
        toExpand.add(node.id);
        if (node.children && node.children.length > 0) {
          expandFirst(node.children[0]);
        }
      };
      expandFirst(nodes[0]);
    }

    setExpandedValue(Array.from([...expandedValue, ...toExpand]));
  }, [operations, isPlanView]);

  if (!treeCollection) return <></>;

  return (
    <TreeViewRoot
      collection={treeCollection}
      expandedValue={expandedValue}
      selectedValue={selectedValue}
      onSelectionChange={(details: any) => {
        const values = details?.selectedValue ?? [];
        setSelectedValue(values);
        setExpandedValue((prev) => Array.from(new Set([...prev, ...values])));

        if (!details.selectedNodes || details.selectedNodes.length === 0)
          return;
        onSelect && onSelect(details.selectedNodes[0].backup);
      }}
    >
      <TreeViewTree>
        <TreeViewNode<OpTreeNode>
          render={({ node, nodeState }) =>
            nodeState.isBranch ? (
              <TreeViewBranchControl
                cursor="pointer"
                onClick={() => {
                  setExpandedValue((prev) =>
                    prev.includes(node.id)
                      ? prev.filter((id) => id !== node.id)
                      : [...prev, node.id],
                  );
                }}
              >
                <Box
                  transform={nodeState.expanded ? "rotate(90deg)" : undefined}
                  transition="transform 0.2s"
                  display="inline-flex"
                  alignItems="center"
                  justifyContent="center"
                  w="20px"
                  flexShrink={0}
                >
                  <LuChevronRight size="14px" />
                </Box>
                <TreeViewBranchTrigger>
                  <Box
                    display="inline-flex"
                    alignItems="center"
                    justifyContent="center"
                    w="24px"
                    flexShrink={0}
                  >
                    <LuFolder />
                  </Box>
                </TreeViewBranchTrigger>
                <TreeViewBranchText>{node.label}</TreeViewBranchText>
                <TreeViewBranchContent />
              </TreeViewBranchControl>
            ) : (
              <TreeViewItem>
                {/* Spacer to match chevron width */}
                <Box w="20px" flexShrink={0} />
                <Box
                  display="inline-flex"
                  alignItems="center"
                  justifyContent="center"
                  w="24px"
                  flexShrink={0}
                >
                  {node.icon ? node.icon : <LuFile />}
                </Box>
                <TreeViewItemText>
                  {node.backup ? (
                    <VStack align="start" gap="0">
                      <Text>
                        {displayTypeToString(node.backup.type)}{" "}
                        {formatTime(node.backup.displayTime)}
                      </Text>
                      {node.backup.subtitleComponents &&
                        node.backup.subtitleComponents.length > 0 && (
                          <Text
                            color="fg.muted"
                            fontSize="xs"
                            fontFamily="mono"
                          >
                            {node.backup.subtitleComponents.join(", ")}
                          </Text>
                        )}
                    </VStack>
                  ) : (
                    node.label
                  )}
                </TreeViewItemText>
              </TreeViewItem>
            )
          }
        />
      </TreeViewTree>
    </TreeViewRoot>
  );
};

const BackupViewContainer = ({ children }: { children: React.ReactNode }) => {
  return (
    <Box position="sticky" top="0" width="100%">
      {children}
    </Box>
  );
};

const BackupView = ({ backup }: { backup?: FlowDisplayInfo }) => {
  if (!backup) {
    return (
      <EmptyState.Root>
        <EmptyState.Title>Backup not found.</EmptyState.Title>
      </EmptyState.Root>
    );
  } else {
    const doDeleteSnapshot = async () => {
      try {
        await backrestService.forget(
          create(ForgetRequestSchema, {
            planId: backup.planID!,
            repoId: backup.repoID!,
            snapshotId: backup.snapshotID!,
          }),
        );
        alerts.success("Snapshot forgotten.");
      } catch (e: any) {
        alerts.error("Failed to forget snapshot: " + e);
      }
    };

    const snapshotInFlow = backup?.operations.find(
      (op) => op.op.case === "operationIndexSnapshot",
    );

    const deleteButton =
      snapshotInFlow && snapshotInFlow.snapshotId ? (
        <ConfirmButton
          variant="ghost"
          confirmTitle="Confirm forget?"
          confirmTimeout={2000}
          onClickAsync={doDeleteSnapshot}
          colorPalette="red"
        >
          Forget (Destructive)
        </ConfirmButton>
      ) : (
        <ConfirmButton
          variant="ghost"
          confirmTitle="Confirm clear?"
          onClickAsync={async () => {
            backrestService.clearHistory(
              create(ClearHistoryRequestSchema, {
                selector: {
                  flowId: backup.flowID,
                },
              }),
            );
          }}
        >
          Delete Event
        </ConfirmButton>
      );

    return (
      <Box width="100%">
        <Flex
          alignItems="center"
          direction="row"
          width="100%"
          height="60px"
          position="relative"
        >
          <Heading size="md">{formatTime(backup.displayTime)}</Heading>
          <Box position="absolute" right="20px">
            {backup.status !== OperationStatus.STATUS_PENDING &&
            backup.status !== OperationStatus.STATUS_INPROGRESS
              ? deleteButton
              : null}
          </Box>
        </Flex>
        <OperationListView
          key={backup.flowID}
          useOperations={backup.operations}
        />
      </Box>
    );
  }
};
