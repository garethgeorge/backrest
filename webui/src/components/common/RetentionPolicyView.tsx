import { Flex, Stack, Card, Grid } from "@chakra-ui/react";
import { useMemo } from "react";
import * as m from "../../paraglide/messages";
import { Button } from "../ui/button";
import { Tooltip } from "../ui/tooltip";
import { NumberInputField } from "./NumberInput";

export const RetentionPolicyView = ({
  schedule,
  retention,
  onChange,
}: {
  schedule?: any;
  retention: any;
  onChange: (v: any) => void;
}) => {
  const determineMode = () => {
    if (!retention) return "policyTimeBucketed";
    if (retention.policyKeepLastN) return "policyKeepLastN";
    if (retention.policyKeepAll) return "policyKeepAll";
    if (retention.policyTimeBucketed) return "policyTimeBucketed";
    return "policyTimeBucketed";
  };
  const mode = determineMode();

  const handleModeChange = (newMode: string) => {
    if (newMode === "policyKeepLastN") {
      onChange({ policyKeepLastN: 30 });
    } else if (newMode === "policyTimeBucketed") {
      onChange({
        policyTimeBucketed: {
          yearly: 0,
          monthly: 3,
          weekly: 4,
          daily: 7,
          hourly: 24,
          keepLastN: 0,
        },
      });
    } else {
      onChange({ policyKeepAll: true });
    }
  };

  const cronIsSubHourly = useMemo(
    () =>
      schedule?.schedule?.value &&
      !/^\d+ /.test(schedule.schedule.value) &&
      schedule.schedule.case === "cron",
    [schedule],
  );

  const updateRetentionField = (path: string[], val: any) => {
    const next = { ...retention };
    let curr = next;
    for (let i = 0; i < path.length - 1; i++) {
      curr[path[i]] = curr[path[i]] ? { ...curr[path[i]] } : {};
      curr = curr[path[i]];
    }
    curr[path[path.length - 1]] = val;
    onChange(next);
  };

  return (
    <Stack gap={4}>
      <Card.Root variant="subtle" width="fit-content">
        <Card.Header pb={0}>
          <Flex gap={2} wrap="wrap">
            {[
              {
                value: "policyKeepLastN",
                label: m.add_plan_modal_retention_policy_mode_count_label(),
                tooltip:
                  m.add_plan_modal_retention_policy_keep_last_n_tooltip(),
              },
              {
                value: "policyTimeBucketed",
                label: m.add_plan_modal_retention_policy_mode_time_label(),
                tooltip:
                  m.add_plan_modal_retention_policy_time_bucketed_tooltip(),
              },
              {
                value: "policyKeepAll",
                label: m.add_plan_modal_retention_policy_mode_none_label(),
                tooltip: m.add_plan_modal_retention_policy_keep_all_tooltip(),
              },
            ].map((option) => (
              <Tooltip key={option.value} content={option.tooltip}>
                <Button
                  size="sm"
                  variant={mode === option.value ? "solid" : "outline"}
                  onClick={() => handleModeChange(option.value)}
                >
                  {option.label}
                </Button>
              </Tooltip>
            ))}
          </Flex>
        </Card.Header>

        <Card.Body>
          {mode === "policyKeepAll" && (
            <p>{m.add_plan_modal_retention_policy_keep_all_warning()}</p>
          )}

          {mode === "policyKeepLastN" && (
            <NumberInputField
              label={m.add_plan_modal_retention_policy_keep_last_n_snapshots_label()}
              value={retention?.policyKeepLastN || 0}
              onValueChange={(e: any) =>
                onChange({ ...retention, policyKeepLastN: e.valueAsNumber })
              }
            />
          )}

          {mode === "policyTimeBucketed" && (
            <Grid templateColumns="repeat(3, 180px)" gap={4}>
              <NumberInputField
                label={m.add_plan_modal_retention_policy_hourly_label()}
                value={retention?.policyTimeBucketed?.hourly || 0}
                onValueChange={(e: any) =>
                  updateRetentionField(
                    ["policyTimeBucketed", "hourly"],
                    e.valueAsNumber,
                  )
                }
              />
              <NumberInputField
                label={m.add_plan_modal_retention_policy_daily_label()}
                value={retention?.policyTimeBucketed?.daily || 0}
                onValueChange={(e: any) =>
                  updateRetentionField(
                    ["policyTimeBucketed", "daily"],
                    e.valueAsNumber,
                  )
                }
              />
              <NumberInputField
                label={m.add_plan_modal_retention_policy_weekly_label()}
                value={retention?.policyTimeBucketed?.weekly || 0}
                onValueChange={(e: any) =>
                  updateRetentionField(
                    ["policyTimeBucketed", "weekly"],
                    e.valueAsNumber,
                  )
                }
              />
              <NumberInputField
                label={m.add_plan_modal_retention_policy_monthly_label()}
                value={retention?.policyTimeBucketed?.monthly || 0}
                onValueChange={(e: any) =>
                  updateRetentionField(
                    ["policyTimeBucketed", "monthly"],
                    e.valueAsNumber,
                  )
                }
              />
              <NumberInputField
                label={m.add_plan_modal_retention_policy_yearly_label()}
                value={retention?.policyTimeBucketed?.yearly || 0}
                onValueChange={(e: any) =>
                  updateRetentionField(
                    ["policyTimeBucketed", "yearly"],
                    e.valueAsNumber,
                  )
                }
              />
              <NumberInputField
                label={m.add_plan_modal_retention_policy_latest_label()}
                helperText={
                  cronIsSubHourly
                    ? "Keep recent snapshots (High-frequency schedule detected)"
                    : m.add_plan_modal_retention_policy_keep_regardless_label()
                }
                value={retention?.policyTimeBucketed?.keepLastN || 0}
                onValueChange={(e: any) =>
                  updateRetentionField(
                    ["policyTimeBucketed", "keepLastN"],
                    e.valueAsNumber,
                  )
                }
              />
            </Grid>
          )}
        </Card.Body>
      </Card.Root>
    </Stack>
  );
};
