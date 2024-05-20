import React, { useEffect, useState } from "react";
import { LineChart } from "@mui/x-charts/LineChart";
import { formatBytes, formatDate } from "../lib/formatting";
import { Col, Empty, Row } from "antd";
import {
  Operation,
  OperationStats,
  OperationStatus,
} from "../../gen/ts/v1/operations_pb";
import { useAlertApi } from "./Alerts";
import { BackupInfoCollector, getOperations } from "../state/oplog";
import { MAX_OPERATION_HISTORY } from "../constants";
import { GetOperationsRequest, OpSelector } from "../../gen/ts/v1/service_pb";

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
        selector: new OpSelector({
          repoId: repoId,
        }),
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
            width={600}
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

export default StatsPanel;
