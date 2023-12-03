import React, { useEffect, useState } from "react";
import {
  Operation,
  OperationEvent,
  OperationStatus,
} from "../../gen/ts/v1/operations.pb";
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
  DeleteOutlined,
} from "@ant-design/icons";
import { BackupProgressEntry, ResticSnapshot } from "../../gen/ts/v1/restic.pb";
import {
  BackupInfo,
  BackupInfoCollector,
  EOperation,
  buildOperationListListener,
  getOperations,
  shouldHideStatus,
  subscribeToOperations,
  toEop,
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
import { useAlertApi } from "./Alerts";

export const OperationList = ({
  req,
  useBackups,
}: React.PropsWithoutRef<{
  req?: GetOperationsRequest;
  useBackups?: BackupInfo[];
}>) => {
  const alertApi = useAlertApi();

  let backups: BackupInfo[] = [];
  if (req) {
    const [backupState, setBackups] = useState<BackupInfo[]>(useBackups || []);
    backups = backupState;

    // track backups for this operation tree view.
    useEffect(() => {
      if (!req) {
        return;
      }

      const backupCollector = new BackupInfoCollector();
      const lis = (opEvent: OperationEvent) => {
        if (!!req.planId && opEvent.operation!.planId !== req.planId) {
          return;
        }
        if (!!req.repoId && opEvent.operation!.repoId !== req.repoId) {
          return;
        }
        backupCollector.addOperation(opEvent.type!, opEvent.operation!);
      };
      subscribeToOperations(lis);

      backupCollector.subscribe(() => {
        let backups = backupCollector.getAll();
        backups = backups.filter((b) => {
          return !shouldHideStatus(b.status);
        });
        backups.sort((a, b) => {
          return b.startTimeMs - a.startTimeMs;
        });
        setBackups(backups);
      });

      getOperations(req)
        .then((ops) => {
          backupCollector.bulkAddOperations(ops);
        })
        .catch((e) => {
          alertApi!.error("Failed to fetch operations: " + e.message);
        });
      return () => {
        unsubscribeFromOperations(lis);
      };
    }, [JSON.stringify(req)]);
  } else {
    backups = useBackups || [];
  }

  if (backups.length === 0) {
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
      dataSource={backups}
      renderItem={(backup) => {
        const ops = backup.operations;
        if (ops.length === 1) {
          return <OperationRow key={ops[0].id!} operation={toEop(ops[0])} />;
        }

        return (
          <Card size="small" style={{ margin: "5px" }}>
            {ops.map((op) => (
              <OperationRow key={op.id!} operation={toEop(op)} />
            ))}
          </Card>
        );
      }}
      pagination={
        backups.length > 50
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
      desc += " is still running.";
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
  } else if (operation.operationForget) {
    const forgetOp = operation.operationForget;
    if (forgetOp.forget?.length === 0) {
      return null;
    }

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
      <List.Item>
        <List.Item.Meta
          title={
            <>{formatTime(operation.unixTimeStartMs!)} - Forget Operation</>
          }
          avatar={<DeleteOutlined style={{ color }} />}
          description={
            <>
              <p></p>
              <Collapse
                size="small"
                destroyInactivePanel
                items={[
                  {
                    key: 1,
                    label:
                      "Removed " +
                      forgetOp.forget?.length +
                      " Snapshots (Policy Details)",
                    children: (
                      <div>
                        <ul>
                          {policyDesc.map((desc) => (
                            <li>{desc}</li>
                          ))}
                        </ul>
                      </div>
                    ),
                  },
                ]}
              />
            </>
          }
        />
      </List.Item>
    );
  } else if (operation.operationPrune) {
    const prune = operation.operationPrune;
    return (
      <List.Item>
        <List.Item.Meta
          title={<>Prune Operation</>}
          avatar={<DeleteOutlined style={{ color }} />}
          description={
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
