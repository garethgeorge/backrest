import {
  Button,
  Card,
  Col,
  Collapse,
  Descriptions,
  Divider,
  Empty,
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
import {
  formatBytes,
  formatDate,
  formatDuration,
  formatTime,
} from "../lib/formatting";
import {
  Bar,
  BarChart,
  Cell,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { colorForStatus } from "../state/flowdisplayaggregator";
import { OperationStatus } from "../../gen/ts/v1/operations_pb";
import { isMobile } from "../lib/browserutil";

export const SummaryDashboard = () => {
  const config = useConfig()[0];
  const setContent = useSetContent();
  const alertApi = useAlertApi()!;

  const [summaryData, setSummaryData] =
    useState<SummaryDashboardResponse | null>();

  const showGettingStarted = async () => {
    const { GettingStartedGuide } = await import("./GettingStartedGuide");
    setContent(
      <React.Suspense fallback={<Spin />}>
        <GettingStartedGuide />
      </React.Suspense>,
      [
        {
          title: "Getting Started",
        },
      ]
    );
  };

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

  useEffect(() => {
    if (!config) {
      return;
    }

    if (config.repos.length === 0 && config.plans.length === 0) {
      showGettingStarted();
    }
  }, [config]);

  if (!summaryData) {
    return <Spin />;
  }

  return (
    <>
      <Flex gap={16} vertical>
        <Typography.Title level={3}>Repos</Typography.Title>
        {summaryData && summaryData.repoSummaries.length > 0 ? (
          summaryData.repoSummaries.map((summary) => (
            <SummaryPanel summary={summary} />
          ))
        ) : (
          <Empty description="No repos found" />
        )}
        <Typography.Title level={3}>Plans</Typography.Title>
        {summaryData && summaryData.planSummaries.length > 0 ? (
          summaryData.planSummaries.map((summary) => (
            <SummaryPanel summary={summary} />
          ))
        ) : (
          <Empty description="No plans found" />
        )}
        <Divider />
        <Typography.Title level={3}>System Info</Typography.Title>
        <Descriptions
          layout="vertical"
          column={2}
          items={[
            {
              key: 1,
              label: "Config Path",
              children: summaryData.configPath,
            },
            {
              key: 2,
              label: "Data Directory",
              children: summaryData.dataPath,
            },
          ]}
        />
      </Flex>
    </>
  );
};

const SummaryPanel = ({
  summary,
}: {
  summary: SummaryDashboardResponse_Summary;
}) => {
  const recentBackupsChart: {
    idx: number;
    time: number;
    durationMs: number;
    color: string;
    bytesAdded: number;
  }[] = [];
  const recentBackups = summary.recentBackups!;
  for (let i = 0; i < recentBackups.timestampMs.length; i++) {
    const color = colorForStatus(recentBackups.status[i]);
    recentBackupsChart.push({
      idx: i,
      time: Number(recentBackups.timestampMs[i]),
      durationMs: Number(recentBackups.durationMs[i]),
      color: color,
      bytesAdded: Number(recentBackups.bytesAdded[i]),
    });
  }
  while (recentBackupsChart.length < 60) {
    recentBackupsChart.push({
      idx: recentBackupsChart.length,
      time: 0,
      durationMs: 0,
      color: "white",
      bytesAdded: 0,
    });
  }

  const BackupChartTooltip = ({ active, payload, label }: any) => {
    const idx = Number(label);

    const entry = recentBackupsChart[idx];
    if (!entry || entry.idx > recentBackups.timestampMs.length) {
      return null;
    }

    const isPending =
      recentBackups.status[idx] === OperationStatus.STATUS_PENDING;

    return (
      <Card style={{ opacity: 0.9 }} size="small">
        <Typography.Text>Backup at {formatTime(entry.time)}</Typography.Text>{" "}
        <br />
        {isPending ? (
          <Typography.Text type="secondary">
            Scheduled, waiting.
          </Typography.Text>
        ) : (
          <Typography.Text type="secondary">
            Took {formatDuration(entry.durationMs)}, added{" "}
            {formatBytes(entry.bytesAdded)}
          </Typography.Text>
        )}
      </Card>
    );
  };

  const cardInfo: { key: number; label: string; children: React.ReactNode }[] =
    [];

  cardInfo.push(
    {
      key: 1,
      label: "Backups (30d)",
      children: (
        <>
          {summary.backupsSuccessLast30days && (
            <Typography.Text type="success" style={{ marginRight: "5px" }}>
              {summary.backupsSuccessLast30days + ""} ok
            </Typography.Text>
          )}
          {summary.backupsFailed30days && (
            <Typography.Text type="danger">
              {summary.backupsFailed30days + ""} failed
            </Typography.Text>
          )}
          {summary.backupsWarningLast30days && (
            <Typography.Text type="warning">
              {summary.backupsWarningLast30days + ""} warning
            </Typography.Text>
          )}
        </>
      ),
    },
    {
      key: 2,
      label: "Bytes Scanned (30d)",
      children: formatBytes(Number(summary.bytesScannedLast30days)),
    },
    {
      key: 3,
      label: "Bytes Added (30d)",
      children: formatBytes(Number(summary.bytesAddedLast30days)),
    }
  );

  // check if mobile layout
  if (!isMobile()) {
    cardInfo.push(
      {
        key: 4,
        label: "Next Scheduled Backup",
        children: summary.nextBackupTimeMs
          ? formatTime(Number(summary.nextBackupTimeMs))
          : "None Scheduled",
      },
      {
        key: 5,
        label: "Bytes Scanned Avg",
        children: formatBytes(Number(summary.bytesScannedAvg)),
      },
      {
        key: 6,
        label: "Bytes Added Avg",
        children: formatBytes(Number(summary.bytesAddedAvg)),
      }
    );
  }

  return (
    <Card title={summary.id} style={{ width: "100%" }}>
      <Row gutter={16} key={1}>
        <Col span={10}>
          <Descriptions
            layout="vertical"
            column={3}
            items={cardInfo}
          ></Descriptions>
        </Col>
        <Col span={14}>
          <ResponsiveContainer width="100%" height={140}>
            <BarChart data={recentBackupsChart}>
              <Bar dataKey="durationMs">
                {recentBackupsChart.map((entry, index) => (
                  <Cell cursor="pointer" fill={entry.color} key={`${index}`} />
                ))}
              </Bar>
              <YAxis dataKey="durationMs" hide />
              <XAxis dataKey="idx" hide />
              <Tooltip content={<BackupChartTooltip />} cursor={false} />
            </BarChart>
          </ResponsiveContainer>
        </Col>
      </Row>
    </Card>
  );
};
