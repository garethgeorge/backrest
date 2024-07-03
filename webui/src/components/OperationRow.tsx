import React, { useEffect, useState } from "react";
import {
  Operation,
  OperationEvent,
  OperationEventType,
  OperationForget,
  OperationRunHook,
  OperationStatus,
} from "../../gen/ts/v1/operations_pb";
import {
  Button,
  Col,
  Collapse,
  Empty,
  List,
  Modal,
  Progress,
  Row,
  Typography,
} from "antd";
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

export const OperationRow = ({
  operation,
  alertApi,
  showPlan,
}: React.PropsWithoutRef<{
  operation: Operation;
  alertApi?: MessageInstance;
  showPlan: boolean;
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
          onClick={() => {
            backrestService
              .cancel({ value: operation.id! })
              .then(() => {
                alertApi?.success("Requested to cancel operation");
              })
              .catch((e) => {
                alertApi?.error("Failed to cancel operation: " + e.message);
              });
          }}
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
            onClick={() => {
              showModal(
                <Modal
                  width="50%"
                  title={
                    "Logs for operation " +
                    opName +
                    " at " +
                    formatTime(Number(operation.unixTimeStartMs))
                  }
                  visible={true}
                  footer={null}
                  onCancel={() => {
                    showModal(null);
                  }}
                >
                  <BigOperationDataVerbatim logref={operation.logref!} />
                </Modal>
              );
            }}
          >
            [View Logs]
          </Button>
        </small>
      </>
    );
  }

  let body: React.ReactNode | undefined;
  let displayMessage = operation.displayMessage;

  if (operation.op.case === "operationBackup") {
    const backupOp = operation.op.value;
    const items: { key: number; label: string; children: React.ReactNode }[] = [
      {
        key: 1,
        label: "Backup Details",
        children: <BackupOperationStatus status={backupOp.lastStatus} />,
      },
    ];

    if (backupOp.errors.length > 0) {
      items.splice(0, 0, {
        key: 2,
        label: "Item Errors",
        children: (
          <pre>
            {backupOp.errors.map((e) => "Error on item: " + e.item).join("\n")}
          </pre>
        ),
      });
    }

    body = (
      <>
        <Collapse
          size="small"
          destroyInactivePanel
          defaultActiveKey={[1]}
          items={items}
        />
      </>
    );
  } else if (operation.op.case === "operationIndexSnapshot") {
    const snapshotOp = operation.op.value;
    body = (
      <SnapshotInfo
        snapshot={snapshotOp.snapshot!}
        repoId={operation.repoId!}
        planId={operation.planId}
      />
    );
  } else if (operation.op.case === "operationForget") {
    const forgetOp = operation.op.value;
    body = <ForgetOperationDetails forgetOp={forgetOp} />;
  } else if (operation.op.case === "operationPrune") {
    const prune = operation.op.value;
    body = (
      <Collapse
        size="small"
        destroyInactivePanel
        items={[
          {
            key: 1,
            label: "Prune Output",
            children: <pre>{prune.output}</pre>,
          },
        ]}
      />
    );
  } else if (operation.op.case === "operationCheck") {
    const check = operation.op.value;
    body = (
      <Collapse
        size="small"
        destroyInactivePanel
        items={[
          {
            key: 1,
            label: "Check Output",
            children: <pre>{check.output}</pre>,
          },
        ]}
      />
    );
  } else if (operation.op.case === "operationRestore") {
    const restore = operation.op.value;
    const progress = Math.round((details.percentage || 0) * 10) / 10;
    const st = restore.lastStatus! || {};

    body = (
      <>
        Restore {restore.path} to {restore.target}
        {details.percentage !== undefined ? (
          <Progress percent={progress} status="active" />
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
                    alertApi?.error(
                      "Failed to fetch download URL: " + e.message
                    );
                  });
              }}
            >
              Download File(s)
            </Button>
          </>
        ) : null}
        <br />
        Snapshot ID: {normalizeSnapshotId(operation.snapshotId!)}
        <Row gutter={16}>
          <Col span={12}>
            <Typography.Text strong>Bytes Done/Total</Typography.Text>
            <br />
            {formatBytes(Number(st.bytesRestored))}/
            {formatBytes(Number(st.totalBytes))}
          </Col>
          <Col span={12}>
            <Typography.Text strong>Files Done/Total</Typography.Text>
            <br />
            {Number(st.filesRestored)}/{Number(st.totalFiles)}
          </Col>
        </Row>
      </>
    );
  } else if (operation.op.case === "operationRunHook") {
    const hook = operation.op.value;
    const triggeringCondition = proto3
      .getEnumType(Hook_Condition)
      .findNumber(hook.condition);
    if (triggeringCondition !== undefined) {
      displayMessage += "\ntriggered by condition: " + triggeringCondition.name;
    }
  }

  const children = [];

  if (operation.displayMessage) {
    children.push(
      <div key="message">
        <pre>
          {details.state ? details.state + ": " : null}
          {displayMessage}
        </pre>
      </div>
    );
  }

  children.push(<div key="body">{body}</div>);

  return (
    <List.Item key={operation.id}>
      <List.Item.Meta title={title} avatar={avatar} description={children} />
    </List.Item>
  );
};

const SnapshotInfo = ({
  snapshot,
  repoId,
  planId,
}: {
  snapshot: ResticSnapshot;
  repoId: string;
  planId?: string;
}) => {
  return (
    <Collapse
      size="small"
      defaultActiveKey={[1]}
      items={[
        {
          key: 1,
          label: "Snapshot Details",
          children: (
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
          ),
        },
        {
          key: 2,
          label: "Browse and Restore Files in Backup",
          children: (
            <SnapshotBrowser
              snapshotId={snapshot.id!}
              repoId={repoId}
              planId={planId}
            />
          ),
        },
      ]}
    />
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
    <Collapse
      size="small"
      destroyInactivePanel
      items={[
        {
          key: 1,
          label: "Removed " + forgetOp.forget?.length + " Snapshots",
          children: (
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
          ),
        },
      ]}
    />
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
