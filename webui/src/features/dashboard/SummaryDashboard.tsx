import {
  Card,
  Flex,
  Stack,
  Heading,
  Text,
  Box,
  SimpleGrid,
  Spinner,
  Center,
} from "@chakra-ui/react";
import {
  AccordionRoot,
  AccordionItem,
  AccordionItemTrigger,
  AccordionItemContent,
} from "../../components/ui/accordion";
import React, { useEffect, useState, useMemo } from "react";
import { useConfig } from "../../app/provider";
import {
  SummaryDashboardResponse,
  SummaryDashboardResponse_Summary,
} from "../../../gen/ts/v1/service_pb";
import { backrestService } from "../../api/client";
import { alerts } from "../../components/common/Alerts";
import { formatBytes, formatDuration, formatTime } from "../../lib/formatting";
import {
  Bar,
  BarChart,
  Cell,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { colorForStatus } from "../../api/flowDisplayAggregator";
import { OperationStatus } from "../../../gen/ts/v1/operations_pb";
import { isMobile } from "../../lib/browserUtil";
import { useNavigate } from "react-router";
import { toJsonString } from "@bufbuild/protobuf";
import { ConfigSchema, Multihost } from "../../../gen/ts/v1/config_pb";
import { useSyncStates } from "../../state/peerStates";
import { PeerState } from "../../../gen/ts/v1sync/syncservice_pb";
import { PeerStateConnectionStatusIcon } from "../../components/common/SyncStateIcon";
import * as m from "../../paraglide/messages";
import { DataListRoot, DataListItem } from "../../components/ui/data-list";
import { EmptyState } from "../../components/ui/empty-state";
import { FiDatabase, FiServer } from "react-icons/fi";

export const SummaryDashboard = () => {
  const [config] = useConfig();
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
      } catch (e: any) {
        alerts.error(m.dashboard_error_fetch() + e);
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
    return (
      <Center h="200px">
        <Spinner size="lg" />
      </Center>
    );
  }

  return (
    <Stack gap={8} width="full">
      {/* Multihost summary if any available */}
      <MultihostSummary multihostConfig={config?.multihost || null} />

      {/* Repos and plans section */}
      <Stack gap={4}>
        <Heading size="md">{m.dashboard_repos_title()}</Heading>
        {summaryData && summaryData.repoSummaries.length > 0 ? (
          summaryData.repoSummaries.map((summary) => (
            <SummaryPanel summary={summary} key={summary.id} />
          ))
        ) : (
          <EmptyState title={m.dashboard_repos_empty()} icon={<FiDatabase />} />
        )}
      </Stack>

      <Stack gap={4}>
        <Heading size="md">{m.dashboard_plans_title()}</Heading>
        {summaryData && summaryData.planSummaries.length > 0 ? (
          summaryData.planSummaries.map((summary) => (
            <SummaryPanel summary={summary} key={summary.id} />
          ))
        ) : (
          <EmptyState title={m.dashboard_plans_empty()} icon={<FiServer />} />
        )}
      </Stack>

      {/* System Info Section */}
      <Stack gap={4}>
        <Heading size="md">{m.dashboard_system_info_title()}</Heading>
        <DataListRoot orientation="horizontal">
          <DataListItem
            label={m.dashboard_config_path()}
            value={summaryData.configPath}
          />
          <DataListItem
            label={m.dashboard_data_dir()}
            value={summaryData.dataPath}
          />
        </DataListRoot>

        <AccordionRoot collapsible variant="plain">
          <AccordionItem value="config">
            <AccordionItemTrigger>
              {m.dashboard_config_json()}
            </AccordionItemTrigger>
            <AccordionItemContent>
              <Box
                as="pre"
                p={2}
                bg="gray.900"
                color="white"
                borderRadius="md"
                fontSize="xs"
                overflowX="auto"
              >
                {config &&
                  toJsonString(ConfigSchema, config, { prettySpaces: 2 })}
              </Box>
            </AccordionItemContent>
          </AccordionItem>
        </AccordionRoot>
      </Stack>
    </Stack>
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
      color: "transparent", // transparent instead of white for dark mode support
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
      <Box bg="bg.panel" p={2} boxShadow="md" borderRadius="md" opacity={0.9}>
        <Text fontSize="sm">
          {m.dashboard_backup_tooltip_time({ time: formatTime(entry.time) })}
        </Text>
        <br />
        {isPending ? (
          <Text fontSize="xs" color="gray.500">
            {m.dashboard_backup_tooltip_pending()}
          </Text>
        ) : (
          <Text fontSize="xs" color="gray.500">
            {m.dashboard_backup_tooltip_finished({
              duration: formatDuration(entry.durationMs),
              bytes: formatBytes(entry.bytesAdded),
            })}
          </Text>
        )}
      </Box>
    );
  };

  const DataValue = ({ children }: { children: React.ReactNode }) => (
    <Text fontWeight="medium">{children}</Text>
  );

  const CardInfo = () => (
    <DataListRoot orientation="vertical" size="sm">
      <SimpleGrid columns={2} gapX={8} gapY={2}>
        <DataListItem
          label={m.dashboard_card_backups_30d()}
          value={
            <Flex gap={2}>
              {summary.backupsSuccessLast30days ? (
                <Text color="green.500">
                  {summary.backupsSuccessLast30days +
                    " " +
                    m.dashboard_card_backups_ok()}
                </Text>
              ) : undefined}
              {summary.backupsFailed30days ? (
                <Text color="red.500">
                  {summary.backupsFailed30days +
                    " " +
                    m.dashboard_card_backups_failed()}
                </Text>
              ) : undefined}
              {summary.backupsWarningLast30days ? (
                <Text color="orange.500">
                  {summary.backupsWarningLast30days +
                    " " +
                    m.dashboard_card_backups_warning()}
                </Text>
              ) : undefined}
            </Flex>
          }
        />
        <DataListItem
          label={m.dashboard_card_bytes_scanned_30d()}
          value={
            <DataValue>
              {formatBytes(Number(summary.bytesScannedLast30days))}
            </DataValue>
          }
        />
        <DataListItem
          label={m.dashboard_card_bytes_added_30d()}
          value={
            <DataValue>
              {formatBytes(Number(summary.bytesAddedLast30days))}
            </DataValue>
          }
        />

        {!isMobile() && (
          <>
            <DataListItem
              label={m.dashboard_card_next_backup()}
              value={
                <DataValue>
                  {summary.nextBackupTimeMs
                    ? formatTime(Number(summary.nextBackupTimeMs))
                    : m.dashboard_card_none_scheduled()}
                </DataValue>
              }
            />
            <DataListItem
              label={m.dashboard_card_bytes_scanned_avg()}
              value={
                <DataValue>
                  {formatBytes(Number(summary.bytesScannedAvg))}
                </DataValue>
              }
            />
            <DataListItem
              label={m.dashboard_card_bytes_added_avg()}
              value={
                <DataValue>
                  {formatBytes(Number(summary.bytesAddedAvg))}
                </DataValue>
              }
            />
          </>
        )}
      </SimpleGrid>
    </DataListRoot>
  );

  return (
    <Card.Root width="full">
      <Card.Header>
        <Card.Title>{summary.id}</Card.Title>
      </Card.Header>
      <Card.Body>
        <SimpleGrid columns={[1, 1, 2]} gap={4}>
          <Box>
            <CardInfo />
          </Box>
          <Box height="140px">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={recentBackupsChart}>
                <Bar dataKey="durationMs">
                  {recentBackupsChart.map((entry, index) => (
                    <Cell
                      cursor="pointer"
                      fill={entry.color}
                      key={`${index}`}
                    />
                  ))}
                </Bar>
                <YAxis dataKey="durationMs" hide />
                <XAxis dataKey="idx" hide />
                <Tooltip content={<BackupChartTooltip />} cursor={false} />
              </BarChart>
            </ResponsiveContainer>
          </Box>
        </SimpleGrid>
      </Card.Body>
    </Card.Root>
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
      <PeerStateTile peerState={peerState} key={peerState.peerKeyid} />,
    );
  }

  const authorizedClientTiles: JSX.Element[] = [];
  for (const cfgPeer of multihostConfig?.authorizedClients || []) {
    const peerState = peerStates.get(cfgPeer.keyid);
    if (!peerState) {
      continue;
    }
    authorizedClientTiles.push(
      <PeerStateTile peerState={peerState} key={peerState.peerKeyid} />,
    );
  }

  return (
    <Stack gap={8}>
      {knownHostTiles.length > 0 ? (
        <Stack gap={4}>
          <Heading size="md">{m.dashboard_remote_hosts_title()}</Heading>
          <Stack gap={4}>{knownHostTiles}</Stack>
        </Stack>
      ) : null}
      {authorizedClientTiles.length > 0 ? (
        <Stack gap={4}>
          <Heading size="md">{m.dashboard_remote_clients_title()}</Heading>
          <Stack gap={4}>{authorizedClientTiles}</Stack>
        </Stack>
      ) : null}
    </Stack>
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
    <Card.Root key={peerState.peerKeyid} width="full">
      <Card.Header>
        <Flex justify="space-between" align="center">
          <Card.Title>{peerState.peerInstanceId}</Card.Title>
          <Flex align="center" gap={2}>
            <PeerStateConnectionStatusIcon peerState={peerState} />
          </Flex>
        </Flex>
      </Card.Header>
      <Card.Body>
        <DataListRoot orientation="horizontal">
          <DataListItem
            label={m.dashboard_peer_instance_id()}
            value={peerState.peerInstanceId}
          />
          <DataListItem
            label={m.dashboard_peer_public_key_id()}
            value={peerState.peerKeyid}
          />
          <DataListItem
            label={m.dashboard_peer_last_state_update()}
            value={
              <TimeSinceLastHeartbeat
                lastHeartbeatMillis={Number(peerState.lastHeartbeatMillis)}
              />
            }
          />
        </DataListRoot>
      </Card.Body>
    </Card.Root>
  );
};

const TimeSinceLastHeartbeat = ({
  lastHeartbeatMillis,
}: {
  lastHeartbeatMillis: number;
}) => {
  const [timeSince, setTimeSince] = useState(
    lastHeartbeatMillis ? Date.now() - lastHeartbeatMillis : 0,
  );

  useEffect(() => {
    const interval = setInterval(() => {
      setTimeSince(Date.now() - lastHeartbeatMillis);
    }, 1000);
    return () => clearInterval(interval);
  }, [lastHeartbeatMillis]);

  return (
    <Text>
      {formatTime(lastHeartbeatMillis)} ({formatDuration(timeSince)}{" "}
      {m.dashboard_peer_ago()})
    </Text>
  );
};
