import React from "react";
import { Operation, OperationStatus } from "../../gen/ts/v1/operations.pb";
import { Col, Collapse, Empty, List, Progress, Row, Typography } from "antd";
import {
  AlertOutlined,
  DatabaseOutlined,
  ExclamationCircleOutlined,
  ExclamationOutlined,
  PaperClipOutlined,
  SaveOutlined,
} from "@ant-design/icons";
import { BackupProgressEntry, ResticSnapshot } from "../../gen/ts/v1/restic.pb";
import { EOperation } from "../state/oplog";

export const OperationList = ({
  operations,
}: React.PropsWithoutRef<{ operations: EOperation[] }>) => {
  operations.sort((a, b) => b.parsedTime - a.parsedTime);

  const elems = operations.map((operation) => (
    <OperationRow operation={operation} />
  ));

  if (operations.length === 0) {
    return (
      <Empty
        description="No operations yet."
        image={Empty.PRESENTED_IMAGE_SIMPLE}
      ></Empty>
    );
  }

  return (
    <List
      itemLayout="horizontal"
      size="small"
      dataSource={operations}
      renderItem={(item, index) => (
        <OperationRow key={item.parsedId} operation={item} />
      )}
    />
  );
};

export const OperationRow = ({
  operation,
}: React.PropsWithoutRef<{ operation: EOperation }>) => {
  let color = "grey";
  if (operation.status === OperationStatus.STATUS_SUCCESS) {
    color = "green";
  } else if (operation.status === OperationStatus.STATUS_ERROR) {
    color = "red";
  } else if (operation.status === OperationStatus.STATUS_INPROGRESS) {
    color = "blue";
  }

  if (operation.displayMessage) {
    return (
      <List.Item>
        <List.Item.Meta
          title={<>Message</>}
          avatar={<ExclamationCircleOutlined style={{ color }} />}
          description={operation.displayMessage}
        />
      </List.Item>
    );
  } else if (operation.operationBackup) {
    const backupOp = operation.operationBackup;
    let desc = `Backup at ${formatTime(operation.unixTimeStartMs!)}`;
    if (operation.status !== OperationStatus.STATUS_INPROGRESS) {
      desc += ` and finished at ${formatTime(operation.unixTimeEndMs!)}`;
    } else {
      desc += " and is still running.";
    }

    return (
      <List.Item>
        <List.Item.Meta
          title={desc}
          avatar={
            <SaveOutlined
              style={{ color }}
              spin={operation.status === OperationStatus.STATUS_INPROGRESS}
            />
          }
          description={
            <>
              <Collapse
                size="small"
                defaultActiveKey={
                  operation.status === OperationStatus.STATUS_INPROGRESS
                    ? [1]
                    : undefined
                }
                items={[
                  {
                    key: 1,
                    label: "Details",
                    children: (
                      <BackupOperationStatus status={backupOp.lastStatus} />
                    ),
                  },
                ]}
              />
            </>
          }
        />
      </List.Item>
    );
  } else if (operation.operationIndexSnapshot) {
    const snapshotOp = operation.operationIndexSnapshot;
    return (
      <List.Item>
        <List.Item.Meta
          title={
            <>Snapshot at {formatTime(snapshotOp.snapshot!.unixTimeMs!)}</>
          }
          avatar={<PaperClipOutlined style={{ color }} />}
          description={<SnapshotInfo snapshot={snapshotOp.snapshot!} />}
        />
      </List.Item>
    );
  }
};

const SnapshotInfo = ({ snapshot }: { snapshot: ResticSnapshot }) => {
  return (
    <Collapse
      size="small"
      items={[
        {
          key: 1,
          label: "Details",
          children: (
            <>
              <Typography.Text>
                <Typography.Text strong>Snapshot ID: </Typography.Text>
                {snapshot.id?.substring(0, 8)}
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
          label: "Browse",
          children: null,
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

  if (status.status) {
    const st = status.status;
    const progress =
      Math.round(
        (parseInt(st.bytesDone!) / Math.max(parseInt(st.totalBytes!), 1)) * 1000
      ) / 10;
    return (
      <>
        <Progress percent={progress} status="active" />
        <br />
        <Row gutter={16}>
          <Col span={12}>
            <Typography.Text strong>Bytes Done/Total</Typography.Text>
            <br />
            {formatBytes(st.bytesDone)}/{formatBytes(st.totalBytes)}
          </Col>
          <Col span={12}>
            <Typography.Text strong>Files Done/Total</Typography.Text>
            <br />
            {st.filesDone}/{st.totalFiles}
          </Col>
        </Row>
      </>
    );
  } else if (status.summary) {
    const sum = status.summary;
    return (
      <>
        <Typography.Text>
          <Typography.Text strong>Snapshot ID: </Typography.Text>
          {sum.snapshotId}
        </Typography.Text>
        <Row gutter={16}>
          <Col span={8}>
            <Typography.Text strong>Files Added</Typography.Text>
            <br />
            {sum.filesNew}
          </Col>
          <Col span={8}>
            <Typography.Text strong>Files Changed</Typography.Text>
            <br />
            {sum.filesChanged}
          </Col>
          <Col span={8}>
            <Typography.Text strong>Files Unmodified</Typography.Text>
            <br />
            {sum.filesChanged}
          </Col>
        </Row>
        <Row gutter={16}>
          <Col span={8}>
            <Typography.Text strong>Bytes Added</Typography.Text>
            <br />
            {formatBytes(sum.dataAdded)}
          </Col>
          <Col span={8}>
            <Typography.Text strong>Total Bytes Processed</Typography.Text>
            <br />
            {formatBytes(sum.totalBytesProcessed)}
          </Col>
          <Col span={8}>
            <Typography.Text strong>Total Files Processed</Typography.Text>
            <br />
            {sum.totalFilesProcessed}
          </Col>
        </Row>
      </>
    );
  } else {
    console.error("GOT UNEXPECTED STATUS: ", status);
    return <>No fields set. This shouldn't happen</>;
  }
};

const formatBytes = (bytes?: number | string) => {
  if (!bytes) {
    return 0;
  }
  if (typeof bytes === "string") {
    bytes = parseInt(bytes);
  }

  const units = ["B", "KB", "MB", "GB", "TB", "PB"];
  let unit = 0;
  while (bytes > 1024) {
    bytes /= 1024;
    unit++;
  }
  return `${Math.round(bytes * 100) / 100} ${units[unit]}`;
};

const formatTime = (time: number | string) => {
  if (typeof time === "string") {
    time = parseInt(time);
  }
  const d = new Date();
  d.setTime(time);
  return d.toLocaleString();
};
