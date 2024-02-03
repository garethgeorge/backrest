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
import { MinusCircleOutlined, PlusOutlined } from "@ant-design/icons";
import { useAlertApi } from "../components/Alerts";
import { namePattern, validateForm } from "../lib/formutil";
import { useConfig } from "../components/ConfigProvider";
import { authenticationService, backrestService } from "../api";

interface FormData {
  auth: {
    users: ({
      name: string;
      passwordBcrypt: string;
      needsBcrypt?: boolean;
    })[];
  }
}

export const SettingsModal = () => {
  let [config, setConfig] = useConfig();
  const configObj = JSON.parse(config!.toJsonString());
  const showModal = useShowModal();
  const alertsApi = useAlertApi()!;
  const [form] = Form.useForm<FormData>();

  if (!config) {
    return null;
  }

  const handleOk = async () => {
    try {
      // Validate form
      let formData = await validateForm(form);

      for (const user of formData.auth?.users) {
        if (user.needsBcrypt) {
          const hash = await authenticationService.hashPassword({ value: user.passwordBcrypt });
          user.passwordBcrypt = hash.value;
          delete user.needsBcrypt;
        }
      }

      // Update configuration
      let newConfig = config!.clone();
      newConfig.auth = new Auth().fromJson(formData.auth, { ignoreUnknownFields: false });

      setConfig(await backrestService.setConfig(newConfig));
      alertsApi.success("Settings updated", 5);
      setTimeout(() => {
        window.location.reload();
      }, 500);
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
          {config.auth?.users?.length === 0 ? (
            <>
              <strong>Initial backrest setup! </strong>
              <p>
                Backrest has detected that you do not have any users configured, please add at least one user to secure the web interface.
              </p>
              <p>
                You can add more users later or can reset users by editing the configuration file.
              </p>
            </>
          ) : null}
          <Form.Item label="Users" required={true}>
            <Form.List
              name={["auth", "users"]}
              rules={[
                {
                  validator: async (_, users) => {
                    if (!users || users.length < 1) {
                      return Promise.reject(new Error("At least one user is required"));
                    }
                  },
                },
              ]}
              initialValue={configObj.auth?.users || []}
            >
              {(fields, { add, remove }) => (
                <>
                  {fields.map((field, index) => {

                    return (
                      <Row key={field.key} gutter={16}>
                        <Col span={11}>
                          <Form.Item
                            name={[field.name, "name"]}
                            rules={[{ required: true }, { pattern: namePattern, message: "Name must be alphanumeric with dashes or underscores as separators" }]}
                          >
                            <Input placeholder="Username" />
                          </Form.Item>
                        </Col>
                        <Col span={11}>
                          <Form.Item
                            name={[field.name, "passwordBcrypt"]}
                            rules={[{ required: true }]}
                          >
                            <Input.Password placeholder="Password" onFocus={() => {
                              form.setFieldValue(["auth", "users", index, "needsBcrypt"], true);
                              form.setFieldValue(["auth", "users", index, "passwordBcrypt"], "");
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
                    )
                  })}
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
