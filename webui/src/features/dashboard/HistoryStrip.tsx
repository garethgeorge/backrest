import { Box, Flex, Text } from "@chakra-ui/react";
import { OperationStatus } from "../../../gen/ts/v1/operations_pb";
import { SummaryDashboardResponse_DayStatusBucket } from "../../../gen/ts/v1/service_pb";
import * as m from "../../paraglide/messages";

const HISTORY_DAYS = 30;

// Display category for one day; drives color, border, dimming, and tooltip via CELL_STYLE.
type CellKind = "beforeStart" | "missed" | "ok" | "warn" | "err" | "other";

interface DayCell {
  kind: CellKind;
  label: string; // local date label for the tooltip
  isToday: boolean;
}

const CELL_STYLE: Record<
  CellKind,
  { bg: string; dashed: boolean; dim: boolean; tooltip: () => string }
> = {
  beforeStart: {
    bg: "bg.muted",
    dashed: false,
    dim: true,
    tooltip: m.dashboard_history_tooltip_before_start,
  },
  missed: {
    bg: "transparent",
    dashed: true,
    dim: false,
    tooltip: m.dashboard_history_tooltip_no_backup,
  },
  ok: {
    bg: "green.500",
    dashed: false,
    dim: false,
    tooltip: m.dashboard_history_legend_backed_up,
  },
  warn: {
    bg: "orange.400",
    dashed: false,
    dim: false,
    tooltip: m.dashboard_activity_row_completed_warnings,
  },
  err: {
    bg: "red.500",
    dashed: false,
    dim: false,
    tooltip: m.dashboard_state_label_err,
  },
  other: {
    bg: "bg.muted",
    dashed: false,
    dim: false,
    tooltip: m.dashboard_state_label_err,
  },
};

// rank: higher = worse, so the worst status wins for a day with mixed results.
function rank(s: OperationStatus): number {
  if (
    s === OperationStatus.STATUS_ERROR ||
    s === OperationStatus.STATUS_SYSTEM_CANCELLED
  )
    return 3;
  if (s === OperationStatus.STATUS_WARNING) return 2;
  if (s === OperationStatus.STATUS_SUCCESS) return 1;
  return 0;
}

function cellKind(
  bucket: SummaryDashboardResponse_DayStatusBucket | undefined,
): CellKind {
  let worst: OperationStatus | undefined;
  for (const sc of bucket?.statusCounts ?? []) {
    if (worst === undefined || rank(sc.status) > rank(worst)) worst = sc.status;
  }
  if (worst === undefined) return "missed";
  if (worst === OperationStatus.STATUS_SUCCESS) return "ok";
  if (worst === OperationStatus.STATUS_WARNING) return "warn";
  if (
    worst === OperationStatus.STATUS_ERROR ||
    worst === OperationStatus.STATUS_SYSTEM_CANCELLED
  )
    return "err";
  return "other";
}

// Fixed 30-cell strip ending today; days before the plan's first backup are left-padded.
function toCells(buckets: SummaryDashboardResponse_DayStatusBucket[]): DayCell[] {
  const recent = buckets.slice(-HISTORY_DAYS);
  const padCount = HISTORY_DAYS - recent.length;
  const midnight = new Date();
  midnight.setHours(0, 0, 0, 0);

  return Array.from({ length: HISTORY_DAYS }, (_, i): DayCell => {
    const daysAgo = HISTORY_DAYS - 1 - i;
    const date = new Date(midnight);
    date.setDate(midnight.getDate() - daysAgo);
    return {
      kind: i < padCount ? "beforeStart" : cellKind(recent[i - padCount]),
      label: date.toLocaleDateString(),
      isToday: daysAgo === 0,
    };
  });
}

function summaryText(cells: DayCell[]): string {
  const active = cells.filter((c) => c.kind !== "beforeStart");
  if (active.length === 0) return m.dashboard_history_no_data();
  const missed = active.filter((c) => c.kind === "missed").length;
  const issues = active.filter(
    (c) => c.kind === "warn" || c.kind === "err",
  ).length;
  if (missed === 0 && issues === 0) return m.dashboard_history_all_backed_up();
  const parts: string[] = [];
  if (missed) parts.push(m.dashboard_history_missed({ count: missed }));
  if (issues) parts.push(m.dashboard_history_issues({ count: issues }));
  return m.dashboard_history_summary({ details: parts.join(" · ") });
}

export const HistoryStrip = ({
  buckets,
}: {
  buckets: SummaryDashboardResponse_DayStatusBucket[];
}) => {
  const cells = toCells(buckets);

  return (
    <Box mt={4}>
      <Text fontSize="13px" fontWeight="520" mb={2} color="fg.default">
        {summaryText(cells)}
      </Text>
      <Flex gap="3px" w="full">
        {cells.map((c, i) => {
          const style = CELL_STYLE[c.kind];
          return (
            <Box
              key={i}
              flexGrow={1}
              flexShrink={1}
              flexBasis={0}
              minW={0}
              h="22px"
              borderRadius="3px"
              bg={style.bg}
              opacity={style.dim ? 0.35 : 1}
              border={style.dashed ? "1.5px dashed" : "none"}
              borderColor={style.dashed ? "border.subtle" : "transparent"}
              boxShadow={
                c.isToday
                  ? "0 0 0 2px var(--chakra-colors-bg-canvas), 0 0 0 3.5px var(--chakra-colors-fg-muted)"
                  : undefined
              }
              title={`${c.label} — ${style.tooltip()}`}
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
