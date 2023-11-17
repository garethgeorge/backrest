import React from "react";
import { Operation, OperationStatus } from "../../gen/ts/v1/operations.pb";
import {
  Card,
  Col,
  Collapse,
  Empty,
  List,
  Progress,
  Row,
  Typography,
} from "antd";
import {
  ExclamationCircleOutlined,
  PaperClipOutlined,
  SaveOutlined,
} from "@ant-design/icons";
import { BackupProgressEntry, ResticSnapshot } from "../../gen/ts/v1/restic.pb";
import { EOperation } from "../state/oplog";
import { SnapshotBrowser } from "./SnapshotBrowser";

export const OperationList = ({
  operations,
}: React.PropsWithoutRef<{ operations: EOperation[] }>) => {
  operations.sort((a, b) => b.parsedTime - a.parsedTime);

  if (operations.length === 0) {
    return (
      <Empty
        description="No operations yet."
        image={Empty.PRESENTED_IMAGE_SIMPLE}
      ></Empty>
    );
  }

  const groupBy = (ops: EOperation[], keyFunc: (op: EOperation) => string) => {
    const groups: { [key: string]: EOperation[] } = {};

    ops.forEach((op) => {
      const key = keyFunc(op);
      if (!(key in groups)) {
        groups[key] = [];
      }
      groups[key].push(op);
    });

    return Object.values(groups);
  };

  // snapshotKey is a heuristic that tries to find a snapshot ID to group the operation by,
  // if one can not be found the operation ID is the key.
  const snapshotKey = (op: EOperation) => {
    if (
      op.operationBackup &&
      op.operationBackup.lastStatus &&
      op.operationBackup.lastStatus.summary
    ) {
      return normalizeSnapshotId(
        op.operationBackup.lastStatus.summary.snapshotId!
      );
    } else if (op.operationIndexSnapshot) {
      return normalizeSnapshotId(op.operationIndexSnapshot.snapshot!.id!);
    }
    return op.id!;
  };

  const groupedItems = groupBy(operations, snapshotKey);
  groupedItems.sort((a, b) => {
    return b[0].parsedTime - a[0].parsedTime;
  });

  return (
    <List
      itemLayout="horizontal"
      size="small"
      dataSource={groupedItems}
      renderItem={(group, index) => {
        if (group.length === 1) {
          return <OperationRow key={group[0].parsedId} operation={group[0]} />;
        }

        return (
          <Card size="small" style={{ margin: "5px" }}>
            {group.map((op) => (
              <OperationRow key={op.parsedId} operation={op} />
            ))}
          </Card>
        );
      }}
      pagination={
        operations.length > 50
          ? { position: "both", align: "center", defaultPageSize: 50 }
          : {}
      }
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

  if (
    operation.displayMessage &&
    operation.status === OperationStatus.STATUS_ERROR
  ) {
    let opType = "Message";
    if (operation.operationBackup) {
      opType = "Backup";
    } else if (operation.operationIndexSnapshot) {
      opType = "Snapshot";
    }

    return (
      <List.Item>
        <List.Item.Meta
          title={
            <>
              {formatTime(operation.unixTimeStartMs!)} - {opType} Error
            </>
          }
          avatar={<ExclamationCircleOutlined style={{ color }} />}
          description={operation.displayMessage}
        />
      </List.Item>
    );
  } else if (operation.operationBackup) {
    const backupOp = operation.operationBackup;
    let desc = `${formatTime(operation.unixTimeStartMs!)} - Backup`;
    if (operation.status !== OperationStatus.STATUS_INPROGRESS) {
      desc += ` completed in ${formatDuration(
        parseInt(operation.unixTimeEndMs!) -
          parseInt(operation.unixTimeStartMs!)
      )}`;
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
            <>{formatTime(snapshotOp.snapshot!.unixTimeMs!)} - Snapshot </>
          }
          avatar={<PaperClipOutlined style={{ color }} />}
          description={
            <SnapshotInfo
              snapshot={snapshotOp.snapshot!}
              repoId={operation.repoId!}
            />
          }
        />
      </List.Item>
    );
  }
};

const SnapshotInfo = ({
  snapshot,
  repoId,
}: {
  snapshot: ResticSnapshot;
  repoId: string;
}) => {
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
          label: "Browse",
          children: (
            <SnapshotBrowser snapshotId={snapshot.id!} repoId={repoId} />
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

const timezoneOffsetMs = new Date().getTimezoneOffset() * 60 * 1000;
const formatTime = (time: number | string) => {
  if (typeof time === "string") {
    time = parseInt(time);
  }
  const d = new Date();
  d.setTime(time - timezoneOffsetMs);
  const isoStr = d.toISOString();
  return `${isoStr.substring(0, 10)} ${d.getUTCHours()}h${d.getUTCMinutes()}m`;
};

const formatDuration = (ms: number) => {
  const seconds = Math.floor(ms / 100) / 10;
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  if (hours === 0 && minutes === 0) {
    return `${seconds % 60}s`;
  } else if (hours === 0) {
    return `${minutes}m${seconds % 60}s`;
  }
  return `${hours}h${minutes % 60}m${seconds % 60}s`;
};

const normalizeSnapshotId = (id: string) => {
  return id.substring(0, 8);
};
