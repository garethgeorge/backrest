import React, { useEffect, useState } from "react";
import { Repo } from "../../gen/ts/v1/config_pb";
import { Col, Empty, Flex, Row, Spin, TabsProps, Tabs, Tooltip, Typography } from "antd";
import { useRecoilValue } from "recoil";
import { configState } from "../state/config";
import { useAlertApi } from "../components/Alerts";
import { OperationList } from "../components/OperationList";
import { OperationTree } from "../components/OperationTree";
import { MAX_OPERATION_HISTORY, STATS_OPERATION_HISTORY } from "../constants";
import { GetOperationsRequest } from "../../gen/ts/v1/service_pb";
import { getOperations } from "../state/oplog";
import { RepoStats } from "../../gen/ts/v1/restic_pb";
import { formatBytes, formatTime } from "../lib/formatting";
import { Operation } from "../../gen/ts/v1/operations_pb";
import { backrestService } from "../api";
import { StringValue } from "@bufbuild/protobuf";
import { SpinButton } from "../components/SpinButton";

export const RepoView = ({ repo }: React.PropsWithChildren<{ repo: Repo }>) => {
  const alertsApi = useAlertApi()!;
  const [loading, setLoading] = useState(true);
  const [statsOperation, setStatsOperation] = useState<Operation | null>(null);

  useEffect(() => {
    setLoading(true);
    setStatsOperation(null);
    getOperations(new GetOperationsRequest({ repoId: repo.id!, lastN: BigInt(STATS_OPERATION_HISTORY) })).then((operations) => {
      for (const op of operations) {
        if (op.op.case === "operationStats") {
          const stats = op.op.value.stats;
          if (stats) {
            setStatsOperation(op);
          }
        }
      }
    }).catch((e) => {
      console.error(e);
    }).finally(() => {
      setLoading(false);
    });
  }, [repo.id]);

  // Task handlers
  const handleIndexNow = async () => {
    await backrestService.indexSnapshots(new StringValue({ value: repo.id! }));
  }

  // Gracefully handle deletions by checking if the plan is still in the config.
  const config = useRecoilValue(configState);
  let repoInConfig = config.repos?.find((p) => p.id === repo.id);
  if (!repoInConfig) {
    return <p>Repo was deleted.</p>;
  }
  repo = repoInConfig;

  if (loading) {
    return <Spin />;
  }

  const items = [
    {
      key: "1",
      label: "Stats",
      children: (
        <>
          {statsOperation === null ? <Empty description="No data. Have you run a backup yet?" /> :
            <>
              <h3>Repo stats computed on {formatTime(Number(statsOperation.unixTimeStartMs))}</h3>
              {statsOperation.op.case === "operationStats" && <StatsTable stats={statsOperation.op.value.stats!} />}
              <small>Stats are refreshed periodically in the background as new data is added (e.g. every 10GB added or every 50 operations).</small>
            </>
          }
        </>
      ),
      destroyInactiveTabPane: true,
    },
    {
      key: "2",
      label: "Tree View",
      children: (
        <>
          <h3>Browse Backups</h3>
          <OperationTree
            req={new GetOperationsRequest({ repoId: repo.id!, lastN: BigInt(MAX_OPERATION_HISTORY) })}
          />
        </>
      ),
      destroyInactiveTabPane: true,
    },
    {
      key: "3",
      label: "Operation List",
      children: (
        <>
          <h3>Backup Action History</h3>
          <OperationList
            req={new GetOperationsRequest({ repoId: repo.id!, lastN: BigInt(MAX_OPERATION_HISTORY) })}
            showPlan={true}
          />
        </>
      ),
      destroyInactiveTabPane: true,
    },
  ]
  return (
    <>
      <Flex gap="small" align="center" wrap="wrap">
        <Typography.Title>
          {repo.id}
        </Typography.Title>
      </Flex>
      <Flex gap="small" align="center" wrap="wrap">
        <Tooltip title="Indexes the snapshots in the repository. Snapshots are also indexed automatically after each backup.">
          <SpinButton type="default" onClickAsync={handleIndexNow}>
            Index Snapshots
          </SpinButton>
        </Tooltip>
      </Flex>
      <Tabs
        defaultActiveKey={items[0].key}
        items={items}
      />
    </>
  );
};

const StatsTable = ({ stats }: { stats: RepoStats }) => {
  return <Row>
    <Col style={{ paddingRight: "20px" }}>
      <p><strong>Total Size: </strong></p>
      <p><strong>Total Size Uncompressed: </strong></p>
      <p><strong>Blob Count: </strong></p>
      <p><strong>Snapshot Count: </strong></p>
      <p><strong>Compression Ratio: </strong></p>
    </Col>
    <Col>
      <p>{formatBytes(Number(stats.totalSize))}</p>
      <p>{formatBytes(Number(stats.totalUncompressedSize))}</p>
      <p>{Number(stats.totalBlobCount)} blobs</p>
      <p>{Number(stats.snapshotCount)} snapshots</p>
      <p>{Math.round(stats.compressionRatio * 1000) / 1000}</p>
    </Col>
  </Row>
}