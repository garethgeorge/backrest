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
  Checkbox,
} from "antd";
import React, { useEffect, useState } from "react";
import { useShowModal } from "../components/ModalManager";
import { MinusCircleOutlined, PlusOutlined } from "@ant-design/icons";
import { formatErrorAlert, useAlertApi } from "../components/Alerts";
import { namePattern, validateForm } from "../lib/formutil";
import { useConfig } from "../components/ConfigProvider";
import { authenticationService, backrestService } from "../api";
import { clone, fromJson, toJson, toJsonString } from "@bufbuild/protobuf";
import {
  AuthSchema,
  ConfigSchema,
  UserSchema,
} from "../../gen/ts/v1/config_pb";

interface FormData {
  auth: {
    users: {
      name: string;
      passwordBcrypt: string;
      needsBcrypt?: boolean;
    }[];
  };
  instance: string;
}

export const SettingsModal = () => {
  let [config, setConfig] = useConfig();
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

      if (formData.auth?.users) {
        for (const user of formData.auth?.users) {
          if (user.needsBcrypt) {
            const hash = await authenticationService.hashPassword({
              value: user.passwordBcrypt,
            });
            user.passwordBcrypt = hash.value;
            delete user.needsBcrypt;
          }
        }
      }

      // Update configuration
      let newConfig = clone(ConfigSchema, config);
      newConfig.auth = fromJson(AuthSchema, formData.auth, {
        ignoreUnknownFields: false,
      });
      newConfig.instance = formData.instance;

      if (!newConfig.auth?.users && !newConfig.auth?.disabled) {
        throw new Error(
          "At least one user must be configured or authentication must be disabled"
        );
      }

      setConfig(await backrestService.setConfig(newConfig));
      alertsApi.success("Settings updated", 5);
      setTimeout(() => {
        window.location.reload();
      }, 500);
    } catch (e: any) {
      alertsApi.error(formatErrorAlert(e, "Operation error: "), 15);
      console.error(e);
    }
  };

  const handleCancel = () => {
    showModal(null);
  };

  const users = config.auth?.users || [];

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
          <Button key="submit" type="primary" onClick={handleOk}>
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
          {users.length > 0 || config.auth?.disabled ? null : (
            <>
              <strong>Initial backrest setup! </strong>
              <p>
                Backrest has detected that you do not have any users configured,
                please add at least one user to secure the web interface.
              </p>
              <p>
                You can add more users later or, if you forget your password,
                reset users by editing the configuration file (typically in
                $HOME/.backrest/config.json)
              </p>
            </>
          )}
          <Tooltip title="The instance name will be used to identify this backrest install. Pick a value carefully as it cannot be changed later.">
            <Form.Item
              hasFeedback
              name="instance"
              label="Instance ID"
              required
              initialValue={config.instance || ""}
              rules={[
                { required: true, message: "Instance ID is required" },
                {
                  pattern: namePattern,
                  message:
                    "Instance ID must be alphanumeric with '_-.' allowed as separators",
                },
              ]}
            >
              <Input
                placeholder={
                  "Unique instance ID for this instance (e.g. my-backrest-server)"
                }
                disabled={!!config.instance}
              />
            </Form.Item>
          </Tooltip>
          <Form.Item
            label="Disable Authentication"
            name={["auth", "disabled"]}
            valuePropName="checked"
            initialValue={config.auth?.disabled || false}
          >
            <Checkbox />
          </Form.Item>
          <Form.Item label="Users" required={true}>
            <Form.List
              name={["auth", "users"]}
              initialValue={
                config.auth?.users?.map((u) =>
                  toJson(UserSchema, u, { alwaysEmitImplicit: true })
                ) || []
              }
            >
              {(fields, { add, remove }) => (
                <>
                  {fields.map((field, index) => {
                    return (
                      <Row key={field.key} gutter={16}>
                        <Col span={11}>
                          <Form.Item
                            name={[field.name, "name"]}
                            rules={[
                              { required: true, message: "Name is required" },
                              {
                                pattern: namePattern,
                                message:
                                  "Name must be alphanumeric with dashes or underscores as separators",
                              },
                            ]}
                          >
                            <Input placeholder="Username" />
                          </Form.Item>
                        </Col>
                        <Col span={11}>
                          <Form.Item
                            name={[field.name, "passwordBcrypt"]}
                            rules={[
                              {
                                required: true,
                                message: "Password is required",
                              },
                            ]}
                          >
                            <Input.Password
                              placeholder="Password"
                              onFocus={() => {
                                form.setFieldValue(
                                  ["auth", "users", index, "needsBcrypt"],
                                  true
                                );
                                form.setFieldValue(
                                  ["auth", "users", index, "passwordBcrypt"],
                                  ""
                                );
                              }}
                            />
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
                    );
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
                        <pre>
                          {JSON.stringify(form.getFieldsValue(), null, 2)}
                        </pre>
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
