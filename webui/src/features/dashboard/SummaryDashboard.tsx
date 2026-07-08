import {
  Box,
  Card,
  Center,
  Flex,
  Heading,
  SimpleGrid,
  Spinner,
  Stack,
  Text,
} from "@chakra-ui/react";
import { motion } from "framer-motion";
import React, { useCallback, useEffect, useMemo, useState } from "react";
import { FiCheck, FiDatabase, FiRefreshCw, FiServer } from "react-icons/fi";
import { LuTriangle, LuX } from "react-icons/lu";
import { useNavigate } from "react-router";
import { toJsonString } from "@bufbuild/protobuf";
import { ConfigSchema, Multihost } from "../../../gen/ts/v1/config_pb";
import {
  Operation,
  OperationEvent,
  OperationStatus,
} from "../../../gen/ts/v1/operations_pb";
import {
  GetOperationsRequestSchema,
  OpSelectorSchema,
  SummaryDashboardResponse,
  SummaryDashboardResponse_Summary,
} from "../../../gen/ts/v1/service_pb";
import { PeerState } from "../../../gen/ts/v1sync/syncservice_pb";
import { create } from "@bufbuild/protobuf";
import { backrestService } from "../../api/client";
import {
  getOperations,
  subscribeToOperations,
  unsubscribeFromOperations,
} from "../../api/oplog";
import { matchSelector } from "../../api/logState";
import { alerts } from "../../components/common/Alerts";
import { PeerStateConnectionStatusIcon } from "../../components/common/SyncStateIcon";
import {
  AccordionItem,
  AccordionItemContent,
  AccordionItemTrigger,
  AccordionRoot,
} from "../../components/ui/accordion";
import { DataListItem, DataListRoot } from "../../components/ui/data-list";
import { EmptyState } from "../../components/ui/empty-state";
import { formatBytes, formatDuration, formatTime } from "../../lib/formatting";
import { useConfig } from "../../app/provider";
import { useSyncStates } from "../../state/peerStates";
import * as m from "../../paraglide/messages";
import { HistoryStrip } from "./HistoryStrip";

// ─── helpers ────────────────────────────────────────────────────────────────

function prettyPlanId(id: string): string {
  return id.replace(/[-_]/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

/** Relative "X ago" string.  ms = unix epoch in ms. */
function agoText(ms: number): string {
  if (!ms) return m.dashboard_time_never();
  const s = Math.floor((Date.now() - ms) / 1000);
  if (s < 45) return m.dashboard_time_just_now();
  if (s < 90) return m.dashboard_time_a_minute_ago();
  if (s < 3600)
    return m.dashboard_time_minutes_ago({ count: Math.floor(s / 60) });
  if (s < 5400) return m.dashboard_time_an_hour_ago();
  if (s < 86400) {
    const h = Math.floor(s / 3600);
    return h === 1
      ? m.dashboard_time_hour_ago({ count: h })
      : m.dashboard_time_hours_ago({ count: h });
  }
  if (s < 172800) return m.dashboard_time_yesterday();
  return m.dashboard_time_days_ago({ count: Math.floor(s / 86400) });
}

/** "in X" relative string for future times.  ms = unix epoch in ms. */
function untilText(ms: number): string | null {
  if (!ms) return null;
  const s = Math.floor((ms - Date.now()) / 1000);
  if (s <= 0) return m.dashboard_time_due_now();
  if (s < 5400)
    return m.dashboard_time_in_minutes({
      count: Math.max(1, Math.round(s / 60)),
    });
  if (s < 172800)
    return m.dashboard_time_in_hours({ count: Math.round(s / 3600) });
  return m.dashboard_time_in_days({ count: Math.round(s / 86400) });
}

function scheduleText(maxFrequencyHours?: number): string | null {
  if (!maxFrequencyHours) return null;
  if (maxFrequencyHours < 1)
    return m.dashboard_schedule_every_minutes({
      count: Math.round(maxFrequencyHours * 60),
    });
  if (maxFrequencyHours === 1) return m.dashboard_schedule_hourly();
  if (maxFrequencyHours === 24) return m.dashboard_schedule_daily();
  return m.dashboard_schedule_every_hours({ count: maxFrequencyHours });
}

function retentionText(r: {
  hourly: number;
  daily: number;
  weekly: number;
  monthly: number;
  yearly: number;
}): string | null {
  const buckets: [number, (p: { count: number }) => string][] = [
    [r.hourly, m.dashboard_retention_hourly],
    [r.daily, m.dashboard_retention_daily],
    [r.weekly, m.dashboard_retention_weekly],
    [r.monthly, m.dashboard_retention_monthly],
    [r.yearly, m.dashboard_retention_yearly],
  ];
  const parts = buckets
    .filter(([count]) => count > 0)
    .map(([count, fmt]) => fmt({ count }));
  return parts.length
    ? m.dashboard_card_retention({ parts: parts.join(", ") })
    : null;
}

// Derived state for one plan card
type PlanState = "ok" | "warn" | "err" | "run" | "idle";

function planState(
  latestStatus: OperationStatus | undefined,
  running: boolean,
): PlanState {
  if (running) return "run";
  if (latestStatus === OperationStatus.STATUS_SUCCESS) return "ok";
  if (latestStatus === OperationStatus.STATUS_WARNING) return "warn";
  if (
    latestStatus === OperationStatus.STATUS_ERROR ||
    latestStatus === OperationStatus.STATUS_SYSTEM_CANCELLED
  )
    return "err";
  return "idle";
}

const STATE_COLORS: Record<PlanState, string> = {
  ok: "green.500",
  warn: "orange.400",
  err: "red.500",
  run: "blue.500",
  idle: "gray.400",
};

const STATE_BG: Record<PlanState, string> = {
  ok: "green.50",
  warn: "orange.50",
  err: "red.50",
  run: "blue.50",
  idle: "gray.100",
};

// Worst-wins ordering used to derive the hero state across all plans.
const STATE_SEVERITY: Record<PlanState, number> = {
  err: 3,
  warn: 2,
  run: 1,
  ok: 0,
  idle: -1,
};

const STATE_LABEL: Record<PlanState, () => string> = {
  ok: m.dashboard_state_label_ok,
  warn: m.dashboard_state_label_warn,
  err: m.dashboard_state_label_err,
  run: m.dashboard_state_label_run,
  idle: m.dashboard_state_label_idle,
};

// Status fields shared by plan cards, repo cards, and the hero banner,
// derived from a summary's most recent backup.
interface SummaryStatus {
  latestTs: number;
  running: boolean;
  state: PlanState;
  color: string;
}

function summaryStatus(
  summary: SummaryDashboardResponse_Summary,
): SummaryStatus {
  const rb = summary.recentBackups;
  const latestStatus = rb?.status[0];
  const latestTs = Number(rb?.timestampMs[0] ?? 0);
  const running =
    latestStatus === OperationStatus.STATUS_INPROGRESS ||
    latestStatus === OperationStatus.STATUS_PENDING;
  const state = planState(latestStatus, running);
  return { latestTs, running, state, color: STATE_COLORS[state] };
}

// ─── Progress bar (animated shimmer) ─────────────────────────────────────────

const ProgressBar = ({ pct }: { pct: number }) => (
  <Box h="7px" borderRadius="full" bg="bg.muted" overflow="hidden" my={3}>
    <motion.div
      initial={{ width: 0 }}
      animate={{ width: `${Math.max(2, pct)}%` }}
      transition={{ duration: 0.6, ease: "easeOut" }}
      style={{
        height: "100%",
        borderRadius: "999px",
        background:
          "linear-gradient(90deg, var(--chakra-colors-blue-500) 0%, var(--chakra-colors-blue-300) 100%)",
      }}
    />
  </Box>
);

// ─── Hero banner ──────────────────────────────────────────────────────────────

const HERO_ICON: Record<PlanState, React.ReactNode> = {
  ok: <FiCheck strokeWidth={2.4} />,
  run: <FiRefreshCw strokeWidth={2.4} />,
  warn: <LuTriangle strokeWidth={2.4} />,
  err: <LuX strokeWidth={2.4} />,
  idle: <FiDatabase strokeWidth={2.4} />,
};

const HERO_TITLE: Record<PlanState, () => string> = {
  ok: m.dashboard_hero_ok,
  run: m.dashboard_hero_run,
  warn: m.dashboard_hero_warn,
  err: m.dashboard_hero_err,
  idle: m.dashboard_hero_idle,
};

interface HeroStats {
  state: PlanState;
  newestMs: number;
  nextMs: number | null;
}

// Worst state across all plans (a running backup takes over the hero), the
// newest backup time, and the soonest upcoming backup time.
function heroStats(plans: SummaryDashboardResponse_Summary[]): HeroStats {
  let state: PlanState = "idle";
  let newestMs = 0;
  let nextMs: number | null = null;
  let anyRunning = false;

  for (const summary of plans) {
    const status = summaryStatus(summary);
    anyRunning ||= status.running;
    newestMs = Math.max(newestMs, status.latestTs);
    if (STATE_SEVERITY[status.state] > STATE_SEVERITY[state]) {
      state = status.state;
    }
    const next = Number(summary.nextBackupTimeMs ?? 0);
    if (next > 0 && (nextMs === null || next < nextMs)) nextMs = next;
  }

  if (anyRunning) state = "run";
  return { state, newestMs, nextMs };
}

const HeroBanner = ({ state, newestMs, nextMs }: HeroStats) => {
  const nextIn = nextMs ? untilText(nextMs) : null;

  return (
    <Card.Root borderRadius="2xl" shadow="sm" mb={6}>
      <Card.Body py={6} px={7}>
        <Flex align="center" gap={5}>
          <Flex
            flexShrink={0}
            w="60px"
            h="60px"
            borderRadius="full"
            bg={STATE_BG[state]}
            color={STATE_COLORS[state]}
            align="center"
            justify="center"
            fontSize="2xl"
          >
            {HERO_ICON[state]}
          </Flex>
          <Box>
            <Text
              fontSize="23px"
              fontWeight="650"
              letterSpacing="-0.02em"
              lineHeight={1.2}
            >
              {HERO_TITLE[state]()}
            </Text>
            <Text fontSize="14.5px" color="fg.muted" mt="3px">
              {newestMs ? (
                <Text as="span" fontWeight="semibold" color="fg.default">
                  {m.dashboard_hero_last_backup({ ago: agoText(newestMs) })}
                </Text>
              ) : (
                m.dashboard_hero_no_backups()
              )}
              {nextIn && ` · ${m.dashboard_hero_next({ when: nextIn })}`}
            </Text>
          </Box>
        </Flex>
      </Card.Body>
    </Card.Root>
  );
};

// ─── Live progress hook ───────────────────────────────────────────────────────

interface LiveProgress {
  pct: number;
  done: number;
  total: number;
}

// Pulls live backup progress out of a backup operation, if it carries any.
function progressFromOp(op: Operation): LiveProgress | null {
  if (op.op.case !== "operationBackup") return null;
  const entry = op.op.value.lastStatus?.entry;
  if (entry?.case !== "status") return null;
  return {
    pct: Math.round(entry.value.percentDone * 100),
    done: Number(entry.value.bytesDone),
    total: Number(entry.value.totalBytes),
  };
}

const useLiveProgress = (
  planId: string,
  running: boolean,
  onFinished?: () => void,
) => {
  const [progress, setProgress] = useState<LiveProgress | null>(null);

  useEffect(() => {
    if (!running) {
      setProgress(null);
      return;
    }
    let cancelled = false;
    let finished = false;
    const selector = create(OpSelectorSchema, { planId });

    const apply = (op: Operation) => {
      if (op.op.case !== "operationBackup") return;
      // Once the backup leaves the in-progress/pending state it's done: clear the
      // live bar and refresh the summary so the card leaves its running state.
      if (
        op.status !== OperationStatus.STATUS_INPROGRESS &&
        op.status !== OperationStatus.STATUS_PENDING
      ) {
        if (finished) return;
        finished = true;
        setProgress(null);
        onFinished?.();
        return;
      }
      const p = progressFromOp(op);
      if (p) setProgress(p);
    };

    // Seed with the in-flight operation so the bar isn't empty until the first event.
    getOperations(create(GetOperationsRequestSchema, { lastN: 1n, selector }))
      .then((ops) => {
        if (!cancelled && ops[0]) apply(ops[0]);
      })
      .catch(() => {}); // best-effort; live updates follow

    // Then stay current via the shared operation-event subscription.
    const handler = (event?: OperationEvent) => {
      if (
        event?.event.case !== "createdOperations" &&
        event?.event.case !== "updatedOperations"
      ) {
        return;
      }
      for (const op of event.event.value.operations) {
        if (matchSelector(selector, op)) apply(op);
      }
    };
    subscribeToOperations(handler);
    return () => {
      cancelled = true;
      unsubscribeFromOperations(handler);
    };
  }, [planId, running, onFinished]);

  return progress;
};

// ─── Shared card building blocks ─────────────────────────────────────────────

const CardTitle = ({ children }: { children: React.ReactNode }) => (
  <Text fontSize="16px" fontWeight="640" letterSpacing="-0.01em">
    {children}
  </Text>
);

const StatusDot = ({ color, pulsing }: { color: string; pulsing?: boolean }) =>
  pulsing ? (
    <motion.div
      animate={{ opacity: [1, 0.4, 1], scale: [1, 0.82, 1] }}
      transition={{ duration: 1.6, repeat: Infinity, ease: "easeInOut" }}
      style={{
        width: 9,
        height: 9,
        borderRadius: "50%",
        background: "var(--chakra-colors-blue-500)",
      }}
    />
  ) : (
    <Box w="9px" h="9px" borderRadius="full" bg={color} />
  );

// Large colored state label with a muted "X ago" beside it.
const StatusLine = ({ status }: { status: SummaryStatus }) => (
  <Flex align="baseline" gap={2} mt={4} mb={1}>
    <Text
      fontSize="20px"
      fontWeight="660"
      letterSpacing="-0.02em"
      color={status.color}
    >
      {STATE_LABEL[status.state]()}
    </Text>
    {status.latestTs > 0 && !status.running && (
      <Text fontSize="13px" color="fg.muted">
        {agoText(status.latestTs)}
      </Text>
    )}
  </Flex>
);

// One "Label value" pair in a card's meta row.
const MetaItem = ({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) => (
  <Box fontSize="12.5px" color="fg.muted">
    {label}{" "}
    <Text as="span" fontWeight="600" color="fg.default">
      {children}
    </Text>
  </Box>
);

// ─── Plan card ────────────────────────────────────────────────────────────────

const PlanCard = ({
  summary,
}: {
  summary: SummaryDashboardResponse_Summary;
}) => {
  const [config] = useConfig();
  const status = summaryStatus(summary);
  const { running } = status;

  const progress = useLiveProgress(summary.id, running);

  // Plan config fields
  const planCfg = useMemo(
    () => config?.plans.find((p) => p.id === summary.id),
    [config, summary.id],
  );
  const schedLine = scheduleText(
    planCfg?.schedule?.schedule.case === "maxFrequencyHours"
      ? planCfg.schedule.schedule.value
      : undefined,
  );
  const destLabel = planCfg?.repo ?? "";
  const nextMs = Number(summary.nextBackupTimeMs ?? 0);
  const protectedBytes = Number(summary.protectedBytes);
  const lastUploadBytes = Number(summary.recentBackups?.bytesAdded[0] ?? 0);

  let retLine: string | null = null;
  if (planCfg?.retention?.policy.case === "policyTimeBucketed") {
    retLine = retentionText(planCfg.retention.policy.value);
  }

  return (
    <Card.Root
      borderRadius="2xl"
      shadow="sm"
      overflow="hidden"
      position="relative"
    >
      <Card.Body px={5} py={5}>
        {/* Title row */}
        <Flex justify="space-between" align="flex-start" gap={3}>
          <Box>
            <CardTitle>{prettyPlanId(summary.id)}</CardTitle>
            {schedLine && (
              <Text fontSize="12.5px" color="fg.muted" mt="2px">
                {schedLine}
              </Text>
            )}
          </Box>
          <Box mt="6px" flexShrink={0}>
            <StatusDot color={status.color} pulsing={running} />
          </Box>
        </Flex>

        <StatusLine status={status} />

        {/* Progress bar when running */}
        {running && (
          <>
            <ProgressBar pct={progress?.pct ?? 0} />
            <Text fontSize="12.5px" color="fg.muted">
              {progress
                ? progress.total > 0
                  ? m.dashboard_card_progress_detail({
                      pct: progress.pct,
                      done: formatBytes(progress.done),
                      total: formatBytes(progress.total),
                    })
                  : m.dashboard_card_progress_pct({ pct: progress.pct })
                : m.dashboard_card_progress_scanning()}
            </Text>
          </>
        )}

        {/* Meta row (not shown while running) */}
        {!running && (
          <Flex flexWrap="wrap" gap="4px 18px" mt={3}>
            {nextMs > 0 && (
              <MetaItem label={m.dashboard_card_next_run()}>
                {untilText(nextMs) ?? m.dashboard_time_soon()}
              </MetaItem>
            )}
            {destLabel && (
              <MetaItem label={m.dashboard_card_destination()}>
                {destLabel}
              </MetaItem>
            )}
            {protectedBytes > 0 && (
              <MetaItem label={m.dashboard_card_protected()}>
                {formatBytes(protectedBytes)}
              </MetaItem>
            )}
            {lastUploadBytes > 0 && (
              <MetaItem label={m.dashboard_card_last_upload()}>
                {formatBytes(lastUploadBytes)}
              </MetaItem>
            )}
          </Flex>
        )}

        {/* 30-day history strip */}
        <HistoryStrip buckets={summary.historyLast30days} />

        {/* Retention footer */}
        {retLine && (
          <Box
            mt={3}
            pt={3}
            borderTop="1px solid"
            borderColor="border.subtle"
            fontSize="12px"
            color="fg.muted"
          >
            {retLine}
          </Box>
        )}
      </Card.Body>
    </Card.Root>
  );
};

// ─── Repo card (slimmed — no schedule/next-run) ───────────────────────────────

const RepoCard = ({
  summary,
}: {
  summary: SummaryDashboardResponse_Summary;
}) => {
  const status = summaryStatus(summary);
  const protectedBytes = Number(summary.protectedBytes);
  const bytesAdded30d = Number(summary.bytesAddedLast30days);

  return (
    <Card.Root borderRadius="2xl" shadow="sm">
      <Card.Body px={5} py={5}>
        <Flex justify="space-between" align="flex-start" gap={3}>
          <CardTitle>{summary.id}</CardTitle>
          <Box mt="6px" flexShrink={0}>
            <StatusDot color={status.color} />
          </Box>
        </Flex>

        <StatusLine status={status} />

        <Flex flexWrap="wrap" gap="4px 18px" mt={3}>
          <Box fontSize="12.5px" color="fg.muted">
            {m.dashboard_repo_window_30d()}{" "}
            <Text as="span" fontWeight="600" color="green.500">
              {summary.backupsSuccessLast30days
                ? m.dashboard_repo_ok({
                    count: Number(summary.backupsSuccessLast30days),
                  })
                : ""}
            </Text>
            {summary.backupsFailed30days ? (
              <Text as="span" fontWeight="600" color="red.500" ml={2}>
                {m.dashboard_repo_failed({
                  count: Number(summary.backupsFailed30days),
                })}
              </Text>
            ) : null}
          </Box>
          {protectedBytes > 0 && (
            <MetaItem label={m.dashboard_card_protected()}>
              {formatBytes(protectedBytes)}
            </MetaItem>
          )}
          {bytesAdded30d > 0 && (
            <MetaItem label={m.dashboard_repo_added()}>
              {formatBytes(bytesAdded30d)}
            </MetaItem>
          )}
        </Flex>

        {/* 30-day history strip */}
        <HistoryStrip buckets={summary.historyLast30days} />
      </Card.Body>
    </Card.Root>
  );
};

// ─── Recent activity timeline ─────────────────────────────────────────────────

interface ActivityRow {
  planId: string;
  flowId: bigint;
  status: OperationStatus;
  timestampMs: number;
  durationMs: number;
  bytesAdded: number;
}

function rowLabel(status: OperationStatus): string {
  if (status === OperationStatus.STATUS_SUCCESS)
    return m.dashboard_activity_row_completed();
  if (status === OperationStatus.STATUS_WARNING)
    return m.dashboard_activity_row_completed_warnings();
  if (status === OperationStatus.STATUS_ERROR)
    return m.dashboard_activity_row_failed();
  return m.dashboard_activity_row_ran();
}

const RecentActivity = ({
  summaries,
}: {
  summaries: SummaryDashboardResponse_Summary[];
}) => {
  const [config] = useConfig();

  // planId -> destination repo ID, derived from config.
  const destByPlan = useMemo(() => {
    const map = new Map<string, string>();
    for (const plan of config?.plans ?? []) {
      map.set(plan.id, plan.repo ?? "");
    }
    return map;
  }, [config]);

  const rows = useMemo<ActivityRow[]>(() => {
    const all: ActivityRow[] = [];
    for (const s of summaries) {
      const rb = s.recentBackups;
      if (!rb) continue;
      for (let i = 0; i < rb.timestampMs.length; i++) {
        const status = rb.status[i];
        if (
          status === OperationStatus.STATUS_INPROGRESS ||
          status === OperationStatus.STATUS_PENDING
        )
          continue;
        all.push({
          planId: s.id,
          flowId: rb.flowId[i],
          status,
          timestampMs: Number(rb.timestampMs[i]),
          durationMs: Number(rb.durationMs[i]),
          bytesAdded: Number(rb.bytesAdded[i]),
        });
      }
    }
    all.sort((a, b) => b.timestampMs - a.timestampMs);
    return all.slice(0, 8);
  }, [summaries]);

  if (rows.length === 0) {
    return (
      <Card.Root borderRadius="2xl" shadow="sm">
        <Card.Body>
          <Text color="fg.muted" textAlign="center" py={8} fontSize="sm">
            {m.dashboard_activity_no_backups()}
          </Text>
        </Card.Body>
      </Card.Root>
    );
  }

  return (
    <Card.Root borderRadius="2xl" shadow="sm" overflow="hidden">
      {rows.map((row, i) => {
        const dotColor = STATE_COLORS[planState(row.status, false)];
        const dest = destByPlan.get(row.planId);

        // Muted secondary line: relative + absolute time, duration, destination.
        const detailParts: string[] = [agoText(row.timestampMs)];
        detailParts.push(formatTime(row.timestampMs));
        if (row.durationMs > 0) {
          detailParts.push(
            m.dashboard_activity_took({
              duration: formatDuration(row.durationMs),
            }),
          );
        }
        if (dest) detailParts.push(dest);

        return (
          <Box
            key={`${row.planId}-${row.flowId}`}
            px={5}
            py="13px"
            borderTop={i === 0 ? "none" : "1px solid"}
            borderColor="border.subtle"
          >
            <Flex align="center" gap="13px">
              <Box
                w="8px"
                h="8px"
                borderRadius="full"
                bg={dotColor}
                flexShrink={0}
              />
              <Box flex={1} minW={0}>
                <Flex align="baseline" gap="7px" minW={0}>
                  <Text fontSize="14px" fontWeight="550" truncate>
                    {prettyPlanId(row.planId)}
                  </Text>
                  <Text
                    fontSize="12.5px"
                    fontWeight="600"
                    color={dotColor}
                    flexShrink={0}
                  >
                    {rowLabel(row.status)}
                  </Text>
                </Flex>
                <Text fontSize="12.5px" color="fg.muted" truncate>
                  {detailParts.join(" · ")}
                </Text>
              </Box>
              {row.bytesAdded > 0 && (
                <Text
                  fontSize="12.5px"
                  color="fg.muted"
                  fontVariantNumeric="tabular-nums"
                  flexShrink={0}
                >
                  +{formatBytes(row.bytesAdded)}
                </Text>
              )}
            </Flex>
          </Box>
        );
      })}
    </Card.Root>
  );
};

// ─── Root component ───────────────────────────────────────────────────────────

export const SummaryDashboard = () => {
  const [config] = useConfig();
  const navigate = useNavigate();
  const [summaryData, setSummaryData] =
    useState<SummaryDashboardResponse | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      if (document.hidden) return;
      try {
        const data = await backrestService.getSummaryDashboard({});
        setSummaryData(data);
      } catch (e: unknown) {
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
    if (!config) return;
    if (
      config.repos.length === 0 &&
      config.plans.length === 0 &&
      config.multihost?.knownHosts.length === 0 &&
      config.multihost?.authorizedClients.length === 0
    ) {
      navigate("/getting-started");
    }
  }, [config, navigate]);

  if (!summaryData) {
    return (
      <Center h="200px">
        <Spinner size="lg" />
      </Center>
    );
  }

  const plans = summaryData.planSummaries;
  const hero = heroStats(plans);

  return (
    <Stack gap={8} width="full">
      {/* Multihost summary */}
      <MultihostSummary multihostConfig={config?.multihost ?? null} />

      {/* Hero */}
      {plans.length > 0 && <HeroBanner {...hero} />}

      {/* Plan cards */}
      {plans.length > 0 && (
        <Stack gap={4}>
          <Heading size="md">{m.dashboard_plans_title()}</Heading>
          <SimpleGrid columns={{ base: 1, md: 2 }} gap={4}>
            {plans.map((s) => (
              <PlanCard key={s.id} summary={s} />
            ))}
          </SimpleGrid>
        </Stack>
      )}

      {plans.length === 0 && (
        <Stack gap={4}>
          <Heading size="md">{m.dashboard_plans_title()}</Heading>
          <EmptyState title={m.dashboard_plans_empty()} icon={<FiServer />} />
        </Stack>
      )}

      {/* Recent activity */}
      {plans.length > 0 && (
        <Stack gap={4}>
          <Heading size="md">{m.dashboard_activity_title()}</Heading>
          <RecentActivity summaries={plans} />
        </Stack>
      )}

      {/* Repos */}
      <Stack gap={4}>
        <Heading size="md">{m.dashboard_repos_title()}</Heading>
        {summaryData.repoSummaries.length > 0 ? (
          <SimpleGrid columns={{ base: 1, md: 2 }} gap={4}>
            {summaryData.repoSummaries.map((s) => (
              <RepoCard key={s.id} summary={s} />
            ))}
          </SimpleGrid>
        ) : (
          <EmptyState title={m.dashboard_repos_empty()} icon={<FiDatabase />} />
        )}
      </Stack>

      {/* System Info */}
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

// ─── Multihost ────────────────────────────────────────────────────────────────

const MultihostSummary = ({
  multihostConfig,
}: {
  multihostConfig: Multihost | null;
}) => {
  const [config] = useConfig();
  const allPeerStates = useSyncStates();
  const peerStates = useMemo(() => {
    const map = new Map<string, PeerState>();
    for (const state of allPeerStates) {
      map.set(state.peerKeyid, state);
    }
    return map;
  }, [allPeerStates]);

  const sharedReposByHost = useMemo(() => {
    const map = new Map<string, string[]>();
    for (const repo of config?.repos ?? []) {
      if (repo.originInstanceId) {
        const repos = map.get(repo.originInstanceId) ?? [];
        repos.push(repo.id);
        map.set(repo.originInstanceId, repos);
      }
    }
    return map;
  }, [config?.repos]);

  const knownHostTiles: React.JSX.Element[] = [];
  for (const cfgPeer of multihostConfig?.knownHosts ?? []) {
    const peerState = peerStates.get(cfgPeer.keyid);
    if (!peerState) continue;
    knownHostTiles.push(
      <PeerStateTile
        peerState={peerState}
        sharedRepoIds={sharedReposByHost.get(peerState.peerInstanceId)}
        key={peerState.peerKeyid}
      />,
    );
  }

  const authorizedClientTiles: React.JSX.Element[] = [];
  for (const cfgPeer of multihostConfig?.authorizedClients ?? []) {
    const peerState = peerStates.get(cfgPeer.keyid);
    if (!peerState) continue;
    authorizedClientTiles.push(
      <PeerStateTile peerState={peerState} key={peerState.peerKeyid} />,
    );
  }

  return (
    <Stack gap={8}>
      {knownHostTiles.length > 0 && (
        <Stack gap={4}>
          <Heading size="md">{m.dashboard_remote_hosts_title()}</Heading>
          <Stack gap={4}>{knownHostTiles}</Stack>
        </Stack>
      )}
      {authorizedClientTiles.length > 0 && (
        <Stack gap={4}>
          <Heading size="md">{m.dashboard_remote_clients_title()}</Heading>
          <Stack gap={4}>{authorizedClientTiles}</Stack>
        </Stack>
      )}
    </Stack>
  );
};

const PeerStateTile = ({
  peerState,
  sharedRepoIds,
}: {
  peerState: PeerState;
  sharedRepoIds?: string[];
}) => {
  const tickState = useState(1);
  useEffect(() => {
    const interval = setInterval(() => {
      tickState[1]((prev) => prev + 1);
    }, 1000);
    return () => clearInterval(interval);
  }, [peerState.peerKeyid, peerState.lastHeartbeatMillis, tickState[1]]); // eslint-disable-line react-hooks/exhaustive-deps

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
                lastHeartbeatMillis={Number(peerState.lastHeartbeatMillis ?? 0)}
              />
            }
          />
          {peerState.knownRepos.length > 0 && (
            <DataListItem
              label={m.dashboard_peer_shared_repos()}
              value={
                <Flex gap={1} flexWrap="wrap">
                  {peerState.knownRepos.map((repo) => (
                    <Box
                      key={repo.id}
                      px={2}
                      py={0.5}
                      bg="bg.muted"
                      borderRadius="sm"
                      fontSize="xs"
                    >
                      {repo.id}
                    </Box>
                  ))}
                </Flex>
              }
            />
          )}
          {sharedRepoIds && sharedRepoIds.length > 0 && (
            <DataListItem
              label={m.dashboard_peer_shared_repos()}
              value={
                <Flex gap={1} flexWrap="wrap">
                  {sharedRepoIds.map((repoId) => (
                    <Box
                      key={repoId}
                      px={2}
                      py={0.5}
                      bg="bg.muted"
                      borderRadius="sm"
                      fontSize="xs"
                    >
                      {repoId}
                    </Box>
                  ))}
                </Flex>
              }
            />
          )}
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
