import { Flex, Stack, Input, Text, Card, Button } from "@chakra-ui/react";
import { Checkbox } from "../ui/checkbox";
import { Radio, RadioGroup } from "../ui/radio";
import { NumberInputField } from "./NumberInput"; // Assuming I have this wrapper or standard NumberInput
import { Field } from "../ui/field";
import React, { useEffect } from "react";
import {
  Schedule_Clock,
  Schedule_ClockSchema,
} from "../../../gen/ts/v1/config_pb";
import cronstrue from "cronstrue";
import * as m from "../../paraglide/messages"

interface ScheduleDefaults {
  maxFrequencyDays: number;
  maxFrequencyHours: number;
  cron: string;
  clock: Schedule_Clock;
}

export const ScheduleDefaultsInfrequent: ScheduleDefaults = {
  maxFrequencyDays: 30,
  maxFrequencyHours: 30 * 24,
  // midnight on the first day of the month
  cron: "0 0 1 * *",
  clock: Schedule_Clock.LAST_RUN_TIME,
};

export const ScheduleDefaultsDaily: ScheduleDefaults = {
  maxFrequencyDays: 1,
  maxFrequencyHours: 24,
  // every 4 hours
  cron: "0 0/4 * * *",
  clock: Schedule_Clock.LOCAL,
};

type SchedulingMode =
  | ""
  | "disabled"
  | "maxFrequencyDays"
  | "maxFrequencyHours"
  | "cron";

export const ScheduleFormItem = ({
  value,
  onChange,
  defaults = ScheduleDefaultsDaily,
}: {
  value: any;
  onChange: (val: any) => void;
  defaults?: ScheduleDefaults;
}) => {
  // Ensure we have a valid object to work with
  const schedule = value || {};

  // Initialize clock if missing
  useEffect(() => {
    if (schedule && schedule.clock === undefined) {
      onChange({ ...schedule, clock: defaults.clock });
    }
  }, [schedule.clock]); // Check dependency carefully to avoid loops, maybe better to check on render or mount

  const determineMode = (): SchedulingMode => {
    if (!schedule) return "";
    if (schedule.disabled) return "disabled";
    if (schedule.maxFrequencyDays) return "maxFrequencyDays";
    if (schedule.maxFrequencyHours) return "maxFrequencyHours";
    if (schedule.cron) return "cron";
    return ""; // Default or nothing
  };

  const mode = determineMode();

  const handleModeChange = (newMode: string) => {
    let newSchedule: any = { clock: schedule.clock };

    if (newMode === "maxFrequencyDays") {
      newSchedule.maxFrequencyDays = defaults.maxFrequencyDays;
    } else if (newMode === "maxFrequencyHours") {
      newSchedule.maxFrequencyHours = defaults.maxFrequencyHours;
    } else if (newMode === "cron") {
      newSchedule.cron = defaults.cron;
    } else if (newMode === "disabled") {
      newSchedule.disabled = true;
    }
    onChange(newSchedule);
  };

  const handleClockChange = (newClockVals: string[]) => {
    // Assuming RadioGroup returns array or string, usually string if not multiple
    const valStr = newClockVals[0]; // if using standard CheckboxGroup generic logic or just val
    // Actually standard RadioGroup onValueChange gives string
    // But let's check basic RadioGroup usage
  };

  // Helper for clock
  const currentClockName = clockEnumValueToString(
    schedule.clock || defaults.clock,
  );

  return (
    <Stack gap={4}>
      {/* Schedule Mode */}
      <Card.Root variant="subtle" width="fit-content">
        <Card.Header pb={0}>
          <Field label={m.add_plan_modal_schedule_type_label()}>
            <Flex gap={2} wrap="wrap">
              {[
                { value: "disabled", label: m.add_plan_modal_schedule_disabled_label() },
                { value: "maxFrequencyHours", label: m.add_plan_modal_schedule_interval_hours() },
                { value: "maxFrequencyDays", label: m.add_plan_modal_schedule_interval_days() },
                { value: "cron", label: m.add_plan_modal_schedule_cron() },
              ].map((option) => (
                <Button
                  key={option.value}
                  size="sm"
                  variant={mode === option.value ? "solid" : "outline"}
                  onClick={() => handleModeChange(option.value)}
                >
                  {option.label}
                </Button>
              ))}
            </Flex>
          </Field>
        </Card.Header>
        <Card.Body>
          {/* Mode Specific Input */}
          {mode === "cron" && (
            <Field
              label={m.add_plan_modal_schedule_cron_expression()}
              helperText={(() => {
                try {
                  return schedule.cron
                    ? cronstrue.toString(schedule.cron)
                    : "Standard cron syntax (e.g. 0 0 * * *)";
                } catch (e) {
                  return m.add_plan_modal_schedule_invalid_cron();
                }
              })()}
            >
              <Input
                value={schedule.cron || ""}
                onChange={(e) =>
                  onChange({ ...schedule, cron: e.target.value })
                }
                fontFamily="mono"
                width="sm"
              />
            </Field>
          )}

          {mode === "maxFrequencyDays" && (
            <NumberInputField
              label={m.add_plan_modal_schedule_interval_in_days()}
              value={schedule.maxFrequencyDays || 0}
              onValueChange={(e: any) =>
                onChange({ ...schedule, maxFrequencyDays: e.valueAsNumber })
              }
              min={1}
              width="sm"
            />
          )}

          {mode === "maxFrequencyHours" && (
            <NumberInputField
              label={m.add_plan_modal_schedule_interval_in_hours()}
              value={schedule.maxFrequencyHours || 0}
              onValueChange={(e: any) =>
                onChange({ ...schedule, maxFrequencyHours: e.valueAsNumber })
              }
              min={1}
              width="sm"
            />
          )}

          {mode !== "disabled" && (
            /* Clock Selection */
            <Field
              label={m.add_plan_modal_schedule_reference_clock()}
              helperText={m.add_plan_modal_schedule_time_zone()}
            >
              <RadioGroup
                value={clockEnumValueToString(schedule.clock)}
                onValueChange={(e) => {
                  // find enum value
                  const clk = Schedule_ClockSchema.values.find(
                    (v) => v.name === e.value,
                  );
                  if (clk) onChange({ ...schedule, clock: clk.number });
                }}
              >
                <Stack direction="row" gap={4}>
                  <Radio value="CLOCK_LOCAL">{m.add_plan_modal_schedule_time_local()}</Radio>
                  <Radio value="CLOCK_UTC">{m.add_plan_modal_schedule_time_utc()}</Radio>
                  <Radio value="CLOCK_LAST_RUN_TIME">{m.add_plan_modal_schedule_time_last()}</Radio>
                </Stack>
              </RadioGroup>
            </Field>
          )}
          {mode === "disabled" && (
            <Text color="fg.muted" fontSize="sm">
              {m.add_plan_modal_schedule_disabled_description()}
            </Text>
          )}
        </Card.Body>
      </Card.Root>
    </Stack>
  );
};

const clockEnumValueToString = (clock: Schedule_Clock | string | number) => {
  if (typeof clock === "string") return clock;
  return (
    Schedule_ClockSchema.values.find((v) => v.number === clock)?.name ||
    "CLOCK_LOCAL"
  );
};
