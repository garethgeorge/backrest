import {
  Form,
  Modal,
  Input,
  Typography,
  Select,
  Button,
  Tooltip,
  Radio,
  InputNumber,
  Row,
  Card,
  Col,
  Collapse,
} from "antd";
import React, { useEffect, useState } from "react";
import { useShowModal } from "../components/ModalManager";
import { Auth, Config, User } from "../../gen/ts/v1/config_pb";
import { useRecoilState } from "recoil";
import { configState, updateConfig } from "../state/config";
import { MinusCircleOutlined, PlusOutlined } from "@ant-design/icons";
import { useAlertApi } from "../components/Alerts";
import { validateForm } from "../lib/formutil";
import { authenticationService } from "../api";
import { StringValue } from "../../gen/ts/types/value_pb";

interface SettingsFormData {
  users: User[],
}

export const SettingsModal = () => {
  let [config, setConfig] = useRecoilState(configState);
  const showModal = useShowModal();
  const alertsApi = useAlertApi()!;
  const [form] = Form.useForm<SettingsFormData>();
  form.setFieldsValue({
    users: config.auth?.users || [],
  });

  const data: SettingsFormData = {
    users: config.auth?.users || [],
  };

  data.users[0].password.value

  const handleOk = async () => {
    try {
      let formData = await validateForm<SettingsFormData>(form);

      let copy = config.clone();
      copy.auth ||= new Auth();
      copy.auth.users = formData.users.map((u) => {
        return new User({
          name: u.name,
          password: u.password,
        });
      });

      // Update config and notify success.
      setConfig(await updateConfig(copy));
      showModal(null);
      alertsApi.success("Settings updated", 5);
    } catch (e: any) {
      alertsApi.error("Operation failed: " + e.message, 15);
      console.error(e);
    }
  };

  const handleCancel = () => {
    showModal(null);
  };

  return (
    <>
      <Modal
        open={true}
        onCancel={handleCancel}
        title={"Settings"}
        width="40vw"
        footer={[
          <Button key="back" onClick={handleCancel}>
            Cancel
          </Button>,
          <Button
            key="submit"
            type="primary"
            onClick={handleOk}
          >
            Submit
          </Button>,
        ]}
      >
        <Form
          autoComplete="off"
          form={form}
          labelCol={{ span: 6 }}
          wrapperCol={{ span: 16 }}
        >
          <Form.Item label="Users" required={true}>
            <Form.List
              name={["users"]}
              rules={[
                {
                  validator: async (_, users) => {
                    if (!users || users.length < 1) {
                      return Promise.reject(new Error("At least one user is required"));
                    }
                  },
                },
              ]}
              initialValue={config.auth?.users || []}
            >
              {(fields, { add, remove }) => (
                <>
                  {fields.map((field, index) => (
                    <Row key={field.key} gutter={16}>
                      <Col span={11}>
                        <Form.Item
                          name={[field.name, "name"]}
                          rules={[{ required: true }]}
                        >
                          <Input placeholder="Username" />
                        </Form.Item>
                      </Col>
                      <Col span={11}>
                        <Form.Item
                          name={[field.name, "password", "value"]}
                          rules={[{ required: true }]}
                        >
                          <Input.Password placeholder="Password" onFocus={(e) => {
                            form.setFieldValue(["users", field.name], new User({
                              name: form.getFieldValue(["users", field.name, "name"]),
                              password: {
                                case: "passwordBcrypt",
                                value: "",
                              },
                            }));
                          }} onBlur={(e) => {
                            authenticationService.hashPassword(new StringValue({ value: e.target.value })).then((res) => {
                              form.setFieldValue(["users", field.name, "password", "value"], res.value);
                            })
                          }} />
                        </Form.Item>
                      </Col>
                      <Col span={2}>
                        <MinusCircleOutlined
                          onClick={() => {
                            remove(field.name);
                          }}
                        />
                      </Col>
                    </Row>
                  ))}
                  <Form.Item>
                    <Button
                      type="dashed"
                      onClick={() => {
                        add();
                      }}
                      block
                    >
                      <PlusOutlined /> Add user
                    </Button>
                  </Form.Item>
                </>
              )}
            </Form.List>
          </Form.Item>

          <Form.Item shouldUpdate label="Preview">
            {() => (
              <Collapse
                size="small"
                items={[
                  {
                    key: "1",
                    label: "Config as JSON",
                    children: (
                      <Typography>
                        <pre>{JSON.stringify(form.getFieldsValue(), null, 2)}</pre>
                      </Typography>
                    ),
                  },
                ]}
              />
            )}
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};
