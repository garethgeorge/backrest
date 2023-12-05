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
  Typography,
} from "antd";
import {
  PaperClipOutlined,
  SaveOutlined,
  DeleteOutlined,
  DownloadOutlined,
} from "@ant-design/icons";
import { BackupProgressEntry, ResticSnapshot } from "../../gen/ts/v1/restic.pb";
import {
  BackupInfo,
  BackupInfoCollector,
  DisplayType,
  EOperation,
  detailsForOperation,
  displayTypeToString,
  getOperations,
  getTypeForDisplay,
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
  const details = detailsForOperation(operation);
  const displayType = getTypeForDisplay(operation);
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
  }

  const opName = displayTypeToString(getTypeForDisplay(operation));
  const title = (
    <>
      {formatTime(operation.unixTimeStartMs!)} - {opName}{" "}
      <span className="resticui operation-details">{details.displayState}</span>
    </>
  );

  let body: React.ReactNode | undefined;

  if (
    operation.displayMessage &&
    operation.status === OperationStatus.STATUS_ERROR
  ) {
    body = <pre>{operation.displayMessage}</pre>;
  } else if (operation.operationBackup) {
    const backupOp = operation.operationBackup;
    body = (
      <>
        <Collapse
          size="small"
          destroyInactivePanel
          defaultActiveKey={[1]}
          items={[
            {
              key: 1,
              label: "Backup Details",
              children: <BackupOperationStatus status={backupOp.lastStatus} />,
            },
          ]}
        />
      </>
    );
  } else if (operation.operationIndexSnapshot) {
    const snapshotOp = operation.operationIndexSnapshot;
    body = (
      <SnapshotInfo
        snapshot={snapshotOp.snapshot!}
        repoId={operation.repoId!}
        planId={operation.planId}
      />
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

    body = (
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
    );
  } else if (operation.operationPrune) {
    const prune = operation.operationPrune;
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
  } else if (operation.operationRestore) {
    const restore = operation.operationRestore;
    body = (
      <>
        Restore {restore.path} to {restore.target}
        {details.percentage !== undefined ? (
          <Progress percent={details.percentage || 0} status="active" />
        ) : null}
      </>
    );
  }

  return (
    <List.Item>
      <List.Item.Meta title={title} avatar={avatar} description={body} />
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
        {st.currentFile && st.currentFile.length > 0 ? (
          <pre>Current file: {st.currentFile.join("\n")}</pre>
        ) : null}
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
