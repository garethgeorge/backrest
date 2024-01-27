import React, { useState } from 'react';
import { Hook, Hook_Command } from '../../gen/ts/v1/config_pb';
import { Button, Card, Collapse, CollapseProps, Form, FormListFieldData, Input, Radio, Row, Tooltip } from 'antd';
import { MinusCircleOutlined, PlusOutlined } from '@ant-design/icons';

export const HooksFormList = ({ hooks }: { hooks: Hook[] }) => {
  return <Form.List name="hooks" initialValue={hooks || []}>
    {(fields, { add, remove }, { errors }) => (
      <>
        {fields.map((field, index) => {
          // const items: CollapseProps["items"] = [
          //   {
          //     key: "1",
          //     label: <>
          //       <strong>Hook {(index + 1)}</strong>
          //       <MinusCircleOutlined
          //         className="dynamic-delete-button"
          //         onClick={() => remove(field.name)}
          //         style={{ marginRight: "5px", float: "right" }}
          //       />
          //     </>,
          //     children: 
          //   }
          // ];

          return <Card key={field.key}>
            <HookBuilder hook={hooks[index]} field={field} />
          </Card>
        })}
        <Form.Item>
          <Button
            type="dashed"
            onClick={() => add()}
            style={{ width: "60%" }}
            icon={<PlusOutlined />}
          >
            Add Hook
          </Button>
          <Form.ErrorList errors={errors} />
        </Form.Item>
      </>
    )}
  </Form.List >
}

const HookBuilder = ({ field, hook }: { field: FormListFieldData, hook?: Hook }) => {
  const [hookVal, setHookVal] = useState<Hook>(hook || new Hook());

  let component: React.ReactNode;
  switch (hookVal.action.case) {
    case "actionDiscord":
      component = <Form.Item name={[field.name, "actionDiscord", "webhookUrl"]} initialValue={hookVal.action.value ? hookVal.action.value.webhookUrl : ""}>
        <Input addonBefore={<div style={{ width: "8em" }}>Webhook URL</div>} />
      </Form.Item>
      break;
    case "actionWebhook":
      component = <Form.Item name={[field.name, "actionWebhook", "webhookUrl"]} initialValue={hookVal.action.value ? hookVal.action.value.webhookUrl : ""}>
        <Input addonBefore={<div style={{ width: "8em" }}>Webhook URL</div>} />
      </Form.Item>
      break;
    case "actionCommand":
    default:
      component = <>
        <Tooltip title="Script to execute. Commands will not work in the docker build of Backrest.">
          Script:
        </Tooltip>
        <Form.Item name={[field.name, "actionCommand", "command"]} initialValue={hookVal.action.value ? hookVal.action.value.command : "#!/bin/sh\n"}>
          <Input.TextArea style={{ width: "100%" }} />
        </Form.Item>
      </>
  }

  return <Form.Item>
    <Row>
      <Radio.Group onChange={(val) => {
        const hook = new Hook(hookVal);
        hook.action = {
          case: val.target.value,
          value: undefined,
        }
        setHookVal(hook);
      }}>
        <Radio.Button value="actionCommand">Command</Radio.Button>
        <Radio.Button value="actionWebhook">Webhook</Radio.Button>
        <Radio.Button value="actionDiscord">Discord</Radio.Button>
      </Radio.Group>
    </Row>
    <br />
    <Row>
      {component}
    </Row>
  </Form.Item>
}