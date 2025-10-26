import React, { useState } from "react";
import {
  Hook_Condition,
  Hook_ConditionSchema,
  Hook_OnError,
  Hook_OnErrorSchema,
} from "../../gen/ts/v1/config_pb";
import {
  Button,
  Card,
  Form,
  FormListFieldData,
  Input,
  Popover,
  Select,
  Tooltip,
} from "antd";
import { MinusCircleOutlined, PlusOutlined } from "@ant-design/icons";
import { Rule } from "antd/es/form";

export interface HookFormData {
  hooks: {
    conditions: string[];
  }[];
}

export interface HookFields {
  conditions: string[];
  actionCommand?: any;
  actionGotify?: any;
  actionDiscord?: any;
  actionWebhook?: any;
  actionSlack?: any;
  actionShoutrrr?: any;
  actionHealthchecks?: any;
  actionTelegram?: any;
}

export const hooksListTooltipText = (
  <>
    Hooks let you configure actions e.g. notifications and scripts that run in
    response to the backup lifecycle. See{" "}
    <a
      href="https://garethgeorge.github.io/backrest/docs/hooks"
      target="_blank"
    >
      the hook documentation
    </a>{" "}
    for available options, or
    <a
      href="https://garethgeorge.github.io/backrest/cookbooks/command-hook-examples"
      target="_blank"
    >
      the cookbook
    </a>
    for scripting examples.
  </>
);

/**
 * HooksFormList is a UI component for editing a list of hooks that can apply either at the repo level or at the plan level.
 */
export const HooksFormList = () => {
  const form = Form.useFormInstance();

  return (
    <Form.List name="hooks">
      {(fields, { add, remove }, { errors }) => (
        <>
          {fields.map((field, index) => {
            const hookData = form.getFieldValue([
              "hooks",
              field.name,
            ]) as HookFields;

            return (
              <Card
                key={index}
                title={
                  <>
                    Hook {index} {findHookTypeName(hookData)}
                    <MinusCircleOutlined
                      className="dynamic-delete-button"
                      onClick={() => remove(field.name)}
                      style={{
                        marginRight: "5px",
                        marginTop: "2px",
                        float: "right",
                      }}
                    />
                  </>
                }
                size="small"
                style={{ marginBottom: "5px" }}
              >
                <HookConditionsTooltip>
                  <Form.Item name={[field.name, "conditions"]}>
                    <Select
                      mode="multiple"
                      allowClear
                      style={{ width: "100%" }}
                      placeholder="Runs when..."
                      options={Hook_ConditionSchema.values.map((v) => ({
                        label: v.name,
                        value: v.name,
                      }))}
                    />
                  </Form.Item>
                </HookConditionsTooltip>
                <Form.Item
                  shouldUpdate={(prevValues, curValues) => {
                    return prevValues.hooks[index] !== curValues.hooks[index];
                  }}
                >
                  <HookBuilder field={field} />
                </Form.Item>
              </Card>
            );
          })}
          <Form.Item>
            <Popover
              content={
                <>
                  {hookTypes.map((hookType, index) => {
                    return (
                      <Button
                        key={index}
                        onClick={() => {
                          add(structuredClone(hookType.template));
                        }}
                      >
                        {hookType.name}
                      </Button>
                    );
                  })}
                </>
              }
              style={{ width: "60%" }}
              placement="bottom"
            >
              <Button
                type="dashed"
                icon={<PlusOutlined />}
                style={{ width: "100%" }}
              >
                Add Hook
              </Button>
            </Popover>
            <Form.ErrorList errors={errors} />
          </Form.Item>
        </>
      )}
    </Form.List>
  );
};

const hookTypes: {
  name: string;
  template: HookFields;
  oneofKey: string;
  component: ({ field }: { field: FormListFieldData }) => React.ReactNode;
}[] = [
  {
    name: "Command",
    template: {
      actionCommand: {
        command: "echo {{ .ShellEscape .Summary }}",
      },
      conditions: [],
    },
    oneofKey: "actionCommand",
    component: ({ field }: { field: FormListFieldData }) => {
      return (
        <>
          <Tooltip title="Script to execute.">Script:</Tooltip>
          <Form.Item
            name={[field.name, "actionCommand", "command"]}
            rules={[requiredField("command is required")]}
          >
            <Input.TextArea
              style={{ width: "100%", fontFamily: "monospace" }}
            />
          </Form.Item>
          <ItemOnErrorSelector field={field} />
        </>
      );
    },
  },
  {
    name: "Shoutrrr",
    template: {
      actionShoutrrr: {
        template: "{{ .Summary }}",
      },
      conditions: [],
    },
    oneofKey: "actionShoutrrr",
    component: ({ field }: { field: FormListFieldData }) => {
      return (
        <>
          <Form.Item
            name={[field.name, "actionShoutrrr", "shoutrrrUrl"]}
            rules={[requiredField("shoutrrr URL is required")]}
          >
            <Input
              addonBefore={
                <Tooltip
                  title={
                    <>
                      Shoutrrr is a multi-platform notification service,{" "}
                      <a
                        href="https://containrrr.dev/shoutrrr/v0.8/services/overview/"
                        target="_blank"
                      >
                        see docs
                      </a>{" "}
                      to learn more about supported services
                    </>
                  }
                >
                  <div style={{ width: "8em" }}>Shoutrrr URL</div>
                </Tooltip>
              }
            />
          </Form.Item>
          Text Template:
          <Form.Item name={[field.name, "actionShoutrrr", "template"]}>
            <Input.TextArea
              style={{ width: "100%", fontFamily: "monospace" }}
            />
          </Form.Item>
        </>
      );
    },
  },
  {
    name: "Discord",
    template: {
      actionDiscord: {
        webhookUrl: "",
        template: "{{ .Summary }}",
      },
      conditions: [],
    },
    oneofKey: "actionDiscord",
    component: ({ field }: { field: FormListFieldData }) => {
      return (
        <>
          <Form.Item
            name={[field.name, "actionDiscord", "webhookUrl"]}
            rules={[requiredField("webhook URL is required")]}
          >
            <Input
              addonBefore={<div style={{ width: "8em" }}>Discord Webhook</div>}
            />
          </Form.Item>
          Text Template:
          <Form.Item name={[field.name, "actionDiscord", "template"]}>
            <Input.TextArea
              style={{ width: "100%", fontFamily: "monospace" }}
            />
          </Form.Item>
        </>
      );
    },
  },
  {
    name: "Gotify",
    template: {
      actionGotify: {
        baseUrl: "",
        token: "",
        template: "{{ .Summary }}",
        titleTemplate:
          "Backrest {{ .EventName .Event }} in plan {{ .Plan.Id }}",
        priority: 5,
      },
      conditions: [],
    },
    oneofKey: "actionGotify",
    component: ({ field }: { field: FormListFieldData }) => {
      return (
        <>
          <Form.Item
            name={[field.name, "actionGotify", "baseUrl"]}
            rules={[
              requiredField("gotify base URL is required"),
              { type: "string" },
            ]}
          >
            <Input
              addonBefore={<div style={{ width: "8em" }}>Gotify Base URL</div>}
            />
          </Form.Item>
          <Form.Item
            name={[field.name, "actionGotify", "token"]}
            rules={[requiredField("gotify token is required")]}
          >
            <Input
              addonBefore={<div style={{ width: "8em" }}>Gotify Token</div>}
            />
          </Form.Item>
          <Form.Item
            name={[field.name, "actionGotify", "titleTemplate"]}
            rules={[requiredField("gotify title template is required")]}
          >
            <Input
              addonBefore={<div style={{ width: "8em" }}>Title Template</div>}
            />
          </Form.Item>
          Text Template:
          <Form.Item name={[field.name, "actionGotify", "template"]}>
            <Input.TextArea
              style={{ width: "100%", fontFamily: "monospace" }}
            />
          </Form.Item>
          <Form.Item name={[field.name, "actionGotify", "priority"]}>
            <Select
              allowClear
              style={{ width: "100%" }}
              placeholder={"Set priority"}
              options={[
                  {label: "0 - No notification", value: 0},
                  {label: "1 - Icon in notification bar", value: 1},
                  {label: "4 - Icon in notification bar + Sound", value: 4},
                  {label: "8 - Icon in notification bar + Sound + Vibration", value: 8},
              ]}
            />
          </Form.Item>
        </>
      );
    },
  },
  {
    name: "Slack",
    template: {
      actionSlack: {
        webhookUrl: "",
        template: "{{ .Summary }}",
      },
      conditions: [],
    },
    oneofKey: "actionSlack",
    component: ({ field }: { field: FormListFieldData }) => {
      return (
        <>
          <Form.Item
            name={[field.name, "actionSlack", "webhookUrl"]}
            rules={[requiredField("webhook URL is required")]}
          >
            <Input
              addonBefore={<div style={{ width: "8em" }}>Slack Webhook</div>}
            />
          </Form.Item>
          Text Template:
          <Form.Item name={[field.name, "actionSlack", "template"]}>
            <Input.TextArea
              style={{ width: "100%", fontFamily: "monospace" }}
            />
          </Form.Item>
        </>
      );
    },
  },
  {
    name: "Healthchecks",
    template: {
      actionHealthchecks: {
        webhookUrl: "",
        template: "{{ .Summary }}",
      },
      conditions: [],
    },
    oneofKey: "actionHealthchecks",
    component: ({ field }: { field: FormListFieldData }) => {
      return (
        <>
          <Form.Item
            name={[field.name, "actionHealthchecks", "webhookUrl"]}
            rules={[requiredField("Ping URL is required")]}
          >
            <Input addonBefore={<div style={{ width: "8em" }}>Ping URL</div>} />
          </Form.Item>
          Text Template:
          <Form.Item name={[field.name, "actionHealthchecks", "template"]}>
            <Input.TextArea
              style={{ width: "100%", fontFamily: "monospace" }}
            />
          </Form.Item>
        </>
      );
    },
  },
  {
    name: "Telegram",
    template: {
      actionTelegram: {
        botToken: "",
        chatId: "",
        template: "{{ .Summary }}",
      },
      conditions: [],
    },
    oneofKey: "actionTelegram",
    component: ({ field }: { field: FormListFieldData }) => {
      return (
        <>
          <Form.Item
            name={[field.name, "actionTelegram", "botToken"]}
            rules={[requiredField("bot token is required")]}
          >
            <Input
              addonBefore={
                <Tooltip
                  title={
                    <>
                      Create a bot with{" "}
                      <a
                        href="https://t.me/botfather"
                        target="_blank"
                      >
                        @BotFather
                      </a>{" "}
                      and use the token provided
                    </>
                  }
                >
                  <div style={{ width: "8em" }}>Bot Token</div>
                </Tooltip>
              }
            />
          </Form.Item>
          <Form.Item
            name={[field.name, "actionTelegram", "chatId"]}
            rules={[requiredField("chat ID is required")]}
          >
            <Input
              addonBefore={
                <Tooltip
                  title={
                    <>
                      Chat ID can be a user ID, group ID, or channel ID. Use{" "}
                      <a
                        href="https://t.me/userinfobot"
                        target="_blank"
                      >
                        @userinfobot
                      </a>{" "}
                      to find your user ID
                    </>
                  }
                >
                  <div style={{ width: "8em" }}>Chat ID</div>
                </Tooltip>
              }
            />
          </Form.Item>
          Text Template:
          <Form.Item name={[field.name, "actionTelegram", "template"]}>
            <Input.TextArea
              style={{ width: "100%", fontFamily: "monospace" }}
            />
          </Form.Item>
        </>
      );
    },
  },
];

const findHookTypeName = (field: HookFields): string => {
  if (!field) {
    return "Unknown";
  }
  for (const hookType of hookTypes) {
    if (hookType.oneofKey in field) {
      return hookType.name;
    }
  }
  return "Unknown";
};

const HookBuilder = ({ field }: { field: FormListFieldData }) => {
  const form = Form.useFormInstance();
  const hookData = form.getFieldValue(["hooks", field.name]) as HookFields;

  if (!hookData) {
    return <p>Unknown hook type</p>;
  }

  for (const hookType of hookTypes) {
    if (hookType.oneofKey in hookData) {
      return hookType.component({ field });
    }
  }

  return <p>Unknown hook type</p>;
};

const ItemOnErrorSelector = ({ field }: { field: FormListFieldData }) => {
  return (
    <>
      <Tooltip
        title={
          <>
            What happens when the hook fails (only effective on start hooks e.g.
            backup start, prune start, check start)
            <ul>
              <li>
                IGNORE - the failure is ignored, subsequent hooks and the backup
                operation will run as normal.
              </li>
              <li>
                FATAL - stops the backup with an error status (triggers an error
                notification). Skips running all subsequent hooks.
              </li>
              <li>
                CANCEL - marks the backup as cancelled but does not trigger any
                error notification. Skips running all subsequent hooks.
              </li>
            </ul>
          </>
        }
      >
        Error Behavior:
      </Tooltip>
      <Form.Item name={[field.name, "onError"]}>
        <Select
          allowClear
          style={{ width: "100%" }}
          placeholder={"Specify what happens when this hook fails..."}
          options={Hook_OnErrorSchema.values.map((v) => ({
            label: v.name,
            value: v.name,
          }))}
        />
      </Form.Item>
    </>
  );
};

const requiredField = (message: string, extra?: Rule) => ({
  required: true,
  message: message,
});

const HookConditionsTooltip = ({ children }: { children: React.ReactNode }) => {
  return (
    <Tooltip
      title={
        <div>
          Available conditions
          <ul>
            <li>CONDITION_ANY_ERROR - error executing any task</li>
            <li>CONDITION_SNAPSHOT_START - start of a backup operation</li>
            <li>
              CONDITION_SNAPSHOT_END - end of backup operation (success or
              failure)
            </li>
            <li>
              CONDITION_SNAPSHOT_SUCCESS - end of successful backup operation
            </li>
            <li>CONDITION_SNAPSHOT_ERROR - end of failed backup</li>
            <li>CONDITION_SNAPSHOT_WARNING - end of partial backup</li>
            <li>CONDITION_PRUNE_START - start of prune operation</li>
            <li>CONDITION_PRUNE_SUCCESS - end of successful prune</li>
            <li>CONDITION_PRUNE_ERROR - end of failed prune</li>
            <li>CONDITION_CHECK_START - start of check operation</li>
            <li>CONDITION_CHECK_SUCCESS - end of successful check</li>
            <li>CONDITION_CHECK_ERROR - end of failed check</li>
          </ul>
          for more info see the{" "}
          <a
            href="https://garethgeorge.github.io/backrest/docs/hooks"
            target="_blank"
          >
            documentation
          </a>
        </div>
      }
    >
      {children}
    </Tooltip>
  );
};
