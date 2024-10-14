import {
  Card,
  Col,
  Collapse,
  Descriptions,
  Divider,
  Flex,
  Row,
  Spin,
  Typography,
} from "antd";
import React, { useEffect, useState } from "react";
import { useConfig } from "../components/ConfigProvider";
import { useSetContent } from "./MainContentArea";
import {
  SummaryDashboardResponse,
  SummaryDashboardResponse_Summary,
} from "../../gen/ts/v1/service_pb";
import { backrestService } from "../api";
import { useAlertApi } from "../components/Alerts";
import { formatBytes, formatDate } from "../lib/formatting";
import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

export const SummaryDashboard = () => {
  const config = useConfig()[0];
  const setContent = useSetContent();
  const alertApi = useAlertApi()!;

  const [summaryData, setSummaryData] =
    useState<SummaryDashboardResponse | null>();

  useEffect(() => {
    // Fetch summary data
    const fetchData = async () => {
      // check if the tab is in the foreground
      if (document.hidden) {
        return;
      }

      try {
        const data = await backrestService.getSummaryDashboard({});
        setSummaryData(data);
      } catch (e) {
        alertApi.error("Failed to fetch summary data", e);
      }
    };

    fetchData();

    const interval = setInterval(fetchData, 60000);
    return () => clearInterval(interval);
  }, []);

  return (
    <>
      <Typography.Title level={2}>Dashboard</Typography.Title>
      <Divider />

      <Flex gap={16} vertical>
        <Typography.Title level={3}>Repos</Typography.Title>
        {summaryData ? (
          summaryData.repoSummaries.map((summary) => (
            <SummaryPanel summary={summary} />
          ))
        ) : (
          <Spin />
        )}
        <Typography.Title level={3}>Plans</Typography.Title>
        {summaryData ? (
          summaryData.planSummaries.map((summary) => (
            <SummaryPanel summary={summary} />
          ))
        ) : (
          <Spin />
        )}
      </Flex>
    </>
  );
};

const SummaryPanel = ({
  summary,
}: {
  summary: SummaryDashboardResponse_Summary;
}) => {
  const bytesAddedDataset = summary.bytesAdded.map((val) => {
    return {
      time: Number(val.timestampMillis),
      value: Number(val.value),
    };
  });

  bytesAddedDataset.sort((a, b) => a.time - b.time);

  return (
    <Card title={summary.id} style={{ width: "100%" }}>
      <Row gutter={16} key={1}>
        <Col span={10}>
          <Descriptions
            layout="vertical"
            items={[
              {
                key: 1,
                label: "Backups (90d)",
                children: "" + summary.backupsLast90days,
              },
              {
                key: 2,
                label: "Bytes Scanned (90d)",
                children: formatBytes(Number(summary.bytesScannedLast90days)),
              },
              {
                key: 3,
                label: "Bytes Added (90d)",
                children: formatBytes(Number(summary.bytesAddedLast90days)),
              },
            ]}
          ></Descriptions>
        </Col>
        <Col span={14}>
          <ResponsiveContainer width="100%" height={200}>
            <LineChart data={bytesAddedDataset}>
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
                dataKey="value"
                tick={false}
              />
              <Line
                yAxisId="left"
                type="monotone"
                dataKey="value"
                stroke="#00BBBB"
                name="Bytes Added"
              ></Line>
              <Tooltip labelFormatter={(v) => formatBytes(v as number)} />
            </LineChart>
          </ResponsiveContainer>
        </Col>
      </Row>
    </Card>
  );
};
