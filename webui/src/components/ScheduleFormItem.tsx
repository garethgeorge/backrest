import { Checkbox, Form, InputNumber, Radio, Row, Tooltip } from "antd";
import React from "react";
import Cron, { CronType, PeriodType } from "react-js-cron";
import { Schedule_Clock } from "../../gen/ts/v1/config_pb";

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

  defaults = defaults || ScheduleDefaultsInfrequent;

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
    <>
      <Row>
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
          {(!allowedModes || allowedModes.includes("disabled")) && (
            <Radio.Button value={"disabled"}>
              <Tooltip title="Schedule is disabled, will never run.">
                Disabled
              </Tooltip>
            </Radio.Button>
          )}
          {(!allowedModes || allowedModes.includes("maxFrequencyHours")) && (
            <Radio.Button value={"maxFrequencyHours"}>
              <Tooltip title="Schedule will run at the specified interval in hours relative to Backrest's start time.">
                Interval Hours
              </Tooltip>
            </Radio.Button>
          )}
          {(!allowedModes || allowedModes.includes("maxFrequencyDays")) && (
            <Radio.Button value={"maxFrequencyDays"}>
              <Tooltip title="Schedule will run at the specified interval in days relative to Backrest's start time.">
                Interval Days
              </Tooltip>
            </Radio.Button>
          )}
          {(!allowedModes || allowedModes.includes("cron")) && (
            <Radio.Button value={"cron"}>
              <Tooltip title="Schedule will run based on a cron schedule evaluated relative to Backrest's start time.">
                Startup Relative Cron
              </Tooltip>
            </Radio.Button>
          )}
        </Radio.Group>
      </Row>
      <div style={{ height: "0.5em" }} />
      <Row>
        <Form.Item>{elem}</Form.Item>
      </Row>
    </>
  );
};
