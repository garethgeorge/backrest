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
import { BackupProgressEntry, ResticSnapshot } from "../../gen/ts/v1/restic_pb";
import {
  BackupInfo,
  BackupInfoCollector,
  DisplayType,
  detailsForOperation,
  displayTypeToString,
  getOperations,
  getTypeForDisplay,
  shouldHideOperation,
  subscribeToOperations,
  unsubscribeFromOperations,
} from "../state/oplog";
import { SnapshotBrowser } from "./SnapshotBrowser";
import {
  formatBytes,
  formatTime,
  normalizeSnapshotId,
} from "../lib/formatting";
import _ from "lodash";
import { GetOperationsRequest, OperationDataRequest } from "../../gen/ts/v1/service_pb";
import { useAlertApi } from "./Alerts";
import { MessageInstance } from "antd/es/message/interface";
import { backrestService } from "../api";

// OperationList displays a list of operations that are either fetched based on 'req' or passed in via 'useBackups'.
// If showPlan is provided the planId will be displayed next to each operation in the operation list.
export const OperationList = ({
  req,
  useBackups,
  showPlan,
}: React.PropsWithoutRef<{
  req?: GetOperationsRequest;
  useBackups?: BackupInfo[];
  showPlan?: boolean,
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
        if (opEvent.type !== OperationEventType.EVENT_DELETED) {
          backupCollector.addOperation(opEvent.type!, opEvent.operation!);
        } else {
          backupCollector.removeOperation(opEvent.operation!);
        }
      };
      subscribeToOperations(lis);

      backupCollector.subscribe(_.debounce(() => {
        let backups = backupCollector.getAll();
        backups.sort((a, b) => {
          return b.startTimeMs - a.startTimeMs;
        });
        setBackups(backups);
      }, 50));

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
    backups = [...(useBackups || [])];
    backups.sort((a, b) => {
      return b.startTimeMs - a.startTimeMs;
    });
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
        const ops = [...backup.operations];
        ops.reverse();
        return (
          <Card size="small" style={{ margin: "5px" }}>
            {ops.map((op) => {
              if (shouldHideOperation(op)) {
                return null;
              }
              return <OperationRow alertApi={alertApi!} key={op.id} operation={op} showPlan={showPlan || false} />
            })}
          </Card>
        );
      }}
      pagination={
        backups.length > 10
          ? { position: "both", align: "center", defaultPageSize: 10 }
          : undefined
      }
    />
  );
};

export const OperationRow = ({
  operation,
  alertApi,
  showPlan,
}: React.PropsWithoutRef<{ operation: Operation, alertApi?: MessageInstance, showPlan: boolean }>) => {
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
  let title = (
    <>
      {showPlan ? operation.planId + " - " : undefined} {formatTime(Number(operation.unixTimeStartMs))} - {opName}{" "}
      <span className="backrest operation-details">{details.displayState}</span>
    </>
  );

  if (operation.status === OperationStatus.STATUS_PENDING || operation.status == OperationStatus.STATUS_INPROGRESS) {
    title = <>
      {title}
      <Button type="link" size="small" onClick={() => {
        backrestService.cancel({ value: operation.id! }).then(() => {
          alertApi?.success("Requested to cancel operation");
        }).catch((e) => {
          alertApi?.error("Failed to cancel operation: " + e.message);
        });
      }}>[Cancel Operation]</Button>
    </>
  }

  let body: React.ReactNode | undefined;

  if (operation.op.case === "operationBackup") {
    const backupOp = operation.op.value;
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
    body = <ForgetOperationDetails forgetOp={forgetOp} />
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
  } else if (operation.op.case === "operationRestore") {
    const restore = operation.op.value;
    body = (
      <>
        Restore {restore.path} to {restore.target}
        {details.percentage !== undefined ? (
          <Progress percent={details.percentage || 0} status="active" />
        ) : null}
      </>
    );
  } else if (operation.op.case === "operationRunHook") {
    const hook = operation.op.value;
    body = <RunHookOperationStatus op={operation} />
  }

  if (operation.displayMessage) {
    body = (
      <>
        <pre>{details.state}: {operation.displayMessage}</pre>
        {body}
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
            {formatBytes(Number(st.bytesDone))}/{formatBytes(Number(st.totalBytes))}
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

const ForgetOperationDetails = ({ forgetOp }: { forgetOp: OperationForget }) => {
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
          children: <>
            Removed snapshots:
            <pre>{forgetOp.forget?.map((f) => (
              <div key={f.id}>
                {"removed snapshot " + normalizeSnapshotId(f.id!) + " taken at " + formatTime(Number(f.unixTimeMs))} <br />
              </div>
            ))}</pre>
            Policy:
            <ul>
              {policyDesc.map((desc, idx) => (
                <li key={idx}>{desc}</li>
              ))}
            </ul>
          </>,
        },
      ]}
    />
  );
}

const RunHookOperationStatus = ({ op }: { op: Operation }) => {
  const [output, setOutput] = useState<string | undefined>(undefined);

  if (op.op.case !== "operationRunHook") {
    return <>Wrong operation type</>;
  }

  const hook = op.op.value;

  useEffect(() => {
    if (!hook.outputRef) {
      return;
    }
    backrestService.getBigOperationData(new OperationDataRequest({
      id: op.id,
      key: hook.outputRef,
    })).then((resp) => {
      setOutput(new TextDecoder("utf-8").decode(resp.value));
    }).catch((e) => {
      console.error("Failed to fetch hook output: ", e);
    });
  }, [hook.outputRef]);

  return <>
    Hook: {hook.name} <br />
    Output: <br />
    <pre>
      {output}
    </pre>
  </>
}