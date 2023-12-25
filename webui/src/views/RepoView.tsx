import React, { useEffect, useState } from "react";
import { Repo } from "../../gen/ts/v1/config_pb";
import { Col, Empty, Flex, Row, Tabs, Typography } from "antd";
import { useRecoilValue } from "recoil";
import { configState } from "../state/config";
import { useAlertApi } from "../components/Alerts";
import { OperationList } from "../components/OperationList";
import { OperationTree } from "../components/OperationTree";
import { MAX_OPERATION_HISTORY } from "../constants";
import { GetOperationsRequest } from "../../gen/ts/v1/service_pb";
import { getOperations } from "../state/oplog";
import { RepoStats } from "../../gen/ts/v1/restic_pb";
import { formatBytes } from "../lib/formatting";

export const RepoView = ({ repo }: React.PropsWithChildren<{ repo: Repo }>) => {
  const alertsApi = useAlertApi()!;
  const [stats, setStats] = useState<RepoStats | null>(null);

  useEffect(() => {
    setStats(null);
    getOperations(new GetOperationsRequest({ repoId: repo.id!, lastN: BigInt(10) })).then((operations) => {
      for (const op of operations) {
        if (op.op.case === "operationStats") {
          const stats = op.op.value.stats;
          if (stats) {
            setStats(stats);
          }
        }
      }
    }).catch((e) => {
      console.error(e);
    });
  }, [repo.id]);


  // Gracefully handle deletions by checking if the plan is still in the config.
  const config = useRecoilValue(configState);
  let repoInConfig = config.repos?.find((p) => p.id === repo.id);
  if (!repoInConfig) {
    return <p>Repo was deleted.</p>;
  }
  repo = repoInConfig;

  if (!stats) {
    return <Empty description="No stats available. Run a backup." />;
  }

  const items = [
    {
      key: "1",
      label: "Stats",
      children: (
        <>
          <h3>Repo Stats</h3>
          <Row>
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
        </>
      ),
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
    },
  ]
  return (
    <>
      <Flex gap="small" align="center" wrap="wrap">
        <Typography.Title>
          {repo.id}
        </Typography.Title>
      </Flex>
      <Tabs
        defaultActiveKey={items[0].key}
        items={items}
      />
    </>
  );
};
