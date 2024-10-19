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
import { LogDataRequest } from "../../gen/ts/v1/service_pb";
import { MessageInstance } from "antd/es/message/interface";
import { backrestService } from "../api";
import { useShowModal } from "./ModalManager";
import { useAlertApi } from "./Alerts";
import { OperationList } from "./OperationList";
import {
  displayTypeToString,
  getTypeForDisplay,
  nameForStatus,
} from "../state/flowdisplayaggregator";
import { OperationIcon } from "./OperationIcon";
import { LogView } from "./LogView";

export const OperationRow = ({
  operation,
  alertApi,
  showPlan,
  hookOperations,
}: React.PropsWithoutRef<{
  operation: Operation;
  alertApi?: MessageInstance;
  showPlan?: boolean;
  hookOperations?: Operation[];
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

  const doCancel = () => {
    backrestService
      .cancel({ value: operation.id! })
      .then(() => {
        alertApi?.success("Requested to cancel operation");
      })
      .catch((e) => {
        alertApi?.error("Failed to cancel operation: " + e.message);
      });
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
  let title = (
    <>
      {showPlan ? operation.planId + " - " : undefined}{" "}
      {formatTime(Number(operation.unixTimeStartMs))} - {opName}{" "}
      <span className="backrest operation-details">{details}</span>
    </>
  );

  if (
    operation.status === OperationStatus.STATUS_INPROGRESS ||
    operation.status === OperationStatus.STATUS_PENDING
  ) {
    title = (
      <>
        {title}
        <Button
          type="link"
          size="small"
          className="backrest operation-details"
          onClick={doCancel}
        >
          [Cancel Operation]
        </Button>
      </>
    );
  }

  if (operation.logref) {
    title = (
      <>
        {title}
        <small>
          <Button
            type="link"
            size="middle"
            className="backrest operation-details"
            onClick={doShowLogs}
          >
            [View Logs]
          </Button>
        </small>
      </>
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
          repoId={operation.repoId}
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
        <OperationList
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
    <List.Item key={operation.id}>
      <List.Item.Meta
        title={title}
        avatar={<OperationIcon type={displayType} status={operation.status} />}
        description={
          <>
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
              destroyInactivePanel={true}
              items={bodyItems}
              defaultActiveKey={expandedBodyItems}
            />
          </>
        }
      />
    </List.Item>
  );
};

const SnapshotDetails = ({ snapshot }: { snapshot: ResticSnapshot }) => {
  const summary: Partial<SnapshotSummary> = snapshot.summary || {};

  const rows: React.ReactNode[] = [
    <Row gutter={16} key={1}>
      <Col span={8}>
        <Typography.Text strong>User and Host</Typography.Text>
        <br />
        {snapshot.username}@{snapshot.hostname}
      </Col>
      <Col span={12}>
        <Typography.Text strong>Tags</Typography.Text>
        <br />
        {snapshot.tags.join(", ")}
      </Col>
    </Row>,
  ];

  if (
    summary.filesNew ||
    summary.filesChanged ||
    summary.filesUnmodified ||
    summary.dataAdded ||
    summary.totalFilesProcessed ||
    summary.totalBytesProcessed
  ) {
    rows.push(
      <Row gutter={16} key={2}>
        <Col span={8}>
          <Typography.Text strong>Files Added</Typography.Text>
          <br />
          {"" + summary.filesNew}
        </Col>
        <Col span={8}>
          <Typography.Text strong>Files Changed</Typography.Text>
          <br />
          {"" + summary.filesChanged}
        </Col>
        <Col span={8}>
          <Typography.Text strong>Files Unmodified</Typography.Text>
          <br />
          {"" + summary.filesUnmodified}
        </Col>
      </Row>
    );
    rows.push(
      <Row gutter={16} key={3}>
        <Col span={8}>
          <Typography.Text strong>Bytes Added</Typography.Text>
          <br />
          {formatBytes(Number(summary.dataAdded))}
        </Col>
        <Col span={8}>
          <Typography.Text strong>Bytes Processed</Typography.Text>
          <br />
          {formatBytes(Number(summary.totalBytesProcessed))}
        </Col>
        <Col span={8}>
          <Typography.Text strong>Files Processed</Typography.Text>
          <br />
          {"" + summary.totalFilesProcessed}
        </Col>
      </Row>
    );
  }

  return (
    <>
      <Typography.Text>
        <Typography.Text strong>Snapshot ID: </Typography.Text>
        {normalizeSnapshotId(snapshot.id!)} <br />
        {rows}
      </Typography.Text>
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
        <Row gutter={16}>
          <Col span={12}>
            <Typography.Text strong>Bytes Done/Total</Typography.Text>
            <br />
            {formatBytes(Number(st.bytesDone))}/
            {formatBytes(Number(st.totalBytes))}
          </Col>
          <Col span={12}>
            <Typography.Text strong>Files Done/Total</Typography.Text>
            <br />
            {Number(st.filesDone)}/{Number(st.totalFiles)}
          </Col>
        </Row>
        {st.currentFile && st.currentFile.length > 0 ? (
          <pre>Current file: {st.currentFile.join("\n")}</pre>
        ) : null}
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
        <Row gutter={16}>
          <Col span={8}>
            <Typography.Text strong>Files Added</Typography.Text>
            <br />
            {sum.filesNew.toString()}
          </Col>
          <Col span={8}>
            <Typography.Text strong>Files Changed</Typography.Text>
            <br />
            {sum.filesChanged.toString()}
          </Col>
          <Col span={8}>
            <Typography.Text strong>Files Unmodified</Typography.Text>
            <br />
            {sum.filesUnmodified.toString()}
          </Col>
        </Row>
        <Row gutter={16}>
          <Col span={8}>
            <Typography.Text strong>Bytes Added</Typography.Text>
            <br />
            {formatBytes(Number(sum.dataAdded))}
          </Col>
          <Col span={8}>
            <Typography.Text strong>Total Bytes Processed</Typography.Text>
            <br />
            {formatBytes(Number(sum.totalBytesProcessed))}
          </Col>
          <Col span={8}>
            <Typography.Text strong>Total Files Processed</Typography.Text>
            <br />
            {sum.totalFilesProcessed.toString()}
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
        policyDesc.push(`Keep hourly for ${val.hourly} hours`);
      }
      if (val.daily) {
        policyDesc.push(`Keep daily for ${val.daily} days`);
      }
      if (val.weekly) {
        policyDesc.push(`Keep weekly for ${val.weekly} weeks`);
      }
      if (val.monthly) {
        policyDesc.push(`Keep monthly for ${val.monthly} months`);
      }
      if (val.yearly) {
        policyDesc.push(`Keep yearly for ${val.yearly} years`);
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
