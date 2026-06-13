import { Box, Flex, Text } from "@chakra-ui/react";
import React from "react";
import { OperationStatus } from "../../../gen/ts/v1/operations_pb";
import * as m from "../../paraglide/messages";

export interface DayBucket {
  /** worst OperationStatus for this day, or undefined if no ops */
  status?: OperationStatus;
  /** true if this day is before the plan's first ever operation */
  beforeStart: boolean;
  /** local date label for tooltip */
  label: string;
  isToday: boolean;
}

/** Build 30 DayBuckets from a raw ops list for one plan. */
export function buildDayBuckets(
  ops: Array<{ unixTimeStartMs: bigint; status: OperationStatus }>,
): { buckets: DayBucket[]; firstMs: number | null; protectedBytes: number } {
  // rank: higher = worse  (we want worst-day semantics)
  const rank = (s: OperationStatus): number => {
    if (
      s === OperationStatus.STATUS_ERROR ||
      s === OperationStatus.STATUS_SYSTEM_CANCELLED
    )
      return 3;
    if (s === OperationStatus.STATUS_WARNING) return 2;
    if (s === OperationStatus.STATUS_SUCCESS) return 1;
    return 0;
  };

  const dayMap = new Map<string, OperationStatus>();
  let firstMs: number | null = null;
  let protectedBytes = 0;

  for (const op of ops) {
    const t = Number(op.unixTimeStartMs);
    if (!t) continue;
    if (firstMs === null || t < firstMs) firstMs = t;
    if (rank(op.status) > 0) {
      const d = new Date(t);
      d.setHours(0, 0, 0, 0);
      const key = d.toISOString();
      const cur = dayMap.get(key);
      if (!cur || rank(op.status) > rank(cur)) {
        dayMap.set(key, op.status);
      }
    }
  }

  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const firstDay = firstMs !== null ? new Date(firstMs) : null;
  if (firstDay) firstDay.setHours(0, 0, 0, 0);

  const buckets: DayBucket[] = [];
  for (let i = 29; i >= 0; i--) {
    const d = new Date(today);
    d.setDate(today.getDate() - i);
    const beforeStart = firstDay === null || d < firstDay;
    const key = d.toISOString();
    const status = beforeStart ? undefined : dayMap.get(key);
    buckets.push({
      status,
      beforeStart,
      label: d.toLocaleDateString(),
      isToday: i === 0,
    });
  }

  return { buckets, firstMs, protectedBytes };
}

function cellBg(bucket: DayBucket): string {
  if (bucket.beforeStart) return "transparent";
  if (!bucket.status) return "transparent"; // missed — dashed border below
  if (bucket.status === OperationStatus.STATUS_SUCCESS) return "green.500";
  if (bucket.status === OperationStatus.STATUS_WARNING) return "orange.400";
  if (
    bucket.status === OperationStatus.STATUS_ERROR ||
    bucket.status === OperationStatus.STATUS_SYSTEM_CANCELLED
  )
    return "red.500";
  return "bg.muted";
}

function summaryText(buckets: DayBucket[]): string {
  const active = buckets.filter((b) => !b.beforeStart);
  if (active.length === 0) return m.dashboard_history_no_data();
  const missed = active.filter((b) => !b.status).length;
  const issues = active.filter(
    (b) =>
      b.status &&
      b.status !== OperationStatus.STATUS_SUCCESS &&
      b.status !== OperationStatus.STATUS_INPROGRESS &&
      b.status !== OperationStatus.STATUS_PENDING,
  ).length;
  if (missed === 0 && issues === 0) return m.dashboard_history_all_backed_up();
  const parts: string[] = [];
  if (missed) parts.push(m.dashboard_history_missed({ count: missed }));
  if (issues) parts.push(m.dashboard_history_issues({ count: issues }));
  return m.dashboard_history_summary({ details: parts.join(" · ") });
}

export const HistoryStrip = ({ buckets }: { buckets: DayBucket[] }) => {
  const summary = summaryText(buckets);

  return (
    <Box mt={4}>
      <Text fontSize="13px" fontWeight="520" mb={2} color="fg.default">
        {summary}
      </Text>
      <Flex gap="3px" w="full">
        {buckets.map((b, i) => {
          const bg = cellBg(b);
          const isMissed = !b.beforeStart && !b.status;
          return (
            <Box
              key={i}
              flexGrow={1}
              flexShrink={1}
              flexBasis={0}
              minW={0}
              h="22px"
              borderRadius="3px"
              bg={b.beforeStart ? "bg.muted" : bg}
              opacity={b.beforeStart ? 0.35 : 1}
              border={isMissed ? "1.5px dashed" : "none"}
              borderColor={isMissed ? "border.subtle" : "transparent"}
              boxShadow={
                b.isToday
                  ? "0 0 0 2px var(--chakra-colors-bg-canvas), 0 0 0 3.5px var(--chakra-colors-fg-muted)"
                  : undefined
              }
              title={`${b.label} — ${
                b.beforeStart
                  ? m.dashboard_history_tooltip_before_start()
                  : !b.status
                    ? m.dashboard_history_tooltip_no_backup()
                    : b.status === OperationStatus.STATUS_SUCCESS
                      ? m.dashboard_history_legend_backed_up()
                      : b.status === OperationStatus.STATUS_WARNING
                        ? m.dashboard_activity_row_completed_warnings()
                        : m.dashboard_state_label_err()
              }`}
            />
          );
        })}
      </Flex>
      {/* Legend */}
      <Flex gap="14px" mt={2} flexWrap="wrap">
        {[
          { label: m.dashboard_history_legend_backed_up(), color: "green.500" },
          { label: m.dashboard_history_legend_issue(), color: "orange.400" },
        ].map(({ label, color }) => (
          <Flex key={label} align="center" gap="5px">
            <Box w="9px" h="9px" borderRadius="2px" bg={color} flexShrink={0} />
            <Text fontSize="11px" color="fg.muted">
              {label}
            </Text>
          </Flex>
        ))}
        <Flex align="center" gap="5px">
          <Box
            w="9px"
            h="9px"
            borderRadius="2px"
            border="1.5px dashed"
            borderColor="border.subtle"
            flexShrink={0}
          />
          <Text fontSize="11px" color="fg.muted">
            {m.dashboard_history_legend_missed()}
          </Text>
        </Flex>
      </Flex>
    </Box>
  );
};
