import {
  Flex,
  Stack,
  Input,
  Text,
  Card,
  Button,
} from "@chakra-ui/react";
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
  // midnight every day
  cron: "0 0 * * *",
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
      <Field label="Schedule Type">
        <Card.Root variant="subtle" width="fit-content">
          <Card.Header pb={0}>
            <Flex gap={2} wrap="wrap">
              {[
                { value: "disabled", label: "Disabled" },
                { value: "maxFrequencyHours", label: "Interval (Hours)" },
                { value: "maxFrequencyDays", label: "Interval (Days)" },
                { value: "cron", label: "Cron" },
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
          </Card.Header>

          <Card.Body>
            {/* Mode Specific Input */}
            {mode === "cron" && (
              <Field
                label="Cron Expression"
                helperText={(() => {
                  try {
                    return schedule.cron
                      ? cronstrue.toString(schedule.cron)
                      : "Standard cron syntax (e.g. 0 0 * * *)";
                  } catch (e) {
                    return "Invalid cron expression";
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
                label="Interval in Days"
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
                label="Interval in Hours"
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
                label="Reference Clock"
                helperText="Time zone or reference point for the schedule."
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
                    <Radio value="LOCAL">Local</Radio>
                    <Radio value="UTC">UTC</Radio>
                    <Radio value="LAST_RUN_TIME">Last Run Time</Radio>
                  </Stack>
                </RadioGroup>
              </Field>
            )}
            {mode === "disabled" && (
              <Text color="fg.muted" fontSize="sm">
                Automatic snapshots are disabled for this plan. You can still
                run backups manually.
              </Text>
            )}
          </Card.Body>
        </Card.Root>
      </Field>
    </Stack>
  );
};

const clockEnumValueToString = (clock: Schedule_Clock) =>
  Schedule_ClockSchema.values.find((v) => v.number === clock)?.name || "LOCAL";
