import React, { useEffect, useState } from "react";
import { Operation, OperationStatus } from "../../gen/ts/v1/operations.pb";
import {
  Card,
  Col,
  Collapse,
  Empty,
  List,
  Progress,
  Row,
  Spin,
  Typography,
} from "antd";
import {
  ExclamationCircleOutlined,
  PaperClipOutlined,
  SaveOutlined,
} from "@ant-design/icons";
import { BackupProgressEntry, ResticSnapshot } from "../../gen/ts/v1/restic.pb";
import {
  EOperation,
  buildOperationListListener,
  subscribeToOperations,
  unsubscribeFromOperations,
} from "../state/oplog";
import { SnapshotBrowser } from "./SnapshotBrowser";
import {
  formatBytes,
  formatDuration,
  formatTime,
  normalizeSnapshotId,
} from "../lib/formatting";
import _ from "lodash";
import { GetOperationsRequest } from "../../gen/ts/v1/service.pb";

export const OperationList = ({
  req,
}: React.PropsWithoutRef<{ req: GetOperationsRequest }>) => {
  const [operations, setOperations] = useState<EOperation[]>([]);
  console.log("operation list with req: ", req);

  useEffect(() => {
    const lis = buildOperationListListener(req, (event, operation, opList) => {
      console.log("got list: ", JSON.stringify(opList, null, 2));
      setOperations(opList);
    });
    subscribeToOperations(lis);

    return () => {
      unsubscribeFromOperations(lis);
    };
  }, [JSON.stringify(req)]);

  if (operations.length === 0) {
    return (
      <Empty
        description="No operations yet."
        image={Empty.PRESENTED_IMAGE_SIMPLE}
      ></Empty>
    );
  }

  // groups items by snapshotID if one can be identified, otherwise by operation ID.
  const groupedItems = _.values(
    _.groupBy(operations, (op: EOperation) => {
      return getSnapshotId(op) || op.id!;
    })
  );
  groupedItems.sort((a, b) => {
    return b[0].parsedTime - a[0].parsedTime;
  });
  groupedItems.forEach((group) => {
    group.sort((a, b) => {
      return b.parsedTime - a.parsedTime;
    });
  });

  return (
    <List
      itemLayout="horizontal"
      size="small"
      dataSource={groupedItems}
      renderItem={(group, index) => {
        if (group.length === 1) {
          return <OperationRow key={group[0].id!} operation={group[0]} />;
        }

        return (
          <Card size="small" style={{ margin: "5px" }}>
            {group.map((op) => (
              <OperationRow key={op.id!} operation={op} />
            ))}
          </Card>
        );
      }}
      pagination={
        operations.length > 50
          ? { position: "both", align: "center", defaultPageSize: 50 }
          : undefined
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

  let opType = "Message";
  if (operation.operationBackup) {
    opType = "Backup";
  } else if (operation.operationIndexSnapshot) {
    opType = "Snapshot";
  }

  if (
    operation.displayMessage &&
    operation.status === OperationStatus.STATUS_ERROR
  ) {
    return (
      <List.Item>
        <List.Item.Meta
          title={
            <>
              {formatTime(operation.unixTimeStartMs!)} - {opType} Error
            </>
          }
          avatar={<ExclamationCircleOutlined style={{ color }} />}
          description={<pre>{operation.displayMessage}</pre>}
        />
      </List.Item>
    );
  } else if (operation.operationBackup) {
    const backupOp = operation.operationBackup;
    let desc = `${formatTime(operation.unixTimeStartMs!)} - Backup`;
    if (operation.status == OperationStatus.STATUS_SUCCESS) {
      desc += ` completed in ${formatDuration(
        parseInt(operation.unixTimeEndMs!) -
          parseInt(operation.unixTimeStartMs!)
      )}`;
    } else if (operation.status === OperationStatus.STATUS_INPROGRESS) {
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
                destroyInactivePanel
                defaultActiveKey={[1]}
                items={[
                  {
                    key: 1,
                    label: "Backup Details",
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
          {normalizeSnapshotId(sum.snapshotId!)}
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

const getSnapshotId = (op: EOperation): string | null => {
  if (op.operationBackup) {
    const ob = op.operationBackup;
    if (ob.lastStatus && ob.lastStatus.summary) {
      return normalizeSnapshotId(ob.lastStatus.summary.snapshotId!);
    }
  } else if (op.operationIndexSnapshot) {
    return normalizeSnapshotId(op.operationIndexSnapshot.snapshot!.id!);
  }
  return null;
};
