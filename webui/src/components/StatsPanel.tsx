import React, { useEffect, useState } from "react";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from "recharts";
import { formatBytes, formatDate } from "../lib/formatting";
import { Col, Empty, Row } from "antd";
import { Operation, OperationStats } from "../../gen/ts/v1/operations_pb";
import { useAlertApi } from "./Alerts";
import { getOperations } from "../state/oplog";
import {
  GetOperationsRequestSchema,
  OpSelector,
} from "../../gen/ts/v1/service_pb";
import { create } from "@bufbuild/protobuf";

const StatsPanel = ({ selector }: { selector: OpSelector }) => {
  const [operations, setOperations] = useState<Operation[]>([]);
  const alertApi = useAlertApi();

  useEffect(() => {
    const req = create(GetOperationsRequestSchema, {
      selector,
    });

    getOperations(req)
      .then((res) => {
        const ops = res.filter((op) => {
          return op.op.case === "operationStats";
        });
        setOperations(ops);
      })
      .catch((e) => {
        alertApi!.error("Failed to fetch operations: " + e.message);
      });
  }, [JSON.stringify(selector)]);

  if (operations.length === 0) {
    return (
      <Empty description="No stats available. Have you run a stats operation yet?" />
    );
  }

  const statsOperations = operations.filter((v) => {
    return v.op.case === "operationStats" && v.op.value.stats;
  });

  const dataset: {
    time: number;
    totalSizeBytes: number;
    compressionRatio: number;
    snapshotCount: number;
    totalBlobCount: number;
  }[] = statsOperations
    .map((op) => {
      const stats = (op.op.value! as OperationStats).stats!;
      return {
        time: Number(op.unixTimeEndMs!),
        totalSizeBytes: Number(stats.totalSize),
        compressionRatio: Number(stats.compressionRatio),
        snapshotCount: Number(stats.snapshotCount),
        totalBlobCount: Number(stats.totalBlobCount),
      };
    })
    .sort((a, b) => a.time - b.time);

  return (
    <>
      <Row>
        <Col span={12}>
          <ResponsiveContainer width="100%" height={300}>
            <LineChart width={600} height={300} data={dataset}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis
                dataKey="time"
                type="number"
                domain={["dataMin", "dataMax"]}
                tickFormatter={(v) => formatDate(v as number)}
              />
              <YAxis
                yAxisId="left"
                type="number"
                dataKey="totalSizeBytes"
                tickFormatter={(v) => formatBytes(v)}
              />
              <Tooltip labelFormatter={(x) => formatDate(x as number)}
                       formatter={(y) => [formatBytes(y as number), 'Total Size']}/>
              <Legend />
              <Line
                yAxisId="left"
                type="monotone"
                dataKey="totalSizeBytes"
                stroke="#8884d8"
                name="Total Size"
              ></Line>
            </LineChart>
          </ResponsiveContainer>
        </Col>
        <Col span={12}>
          <ResponsiveContainer width="100%" height={300}>
            <LineChart width={600} height={300} data={dataset}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis
                dataKey="time"
                type="number"
                domain={["dataMin", "dataMax"]}
                tickFormatter={(v) => formatDate(v as number)}
              />
              <YAxis
                yAxisId="left"
                type="number"
                dataKey="compressionRatio"
                domain={[0.95, "dataMax + 0.05"]}
              />
              <Tooltip labelFormatter={(v) => formatDate(v as number)} />
              <Legend />
              <Line
                yAxisId="left"
                type="monotone"
                dataKey="compressionRatio"
                stroke="#82ca9d"
                name="Compression Ratio"
              ></Line>
            </LineChart>
          </ResponsiveContainer>
        </Col>
      </Row>
      <Row>
        <Col span={12}>
          <ResponsiveContainer width="100%" height={300}>
            <LineChart width={600} height={300} data={dataset}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis
                dataKey="time"
                type="number"
                domain={["dataMin", "dataMax"]}
                tickFormatter={(v) => formatDate(v as number)}
              />
              <YAxis yAxisId="left" type="number" dataKey="snapshotCount" />
              <Tooltip labelFormatter={(v) => formatDate(v as number)} />
              <Legend />
              <Line
                yAxisId="left"
                type="monotone"
                dataKey="snapshotCount"
                stroke="#ff7300"
                name="Snapshot Count"
              ></Line>
            </LineChart>
          </ResponsiveContainer>
        </Col>
        <Col span={12}>
          <ResponsiveContainer width="100%" height={300}>
            <LineChart width={600} height={300} data={dataset}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis
                dataKey="time"
                type="number"
                domain={["dataMin", "dataMax"]}
                tickFormatter={(v) => formatDate(v as number)}
              />
              <YAxis yAxisId="left" type="number" dataKey="totalBlobCount" />
              <Tooltip labelFormatter={(v) => formatDate(v as number)} />
              <Legend />
              <Line
                yAxisId="left"
                type="monotone"
                dataKey="totalBlobCount"
                stroke="#00BBBB"
                name="Total Blob Count"
              ></Line>
            </LineChart>
          </ResponsiveContainer>
        </Col>
      </Row>
    </>
  );
};

export default StatsPanel;
