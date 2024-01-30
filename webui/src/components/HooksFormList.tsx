import React, { useState } from 'react';
import { Hook, Hook_Command, Hook_Condition, Hook_Discord, Hook_Webhook } from '../../gen/ts/v1/config_pb';
import { Button, Card, Collapse, CollapseProps, Form, FormListFieldData, Input, Popover, Radio, Row, Select, Tooltip } from 'antd';
import { MinusCircleOutlined, PlusOutlined } from '@ant-design/icons';

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
export const HooksFormList = (props: { hooks: Hook[] }) => {
  const [hooks, _] = useState([...props.hooks] || []);

  return <Form.List name="hooks" initialValue={props.hooks || []}>
    {(fields, { add, remove }, { errors }) => (
      <>
        {fields.map((field, index) => {
          console.log(index, field);
          const hook = hooks[index];
          if (!hook) return null;
          return <Card key={index} title={<>
            Hook {index}
            <MinusCircleOutlined
              className="dynamic-delete-button"
              onClick={() => remove(field.name)}
              style={{ marginRight: "5px", marginTop: "2px", float: "right" }}
            />
          </>
          } size="small" >
            <Form.Item name={[field.name, "conditions"]} initialValue={hook.conditions}>
              <Select
                mode="multiple"
                allowClear
                style={{ width: '100%' }}
                placeholder="Runs when..."
                defaultValue={hook.conditions}
                options={[
                  { label: "On Finish Snapshot", value: Hook_Condition.SNAPSHOT_END },
                  { label: "On Start Snapshot", value: Hook_Condition.SNAPSHOT_START },
                  { label: "On Snapshot Error", value: Hook_Condition.SNAPSHOT_ERROR },
                  { label: "On Any Error", value: Hook_Condition.ANY_ERROR },
                ]}
              />
            </Form.Item>
            <HookBuilder hook={hook} field={field} />
          </Card>
        })}
        <Form.Item>
          <Popover
            content={<>
              {hookTypes.map((hookType, index) => {
                return <Button key={index} onClick={() => {
                  const hook = new Hook({
                    action: hookType.action
                  });
                  hooks.push(hook);
                  add(hook);
                }}>
                  {hookType.name}
                </Button>
              })}
            </>}
            style={{ width: "60%" }}
            placement="bottom"
          >
            <Button type="dashed" icon={<PlusOutlined />}>
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
  action: typeof Hook.prototype.action,
}[] = [
    {
      name: "Command", action: {
        case: "actionCommand",
        value: new Hook_Command({
          command: "echo {{ .ShellEscape .Summary }}",
        }),
      }
    },
    {
      name: "Discord", action: {
        case: "actionDiscord",
        value: new Hook_Discord({
          webhookUrl: "",
          template: "{{ .Summary }}",
        }),
      }
    },
  ];

const HookBuilder = ({ field, hook }: { field: FormListFieldData, hook: Hook }) => {
  let component: React.ReactNode;
  switch (hook.action.case) {
    case "actionDiscord":
      return <>
        <Form.Item name={[field.name, "action", "value", "webhookUrl"]} required={true}>
          <Input addonBefore={<div style={{ width: "8em" }}>Discord Webhook</div>} />
        </Form.Item>
        Text:
        <Form.Item name={[field.name, "action", "value", "template"]} required={true}>
          <Input.TextArea style={{ width: "100%", fontFamily: "monospace" }} />
        </Form.Item>
      </>
    case "actionCommand":
      return <>
        <Tooltip title="Script to execute. Commands will not work in the docker build of Backrest.">
          Script:
        </Tooltip>
        <Form.Item name={[field.name, "action", "value", "command"]}>
          <Input.TextArea style={{ width: "100%", fontFamily: "monospace" }} />
        </Form.Item>
      </>
    default:
      return <p>Unknown hook {hook.action.case}</p>
  }
}