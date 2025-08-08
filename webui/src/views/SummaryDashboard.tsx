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
import React, { useEffect, useState, useMemo } from "react";
import { useConfig } from "../components/ConfigProvider";
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
import { useNavigate } from "react-router";
import { toJsonString } from "@bufbuild/protobuf";
import { ConfigSchema, Multihost } from "../../gen/ts/v1/config_pb";
import { useSyncStates } from "../state/peerstates";
import { PeerState } from "../../gen/ts/v1/syncservice_pb";
import { PeerStateConnectionStatusIcon } from "../components/SyncStateIcon";
import { last } from "lodash";

export const SummaryDashboard = () => {
  const config = useConfig()[0];
  const alertApi = useAlertApi()!;
  const navigate = useNavigate();

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
        alertApi.error("Failed to fetch summary data: " + e);
      }
    };

    fetchData();

    document.addEventListener("visibilitychange", fetchData);
    const interval = setInterval(fetchData, 60000);
    return () => {
      document.removeEventListener("visibilitychange", fetchData);
      clearInterval(interval);
    };
  }, []);

  useEffect(() => {
    if (!config) {
      return;
    }

    if (
      config.repos.length === 0 &&
      config.plans.length === 0 &&
      config.multihost?.knownHosts.length === 0 &&
      config.multihost?.authorizedClients.length === 0
    ) {
      navigate("/getting-started");
    }
  }, [config]);

  if (!summaryData) {
    return <Spin className="summary-dashboard-spin" />;
  }

  return (
    <>
      <Flex gap={16} vertical>
        {/* Multihost summary if any available */}
        <MultihostSummary multihostConfig={config?.multihost || null} />

        {/* Repos and plans section */}
        <Typography.Title level={3}>Repos</Typography.Title>
        {summaryData && summaryData.repoSummaries.length > 0 ? (
          summaryData.repoSummaries.map((summary) => (
            <SummaryPanel summary={summary} key={summary.id} />
          ))
        ) : (
          <Empty description="No repos found" />
        )}
        <Typography.Title level={3}>Plans</Typography.Title>
        {summaryData && summaryData.planSummaries.length > 0 ? (
          summaryData.planSummaries.map((summary) => (
            <SummaryPanel summary={summary} key={summary.id} />
          ))
        ) : (
          <Empty description="No plans found" />
        )}

        {/* System Info Section */}
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
        <Collapse
          size="small"
          items={[
            {
              label: "Config as JSON",
              children: (
                <pre>
                  {config &&
                    toJsonString(ConfigSchema, config, { prettySpaces: 2 })}
                </pre>
              ),
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
      <Card style={{ opacity: 0.9 }} size="small" key={label}>
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
          {summary.backupsSuccessLast30days ? (
            <Typography.Text type="success" style={{ marginRight: "5px" }}>
              {summary.backupsSuccessLast30days + ""} ok
            </Typography.Text>
          ) : undefined}
          {summary.backupsFailed30days ? (
            <Typography.Text type="danger" style={{ marginRight: "5px" }}>
              {summary.backupsFailed30days + ""} failed
            </Typography.Text>
          ) : undefined}
          {summary.backupsWarningLast30days ? (
            <Typography.Text type="warning" style={{ marginRight: "5px" }}>
              {summary.backupsWarningLast30days + ""} warning
            </Typography.Text>
          ) : undefined}
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

const MultihostSummary = ({
  multihostConfig,
}: {
  multihostConfig: Multihost | null;
}) => {
  const allPeerStates = useSyncStates();
  const peerStates = useMemo(() => {
    const map = new Map<string, PeerState>();
    for (const state of allPeerStates) {
      map.set(state.peerKeyid, state);
    }
    return map;
  }, [allPeerStates]);

  const knownHostTiles: JSX.Element[] = [];
  for (const cfgPeer of multihostConfig?.knownHosts || []) {
    const peerState = peerStates.get(cfgPeer.keyid);
    if (!peerState) {
      continue;
    }
    knownHostTiles.push(
      <PeerStateTile peerState={peerState} key={peerState.peerKeyid} />
    );
  }

  const authorizedClientTiles: JSX.Element[] = [];
  for (const cfgPeer of multihostConfig?.authorizedClients || []) {
    const peerState = peerStates.get(cfgPeer.keyid);
    if (!peerState) {
      continue;
    }
    authorizedClientTiles.push(
      <PeerStateTile peerState={peerState} key={peerState.peerKeyid} />
    );
  }

  return (
    <>
      {knownHostTiles.length > 0 ? (
        <>
          <Typography.Title level={3}>Remote Hosts</Typography.Title>
          <Flex gap={16} vertical>
            {knownHostTiles}
          </Flex>
        </>
      ) : null}
      {authorizedClientTiles.length > 0 ? (
        <>
          <Typography.Title level={3}>Remote Clients</Typography.Title>
          <Flex gap={16} vertical>
            {authorizedClientTiles}
          </Flex>
        </>
      ) : null}
    </>
  );
};

const PeerStateTile = ({ peerState }: { peerState: PeerState }) => {
  const state = useState(1);
  useEffect(() => {
    // Force rerender every second to update the last heartbeat time
    const interval = setInterval(() => {
      state[1]((prev) => prev + 1);
    }, 1000);
    return () => clearInterval(interval);
  }, [peerState.peerKeyid, peerState.lastHeartbeatMillis, state[1]]);

  return (
    <Card
      key={peerState.peerKeyid}
      title={
        <>
          {peerState.peerInstanceId}
          <div
            style={{
              position: "absolute",
              top: "8px",
              right: "8px",
              display: "flex",
              alignItems: "center",
              gap: "8px",
            }}
          >
            <PeerStateConnectionStatusIcon peerState={peerState} />
          </div>
        </>
      }
      style={{ marginBottom: "16px" }}
    >
      <Descriptions
        layout="vertical"
        column={2}
        items={[
          {
            key: 1,
            label: "Instance ID",
            children: peerState.peerInstanceId,
          },
          {
            key: 2,
            label: "Public Key ID",
            children: peerState.peerKeyid,
          },
          {
            key: 3,
            label: "Last State Update",
            children: (
              <TimeSinceLastHeartbeat
                lastHeartbeatMillis={Number(peerState.lastHeartbeatMillis)}
              />
            ),
          },
        ]}
      />
    </Card>
  );
};

const TimeSinceLastHeartbeat = ({
  lastHeartbeatMillis,
}: {
  lastHeartbeatMillis: number;
}) => {
  const [timeSince, setTimeSince] = useState(
    lastHeartbeatMillis ? Date.now() - lastHeartbeatMillis : 0
  );

  useEffect(() => {
    const interval = setInterval(() => {
      setTimeSince(Date.now() - lastHeartbeatMillis);
    }, 1000);
    return () => clearInterval(interval);
  }, [lastHeartbeatMillis]);

  return (
    formatTime(lastHeartbeatMillis) + " (" + formatDuration(timeSince) + " ago)"
  );
};
