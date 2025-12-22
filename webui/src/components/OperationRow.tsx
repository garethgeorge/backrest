import React, { useEffect, useState } from "react";
import {
  Operation,
  OperationForget,
  OperationRestore,
  OperationStatus,
} from "../../gen/ts/v1/operations_pb";
import {
  Button,
  Col,
  Collapse,
  List,
  Modal,
  Progress,
  Row,
  Typography,
} from "antd";
import type { ItemType } from "rc-collapse/es/interface";
import {
  BackupProgressEntry,
  ResticSnapshot,
  SnapshotSummary,
} from "../../gen/ts/v1/restic_pb";
import { SnapshotBrowser } from "./SnapshotBrowser";
import {
  formatBytes,
  formatDuration,
  formatTime,
  normalizeSnapshotId,
} from "../lib/formatting";
import { ClearHistoryRequestSchema } from "../../gen/ts/v1/service_pb";
import { MessageInstance } from "antd/es/message/interface";
import { backrestService } from "../api";
import { useShowModal } from "./ModalManager";
import { useAlertApi } from "./Alerts";
import {
  displayTypeToString,
  getTypeForDisplay,
  nameForStatus,
} from "../state/flowdisplayaggregator";
import { OperationIcon } from "./OperationIcon";
import { LogView } from "./LogView";
import { ConfirmButton } from "./SpinButton";
import { create } from "@bufbuild/protobuf";
import { OperationListView } from "./OperationListView";
import * as m from "../paraglide/messages";

export const OperationRow = ({
  operation,
  alertApi,
  showPlan,
  hookOperations,
  showDelete,
}: React.PropsWithoutRef<{
  operation: Operation;
  alertApi?: MessageInstance;
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
        })
      );
      alertApi?.success(m.op_row_deleted_success());
    } catch (e: any) {
      alertApi?.error(m.op_row_deleted_error() + e.message);
    }
  };

  const doCancel = async () => {
    try {
      await backrestService.cancel({ value: operation.id! });
      alertApi?.success(m.op_row_cancel_success());
    } catch (e: any) {
      alertApi?.error(m.op_row_cancel_error() + e.message);
    }
  };

  const doShowLogs = () => {
    showModal(
      <Modal
        width="70%"
        title={m.op_row_logs_title({
          name: opName,
          time: formatTime(Number(operation.unixTimeStartMs)),
        })}
        open={true}
        footer={null}
        onCancel={() => {
          showModal(null);
        }}
      >
        <LogView logref={operation.logref!} />
      </Modal>
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
        Number(operation.unixTimeEndMs - operation.unixTimeStartMs)
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

  if (operation.logref) {
    title.push(
      <Button
        key="logs"
        type="link"
        size="small"
        className="backrest operation-details"
        onClick={doShowLogs}
      >
        {m.op_row_view_logs()}
      </Button>
    );
  }

  if (
    operation.status === OperationStatus.STATUS_INPROGRESS ||
    operation.status === OperationStatus.STATUS_PENDING
  ) {
    title.push(
      <ConfirmButton
        key="cancel"
        type="link"
        size="small"
        className="backrest operation-details"
        confirmTitle={m.op_row_confirm_cancel()}
        onClickAsync={doCancel}
      >
        {m.op_row_cancel_op()}
      </ConfirmButton>
    );
  } else if (showDelete) {
    title.push(
      <ConfirmButton
        key="delete"
        type="link"
        size="small"
        className="backrest operation-details hidden-child"
        confirmTitle={m.op_row_confirm_delete()}
        onClickAsync={doDelete}
      >
        {m.op_row_delete()}
      </ConfirmButton>
    );
  }

  let displayMessage = operation.displayMessage;

  const bodyItems: ItemType[] = [];
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
          <pre>
            {backupOp.errors
              .map((e) => m.op_row_error_on_item({ item: e.item }))
              .join("\n")}
          </pre>
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
    <div className="backrest visible-on-hover">
      <List.Item key={operation.id}>
        <List.Item.Meta
          title={
            <div style={{ display: "flex", flexDirection: "row" }}>{title}</div>
          }
          avatar={
            <OperationIcon type={displayType} status={operation.status} />
          }
          description={
            <div className="backrest" style={{ width: "100%", height: "100%" }}>
              {operation.displayMessage && (
                <div key="message">
                  <pre>
                    {operation.status !== OperationStatus.STATUS_SUCCESS &&
                      nameForStatus(operation.status) + ": "}
                    {displayMessage}
                  </pre>
                </div>
              )}
              <Collapse
                size="small"
                destroyOnHidden={true}
                items={bodyItems}
                defaultActiveKey={expandedBodyItems}
              />
            </div>
          }
        />
      </List.Item>
    </div>
  );
};

const SnapshotDetails = ({ snapshot }: { snapshot: ResticSnapshot }) => {
  const summary = snapshot.summary;

  return (
    <>
      <Typography.Text>
        <Typography.Text strong>{m.op_row_snapshot_id()}</Typography.Text>
        {normalizeSnapshotId(snapshot.id!)}
      </Typography.Text>
      <Row gutter={[16, 8]} style={{ marginTop: 8 }}>
        <Col span={8}>
          <Typography.Text strong>{m.op_row_user_host()}</Typography.Text>
          <br />
          <Typography.Text type="secondary">
            {snapshot.username}@{snapshot.hostname}
          </Typography.Text>
        </Col>
        <Col span={12}>
          <Typography.Text strong>{m.op_row_tags()}</Typography.Text>
          <br />
          <Typography.Text type="secondary">
            {snapshot.tags.join(", ")}
          </Typography.Text>
        </Col>
      </Row>

      {summary && (
        <>
          <Row gutter={[16, 8]} style={{ marginTop: 8 }}>
            <Col span={8}>
              <Typography.Text strong>{m.op_row_files_added()}</Typography.Text>
              <br />
              <Typography.Text type="secondary">
                {summary.filesNew.toLocaleString()}
              </Typography.Text>
            </Col>
            <Col span={8}>
              <Typography.Text strong>{m.op_row_files_changed()}</Typography.Text>
              <br />
              <Typography.Text type="secondary">
                {summary.filesChanged.toLocaleString()}
              </Typography.Text>
            </Col>
            <Col span={8}>
              <Typography.Text strong>
                {m.op_row_files_unmodified()}
              </Typography.Text>
              <br />
              <Typography.Text type="secondary">
                {summary.filesUnmodified.toLocaleString()}
              </Typography.Text>
            </Col>
          </Row>
          <Row gutter={[16, 8]}>
            <Col span={8}>
              <Typography.Text strong>{m.op_row_bytes_added()}</Typography.Text>
              <br />
              <Typography.Text type="secondary">
                {formatBytes(Number(summary.dataAdded))}
              </Typography.Text>
            </Col>
            <Col span={8}>
              <Typography.Text strong>{m.op_row_total_bytes()}</Typography.Text>
              <br />
              <Typography.Text type="secondary">
                {formatBytes(Number(summary.totalBytesProcessed))}
              </Typography.Text>
            </Col>
            <Col span={8}>
              <Typography.Text strong>{m.op_row_total_files()}</Typography.Text>
              <br />
              <Typography.Text type="secondary">
                {summary.totalFilesProcessed.toLocaleString()}
              </Typography.Text>
            </Col>
          </Row>
        </>
      )}
    </>
  );
};

const RestoreOperationStatus = ({ operation }: { operation: Operation }) => {
  const restoreOp = operation.op.value as OperationRestore;
  const isDone = restoreOp.lastStatus?.messageType === "summary";
  const progress = restoreOp.lastStatus?.percentDone || 0;
  const alertApi = useAlertApi();
  const lastStatus = restoreOp.lastStatus;

  return (
    <>
      {m.op_row_restore_desc({
        path: restoreOp.path,
        target: restoreOp.target,
      })}
      {!isDone ? (
        <Progress percent={Math.round(progress * 1000) / 10} status="active" />
      ) : null}
      {operation.status == OperationStatus.STATUS_SUCCESS ? (
        <>
          <Button
            type="link"
            onClick={() => {
              backrestService
                .getDownloadURL({ opId: operation.id!, filePath: "" })
                .then((resp) => {
                  window.open(resp.value, "_blank");
                })
                .catch((e) => {
                  alertApi?.error(
                    m.op_row_fetch_download_error() + e.message
                  );
                });
            }}
          >
            {m.op_row_download_files()}
          </Button>
        </>
      ) : null}
      <br />
      {m.op_row_restored_snapshot_id({
        id: normalizeSnapshotId(operation.snapshotId!),
      })}
      {lastStatus && (
        <Row gutter={16}>
          <Col span={12}>
            <Typography.Text strong>
              {m.op_row_bytes_done_total()}
            </Typography.Text>
            <br />
            {formatBytes(Number(lastStatus.bytesRestored))}/
            {formatBytes(Number(lastStatus.totalBytes))}
          </Col>
          <Col span={12}>
            <Typography.Text strong>
              {m.op_row_files_done_total()}
            </Typography.Text>
            <br />
            {Number(lastStatus.filesRestored)}/{Number(lastStatus.totalFiles)}
          </Col>
        </Row>
      )}
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
        (Number(st.bytesDone) / Math.max(Number(st.totalBytes), 1)) * 1000
      ) / 10;
    return (
      <>
        <Progress percent={progress} status="active" />
        <br />
        <Row gutter={[16, 8]}>
          <Col span={12}>
            <Typography.Text strong>
              {m.op_row_bytes_done_total()}
            </Typography.Text>
            <br />
            <Typography.Text type="secondary">
              {formatBytes(Number(st.bytesDone))} /{" "}
              {formatBytes(Number(st.totalBytes))}
            </Typography.Text>
          </Col>
          <Col span={12}>
            <Typography.Text strong>
              {m.op_row_files_done_total()}
            </Typography.Text>
            <br />
            <Typography.Text type="secondary">
              {Number(st.filesDone).toLocaleString()} /{" "}
              {Number(st.totalFiles).toLocaleString()}
            </Typography.Text>
          </Col>
        </Row>
        {st.currentFile && st.currentFile.length > 0 && (
          <div style={{ marginTop: 8 }}>
            <Typography.Text strong>{m.op_row_current_files()}</Typography.Text>
            <pre
              style={{
                marginTop: 4,
                padding: 8,
                borderRadius: 4,
                borderColor: "#d9d9d9",
                fontSize: "0.85em",
              }}
            >
              {st.currentFile.join("\n")}
            </pre>
          </div>
        )}
      </>
    );
  } else if (status.entry.case === "summary") {
    const sum = status.entry.value;
    return (
      <>
        <Typography.Text>
          <Typography.Text strong>{m.op_row_snapshot_id()}</Typography.Text>
          {sum.snapshotId !== ""
            ? normalizeSnapshotId(sum.snapshotId!)
            : m.op_row_no_snapshot()}
        </Typography.Text>
        <Row gutter={[16, 8]}>
          <Col span={8}>
            <Typography.Text strong>{m.op_row_files_added()}</Typography.Text>
            <br />
            <Typography.Text type="secondary">
              {sum.filesNew.toString()}
            </Typography.Text>
          </Col>
          <Col span={8}>
            <Typography.Text strong>{m.op_row_files_changed()}</Typography.Text>
            <br />
            <Typography.Text type="secondary">
              {sum.filesChanged.toString()}
            </Typography.Text>
          </Col>
          <Col span={8}>
            <Typography.Text strong>
              {m.op_row_files_unmodified()}
            </Typography.Text>
            <br />
            <Typography.Text type="secondary">
              {sum.filesUnmodified.toString()}
            </Typography.Text>
          </Col>
        </Row>
        <Row gutter={[16, 8]}>
          <Col span={8}>
            <Typography.Text strong>{m.op_row_bytes_added()}</Typography.Text>
            <br />
            <Typography.Text type="secondary">
              {formatBytes(Number(sum.dataAdded))}
            </Typography.Text>
          </Col>
          <Col span={8}>
            <Typography.Text strong>{m.op_row_total_bytes()}</Typography.Text>
            <br />
            <Typography.Text type="secondary">
              {formatBytes(Number(sum.totalBytesProcessed))}
            </Typography.Text>
          </Col>
        </Row>
      </>
    );
  } else {
    console.error("GOT UNEXPECTED STATUS: ", status);
    return (
      <>
        {m.op_row_unexpected_status() + JSON.stringify(status)}
      </>
    );
  }
};

const ForgetOperationDetails = ({
  forgetOp,
}: {
  forgetOp: OperationForget;
}) => {
  const policy = forgetOp.policy! || {};
  const policyDesc = [];
  if (policy.policy) {
    if (policy.policy.case === "policyKeepAll") {
      policyDesc.push(m.op_row_policy_keep_all());
    } else if (policy.policy.case === "policyKeepLastN") {
      policyDesc.push(
        m.op_row_policy_keep_last_n({ value: policy.policy.value })
      );
    } else if (policy.policy.case == "policyTimeBucketed") {
      const val = policy.policy.value;
      if (val.hourly) {
        policyDesc.push(m.op_row_policy_hourly({ count: val.hourly }));
      }
      if (val.daily) {
        policyDesc.push(m.op_row_policy_daily({ count: val.daily }));
      }
      if (val.weekly) {
        policyDesc.push(m.op_row_policy_weekly({ count: val.weekly }));
      }
      if (val.monthly) {
        policyDesc.push(m.op_row_policy_monthly({ count: val.monthly }));
      }
      if (val.yearly) {
        policyDesc.push(m.op_row_policy_yearly({ count: val.yearly }));
      }
      if (val.keepLastN) {
        policyDesc.push(
          m.op_row_policy_keep_latest({ count: val.keepLastN })
        );
      }
    }
  }

  return (
    <>
      Removed snapshots:
      <pre>
        {forgetOp.forget?.map((f) => (
          <div key={f.id}>
            {m.op_row_removed_snapshot_desc({
              id: normalizeSnapshotId(f.id!),
              time: formatTime(Number(f.unixTimeMs)),
            })}{" "}
            <br />
          </div>
        ))}
      </pre>
      {m.op_row_policy_header()}
      <ul>
        {policyDesc.map((desc, idx) => (
          <li key={idx}>{desc}</li>
        ))}
      </ul>
    </>
  );
};
