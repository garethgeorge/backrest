import React from "react";
import { Operation, OperationStatus } from "../../gen/ts/v1/operations.pb";
import { Col, Collapse, Empty, List, Progress, Row, Typography } from "antd";
import { AlertOutlined, DatabaseOutlined } from "@ant-design/icons";
import { BackupProgressEntry } from "../../gen/ts/v1/restic.pb";

export const OperationList = ({
  operations,
}: React.PropsWithoutRef<{ operations: Operation[] }>) => {
  interface OpWrapper {
    startTimeMs: number;
    operation: Operation;
  }
  const ops = operations.map((operation) => {
    return {
      time: parseInt(operation.unixTimeStartMs!),
      operation,
    };
  });

  ops.sort((a, b) => b.time - a.time);

  const elems = ops.map(({ operation }) => (
    <OperationRow operation={operation} />
  ));

  if (ops.length === 0) {
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
      dataSource={ops}
      renderItem={(item, index) => (
        <OperationRow key={item.operation.id!} operation={item.operation} />
      )}
    />
  );
};

export const OperationRow = ({
  operation,
}: React.PropsWithoutRef<{ operation: Operation }>) => {
  let contents: React.ReactNode;

  let color = "grey";
  if (operation.status === OperationStatus.STATUS_SUCCESS) {
    color = "green";
  } else if (operation.status === OperationStatus.STATUS_ERROR) {
    color = "red";
  } else if (operation.status === OperationStatus.STATUS_INPROGRESS) {
    color = "blue";
  }

  if (operation.operationBackup) {
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
          avatar={<DatabaseOutlined style={{ color }} />}
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
          avatar={<DatabaseOutlined style={{ color }} />}
          description={<>A snapshot. More info needed</>}
        />
      </List.Item>
    );
  } else if (operation.displayMessage) {
    return (
      <List.Item>
        <List.Item.Meta
          title={<>Message</>}
          avatar={<AlertOutlined style={{ color }} />}
          description={operation.displayMessage}
        />
      </List.Item>
    );
  }
};

const formatTime = (time: number | string) => {
  if (typeof time === "string") {
    time = parseInt(time);
  }
  const d = new Date();
  d.setTime(time);
  return d.toLocaleString();
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
