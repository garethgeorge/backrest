import React, { useState } from 'react';
import { Hook, Hook_Command, Hook_Condition, Hook_Discord, Hook_Gotify, Hook_Webhook } from '../../gen/ts/v1/config_pb';
import { Button, Card, Collapse, CollapseProps, Form, FormListFieldData, Input, Popover, Radio, Row, Select, Tooltip } from 'antd';
import { MinusCircleOutlined, PlusOutlined } from '@ant-design/icons';
import { Rule } from 'antd/es/form';

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
}

export const hooksListTooltipText = <>
  Hooks are actions that can execute on backup lifecycle events.

  Available events are:
  <ul>
    <li>On Finish Snapshot: Runs after a snapshot is finished.</li>
    <li>On Start Snapshot: Runs when a snapshot is started.</li>
    <li>On Snapshot Error: Runs when a snapshot fails.</li>
    <li>On Any Error: Runs when any error occurs.</li>
  </ul>
  Arguments are available to hooks as <a target="_blank" rel="noopener noreferrer" href="https://pkg.go.dev/text/template" >Go template variables</a>
  <ul>
    <li>.Task - the name of the task that triggered the hook.</li>
    <li>.Event - the event that triggered the hook.</li>
    <li>.Repo - the name of the repo the event applies to.</li>
    <li>.Plan - the name of the plan the event applies to.</li>
    <li>.Error - the error if any is available.</li>
    <li>.CurTime - the time of the event.</li>
    <li>.SnapshotId - the restic snapshot structure if this is finish snapshot operation and it completed successfully.</li>
  </ul>
  Functions
  <ul>
    <li>.ShellEscape - escapes a string to be used in a shell command.</li>
    <li>.JsonMarshal - serializes a value to be used in a json string.</li>
    <li>.Summary - prints a formatted summary of the event.</li>
    <li>.FormatTime - prints time formatted as RFC3339.</li>
    <li>.FormatSizeBytes - prints a formatted size in bytes.</li>
  </ul>
</>


/**
 * HooksFormList is a UI component for editing a list of hooks that can apply either at the repo level or at the plan level.
 */
export const HooksFormList = () => {
  const form = Form.useFormInstance();

  return <Form.List name="hooks">
    {(fields, { add, remove }, { errors }) => (
      <>
        {fields.map((field, index) => {
          const hookData = form.getFieldValue(["hooks", field.name]) as HookFields;

          return <Card key={index} title={<>
            Hook {index} <small>{findHookTypeName(hookData)}</small>
            <MinusCircleOutlined
              className="dynamic-delete-button"
              onClick={() => remove(field.name)}
              style={{ marginRight: "5px", marginTop: "2px", float: "right" }}
            />
          </>
          } size="small" style={{ marginBottom: "5px" }}>
            <Form.Item name={[field.name, "conditions"]} >
              <Select
                mode="multiple"
                allowClear
                style={{ width: '100%' }}
                placeholder="Runs when..."
                options={[
                  { label: "On Finish Snapshot", value: Hook_Condition.SNAPSHOT_END },
                  { label: "On Start Snapshot", value: Hook_Condition.SNAPSHOT_START },
                  { label: "On Snapshot Error", value: Hook_Condition.SNAPSHOT_ERROR },
                  { label: "On Any Error", value: Hook_Condition.ANY_ERROR },
                ]}
              />
            </Form.Item>
            <Form.Item shouldUpdate={(prevValues, curValues) => {
              return prevValues.hooks[index] !== curValues.hooks[index];
            }}>
              <HookBuilder field={field} />
            </Form.Item>
          </Card>
        })}
        <Form.Item>
          <Popover
            content={<>
              {hookTypes.map((hookType, index) => {
                return <Button key={index} onClick={() => {
                  add(structuredClone(hookType.template));
                }}>
                  {hookType.name}
                </Button>
              })}
            </>}
            style={{ width: "60%" }}
            placement="bottom"
          >
            <Button type="dashed" icon={<PlusOutlined />} style={{ width: "100%" }}>
              Add Hook
            </Button>
          </Popover>
          <Form.ErrorList errors={errors} />
        </Form.Item>
      </>
    )}
  </Form.List >
}

const hookTypes: {
  name: string,
  template: HookFields,
  oneofKey: string,
  component: ({ field }: { field: FormListFieldData }) => React.ReactNode
}[] = [
    {
      name: "Command", template: {
        actionCommand: {
          command: "echo {{ .ShellEscape .Summary }}",
        },
        conditions: [],
      },
      oneofKey: "actionCommand",
      component: ({ field }: { field: FormListFieldData }) => {
        return <>
          <Tooltip title="Script to execute. Commands will not work in the docker build of Backrest.">
            Script:
          </Tooltip>
          <Form.Item name={[field.name, "actionCommand", "command"]} rules={[requiredField("command is required")]}>
            <Input.TextArea style={{ width: "100%", fontFamily: "monospace" }} />
          </Form.Item>
        </>
      }
    },
    {
      name: "Shoutrrr", template: {
        actionShoutrrr: {
          template: "{{ .Summary }}",
        },
        conditions: [],
      },
      oneofKey: "actionShoutrrr",
      component: ({ field }: { field: FormListFieldData }) => {
        return <>
          <Form.Item name={[field.name, "actionShoutrrr", "shoutrrrUrl"]} rules={[requiredField("shoutrrr URL is required")]} >
            <Input addonBefore={
              <Tooltip title={
                <>Shoutrrr is a multi-platform notification service, <a href="https://containrrr.dev/shoutrrr/v0.8/services/overview/" target="_blank">see docs</a> to learn more about supported services</>
              }>
                <div style={{ width: "8em" }}>Shoutrrr URL</div>
              </Tooltip>}
            />
          </Form.Item >
          Text Template:
          <Form.Item name={[field.name, "actionShoutrrr", "template"]} >
            <Input.TextArea style={{ width: "100%", fontFamily: "monospace" }} />
          </Form.Item >
        </>
      }
    },
    {
      name: "Discord", template: {
        actionDiscord: {
          webhookUrl: "",
          template: "{{ .Summary }}",
        },
        conditions: [],
      },
      oneofKey: "actionDiscord",
      component: ({ field }: { field: FormListFieldData }) => {
        return <>
          <Form.Item name={[field.name, "actionDiscord", "webhookUrl"]} rules={[requiredField("webhook URL is required"), { type: "url" }]} >
            <Input addonBefore={<div style={{ width: "8em" }}>Discord Webhook</div>} />
          </Form.Item >
          Text Template:
          <Form.Item name={[field.name, "actionDiscord", "template"]} >
            <Input.TextArea style={{ width: "100%", fontFamily: "monospace" }} />
          </Form.Item >
        </>
      }
    },
    {
      name: "Gotify", template: {
        actionGotify: {
          baseUrl: "",
          token: "",
          template: "{{ .Summary }}",
          titleTemplate: "Backrest {{ .EventName .Event }} in plan {{ .Plan.Id }}",
        },
        conditions: [],
      },
      oneofKey: "actionGotify",
      component: ({ field }: { field: FormListFieldData }) => {
        return <>
          <Form.Item name={[field.name, "actionGotify", "baseUrl"]} rules={[requiredField("gotify base URL is required"), { type: "url" }]}>
            <Input addonBefore={<div style={{ width: "8em" }}>Gotify Base URL</div>} />
          </Form.Item >
          <Form.Item name={[field.name, "actionGotify", "token"]} rules={[requiredField("gotify token is required")]}>
            <Input addonBefore={<div style={{ width: "8em" }}>Gotify Token</div>} />
          </Form.Item>
          <Form.Item name={[field.name, "actionGotify", "titleTemplate"]} rules={[requiredField("gotify title template is required")]}>
            <Input addonBefore={<div style={{ width: "8em" }}>Title Template</div>} />
          </Form.Item>
          Text Template:
          <Form.Item name={[field.name, "actionGotify", "template"]}>
            <Input.TextArea style={{ width: "100%", fontFamily: "monospace" }} />
          </Form.Item>
        </>
      }
    },
    {
      name: "Slack", template: {
        actionSlack: {
          webhookUrl: "",
          template: "{{ .Summary }}",
        },
        conditions: [],
      },
      oneofKey: "actionSlack",
      component: ({ field }: { field: FormListFieldData }) => {
        return <>
          <Form.Item name={[field.name, "actionSlack", "webhookUrl"]} rules={[requiredField("webhook URL is required"), { type: "url" }]} >
            <Input addonBefore={<div style={{ width: "8em" }}>Slack Webhook</div>} />
          </Form.Item >
          Text Template:
          <Form.Item name={[field.name, "actionSlack", "template"]} >
            <Input.TextArea style={{ width: "100%", fontFamily: "monospace" }} />
          </Form.Item >
        </>
      }
    }
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
}

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
}

const requiredField = (message: string, extra?: Rule) => ({ required: true, message: message });