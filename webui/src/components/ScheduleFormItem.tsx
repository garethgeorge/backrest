import {
  Checkbox,
  Col,
  Form,
  InputNumber,
  Radio,
  Row,
  Segmented,
  Tooltip,
} from "antd";
import { NamePath } from "antd/es/form/interface";
import React from "react";
import Cron from "react-js-cron";

export const ScheduleFormItem = ({ name }: { name: string[] }) => {
  const form = Form.useFormInstance();
  const retention = Form.useWatch(name, { form, preserve: true }) as any;

  const determineMode = () => {
    if (!retention) {
      return "";
    } else if (retention.disabled) {
      return "disabled";
    } else if (retention.maxFrequencyDays) {
      return "maxFrequencyDays";
    } else if (retention.maxFrequencyHours) {
      return "maxFrequencyHours";
    } else if (retention.cron) {
      return "cron";
    }
  };

  const mode = determineMode();

  let elem: React.ReactNode = null;
  if (mode === "cron") {
    elem = (
      <Form.Item
        name={name.concat(["cron"])}
        initialValue={"0 * * * *"}
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
          allowedDropdowns={[
            "period",
            "months",
            "month-days",
            "hours",
            "minutes",
          ]}
          allowedPeriods={["day", "hour", "month"]}
          clearButton={false}
        />
      </Form.Item>
    );
  } else if (mode === "maxFrequencyDays") {
    elem = (
      <Form.Item
        name={name.concat(["maxFrequencyDays"])}
        initialValue={0}
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
        />
      </Form.Item>
    );
  } else if (mode === "maxFrequencyHours") {
    elem = (
      <Form.Item
        name={name.concat(["maxFrequencyHours"])}
        initialValue={0}
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
              form.setFieldValue(name, { maxFrequencyDays: 1 });
            } else if (selected === "maxFrequencyHours") {
              form.setFieldValue(name, { maxFrequencyHours: 1 });
            } else if (selected === "cron") {
              form.setFieldValue(name, { cron: "0 * * * *" });
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
            <Tooltip title="Schedule will run at the specified interval in hours (e.g. N hours after the last run).">
              Max Frequency Hours
            </Tooltip>
          </Radio.Button>
          <Radio.Button value={"maxFrequencyDays"}>
            <Tooltip title="Schedule will run at the specified interval in days (e.g. N days after the last run).">
              Max Frequency Days
            </Tooltip>
          </Radio.Button>
          <Radio.Button value={"cron"}>
            <Tooltip title="Schedule will run based on a cron schedule.">
              Cron
            </Tooltip>
          </Radio.Button>
        </Radio.Group>
      </Row>
      <div style={{ height: "0.5em" }} />
      <Row>
        <Form.Item>{elem}</Form.Item>
      </Row>
    </>
  );
};
