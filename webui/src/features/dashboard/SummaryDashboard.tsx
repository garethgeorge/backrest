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
import React, { useEffect, useMemo, useState } from "react";
import { FiCheck, FiDatabase, FiRefreshCw, FiServer } from "react-icons/fi";
import { LuTriangle, LuX } from "react-icons/lu";
import { useNavigate } from "react-router";
import { toJsonString } from "@bufbuild/protobuf";
import { ConfigSchema, Multihost } from "../../../gen/ts/v1/config_pb";
import { OperationStatus } from "../../../gen/ts/v1/operations_pb";
import {
  SummaryDashboardResponse,
  SummaryDashboardResponse_Summary,
} from "../../../gen/ts/v1/service_pb";
import { PeerState } from "../../../gen/ts/v1sync/syncservice_pb";
import { backrestService } from "../../api/client";
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
import { buildDayBuckets, HistoryStrip } from "./HistoryStrip";

// ─── helpers ────────────────────────────────────────────────────────────────

function prettyPlanId(id: string): string {
  return id.replace(/[-_]/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

/** Relative "X ago" string.  ms = unix epoch in ms. */
function agoText(ms: number): string {
  if (!ms) return "never";
  const s = Math.floor((Date.now() - ms) / 1000);
  if (s < 45) return "just now";
  if (s < 90) return "a minute ago";
  if (s < 3600) return `${Math.floor(s / 60)} min ago`;
  if (s < 5400) return "an hour ago";
  if (s < 86400) {
    const h = Math.floor(s / 3600);
    return `${h} hour${h > 1 ? "s" : ""} ago`;
  }
  if (s < 172800) return "yesterday";
  const dd = Math.floor(s / 86400);
  return `${dd} day${dd > 1 ? "s" : ""} ago`;
}

/** "in X" relative string for future times.  ms = unix epoch in ms. */
function untilText(ms: number): string | null {
  if (!ms) return null;
  const s = Math.floor((ms - Date.now()) / 1000);
  if (s <= 0) return "due now";
  if (s < 5400) return `in ${Math.max(1, Math.round(s / 60))} min`;
  if (s < 172800) return `in ${Math.round(s / 3600)} hours`;
  return `in ${Math.round(s / 86400)} days`;
}

function scheduleText(maxFrequencyHours?: number): string | null {
  if (!maxFrequencyHours) return null;
  if (maxFrequencyHours < 1)
    return `Runs every ${Math.round(maxFrequencyHours * 60)} min`;
  if (maxFrequencyHours === 1) return "Runs hourly";
  if (maxFrequencyHours === 24) return "Runs daily";
  return `Runs every ${maxFrequencyHours}h`;
}

function repoDestLabel(uri: string): string {
  const u = uri.toLowerCase();
  if (u.includes("backblaze") || u.startsWith("s3:") || u.startsWith("b2:"))
    return "Backblaze B2";
  if (u.startsWith("rclone:")) {
    const rem = u.split(":")[1] ?? "";
    if (rem.includes("pcloud")) return "pCloud";
    return rem.charAt(0).toUpperCase() + rem.slice(1);
  }
  if (u.startsWith("sftp:") || u.includes("hetzner")) return "Hetzner";
  return uri.replace(/^[a-z0-9]+:/, "");
}

function retentionText(
  hourly: number,
  daily: number,
  weekly: number,
  monthly: number,
  yearly: number,
): string | null {
  const parts: string[] = [];
  if (hourly) parts.push(`${hourly} hourly`);
  if (daily) parts.push(`${daily} daily`);
  if (weekly) parts.push(`${weekly} weekly`);
  if (monthly) parts.push(`${monthly} monthly`);
  if (yearly) parts.push(`${yearly} yearly`);
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

function stateLabelText(state: PlanState): string {
  if (state === "ok") return m.dashboard_state_label_ok();
  if (state === "warn") return m.dashboard_state_label_warn();
  if (state === "err") return m.dashboard_state_label_err();
  if (state === "run") return m.dashboard_state_label_run();
  return m.dashboard_state_label_idle();
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

type HeroState = "ok" | "run" | "warn" | "err" | "idle";

const HERO_ICON: Record<HeroState, React.ReactNode> = {
  ok: <FiCheck strokeWidth={2.4} />,
  run: <FiRefreshCw strokeWidth={2.4} />,
  warn: <LuTriangle strokeWidth={2.4} />,
  err: <LuX strokeWidth={2.4} />,
  idle: <FiDatabase strokeWidth={2.4} />,
};

function heroTitleText(state: HeroState): string {
  if (state === "ok") return m.dashboard_hero_ok();
  if (state === "run") return m.dashboard_hero_run();
  if (state === "warn") return m.dashboard_hero_warn();
  if (state === "err") return m.dashboard_hero_err();
  return m.dashboard_hero_idle();
}

const HeroBanner = ({
  state,
  newestMs,
  nextMs,
}: {
  state: HeroState;
  newestMs: number;
  nextMs: number | null;
}) => {
  const color = STATE_COLORS[state as PlanState] ?? "gray.400";
  const bgColor = STATE_BG[state as PlanState] ?? "gray.100";
  const subParts: React.ReactNode[] = [];
  if (newestMs) {
    subParts.push(
      <Text as="span" fontWeight="semibold" color="fg.default" key="last">
        {m.dashboard_hero_last_backup({ ago: agoText(newestMs) })}
      </Text>,
    );
  } else {
    subParts.push(
      <Text as="span" key="last">
        {m.dashboard_hero_no_backups()}
      </Text>,
    );
  }
  if (nextMs) {
    const u = untilText(nextMs);
    if (u) {
      subParts.push(
        <Text as="span" key="sep">
          {" · "}
        </Text>,
      );
      subParts.push(
        <Text as="span" key="next">
          {m.dashboard_hero_next({ when: u })}
        </Text>,
      );
    }
  }

  return (
    <Card.Root borderRadius="2xl" shadow="sm" mb={6}>
      <Card.Body py={6} px={7}>
        <Flex align="center" gap={5}>
          <Flex
            flexShrink={0}
            w="60px"
            h="60px"
            borderRadius="full"
            bg={bgColor}
            color={color}
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
              {heroTitleText(state)}
            </Text>
            <Text fontSize="14.5px" color="fg.muted" mt="3px">
              {subParts}
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

const useLiveProgress = (planId: string, running: boolean) => {
  const [progress, setProgress] = useState<LiveProgress | null>(null);

  useEffect(() => {
    if (!running) {
      setProgress(null);
      return;
    }
    let cancelled = false;
    const poll = async () => {
      try {
        const res = await backrestService.getOperations({
          lastN: 1n,
          selector: { planId },
        });
        if (cancelled) return;
        const op = res.operations[0];
        const opBackup =
          op?.op.case === "operationBackup" ? op.op.value : undefined;
        const s = opBackup?.lastStatus;
        if (s?.entry.case === "status") {
          const e = s.entry.value;
          setProgress({
            pct: Math.round(e.percentDone * 100),
            done: Number(e.bytesDone),
            total: Number(e.totalBytes),
          });
        }
      } catch {
        // ignore; progress is best-effort
      }
    };
    poll();
    const id = setInterval(poll, 3000);
    return () => {
      cancelled = true;
      clearInterval(id);
    };
  }, [planId, running]);

  return progress;
};

// ─── Per-plan operations cache ────────────────────────────────────────────────

interface PlanOpsData {
  buckets: ReturnType<typeof buildDayBuckets>["buckets"];
  protectedBytes: number;
  loading: boolean;
}

const usePlanOps = (planIds: string[]) => {
  const [data, setData] = useState<Map<string, PlanOpsData>>(new Map());

  useEffect(() => {
    if (planIds.length === 0) return;

    // Mark all as loading
    setData((prev) => {
      const next = new Map(prev);
      for (const id of planIds) {
        if (!next.has(id))
          next.set(id, { buckets: [], protectedBytes: 0, loading: true });
      }
      return next;
    });

    const loadPlan = async (planId: string) => {
      try {
        const res = await backrestService.getOperations({
          lastN: 300n,
          selector: { planId },
        });
        const backupOps = res.operations.filter(
          (op) => op.op.case === "operationBackup",
        );

        // Build day buckets
        const { buckets } = buildDayBuckets(
          backupOps.map((op) => ({
            unixTimeStartMs: op.unixTimeStartMs,
            status: op.status,
          })),
        );

        // Protected = totalBytesProcessed of most recent SUCCESS/WARNING op
        let protectedBytes = 0;
        let protectedMs = 0;
        for (const op of backupOps) {
          const t = Number(op.unixTimeStartMs);
          if (
            t > protectedMs &&
            (op.status === OperationStatus.STATUS_SUCCESS ||
              op.status === OperationStatus.STATUS_WARNING)
          ) {
            const opBackup =
              op.op.case === "operationBackup" ? op.op.value : undefined;
            const lastStatus = opBackup?.lastStatus;
            if (lastStatus?.entry.case === "summary") {
              protectedMs = t;
              protectedBytes = Number(
                lastStatus.entry.value.totalBytesProcessed,
              );
            }
          }
        }

        setData((prev) => {
          const next = new Map(prev);
          next.set(planId, { buckets, protectedBytes, loading: false });
          return next;
        });
      } catch {
        setData((prev) => {
          const next = new Map(prev);
          next.set(planId, { buckets: [], protectedBytes: 0, loading: false });
          return next;
        });
      }
    };

    for (const id of planIds) loadPlan(id);
  }, [planIds.join(",")]); // eslint-disable-line react-hooks/exhaustive-deps

  return data;
};

// ─── Plan card ────────────────────────────────────────────────────────────────

const PlanCard = ({
  summary,
  opsData,
}: {
  summary: SummaryDashboardResponse_Summary;
  opsData: PlanOpsData | undefined;
}) => {
  const [config] = useConfig();
  const rb = summary.recentBackups;
  const latestStatus = rb?.status[0];
  const latestTs = Number(rb?.timestampMs[0] ?? 0);
  const running =
    latestStatus === OperationStatus.STATUS_INPROGRESS ||
    latestStatus === OperationStatus.STATUS_PENDING;
  const state = planState(latestStatus, running);
  const color = STATE_COLORS[state];

  const progress = useLiveProgress(summary.id, running);

  // Plan config fields
  const planCfg = useMemo(
    () => config?.plans.find((p) => p.id === summary.id),
    [config, summary.id],
  );
  const repoCfg = useMemo(
    () => config?.repos.find((r) => r.id === planCfg?.repo),
    [config, planCfg],
  );
  const schedLine = scheduleText(
    planCfg?.schedule?.schedule.case === "maxFrequencyHours"
      ? planCfg.schedule.schedule.value
      : undefined,
  );
  const destLabel = repoCfg
    ? repoDestLabel(repoCfg.uri)
    : (planCfg?.repo ?? "");
  const nextMs = Number(summary.nextBackupTimeMs ?? 0);
  const lastUploadBytes = Number(rb?.bytesAdded[0] ?? 0);

  let retLine: string | null = null;
  if (planCfg?.retention?.policy.case === "policyTimeBucketed") {
    const r = planCfg.retention.policy.value;
    retLine = retentionText(r.hourly, r.daily, r.weekly, r.monthly, r.yearly);
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
            <Text fontSize="16px" fontWeight="640" letterSpacing="-0.01em">
              {prettyPlanId(summary.id)}
            </Text>
            {schedLine && (
              <Text fontSize="12.5px" color="fg.muted" mt="2px">
                {schedLine}
              </Text>
            )}
          </Box>
          {/* Status dot */}
          <Box mt="6px" flexShrink={0}>
            {running ? (
              <motion.div
                animate={{ opacity: [1, 0.4, 1], scale: [1, 0.82, 1] }}
                transition={{
                  duration: 1.6,
                  repeat: Infinity,
                  ease: "easeInOut",
                }}
                style={{
                  width: 9,
                  height: 9,
                  borderRadius: "50%",
                  background: "var(--chakra-colors-blue-500)",
                }}
              />
            ) : (
              <Box w="9px" h="9px" borderRadius="full" bg={color} />
            )}
          </Box>
        </Flex>

        {/* Big status line */}
        <Flex align="baseline" gap={2} mt={4} mb={1}>
          <Text
            fontSize="20px"
            fontWeight="660"
            letterSpacing="-0.02em"
            color={color}
          >
            {stateLabelText(state)}
          </Text>
          {latestTs > 0 && !running && (
            <Text fontSize="13px" color="fg.muted">
              {agoText(latestTs)}
            </Text>
          )}
        </Flex>

        {/* Progress bar when running */}
        {running && (
          <>
            <ProgressBar pct={progress?.pct ?? 0} />
            <Text fontSize="12.5px" color="fg.muted">
              {progress
                ? progress.total > 0
                  ? `${progress.pct}% · ${formatBytes(progress.done)} of ${formatBytes(progress.total)}`
                  : `${progress.pct}% done`
                : "Scanning…"}
            </Text>
          </>
        )}

        {/* Meta row (not shown while running) */}
        {!running && (
          <Flex flexWrap="wrap" gap="4px 18px" mt={3}>
            {nextMs > 0 && (
              <Box fontSize="12.5px" color="fg.muted">
                {m.dashboard_card_next_run()}{" "}
                <Text as="span" fontWeight="600" color="fg.default">
                  {untilText(nextMs) ?? "soon"}
                </Text>
              </Box>
            )}
            {destLabel && (
              <Box fontSize="12.5px" color="fg.muted">
                {m.dashboard_card_destination()}{" "}
                <Text as="span" fontWeight="600" color="fg.default">
                  {destLabel}
                </Text>
              </Box>
            )}
            {opsData && opsData.protectedBytes > 0 && (
              <Box fontSize="12.5px" color="fg.muted">
                {m.dashboard_card_protected()}{" "}
                <Text as="span" fontWeight="600" color="fg.default">
                  {formatBytes(opsData.protectedBytes)}
                </Text>
              </Box>
            )}
            {lastUploadBytes > 0 && (
              <Box fontSize="12.5px" color="fg.muted">
                {m.dashboard_card_last_upload()}{" "}
                <Text as="span" fontWeight="600" color="fg.default">
                  {formatBytes(lastUploadBytes)}
                </Text>
              </Box>
            )}
          </Flex>
        )}

        {/* 30-day history strip */}
        {opsData && !opsData.loading && (
          <HistoryStrip buckets={opsData.buckets} />
        )}
        {opsData?.loading && (
          <Box mt={4}>
            <Spinner size="xs" />
          </Box>
        )}

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
  const rb = summary.recentBackups;
  const latestStatus = rb?.status[0];
  const latestTs = Number(rb?.timestampMs[0] ?? 0);
  const running =
    latestStatus === OperationStatus.STATUS_INPROGRESS ||
    latestStatus === OperationStatus.STATUS_PENDING;
  const state = planState(latestStatus, running);
  const color = STATE_COLORS[state];

  return (
    <Card.Root borderRadius="2xl" shadow="sm">
      <Card.Body px={5} py={5}>
        <Flex justify="space-between" align="flex-start" gap={3}>
          <Text fontSize="16px" fontWeight="640" letterSpacing="-0.01em">
            {summary.id}
          </Text>
          <Box mt="6px" flexShrink={0}>
            <Box w="9px" h="9px" borderRadius="full" bg={color} />
          </Box>
        </Flex>

        <Flex align="baseline" gap={2} mt={4}>
          <Text
            fontSize="20px"
            fontWeight="660"
            letterSpacing="-0.02em"
            color={color}
          >
            {stateLabelText(state)}
          </Text>
          {latestTs > 0 && !running && (
            <Text fontSize="13px" color="fg.muted">
              {agoText(latestTs)}
            </Text>
          )}
        </Flex>

        <Flex flexWrap="wrap" gap="4px 18px" mt={3}>
          <Box fontSize="12.5px" color="fg.muted">
            30d{" "}
            <Text as="span" fontWeight="600" color="green.500">
              {summary.backupsSuccessLast30days
                ? `${Number(summary.backupsSuccessLast30days)} ok`
                : ""}
            </Text>
            {summary.backupsFailed30days ? (
              <Text as="span" fontWeight="600" color="red.500" ml={2}>
                {Number(summary.backupsFailed30days)} failed
              </Text>
            ) : null}
          </Box>
          {Number(summary.bytesAddedLast30days) > 0 && (
            <Box fontSize="12.5px" color="fg.muted">
              Added{" "}
              <Text as="span" fontWeight="600" color="fg.default">
                {formatBytes(Number(summary.bytesAddedLast30days))}
              </Text>
            </Box>
          )}
        </Flex>
      </Card.Body>
    </Card.Root>
  );
};

// ─── Recent activity timeline ─────────────────────────────────────────────────

interface ActivityRow {
  planId: string;
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
        const stateKey = planState(row.status, false);
        const dotColor = STATE_COLORS[stateKey];
        return (
          <Box
            key={i}
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
                <Text fontSize="14px" fontWeight="550" truncate>
                  {m.dashboard_activity_row_label({
                    plan: prettyPlanId(row.planId),
                    status: rowLabel(row.status),
                  })}
                </Text>
                <Text fontSize="12.5px" color="fg.muted">
                  {agoText(row.timestampMs)}
                  {row.durationMs > 0 &&
                    ` · took ${formatDuration(row.durationMs)}`}
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
  }, [config]); // eslint-disable-line react-hooks/exhaustive-deps

  // Stable array of plan IDs for the ops cache
  const planIds = useMemo(
    () => (summaryData?.planSummaries ?? []).map((s) => s.id),
    [summaryData],
  );
  const opsCache = usePlanOps(planIds);

  if (!summaryData) {
    return (
      <Center h="200px">
        <Spinner size="lg" />
      </Center>
    );
  }

  // Compute hero state: worst across plans, with running override
  const plans = summaryData.planSummaries;
  let heroState: HeroState = "idle";
  let newestMs = 0;
  let nextMs: number | null = null;
  let anyRun = false;

  for (const s of plans) {
    const rb = s.recentBackups;
    const latest = rb?.status[0];
    const running =
      latest === OperationStatus.STATUS_INPROGRESS ||
      latest === OperationStatus.STATUS_PENDING;
    if (running) anyRun = true;

    const ts = Number(rb?.timestampMs[0] ?? 0);
    if (ts > newestMs) newestMs = ts;

    const st = planState(latest, running);
    const severity = { err: 3, warn: 2, run: 1, ok: 0, idle: -1 } as Record<
      PlanState,
      number
    >;
    const heroSeverity = { err: 3, warn: 2, run: 1, ok: 0, idle: -1 } as Record<
      HeroState,
      number
    >;
    if (severity[st] > (heroSeverity[heroState] ?? -1)) {
      heroState = st as HeroState;
    }

    const nx = Number(s.nextBackupTimeMs ?? 0);
    if (nx > 0 && (nextMs === null || nx < nextMs)) nextMs = nx;
  }

  if (anyRun) heroState = "run";

  return (
    <Stack gap={8} width="full">
      {/* Multihost summary */}
      <MultihostSummary multihostConfig={config?.multihost ?? null} />

      {/* Hero */}
      {plans.length > 0 && (
        <HeroBanner state={heroState} newestMs={newestMs} nextMs={nextMs} />
      )}

      {/* Plan cards */}
      {plans.length > 0 && (
        <Stack gap={4}>
          <Heading size="md">{m.dashboard_plans_title()}</Heading>
          <SimpleGrid columns={{ base: 1, md: 2 }} gap={4}>
            {plans.map((s) => (
              <PlanCard key={s.id} summary={s} opsData={opsCache.get(s.id)} />
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
          <Stack gap={4}>
            {summaryData.repoSummaries.map((s) => (
              <RepoCard key={s.id} summary={s} />
            ))}
          </Stack>
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

// ─── Multihost (preserved verbatim from original) ─────────────────────────────

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

  const knownHostTiles: JSX.Element[] = [];
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

  const authorizedClientTiles: JSX.Element[] = [];
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
              label="Shared Repos"
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
              label="Shared Repos"
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
