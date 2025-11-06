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
import _ from "lodash";
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
      alertApi?.success("Deleted operation");
    } catch (e: any) {
      alertApi?.error("Failed to delete operation: " + e.message);
    }
  };

  const doCancel = async () => {
    try {
      await backrestService.cancel({ value: operation.id! });
      alertApi?.success("Requested to cancel operation");
    } catch (e: any) {
      alertApi?.error("Failed to cancel operation: " + e.message);
    }
  };

  const doShowLogs = () => {
    showModal(
      <Modal
        width="70%"
        title={
          "Logs for operation " +
          opName +
          " at " +
          formatTime(Number(operation.unixTimeStartMs))
        }
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
        [View Logs]
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
        confirmTitle="[Confirm Cancel?]"
        onClickAsync={doCancel}
      >
        [Cancel Operation]
      </ConfirmButton>
    );
  } else if (showDelete) {
    title.push(
      <ConfirmButton
        key="delete"
        type="link"
        size="small"
        className="backrest operation-details hidden-child"
        confirmTitle="[Confirm Delete?]"
        onClickAsync={doDelete}
      >
        [Delete]
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
      label: "Backup Details",
      children: <BackupOperationStatus status={backupOp.lastStatus} />,
    });

    if (backupOp.errors.length > 0) {
      bodyItems.push({
        key: "errors",
        label: "Item Errors",
        children: (
          <pre>
            {backupOp.errors.map((e) => "Error on item: " + e.item).join("\n")}
          </pre>
        ),
      });
    }
  } else if (operation.op.case === "operationIndexSnapshot") {
    expandedBodyItems.push("details");
    const snapshotOp = operation.op.value;
    bodyItems.push({
      key: "details",
      label: "Details",
      children: <SnapshotDetails snapshot={snapshotOp.snapshot!} />,
    });
    bodyItems.push({
      key: "browser",
      label: "Snapshot Browser",
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
      label: "Removed " + forgetOp.forget?.length + " Snapshots",
      children: <ForgetOperationDetails forgetOp={forgetOp} />,
    });
  } else if (operation.op.case === "operationPrune") {
    const prune = operation.op.value;
    expandedBodyItems.push("prune");
    bodyItems.push({
      key: "prune",
      label: "Prune Output",
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
      label: "Check Output",
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
        "Command Output" +
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
      label: "Restore Details",
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
        label: "Hook Output",
        children: <LogView logref={operation.logref} />,
      });
    }
  }

  if (hookOperations) {
    bodyItems.push({
      key: "hookOperations",
      label: "Hooks Triggered",
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
        <Typography.Text strong>Snapshot ID: </Typography.Text>
        {normalizeSnapshotId(snapshot.id!)}
      </Typography.Text>
      <Row gutter={[16, 8]} style={{ marginTop: 8 }}>
        <Col span={8}>
          <Typography.Text strong>User and Host</Typography.Text>
          <br />
          <Typography.Text type="secondary">
            {snapshot.username}@{snapshot.hostname}
          </Typography.Text>
        </Col>
        <Col span={12}>
          <Typography.Text strong>Tags</Typography.Text>
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
              <Typography.Text strong>Files Added</Typography.Text>
              <br />
              <Typography.Text type="secondary">
                {summary.filesNew.toLocaleString()}
              </Typography.Text>
            </Col>
            <Col span={8}>
              <Typography.Text strong>Files Changed</Typography.Text>
              <br />
              <Typography.Text type="secondary">
                {summary.filesChanged.toLocaleString()}
              </Typography.Text>
            </Col>
            <Col span={8}>
              <Typography.Text strong>Files Unmodified</Typography.Text>
              <br />
              <Typography.Text type="secondary">
                {summary.filesUnmodified.toLocaleString()}
              </Typography.Text>
            </Col>
          </Row>
          <Row gutter={[16, 8]}>
            <Col span={8}>
              <Typography.Text strong>Bytes Added</Typography.Text>
              <br />
              <Typography.Text type="secondary">
                {formatBytes(Number(summary.dataAdded))}
              </Typography.Text>
            </Col>
            <Col span={8}>
              <Typography.Text strong>Total Bytes Processed</Typography.Text>
              <br />
              <Typography.Text type="secondary">
                {formatBytes(Number(summary.totalBytesProcessed))}
              </Typography.Text>
            </Col>
            <Col span={8}>
              <Typography.Text strong>Total Files Processed</Typography.Text>
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
      Restore {restoreOp.path} to {restoreOp.target}
      {!isDone ? (
        <Progress percent={Math.round(progress * 1000) / 10} status="active" />
      ) : null}
      {operation.status == OperationStatus.STATUS_SUCCESS ? (
        <>
          <Button
            type="link"
            onClick={() => {
              backrestService
                .getDownloadURL({ value: operation.id })
                .then((resp) => {
                  window.open(resp.value, "_blank");
                })
                .catch((e) => {
                  alertApi?.error("Failed to fetch download URL: " + e.message);
                });
            }}
          >
            Download File(s)
          </Button>
        </>
      ) : null}
      <br />
      Restored Snapshot ID: {normalizeSnapshotId(operation.snapshotId!)}
      {lastStatus && (
        <Row gutter={16}>
          <Col span={12}>
            <Typography.Text strong>Bytes Done/Total</Typography.Text>
            <br />
            {formatBytes(Number(lastStatus.bytesRestored))}/
            {formatBytes(Number(lastStatus.totalBytes))}
          </Col>
          <Col span={12}>
            <Typography.Text strong>Files Done/Total</Typography.Text>
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
    return <>No status yet.</>;
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
            <Typography.Text strong>Bytes Done/Total</Typography.Text>
            <br />
            <Typography.Text type="secondary">
              {formatBytes(Number(st.bytesDone))} /{" "}
              {formatBytes(Number(st.totalBytes))}
            </Typography.Text>
          </Col>
          <Col span={12}>
            <Typography.Text strong>Files Done/Total</Typography.Text>
            <br />
            <Typography.Text type="secondary">
              {Number(st.filesDone).toLocaleString()} /{" "}
              {Number(st.totalFiles).toLocaleString()}
            </Typography.Text>
          </Col>
        </Row>
        {st.currentFile && st.currentFile.length > 0 && (
          <div style={{ marginTop: 8 }}>
            <Typography.Text strong>Current Files:</Typography.Text>
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
          <Typography.Text strong>Snapshot ID: </Typography.Text>
          {sum.snapshotId !== ""
            ? normalizeSnapshotId(sum.snapshotId!)
            : "No Snapshot Created"}
        </Typography.Text>
        <Row gutter={[16, 8]}>
          <Col span={8}>
            <Typography.Text strong>Files Added</Typography.Text>
            <br />
            <Typography.Text type="secondary">
              {sum.filesNew.toString()}
            </Typography.Text>
          </Col>
          <Col span={8}>
            <Typography.Text strong>Files Changed</Typography.Text>
            <br />
            <Typography.Text type="secondary">
              {sum.filesChanged.toString()}
            </Typography.Text>
          </Col>
          <Col span={8}>
            <Typography.Text strong>Files Unmodified</Typography.Text>
            <br />
            <Typography.Text type="secondary">
              {sum.filesUnmodified.toString()}
            </Typography.Text>
          </Col>
        </Row>
        <Row gutter={[16, 8]}>
          <Col span={8}>
            <Typography.Text strong>Bytes Added</Typography.Text>
            <br />
            <Typography.Text type="secondary">
              {formatBytes(Number(sum.dataAdded))}
            </Typography.Text>
          </Col>
          <Col span={8}>
            <Typography.Text strong>Total Bytes Processed</Typography.Text>
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
    return <>No fields set. This shouldn't happen</>;
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
      policyDesc.push("Keep all.");
    } else if (policy.policy.case === "policyKeepLastN") {
      policyDesc.push(`Keep last ${policy.policy.value} snapshots`);
    } else if (policy.policy.case == "policyTimeBucketed") {
      const val = policy.policy.value;
      if (val.hourly) {
        policyDesc.push(`Keep ${val.hourly} hourly snapshots`);
      }
      if (val.daily) {
        policyDesc.push(`Keep ${val.daily} daily snapshots`);
      }
      if (val.weekly) {
        policyDesc.push(`Keep ${val.weekly} weekly snapshots`);
      }
      if (val.monthly) {
        policyDesc.push(`Keep ${val.monthly} monthly snapshots`);
      }
      if (val.yearly) {
        policyDesc.push(`Keep ${val.yearly} yearly snapshots`);
      }
      if (val.keepLastN) {
        policyDesc.push(
          `Keep latest ${val.keepLastN} snapshots regardless of age`
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
            {"removed snapshot " +
              normalizeSnapshotId(f.id!) +
              " taken at " +
              formatTime(Number(f.unixTimeMs))}{" "}
            <br />
          </div>
        ))}
      </pre>
      Policy:
      <ul>
        {policyDesc.map((desc, idx) => (
          <li key={idx}>{desc}</li>
        ))}
      </ul>
    </>
  );
};
