import {
  Checkbox,
  Flex,
  Form,
  InputNumber,
  Radio,
  Tooltip,
  Typography,
} from "antd";
import React from "react";
import Cron, { CronType, PeriodType } from "react-js-cron";
import {
  Schedule_Clock,
  Schedule_ClockSchema,
} from "../../gen/ts/v1/config_pb";

interface ScheduleDefaults {
  maxFrequencyDays: number;
  maxFrequencyHours: number;
  cron: string;
  cronPeriods?: PeriodType[];
  cronDropdowns?: CronType[];
  clock: Schedule_Clock;
}

export const ScheduleDefaultsInfrequent: ScheduleDefaults = {
  maxFrequencyDays: 30,
  maxFrequencyHours: 30 * 24,
  // midnight on the first day of the month
  cron: "0 0 1 * *",
  cronDropdowns: ["period", "months", "month-days", "week-days", "hours"],
  cronPeriods: ["month", "week"],
  clock: Schedule_Clock.LAST_RUN_TIME,
};

export const ScheduleDefaultsDaily: ScheduleDefaults = {
  maxFrequencyDays: 1,
  maxFrequencyHours: 24,
  // midnight every day
  cron: "0 0 * * *",
  cronDropdowns: [
    "period",
    "months",
    "month-days",
    "hours",
    "minutes",
    "week-days",
  ],
  cronPeriods: ["day", "hour", "month", "week"],
  clock: Schedule_Clock.LOCAL,
};

type SchedulingMode =
  | ""
  | "disabled"
  | "maxFrequencyDays"
  | "maxFrequencyHours"
  | "cron";
export const ScheduleFormItem = ({
  name,
  defaults,
}: {
  name: string[];
  defaults: ScheduleDefaults;
}) => {
  const form = Form.useFormInstance();
  const schedule = Form.useWatch(name, { form, preserve: true }) as any;

  if (schedule !== undefined && schedule.clock === undefined) {
    form.setFieldValue(
      name.concat("clock"),
      clockEnumValueToString(defaults.clock)
    );
  }

  const determineMode = (): SchedulingMode => {
    if (!schedule) {
      return "";
    } else if (schedule.disabled) {
      return "disabled";
    } else if (schedule.maxFrequencyDays) {
      return "maxFrequencyDays";
    } else if (schedule.maxFrequencyHours) {
      return "maxFrequencyHours";
    } else if (schedule.cron) {
      return "cron";
    }
    return "";
  };

  const mode = determineMode();

  let elem: React.ReactNode = null;
  if (mode === "cron") {
    elem = (
      <Form.Item
        name={name.concat(["cron"])}
        initialValue={defaults.cron}
        validateTrigger={["onChange", "onBlur"]}
        rules={[
          {
            required: true,
            message: "Please provide a valid cron schedule.",
          },
        ]}
      >
        <Cron
          value={form.getFieldValue(name.concat(["cron"]))}
          setValue={(val: string) => {
            form.setFieldValue(name.concat(["cron"]), val);
          }}
          allowedDropdowns={defaults.cronDropdowns}
          allowedPeriods={defaults.cronPeriods}
          clearButton={false}
        />
      </Form.Item>
    );
  } else if (mode === "maxFrequencyDays") {
    elem = (
      <Form.Item
        name={name.concat(["maxFrequencyDays"])}
        initialValue={defaults.maxFrequencyDays}
        validateTrigger={["onChange", "onBlur"]}
        rules={[
          {
            required: true,
            message: "Please input an interval in days",
          },
        ]}
      >
        <InputNumber
          addonBefore={<div style={{ width: "10em" }}>Interval in Days</div>}
          type="number"
          min={1}
        />
      </Form.Item>
    );
  } else if (mode === "maxFrequencyHours") {
    elem = (
      <Form.Item
        name={name.concat(["maxFrequencyHours"])}
        initialValue={defaults.maxFrequencyHours}
        validateTrigger={["onChange", "onBlur"]}
        rules={[
          {
            required: true,
            message: "Please input an interval in hours",
          },
        ]}
      >
        <InputNumber
          addonBefore={<div style={{ width: "10em" }}>Interval in Hours</div>}
          type="number"
          min={1}
        />
      </Form.Item>
    );
  } else if (mode === "disabled") {
    elem = (
      <Form.Item
        name={name.concat(["disabled"])}
        valuePropName="checked"
        initialValue={true}
        hidden={true}
      >
        <Checkbox />
      </Form.Item>
    );
  }

  return (
    <Flex vertical gap="small">
      <div>
        <Radio.Group
          value={mode}
          onChange={(e) => {
            const selected = e.target.value;
            if (selected === "maxFrequencyDays") {
              form.setFieldValue(name, {
                maxFrequencyDays: defaults!.maxFrequencyDays,
              });
            } else if (selected === "maxFrequencyHours") {
              form.setFieldValue(name, {
                maxFrequencyHours: defaults!.maxFrequencyHours,
              });
            } else if (selected === "cron") {
              form.setFieldValue(name, { cron: defaults!.cron });
            } else if (selected === "minHoursSinceLastRun") {
              form.setFieldValue(name, { minHoursSinceLastRun: 1 });
            } else if (selected === "minDaysSinceLastRun") {
              form.setFieldValue(name, { minDaysSinceLastRun: 1 });
            } else if (selected === "cronSinceLastRun") {
              form.setFieldValue(name, { cronSinceLastRun: defaults!.cron });
            } else {
              form.setFieldValue(name, { disabled: true });
            }
          }}
        >
          <Radio.Button value={"disabled"}>
            <Tooltip title="Schedule is disabled, will never run.">
              Disabled
            </Tooltip>
          </Radio.Button>
          <Radio.Button value={"maxFrequencyHours"}>
            <Tooltip title="Schedule will run at the specified interval in hours.">
              Interval Hours
            </Tooltip>
          </Radio.Button>
          <Radio.Button value={"maxFrequencyDays"}>
            <Tooltip title="Schedule will run at the specified interval in days.">
              Interval Days
            </Tooltip>
          </Radio.Button>
          <Radio.Button value={"cron"}>
            <Tooltip title="Schedule will run based on a cron schedule.">
              Cron
            </Tooltip>
          </Radio.Button>
        </Radio.Group>
      </div>
      <Flex align="center" gap="small">
        <Typography.Text>Clock for schedule:</Typography.Text>
        <Tooltip
          title={
            <>
              Clock provides the time that the schedule is evaluated relative to.
              <ul>
                <li>Local - current time in the local timezone.</li>
                <li>UTC - current time in the UTC timezone.</li>
                <li>
                  Last Run Time - relative to the last time the task ran. Good for
                  devices that aren't always powered on e.g. laptops.
                </li>
              </ul>
            </>
          }
        >
          <Form.Item name={name.concat("clock")} noStyle>
            <Radio.Group>
              <Radio.Button
                value={clockEnumValueToString(Schedule_Clock.LOCAL)}
              >
                Local
              </Radio.Button>
              <Radio.Button value={clockEnumValueToString(Schedule_Clock.UTC)}>
                UTC
              </Radio.Button>
              <Radio.Button
                value={clockEnumValueToString(Schedule_Clock.LAST_RUN_TIME)}
              >
                Last Run Time
              </Radio.Button>
            </Radio.Group>
          </Form.Item>
        </Tooltip>
      </Flex>
      {elem && (
        <div style={{ marginTop: "8px" }}>
          <Form.Item noStyle>{elem}</Form.Item>
        </div>
      )}
    </Flex>
  );
};

const clockEnumValueToString = (clock: Schedule_Clock) =>
  Schedule_ClockSchema.values.find((v) => v.number === clock)?.name;
