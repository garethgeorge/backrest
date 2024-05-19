import React, { useContext, useEffect, useState } from "react";
import { Repo } from "../../gen/ts/v1/config_pb";
import {
  Col,
  Empty,
  Flex,
  Row,
  Spin,
  TabsProps,
  Tabs,
  Tooltip,
  Typography,
  Button,
} from "antd";
import { OperationList } from "../components/OperationList";
import { OperationTree } from "../components/OperationTree";
import { MAX_OPERATION_HISTORY, STATS_OPERATION_HISTORY } from "../constants";
import { GetOperationsRequest, OpSelector } from "../../gen/ts/v1/service_pb";
import {
  BackupInfo,
  BackupInfoCollector,
  getOperations,
  shouldHideStatus,
} from "../state/oplog";
import { formatBytes, formatDate, formatTime } from "../lib/formatting";
import {
  Operation,
  OperationStats,
  OperationStatus,
} from "../../gen/ts/v1/operations_pb";
import { backrestService } from "../api";
import { StringValue } from "@bufbuild/protobuf";
import { SpinButton } from "../components/SpinButton";
import { useConfig } from "../components/ConfigProvider";
import { useAlertApi } from "../components/Alerts";
import { LineChart } from "@mui/x-charts";
import { useShowModal } from "../components/ModalManager";

export const RepoView = ({ repo }: React.PropsWithChildren<{ repo: Repo }>) => {
  const [config, setConfig] = useConfig();
  const showModal = useShowModal();

  // Task handlers
  const handleIndexNow = async () => {
    await backrestService.indexSnapshots(new StringValue({ value: repo.id! }));
  };

  const handleStatsNow = async () => {
    await backrestService.stats(new StringValue({ value: repo.id! }));
  };

  // Gracefully handle deletions by checking if the plan is still in the config.
  let repoInConfig = config?.repos?.find((r) => r.id === repo.id);
  if (!repoInConfig) {
    return (
      <>
        Repo was deleted
        <pre>{JSON.stringify(config, null, 2)}</pre>
      </>
    );
  }
  repo = repoInConfig;

  const items = [
    {
      key: "1",
      label: "Tree View",
      children: (
        <>
          <h3>Browse Backups</h3>
          <OperationTree
            req={
              new GetOperationsRequest({
                selector: new OpSelector({
                  repoId: repo.id!,
                }),
                lastN: BigInt(MAX_OPERATION_HISTORY),
              })
            }
          />
        </>
      ),
      destroyInactiveTabPane: true,
    },
    {
      key: "2",
      label: "Operation List",
      children: (
        <>
          <h3>Backup Action History</h3>
          <OperationList
            req={
              new GetOperationsRequest({
                selector: new OpSelector({
                  repoId: repo.id!,
                }),
                lastN: BigInt(MAX_OPERATION_HISTORY),
              })
            }
            showPlan={true}
            filter={(op) => !shouldHideStatus(op.status)}
          />
        </>
      ),
      destroyInactiveTabPane: true,
    },
    {
      key: "3",
      label: "Stats",
      children: (
        <>
          <StatsPanel repoId={repo.id!} />
        </>
      ),
      destroyInactiveTabPane: true,
    },
  ];
  return (
    <>
      <Flex gap="small" align="center" wrap="wrap">
        <Typography.Title>{repo.id}</Typography.Title>
      </Flex>
      <Flex gap="small" align="center" wrap="wrap">
        <Tooltip title="Advanced users: open a restic shell to run commands on the repository. Re-index snapshots to reflect any changes in Backrest.">
          <Button
            type="default"
            onClick={async () => {
              const { RunCommandModal } = await import("./RunCommandModal");
              showModal(<RunCommandModal repoId={repo.id!} />);
            }}
          >
            Run Command
          </Button>
        </Tooltip>

        <Tooltip title="Indexes the snapshots in the repository. Snapshots are also indexed automatically after each backup.">
          <SpinButton type="default" onClickAsync={handleIndexNow}>
            Index Snapshots
          </SpinButton>
        </Tooltip>

        <Tooltip title="Runs restic stats on the repository, this may be a slow operation">
          <SpinButton type="default" onClickAsync={handleStatsNow}>
            Compute Stats
          </SpinButton>
        </Tooltip>
      </Flex>
      <Tabs defaultActiveKey={items[0].key} items={items} />
    </>
  );
};

const StatsPanel = ({ repoId }: { repoId: string }) => {
  const [operations, setOperations] = useState<Operation[]>([]);
  const alertApi = useAlertApi();

  useEffect(() => {
    if (!repoId) {
      return;
    }

    const backupCollector = new BackupInfoCollector((op) => {
      return (
        op.status === OperationStatus.STATUS_SUCCESS &&
        op.op.case === "operationStats" &&
        !!op.op.value.stats
      );
    });

    getOperations(
      new GetOperationsRequest({
        repoId: repoId,
        lastN: BigInt(MAX_OPERATION_HISTORY),
      })
    )
      .then((ops) => {
        backupCollector.bulkAddOperations(ops);

        const operations = backupCollector
          .getAll()
          .flatMap((b) => b.operations);
        operations.sort((a, b) => {
          return Number(b.unixTimeEndMs - a.unixTimeEndMs);
        });
        setOperations(operations);
      })
      .catch((e) => {
        alertApi!.error("Failed to fetch operations: " + e.message);
      });
  }, [repoId]);

  if (operations.length === 0) {
    return (
      <Empty description="No stats available. Have you run a prune operation yet?" />
    );
  }

  const dataset: {
    time: number;
    totalSizeMb: number;
    compressionRatio: number;
    snapshotCount: number;
    totalBlobCount: number;
  }[] = operations.map((op) => {
    const stats = (op.op.value! as OperationStats).stats!;
    return {
      time: Number(op.unixTimeEndMs!),
      totalSizeMb: Number(stats.totalSize) / 1000000,
      compressionRatio: Number(stats.compressionRatio),
      snapshotCount: Number(stats.snapshotCount),
      totalBlobCount: Number(stats.totalBlobCount),
    };
  });

  const minTime = Math.min(...dataset.map((d) => d.time));
  const maxTime = Math.max(...dataset.map((d) => d.time));

  return (
    <>
      <Row>
        <Col span={12}>
          <LineChart
            xAxis={[
              {
                dataKey: "time",
                valueFormatter: (v) => formatDate(v as number),
                min: minTime,
                max: maxTime,
              },
            ]}
            series={[
              {
                dataKey: "totalSizeMb",
                label: "Total Size",
                valueFormatter: (v: any) =>
                  formatBytes((v * 1000000) as number),
              },
            ]}
            height={300}
            dataset={dataset}
          />

          <LineChart
            xAxis={[
              {
                dataKey: "time",
                valueFormatter: (v) => formatDate(v as number),
                min: minTime,
                max: maxTime,
              },
            ]}
            series={[
              {
                dataKey: "compressionRatio",
                label: "Compression Ratio",
              },
            ]}
            height={300}
            dataset={dataset}
          />
        </Col>
        <Col span={12}>
          <LineChart
            xAxis={[
              {
                dataKey: "time",
                valueFormatter: (v) => formatDate(v as number),
                min: minTime,
                max: maxTime,
              },
            ]}
            series={[
              {
                dataKey: "snapshotCount",
                label: "Snapshot Count",
              },
            ]}
            height={300}
            dataset={dataset}
          />

          <LineChart
            xAxis={[
              {
                dataKey: "time",
                valueFormatter: (v) => formatDate(v as number),
                min: minTime,
                max: maxTime,
              },
            ]}
            series={[
              {
                dataKey: "totalBlobCount",
                label: "Blob Count",
              },
            ]}
            height={300}
            dataset={dataset}
          />
        </Col>
      </Row>
    </>
  );
};
