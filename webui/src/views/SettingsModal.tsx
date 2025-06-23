import {
  Form,
  Modal,
  Input,
  Typography,
  Button,
  Row,
  Col,
  Collapse,
  Checkbox,
  FormInstance,
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
  Config,
  ConfigSchema,
  UserSchema,
  MultihostSchema,
  Multihost_PeerSchema,
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
  multihost: {
    identity: {
      keyId: string;
    };
    knownHosts: {
      instanceId: string;
      keyid: string;
      keyidVerified?: boolean;
      instanceUrl: string;
    }[];
    authorizedClients: {
      instanceId: string;
      keyid: string;
      keyidVerified?: boolean;
    }[];
  };
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

      // Update multihost configuration if provided
      if (formData.multihost) {
        if (!newConfig.multihost) {
          newConfig.multihost = fromJson(
            MultihostSchema,
            {},
            {
              ignoreUnknownFields: false,
            }
          );
        }
        if (formData.multihost.authorizedClients) {
          newConfig.multihost.authorizedClients =
            formData.multihost.authorizedClients.map((peer) =>
              fromJson(Multihost_PeerSchema, peer, {
                ignoreUnknownFields: false,
              })
            );
        }
        if (formData.multihost.knownHosts) {
          newConfig.multihost.knownHosts = formData.multihost.knownHosts.map(
            (peer) =>
              fromJson(Multihost_PeerSchema, peer, {
                ignoreUnknownFields: false,
              })
          );
        }
      }

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
        width="60vw"
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
          labelCol={{ span: 4 }}
          wrapperCol={{ span: 20 }}
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
          <Form.Item
            hasFeedback
            name="instance"
            label="Instance ID"
            required
            initialValue={config.instance || ""}
            tooltip="The instance name will be used to identify this backrest install. Pick a value carefully as it cannot be changed later."
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

          <Collapse
            items={[
              {
                key: "1",
                label: "Authentication",
                children: <AuthenticationForm form={form} config={config} />,
              },
              {
                key: "2",
                label: "Multihost Identity and Sharing",
                children: <MultihostIdentityForm form={form} config={config} />,
              },
              {
                key: "last",
                label: "Preview",
                children: (
                  <Form.Item shouldUpdate wrapperCol={{ span: 24 }}>
                    {() => (
                      <Typography>
                        <pre>
                          {JSON.stringify(form.getFieldsValue(), null, 2)}
                        </pre>
                      </Typography>
                    )}
                  </Form.Item>
                ),
              },
            ]}
          />
        </Form>
      </Modal>
    </>
  );
};

const AuthenticationForm: React.FC<{
  config: Config;
  form: FormInstance<FormData>;
}> = ({ form, config }) => {
  return (
    <>
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
                          {
                            required: true,
                            message: "Name is required",
                          },
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
    </>
  );
};

const MultihostIdentityForm: React.FC<{
  config: Config;
  form: FormInstance<FormData>;
}> = ({ form, config }) => {
  return (
    <>
      {/* Show the current instance's identity */}
      <Form.Item
        label="Multihost Identity"
        name={["multihost", "identity", "keyId"]}
        initialValue={config.multihost?.identity?.keyid || ""}
        rules={[
          {
            required: true,
            message: "Multihost identity is required",
          },
        ]}
        tooltip="Multihost identity is used to identify this instance in a multihost setup. It is cryptographically derived from the public key of this instance."
        wrapperCol={{ span: 16 }}
      >
        <Row>
          <Col flex="auto">
            <Input
              placeholder="Unique multihost identity"
              disabled
              value={config.multihost?.identity?.keyid}
            />
          </Col>
          <Col>
            <Button
              type="link"
              onClick={() =>
                navigator.clipboard.writeText(
                  config.multihost?.identity?.keyid || ""
                )
              }
            >
              copy
            </Button>
          </Col>
        </Row>
      </Form.Item>

      {/* Authorized client peers. */}
      <Form.Item
        label="Authorized Clients"
        tooltip="Authorized clients are other Backrest instances that are allowed to access repositories on this instance."
      >
        <PeerFormList
          form={form}
          listName={["multihost", "authorizedClients"]}
          showInstanceUrl={false}
          itemTypeName="Authorized Client"
          initialValue={
            config.multihost?.authorizedClients?.map((peer) =>
              toJson(Multihost_PeerSchema, peer, { alwaysEmitImplicit: true })
            ) || []
          }
        />
      </Form.Item>

      {/* Known host peers. */}
      <Form.Item
        label="Known Hosts"
        tooltip="Known hosts are other Backrest instances that this instance can connect to."
      >
        <PeerFormList
          form={form}
          listName={["multihost", "knownHosts"]}
          showInstanceUrl={true}
          itemTypeName="Known Host"
          initialValue={
            config.multihost?.knownHosts?.map((peer) =>
              toJson(Multihost_PeerSchema, peer, { alwaysEmitImplicit: true })
            ) || []
          }
        />
      </Form.Item>
    </>
  );
};

const PeerFormList: React.FC<{
  form: FormInstance<FormData>;
  listName: string[];
  showInstanceUrl: boolean;
  itemTypeName: string;
  initialValue: any[];
}> = ({ form, listName, showInstanceUrl, itemTypeName, initialValue }) => {
  return (
    <Form.List name={listName} initialValue={initialValue}>
      {(fields, { add, remove }, { errors }) => (
        <>
          {fields.map((field, index) => (
            <PeerFormListItem
              key={field.key}
              form={form}
              fieldName={field.name}
              remove={remove}
              showInstanceUrl={showInstanceUrl}
              index={index}
            />
          ))}
          <Form.Item>
            <Button
              type="dashed"
              onClick={() => add({})}
              block
              icon={<PlusOutlined />}
            >
              Add {itemTypeName || "Peer"}
            </Button>
            <Form.ErrorList errors={errors} />
          </Form.Item>
        </>
      )}
    </Form.List>
  );
};

const PeerFormListItem: React.FC<{
  form: FormInstance<FormData>;
  fieldName: number;
  remove: (index: number | number[]) => void;
  showInstanceUrl: boolean;
  index: number;
}> = ({ form, fieldName, remove, showInstanceUrl, index }) => {
  return (
    <div
      style={{
        border: "1px solid #d9d9d9",
        borderRadius: "6px",
        padding: "16px",
        marginBottom: "16px",
        position: "relative",
      }}
    >
      <MinusCircleOutlined
        style={{
          position: "absolute",
          top: "8px",
          right: "8px",
          color: "#999",
          cursor: "pointer",
        }}
        onClick={() => remove(fieldName)}
      />

      <Row gutter={16}>
        <Col span={10}>
          <Form.Item
            name={[fieldName, "instanceId"]}
            label="Instance ID"
            rules={[
              { required: true, message: "Instance ID is required" },
              {
                pattern: namePattern,
                message:
                  "Instance ID must be alphanumeric with '_-.' allowed as separators",
              },
            ]}
          >
            <Input placeholder="e.g. my-backup-server" />
          </Form.Item>
        </Col>
        <Col span={10}>
          <Form.Item
            name={[fieldName, "keyid"]}
            label="Key ID"
            rules={[{ required: true, message: "Key ID is required" }]}
          >
            <Input placeholder="Public key identifier" />
          </Form.Item>
        </Col>
        <Col span={4}>
          <Form.Item
            name={[fieldName, "keyidVerified"]}
            valuePropName="checked"
          >
            <Checkbox>Verified</Checkbox>
          </Form.Item>
        </Col>
      </Row>

      {showInstanceUrl && (
        <Row gutter={16}>
          <Col span={24}>
            <Form.Item
              name={[fieldName, "instanceUrl"]}
              label="Instance URL"
              rules={[
                {
                  required: showInstanceUrl,
                  message: "Instance URL is required for known hosts",
                },
                { type: "url", message: "Please enter a valid URL" },
              ]}
            >
              <Input placeholder="https://example.com:9898" />
            </Form.Item>
          </Col>
        </Row>
      )}
    </div>
  );
};
