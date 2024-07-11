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
  PaperClipOutlined,
  SaveOutlined,
  DeleteOutlined,
  DownloadOutlined,
  RobotOutlined,
  InfoCircleOutlined,
  FileSearchOutlined,
} from "@ant-design/icons";
import { BackupProgressEntry, ResticSnapshot } from "../../gen/ts/v1/restic_pb";
import {
  DisplayType,
  detailsForOperation,
  displayTypeToString,
  getTypeForDisplay,
} from "../state/oplog";
import { SnapshotBrowser } from "./SnapshotBrowser";
import {
  formatBytes,
  formatTime,
  normalizeSnapshotId,
} from "../lib/formatting";
import _ from "lodash";
import { LogDataRequest } from "../../gen/ts/v1/service_pb";
import { MessageInstance } from "antd/es/message/interface";
import { backrestService } from "../api";
import { useShowModal } from "./ModalManager";
import { proto3 } from "@bufbuild/protobuf";
import { Hook_Condition } from "../../gen/ts/v1/config_pb";
import { useAlertApi } from "./Alerts";
import { OperationList } from "./OperationList";

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
  const details = detailsForOperation(operation);
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

  let avatar: React.ReactNode;
  switch (displayType) {
    case DisplayType.BACKUP:
      avatar = (
        <SaveOutlined
          style={{ color: details.color }}
          spin={operation.status === OperationStatus.STATUS_INPROGRESS}
        />
      );
      break;
    case DisplayType.FORGET:
      avatar = (
        <DeleteOutlined
          style={{ color: details.color }}
          spin={operation.status === OperationStatus.STATUS_INPROGRESS}
        />
      );
      break;
    case DisplayType.SNAPSHOT:
      avatar = <PaperClipOutlined style={{ color: details.color }} />;
      break;
    case DisplayType.RESTORE:
      avatar = <DownloadOutlined style={{ color: details.color }} />;
      break;
    case DisplayType.PRUNE:
      avatar = <DeleteOutlined style={{ color: details.color }} />;
      break;
    case DisplayType.CHECK:
      avatar = <FileSearchOutlined style={{ color: details.color }} />;
    case DisplayType.RUNHOOK:
      avatar = <RobotOutlined style={{ color: details.color }} />;
      break;
    case DisplayType.STATS:
      avatar = <InfoCircleOutlined style={{ color: details.color }} />;
      break;
  }

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
        <BigOperationDataVerbatim logref={operation.logref!} />
      </Modal>
    );
  };

  const opName = displayTypeToString(getTypeForDisplay(operation));
  let title = (
    <>
      {showPlan ? operation.planId + " - " : undefined}{" "}
      {formatTime(Number(operation.unixTimeStartMs))} - {opName}{" "}
      <span className="backrest operation-details">{details.displayState}</span>
    </>
  );

  if (operation.status == OperationStatus.STATUS_INPROGRESS) {
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
    expandedBodyItems.push("details");
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
    bodyItems.push({
      key: "prune",
      label: "Prune Output",
      children: <pre>{prune.output}</pre>,
    });
  } else if (operation.op.case === "operationCheck") {
    const check = operation.op.value;
    bodyItems.push({
      key: "check",
      label: "Check Output",
      children: <pre>{check.output}</pre>,
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
    // TODO: customized view of hook execution info
  }

  if (hookOperations) {
    bodyItems.push({
      key: "hookOperations",
      label: "Hooks Triggered",
      children: <OperationList useOperations={hookOperations} />,
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
        avatar={avatar}
        description={
          <>
            {operation.displayMessage && (
              <div key="message">
                <pre>
                  {details.state ? details.state + ": " : null}
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
  return (
    <>
      <Typography.Text>
        <Typography.Text strong>Snapshot ID: </Typography.Text>
        {normalizeSnapshotId(snapshot.id!)}
      </Typography.Text>
      <Row gutter={16}>
        <Col span={8}>
          <Typography.Text strong>Host</Typography.Text>
          <br />
          {snapshot.hostname}
        </Col>
        <Col span={8}>
          <Typography.Text strong>Username</Typography.Text>
          <br />
          {snapshot.hostname}
        </Col>
        <Col span={8}>
          <Typography.Text strong>Tags</Typography.Text>
          <br />
          {snapshot.tags?.join(", ")}
        </Col>
      </Row>
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
        <Progress
          percent={Math.round(progress * 1000) / 1000}
          status="active"
        />
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
          {normalizeSnapshotId(sum.snapshotId!)}
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
  if (policy.keepLastN) {
    policyDesc.push(`Keep Last ${policy.keepLastN} Snapshots`);
  }
  if (policy.keepHourly) {
    policyDesc.push(`Keep Hourly for ${policy.keepHourly} Hours`);
  }
  if (policy.keepDaily) {
    policyDesc.push(`Keep Daily for ${policy.keepDaily} Days`);
  }
  if (policy.keepWeekly) {
    policyDesc.push(`Keep Weekly for ${policy.keepWeekly} Weeks`);
  }
  if (policy.keepMonthly) {
    policyDesc.push(`Keep Monthly for ${policy.keepMonthly} Months`);
  }
  if (policy.keepYearly) {
    policyDesc.push(`Keep Yearly for ${policy.keepYearly} Years`);
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
      {/* Policy:
            <ul>
              {policyDesc.map((desc, idx) => (
                <li key={idx}>{desc}</li>
              ))}
            </ul> */}
    </>
  );
};

// TODO: refactor this to use the provider pattern
const BigOperationDataVerbatim = ({ logref }: { logref: string }) => {
  const [output, setOutput] = useState<string | undefined>(undefined);

  useEffect(() => {
    if (!logref) {
      return;
    }
    backrestService
      .getLogs(
        new LogDataRequest({
          ref: logref,
        })
      )
      .then((resp) => {
        setOutput(new TextDecoder("utf-8").decode(resp.value));
      })
      .catch((e) => {
        console.error("Failed to fetch hook output: ", e);
      });
  }, [logref]);

  return <pre style={{ whiteSpace: "pre" }}>{output}</pre>;
};
