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
  Tooltip,
  Select,
} from "antd";
import React, { useEffect, useState } from "react";
import { useShowModal } from "../components/ModalManager";
import {
  MinusCircleOutlined,
  PlusOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
  CloseCircleOutlined,
  LoadingOutlined,
  QuestionCircleOutlined,
  StopOutlined,
  ClockCircleOutlined,
  KeyOutlined,
  DisconnectOutlined,
} from "@ant-design/icons";
import { formatErrorAlert, useAlertApi } from "../components/Alerts";
import { namePattern, validateForm } from "../lib/formutil";
import { useConfig } from "../components/ConfigProvider";
import {
  authenticationService,
  backrestService,
  syncStateService,
} from "../api";
import { clone, fromJson, toJson, toJsonString } from "@bufbuild/protobuf";
import {
  AuthSchema,
  Config,
  ConfigSchema,
  UserSchema,
  MultihostSchema,
  Multihost_PeerSchema,
  Multihost_PermissionSchema,
  Multihost_Permission_Type,
} from "../../gen/ts/v1/config_pb";
import {
  SyncConnectionState,
  SyncStateStreamItem,
  SyncStreamItem,
} from "../../gen/ts/v1/syncservice_pb";
import { abort } from "process";
import { subscribeToKnownHostSyncStates } from "../state/syncstate";

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
      permissions?: {
        type: number;
        scopes: string[];
      }[];
    }[];
    authorizedClients: {
      instanceId: string;
      keyid: string;
      keyidVerified?: boolean;
      permissions?: {
        type: number;
        scopes: string[];
      }[];
    }[];
  };
}

export const SettingsModal = () => {
  let [config, setConfig] = useConfig();
  const showModal = useShowModal();
  const alertsApi = useAlertApi()!;
  const [form] = Form.useForm<FormData>();
  const [syncState, setSyncState] = useState<SyncStateStreamItem[]>([]);

  if (!config) {
    return null;
  }

  useEffect(() => {
    const abortController = new AbortController();
    subscribeToKnownHostSyncStates(
      abortController,
      (syncStates: SyncStateStreamItem[]) => {
        setSyncState(syncStates);
      }
    );

    return () => {
      abortController.abort();
    };
  }, []);

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
      newConfig.multihost = fromJson(MultihostSchema, formData.multihost, {
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
                forceRender: true,
                children: <AuthenticationForm form={form} config={config} />,
              },
              {
                key: "2",
                label: "Multihost Identity and Sharing",
                forceRender: true,
                children: (
                  <MultihostIdentityForm
                    form={form}
                    config={config}
                    syncState={syncState}
                  />
                ),
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
  syncState: SyncStateStreamItem[];
}> = ({ form, config, syncState }) => {
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
          showSyncState={false}
          syncState={syncState}
          config={config}
          listType="authorizedClients"
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
          showSyncState={true}
          syncState={syncState}
          config={config}
          listType="knownHosts"
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
  showSyncState: boolean;
  itemTypeName: string;
  syncState: SyncStateStreamItem[];
  initialValue: any[];
  config: Config;
  listType: "knownHosts" | "authorizedClients";
}> = ({
  form,
  listName,
  showInstanceUrl,
  showSyncState,
  itemTypeName,
  syncState,
  initialValue,
  config,
  listType,
}) => {
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
              showSyncState={showSyncState}
              syncState={syncState}
              index={index}
              config={config}
              listType={listType}
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
  showSyncState: boolean;
  syncState: SyncStateStreamItem[];
  index: number;
  config: Config;
  listType: "knownHosts" | "authorizedClients";
}> = ({
  form,
  fieldName,
  remove,
  showInstanceUrl,
  showSyncState,
  syncState,
  index,
  config,
  listType,
}) => {
  // Get the instance ID from the form to find the matching sync state
  const instanceId =
    form.getFieldValue(["multihost", "knownHosts", index, "instanceId"]) ||
    form.getFieldValue(["multihost", "authorizedClients", index, "instanceId"]);

  const peerSyncState = syncState.find(
    (state) => state.peerInstanceId === instanceId
  );

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
      <div
        style={{
          position: "absolute",
          top: "8px",
          right: "8px",
          display: "flex",
          alignItems: "center",
          gap: "8px",
        }}
      >
        {showSyncState && peerSyncState && (
          <SyncStateTile syncState={peerSyncState} />
        )}
        <MinusCircleOutlined
          style={{
            color: "#999",
            cursor: "pointer",
          }}
          onClick={() => remove(fieldName)}
        />
      </div>

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
            name={[fieldName, "keyId"]}
            label="Key ID"
            rules={[{ required: true, message: "Key ID is required" }]}
          >
            <Input placeholder="Public key identifier" />
          </Form.Item>
        </Col>
        <Col span={4}>
          <Form.Item
            name={[fieldName, "keyIdVerified"]}
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

      <PeerPermissionsTile
        form={form}
        fieldName={fieldName}
        listType={listType}
        config={config}
      />
    </div>
  );
};

export const SyncStateTile = ({
  syncState,
}: {
  syncState: SyncStateStreamItem;
}) => {
  const getStatusIcon = () => {
    switch (syncState.state) {
      case SyncConnectionState.CONNECTION_STATE_CONNECTED:
        return (
          <CheckCircleOutlined style={{ color: "#52c41a", fontSize: "16px" }} />
        );

      case SyncConnectionState.CONNECTION_STATE_PENDING:
        return (
          <LoadingOutlined style={{ color: "#1890ff", fontSize: "16px" }} />
        );

      case SyncConnectionState.CONNECTION_STATE_RETRY_WAIT:
        return (
          <ClockCircleOutlined style={{ color: "#faad14", fontSize: "16px" }} />
        );

      case SyncConnectionState.CONNECTION_STATE_DISCONNECTED:
        return (
          <DisconnectOutlined style={{ color: "#d9d9d9", fontSize: "16px" }} />
        );

      case SyncConnectionState.CONNECTION_STATE_ERROR_AUTH:
        return <KeyOutlined style={{ color: "#ff4d4f", fontSize: "16px" }} />;

      case SyncConnectionState.CONNECTION_STATE_ERROR_PROTOCOL:
        return (
          <ExclamationCircleOutlined
            style={{ color: "#ff4d4f", fontSize: "16px" }}
          />
        );

      case SyncConnectionState.CONNECTION_STATE_ERROR_INTERNAL:
        return (
          <CloseCircleOutlined style={{ color: "#ff4d4f", fontSize: "16px" }} />
        );

      case SyncConnectionState.CONNECTION_STATE_UNKNOWN:
      default:
        return (
          <QuestionCircleOutlined
            style={{ color: "#8c8c8c", fontSize: "16px" }}
          />
        );
    }
  };

  const getStatusTooltip = () => {
    const baseMessage = `${syncState.peerInstanceId}: `;

    switch (syncState.state) {
      case SyncConnectionState.CONNECTION_STATE_CONNECTED:
        return (
          baseMessage +
          "Connected" +
          (syncState.statusMessage ? ` - ${syncState.statusMessage}` : "")
        );
      case SyncConnectionState.CONNECTION_STATE_PENDING:
        return (
          baseMessage +
          "Connecting..." +
          (syncState.statusMessage ? ` - ${syncState.statusMessage}` : "")
        );

      case SyncConnectionState.CONNECTION_STATE_RETRY_WAIT:
        return (
          baseMessage +
          "Retrying connection" +
          (syncState.statusMessage ? ` - ${syncState.statusMessage}` : "")
        );

      case SyncConnectionState.CONNECTION_STATE_DISCONNECTED:
        return (
          baseMessage +
          "Disconnected" +
          (syncState.statusMessage ? ` - ${syncState.statusMessage}` : "")
        );

      case SyncConnectionState.CONNECTION_STATE_ERROR_AUTH:
        return (
          baseMessage +
          "Authentication error" +
          (syncState.statusMessage ? ` - ${syncState.statusMessage}` : "")
        );

      case SyncConnectionState.CONNECTION_STATE_ERROR_PROTOCOL:
        return (
          baseMessage +
          "Protocol error" +
          (syncState.statusMessage ? ` - ${syncState.statusMessage}` : "")
        );

      case SyncConnectionState.CONNECTION_STATE_ERROR_INTERNAL:
        return (
          baseMessage +
          "Internal error" +
          (syncState.statusMessage ? ` - ${syncState.statusMessage}` : "")
        );

      case SyncConnectionState.CONNECTION_STATE_UNKNOWN:
      default:
        return (
          baseMessage +
          "Unknown status" +
          (syncState.statusMessage ? ` - ${syncState.statusMessage}` : "")
        );
    }
  };

  return (
    <Tooltip title={getStatusTooltip()} placement="top">
      <span
        style={{ cursor: "help", display: "inline-flex", alignItems: "center" }}
      >
        {getStatusIcon()}
      </span>
    </Tooltip>
  );
};

const PeerPermissionsTile: React.FC<{
  form: FormInstance<FormData>;
  fieldName: number;
  listType: "knownHosts" | "authorizedClients";
  config: Config;
}> = ({ form, fieldName, listType, config }) => {
  const repoOptions = (config.repos || []).map((repo) => ({
    label: repo.id,
    value: `repo:${repo.id}`,
  }));

  return (
    <div
    // style={{
    //   marginTop: "16px",
    //   padding: "12px",
    //   borderRadius: "6px",
    //   border: "1px solid #e8e8e8",
    //   background: "none",
    // }}
    >
      <Typography.Text strong style={{ marginBottom: "8px", display: "block" }}>
        Permissions
      </Typography.Text>

      <Form.List name={[fieldName, "permissions"]}>
        {(
          permissionFields,
          { add: addPermission, remove: removePermission }
        ) => (
          <>
            {permissionFields.map((permissionField) => (
              <div
                key={permissionField.key}
                style={{
                  border: "1px solid #d9d9d9",
                  borderRadius: "4px",
                  padding: "12px",
                  marginBottom: "8px",
                  backgroundColor: "transparent",
                }}
              >
                <Row gutter={8} align="middle">
                  <Col span={8}>
                    <Form.Item
                      name={[permissionField.name, "type"]}
                      label="Permission Type"
                      rules={[
                        {
                          required: true,
                          message: "Permission type is required",
                        },
                      ]}
                    >
                      <Select placeholder="Select permission type">
                        <Select.Option
                          value={
                            Multihost_Permission_Type.PERMISSION_READ_REPO_CONFIG
                          }
                        >
                          Read Repo Config
                        </Select.Option>
                        <Select.Option
                          value={
                            Multihost_Permission_Type.PERMISSION_READ_OPERATIONS
                          }
                        >
                          Read Operations
                        </Select.Option>
                      </Select>
                    </Form.Item>
                  </Col>
                  <Col span={14}>
                    <Form.Item
                      name={[permissionField.name, "scopes"]}
                      label="Repository Scopes"
                      rules={[
                        {
                          required: true,
                          message: "At least one scope is required",
                        },
                      ]}
                    >
                      <Select
                        mode="multiple"
                        placeholder="Select repositories or use * for all"
                        options={[
                          { label: "All Repositories (*)", value: "*" },
                          ...repoOptions,
                        ]}
                      />
                    </Form.Item>
                  </Col>
                  <Col span={2}>
                    <MinusCircleOutlined
                      style={{
                        color: "#999",
                        cursor: "pointer",
                        fontSize: "16px",
                      }}
                      onClick={() => removePermission(permissionField.name)}
                    />
                  </Col>
                </Row>
              </div>
            ))}
            <Button
              type="dashed"
              onClick={() =>
                addPermission({
                  type: Multihost_Permission_Type.PERMISSION_READ_REPO_CONFIG,
                  scopes: [],
                })
              }
              icon={<PlusOutlined />}
              size="small"
              style={{ width: "100%" }}
            >
              Add Permission
            </Button>
          </>
        )}
      </Form.List>
    </div>
  );
};
