import React, { useEffect, useState } from "react";
import {
  Operation,
  OperationForget,
  OperationRestore,
  OperationStatus,
} from "../../../gen/ts/v1/operations_pb";
import { HeavyAccordion } from "../../components/common/HeavyAccordion";
import {
  Button,
  GridItem,
  Collapsible,
  Box,
  Flex,
  Text,
  Stack,
  SimpleGrid,
  Heading,
  Code,
  IconButton,
} from "@chakra-ui/react";
import {
  MenuRoot,
  MenuTrigger,
  MenuContent,
  MenuItem,
} from "../../components/ui/menu";
import { FiFileText, FiMoreVertical, FiTrash2, FiX } from "react-icons/fi";
import { ProgressCircle } from "../../components/ui/progress-circle";
import { ProgressBar, ProgressRoot } from "../../components/ui/progress";
import { toaster } from "../../components/ui/toaster";

import {
  BackupProgressEntry,
  ResticSnapshot,
  SnapshotSummary,
} from "../../../gen/ts/v1/restic_pb";
import { SnapshotBrowser } from "../repositories/SnapshotBrowser";
import {
  formatBytes,
  formatDuration,
  formatTime,
  normalizeSnapshotId,
} from "../../lib/formatting";
import { ClearHistoryRequestSchema } from "../../../gen/ts/v1/service_pb";
import { backrestService } from "../../api/client";
import { useShowModal } from "../../components/common/ModalManager";
import { alerts } from "../../components/common/Alerts";
import {
  displayTypeToString,
  getTypeForDisplay,
  nameForStatus,
  colorForStatus,
} from "../../api/flowDisplayAggregator";
import { OperationIcon } from "./OperationIcon";
import { LogView } from "../../components/common/LogView";

import { create } from "@bufbuild/protobuf";
import { OperationListView } from "./OperationListView";
import * as m from "../../paraglide/messages";
import { FormModal } from "../../components/common/FormModal";

const ConfirmMenuItem = ({
  onConfirm,
  confirmText,
  children,
  ...props
}: {
  onConfirm: () => void;
  confirmText: React.ReactNode;
  children: React.ReactNode;
} & React.ComponentProps<typeof MenuItem>) => {
  const [needsConfirm, setNeedsConfirm] = useState(false);

  return (
    <MenuItem
      {...props}
      closeOnSelect={needsConfirm}
      onMouseLeave={() => setNeedsConfirm(false)}
      onClick={(e) => {
        if (!needsConfirm) {
          e.preventDefault();
          setNeedsConfirm(true);
        } else {
          onConfirm();
        }
      }}
    >
      {needsConfirm ? confirmText : children}
    </MenuItem>
  );
};

export const OperationRow = ({
  operation,
  showPlan,
  hookOperations,
  showDelete,
}: React.PropsWithoutRef<{
  operation: Operation;
  alertApi?: any; // Toaster doesn't need passing, but keeping for compatibility for now
  showPlan?: boolean;
  hookOperations?: Operation[];
  showDelete?: boolean;
}>) => {
  const showModal = useShowModal();
  const displayType = getTypeForDisplay(operation);
  const setRefresh = useState(0)[1];


  useEffect(() => {
    if (operation.status === OperationStatus.STATUS_INPROGRESS) {
      const interval = setInterval(() => {
        setRefresh((x) => x + 1);
      }, 1000);
      return () => clearInterval(interval);
    }
  }, [operation.status]);

  const doDelete = async () => {
    try {
      await backrestService.clearHistory(
        create(ClearHistoryRequestSchema, {
          selector: {
            ids: [operation.id!],
          },
          onlyFailed: false,
        }),
      );
      alerts.success(m.op_row_deleted_success());
    } catch (e: any) {
      alerts.error(m.op_row_deleted_error() + e.message);
    }
  };

  const doCancel = async () => {
    try {
      await backrestService.cancel({ value: operation.id! });
      alerts.success(m.op_row_cancel_success());
    } catch (e: any) {
      alerts.error(m.op_row_cancel_error() + e.message);
    }
  };

  const doShowLogs = () => {
    showModal(
      <FormModal
        size="large"
        title={m.op_row_logs_title({
          name: opName,
          time: formatTime(Number(operation.unixTimeStartMs)),
        })}
        isOpen={true}
        onClose={() => {
          showModal(null);
        }}
        footer={null}
      >
        <LogView logref={operation.logref!} />
      </FormModal>,
    );
  };

  let details: string = "";
  if (operation.status !== OperationStatus.STATUS_SUCCESS) {
    details = nameForStatus(operation.status);
  }
  if (operation.unixTimeEndMs - operation.unixTimeStartMs > 100) {
    details +=
      " in " +
      formatDuration(
        Number(operation.unixTimeEndMs - operation.unixTimeStartMs),
      );
  }

  const opName = displayTypeToString(getTypeForDisplay(operation));

  const title: React.ReactNode[] = [
    <div key="title">
      {showPlan
        ? operation.instanceId + " - " + operation.planId + " - "
        : undefined}{" "}
      {formatTime(Number(operation.unixTimeStartMs))} - {opName}{" "}
      <span className="backrest operation-details">{details}</span>
    </div>,
  ];

  // --- Menu Items Logic ---
  const menuItems: React.ReactNode[] = [];

  if (operation.logref) {
    menuItems.push(
      <MenuItem key="logs" value="logs" onClick={doShowLogs}>
        <FiFileText /> {m.op_row_view_logs()}
      </MenuItem>,
    );
  }

  if (
    operation.status === OperationStatus.STATUS_INPROGRESS ||
    operation.status === OperationStatus.STATUS_PENDING
  ) {
    menuItems.push(
      <ConfirmMenuItem
        key="cancel"
        value="cancel"
        onConfirm={doCancel}
        confirmText={m.op_row_confirm_cancel()}
        color="fg.error"
      >
        {m.op_row_cancel_op()}
      </ConfirmMenuItem>,
    );
  } else if (showDelete) {
    menuItems.push(
      <ConfirmMenuItem
        key="delete"
        value="delete"
        onConfirm={doDelete}
        confirmText={m.op_row_confirm_delete()}
        color="fg.error"
      >
        <FiTrash2 /> {m.op_row_delete()}
      </ConfirmMenuItem>,
    );
  }

  let displayMessage = operation.displayMessage;

  const bodyItems: { key: string; label: string; children: React.ReactNode }[] =
    [];
  const expandedBodyItems: string[] = [];

  if (operation.op.case === "operationBackup") {
    if (operation.status === OperationStatus.STATUS_INPROGRESS) {
      expandedBodyItems.push("details");
    }
    const backupOp = operation.op.value;
    bodyItems.push({
      key: "details",
      label: m.op_row_backup_details(),
      children: <BackupOperationStatus status={backupOp.lastStatus} />,
    });

    if (backupOp.errors.length > 0) {
      bodyItems.push({
        key: "errors",
        label: m.op_row_item_errors(),
        children: (
          <Table.Root size="sm" variant="outline">
            <Table.Header>
              <Table.Row>
                <Table.ColumnHeader>
                  {m.op_row_error_message()}
                </Table.ColumnHeader>
              </Table.Row>
            </Table.Header>
            <Table.Body>
              {backupOp.errors.map((e, idx) => (
                <Table.Row key={idx}>
                  <Table.Cell verticalAlign="top">
                    {e.message || e.item}
                  </Table.Cell>
                </Table.Row>
              ))}
            </Table.Body>
          </Table.Root>
        ),
      });
    }
  } else if (operation.op.case === "operationIndexSnapshot") {
    expandedBodyItems.push("details");
    const snapshotOp = operation.op.value;
    bodyItems.push({
      key: "details",
      label: m.op_row_details(),
      children: <SnapshotDetails snapshot={snapshotOp.snapshot!} />,
    });
    bodyItems.push({
      key: "browser",
      label: m.op_row_snapshot_browser(),
      
      children: (
        <SnapshotBrowser
          snapshotId={snapshotOp.snapshot!.id}
          snapshotOpId={operation.id}
          repoId={operation.repoId}
          repoGuid={operation.repoGuid}
          planId={operation.planId}
        />
      ),
    });
  } else if (operation.op.case === "operationForget") {
    const forgetOp = operation.op.value;
    expandedBodyItems.push("forgot");
    bodyItems.push({
      key: "forgot",
      label: m.op_row_removed_snapshots({
        count: forgetOp.forget?.length || 0,
      }),
      children: <ForgetOperationDetails forgetOp={forgetOp} />,
    });
  } else if (operation.op.case === "operationPrune") {
    const prune = operation.op.value;
    expandedBodyItems.push("prune");
    bodyItems.push({
      key: "prune",
      label: m.op_row_prune_output(),
      children: prune.outputLogref ? (
        <LogView logref={prune.outputLogref} />
      ) : (
        <pre>{prune.output}</pre>
      ),
    });
  } else if (operation.op.case === "operationCheck") {
    const check = operation.op.value;
    expandedBodyItems.push("check");
    bodyItems.push({
      key: "check",
      label: m.op_row_check_output(),
      children: check.outputLogref ? (
        <LogView logref={check.outputLogref} />
      ) : (
        <pre>{check.output}</pre>
      ),
    });
  } else if (operation.op.case === "operationRunCommand") {
    const run = operation.op.value;
    if (run.outputSizeBytes < 64 * 1024) {
      expandedBodyItems.push("run");
    }
    bodyItems.push({
      key: "run",
      label:
        m.op_row_command_output() +
        (run.outputSizeBytes > 0
          ? ` (${formatBytes(Number(run.outputSizeBytes))})`
          : ""),
      children: (
        <>
          <LogView logref={run.outputLogref} />
        </>
      ),
    });
  } else if (operation.op.case === "operationRestore") {
    expandedBodyItems.push("restore");
    bodyItems.push({
      key: "restore",
      label: m.op_row_restore_details(),
      children: <RestoreOperationStatus operation={operation} />,
    });
  } else if (operation.op.case === "operationRunHook") {
    const hook = operation.op.value;
    if (operation.logref) {
      if (operation.status === OperationStatus.STATUS_INPROGRESS) {
        expandedBodyItems.push("logref");
      }
      bodyItems.push({
        key: "logref",
        label: m.op_row_hook_output(),
        children: <LogView logref={operation.logref} />,
      });
    }
  }

  if (hookOperations) {
    bodyItems.push({
      key: "hookOperations",
      label: m.op_row_hooks_triggered(),
      children: (
        <OperationListView
          useOperations={hookOperations}
          displayHooksInline={true}
        />
      ),
    });

    for (const op of hookOperations) {
      if (op.status !== OperationStatus.STATUS_SUCCESS) {
        expandedBodyItems.push("hookOperations");
        break;
      }
    }
  }



  return (
    <Box
      className="backrest visible-on-hover"
      mb={2}
      borderWidth="1px"
      borderRadius="md"
      bg="bg.panel"
      _hover={{ borderColor: "border.emphasized" }}
    >
      <Box p={3}>
        <Flex align="center" gap={3}>
          <OperationIcon type={displayType} status={operation.status} />
          <Box flex={1}>
            <Flex wrap="wrap" align="baseline" gap={2}>
              {title}
            </Flex>
          </Box>
          {menuItems.length > 0 && (
            <MenuRoot>
              <MenuTrigger asChild>
                <IconButton variant="ghost" size="sm" aria-label="Actions">
                  <FiMoreVertical />
                </IconButton>
              </MenuTrigger>
              <MenuContent>{menuItems}</MenuContent>
            </MenuRoot>
          )}
        </Flex>

        {operation.displayMessage && (
          <Box mt={2}>
            <Box
              pl={3}
              borderLeftWidth="4px"
              borderLeftColor={colorForStatus(operation.status)}
              py={1}
            >
              <Text fontSize="xs" whiteSpace="pre-wrap">
                {operation.status !== OperationStatus.STATUS_SUCCESS && (
                  <Text as="span" fontWeight="bold">
                    {nameForStatus(operation.status)}:{" "}
                  </Text>
                )}
                {operation.displayMessage}
              </Text>
            </Box>
          </Box>
        )}

        {bodyItems.length > 0 && (
          <Box mt={2} pl={2}>
            <HeavyAccordion
              items={bodyItems}
              defaultExpanded={expandedBodyItems}
            />
          </Box>
        )}
      </Box>
    </Box>
  );
};

const SnapshotDetails = ({ snapshot }: { snapshot: ResticSnapshot }) => {
  const summary = snapshot.summary;

  return (
    <>
      <Text>
        <Text as="span" fontWeight="bold">
          {m.op_row_snapshot_id()}
        </Text>
        {normalizeSnapshotId(snapshot.id!)}
      </Text>
      <SimpleGrid columns={3} gap={4} mt={2}>
        <GridItem colSpan={1}>
          <Text fontWeight="bold">{m.op_row_user_host()}</Text>
          <Text color="fg.muted">
            {snapshot.username}@{snapshot.hostname}
          </Text>
        </GridItem>
        <GridItem colSpan={2}>
          <Text fontWeight="bold">{m.op_row_tags()}</Text>
          <Text color="fg.muted">{snapshot.tags.join(", ")}</Text>
        </GridItem>
      </SimpleGrid>

      {summary && (
        <>
          <SimpleGrid columns={3} gap={4} mt={2}>
            <Box>
              <Text fontWeight="bold">{m.op_row_files_added()}</Text>
              <Text color="fg.muted">{summary.filesNew.toLocaleString()}</Text>
            </Box>
            <Box>
              <Text fontWeight="bold">{m.op_row_files_changed()}</Text>
              <Text color="fg.muted">
                {summary.filesChanged.toLocaleString()}
              </Text>
            </Box>
            <Box>
              <Text fontWeight="bold">{m.op_row_files_unmodified()}</Text>
              <Text color="fg.muted">
                {summary.filesUnmodified.toLocaleString()}
              </Text>
            </Box>
          </SimpleGrid>
          <SimpleGrid columns={3} gap={4}>
            <Box>
              <Text fontWeight="bold">{m.op_row_bytes_added()}</Text>
              <Text color="fg.muted">
                {formatBytes(Number(summary.dataAdded))}
              </Text>
            </Box>
            <Box>
              <Text fontWeight="bold">{m.op_row_total_bytes()}</Text>
              <Text color="fg.muted">
                {formatBytes(Number(summary.totalBytesProcessed))}
              </Text>
            </Box>
            <Box>
              <Text fontWeight="bold">{m.op_row_total_files()}</Text>
              <Text color="fg.muted">
                {summary.totalFilesProcessed.toLocaleString()}
              </Text>
            </Box>
          </SimpleGrid>
        </>
      )}
    </>
  );
};

const RestoreOperationStatus = ({ operation }: { operation: Operation }) => {
  const restoreOp = operation.op.value as OperationRestore;
  const isDone = restoreOp.lastStatus?.messageType === "summary";
  const progress = restoreOp.lastStatus?.percentDone || 0;
  const lastStatus = restoreOp.lastStatus;

  return (
    <>
      <Stack gap={4} mb={4}>
        <Box>
          <Text fontWeight="bold" fontSize="xs" color="fg.muted" mb={1}>
            {m.op_row_restore_source()}
          </Text>
          <Code
            p={2}
            borderRadius="md"
            width="full"
            display="block"
            whiteSpace="pre-wrap"
          >
            {restoreOp.path}
          </Code>
        </Box>
        <Box>
          <Text fontWeight="bold" fontSize="xs" color="fg.muted" mb={1}>
            {m.op_row_restore_target()}
          </Text>
          <Code
            p={2}
            borderRadius="md"
            width="full"
            display="block"
            whiteSpace="pre-wrap"
          >
            {restoreOp.target}
          </Code>
        </Box>
      </Stack>

      {!isDone ? (
        <ProgressRoot value={progress * 100} max={100} size="sm" mb={4}>
          <ProgressBar />
        </ProgressRoot>
      ) : null}

      {operation.status == OperationStatus.STATUS_SUCCESS ? (
        <Box mb={4}>
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              backrestService
                .getDownloadURL({ opId: operation.id!, filePath: "" })
                .then((resp) => {
                  window.open(resp.value, "_blank");
                })
                .catch((e) => {
                  alerts.error(m.op_row_fetch_download_error() + e.message);
                });
            }}
          >
            {m.op_row_download_files()}
          </Button>
        </Box>
      ) : null}

      <SimpleGrid columns={2} gap={4}>
        <Box>
          <Text fontWeight="bold">{m.op_row_snapshot_id()}</Text>
          <Text fontFamily="mono">
            {normalizeSnapshotId(operation.snapshotId!)}
          </Text>
        </Box>
        {lastStatus && (
          <>
            <Box>
              <Text fontWeight="bold">{m.op_row_bytes_done_total()}</Text>
              <Text color="fg.muted">
                {formatBytes(Number(lastStatus.bytesRestored))}/
                {formatBytes(Number(lastStatus.totalBytes))}
              </Text>
            </Box>
            <Box>
              <Text fontWeight="bold">{m.op_row_files_done_total()}</Text>
              <Text color="fg.muted">
                {Number(lastStatus.filesRestored)}/
                {Number(lastStatus.totalFiles)}
              </Text>
            </Box>
          </>
        )}
      </SimpleGrid>
    </>
  );
};

const BackupOperationStatus = ({
  status,
}: {
  status?: BackupProgressEntry;
}) => {
  if (!status) {
    return <>{m.op_row_no_status()}</>;
  }

  if (status.entry.case === "status") {
    const st = status.entry.value;
    const progress =
      Math.round(
        (Number(st.bytesDone) / Math.max(Number(st.totalBytes), 1)) * 1000,
      ) / 10;
    return (
      <>
        <ProgressRoot value={progress} max={100} size="sm" mb={4}>
          <ProgressBar />
        </ProgressRoot>
        <SimpleGrid columns={2} gap={4}>
          <Box>
            <Text fontWeight="bold">{m.op_row_bytes_done_total()}</Text>
            <Text color="fg.muted">
              {formatBytes(Number(st.bytesDone))} /{" "}
              {formatBytes(Number(st.totalBytes))}
            </Text>
          </Box>
          <Box>
            <Text fontWeight="bold">{m.op_row_files_done_total()}</Text>
            <Text color="fg.muted">
              {Number(st.filesDone).toLocaleString()} /{" "}
              {Number(st.totalFiles).toLocaleString()}
            </Text>
          </Box>
        </SimpleGrid>
        {st.currentFile && st.currentFile.length > 0 && (
          <Box mt={2}>
            <Text fontWeight="bold">{m.op_row_current_files()}</Text>
            <Code
              display="block"
              mt={1}
              p={2}
              borderRadius="md"
              fontSize="xs"
              whiteSpace="pre"
            >
              {st.currentFile.join("\n")}
            </Code>
          </Box>
        )}
      </>
    );
  } else if (status.entry.case === "summary") {
    const sum = status.entry.value;
    return (
      <>
        <Text>
          <Text as="span" fontWeight="bold">
            {m.op_row_snapshot_id()}
          </Text>
          {sum.snapshotId !== ""
            ? normalizeSnapshotId(sum.snapshotId!)
            : m.op_row_no_snapshot()}
        </Text>
        <SimpleGrid columns={{ base: 1, md: 3 }} gap={4} mt={2}>
          <Box>
            <Text fontWeight="bold">{m.op_row_files_added()}</Text>
            <Text color="fg.muted">
              {Number(sum.filesNew).toLocaleString()}
            </Text>
          </Box>
          <Box>
            <Text fontWeight="bold">{m.op_row_files_changed()}</Text>
            <Text color="fg.muted">
              {Number(sum.filesChanged).toLocaleString()}
            </Text>
          </Box>
          <Box>
            <Text fontWeight="bold">{m.op_row_files_unmodified()}</Text>
            <Text color="fg.muted">
              {Number(sum.filesUnmodified).toLocaleString()}
            </Text>
          </Box>
        </SimpleGrid>
        <SimpleGrid columns={{ base: 1, md: 3 }} gap={4} mt={2}>
          <Box>
            <Text fontWeight="bold">{m.op_row_bytes_added()}</Text>
            <Text color="fg.muted">{formatBytes(Number(sum.dataAdded))}</Text>
          </Box>
          <Box>
            <Text fontWeight="bold">{m.op_row_total_bytes()}</Text>
            <Text color="fg.muted">
              {formatBytes(Number(sum.totalBytesProcessed))}
            </Text>
          </Box>
          <Box>
            <Text fontWeight="bold">{m.op_row_total_files()}</Text>
            <Text color="fg.muted">
              {Number(sum.totalFilesProcessed).toLocaleString()}
            </Text>
          </Box>
        </SimpleGrid>
      </>
    );
  } else {
    console.error("GOT UNEXPECTED STATUS: ", status);
    return <>{m.op_row_unexpected_status() + JSON.stringify(status)}</>;
  }
};

import {
  Table,
  TableBody,
  TableCell,
  TableColumnHeader,
  TableHeader,
  TableRow,
} from "@chakra-ui/react";

const ForgetOperationDetails = ({
  forgetOp,
}: {
  forgetOp: OperationForget;
}) => {
  const removedSnapshots = forgetOp.forget || [];

  if (removedSnapshots.length === 0) {
    return (
      <Text color="fg.muted" fontStyle="italic">
        {m.op_row_removed_none()}
      </Text>
    );
  }

  return (
    <>
      <Table.Root size="sm" variant="outline">
        <Table.Header>
          <Table.Row>
            <Table.ColumnHeader>{m.op_row_removed_id_col()}</Table.ColumnHeader>
            <Table.ColumnHeader>
              {m.op_row_removed_time_col()}
            </Table.ColumnHeader>
          </Table.Row>
        </Table.Header>
        <Table.Body>
          {removedSnapshots.map((f) => (
            <Table.Row key={f.id}>
              <Table.Cell fontFamily="mono">
                {normalizeSnapshotId(f.id!)}
              </Table.Cell>
              <Table.Cell>{formatTime(Number(f.unixTimeMs))}</Table.Cell>
            </Table.Row>
          ))}
        </Table.Body>
      </Table.Root>
    </>
  );
};
