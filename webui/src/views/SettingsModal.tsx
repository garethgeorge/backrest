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
import { MinusCircleOutlined, PlusOutlined } from "@ant-design/icons";
import { formatErrorAlert, useAlertApi } from "../components/Alerts";
import { namePattern, validateForm } from "../lib/formutil";
import { useConfig } from "../components/ConfigProvider";
import { authenticationService, backrestService } from "../api";
import { clone, fromJson, toJson } from "@bufbuild/protobuf";
import {
  AuthSchema,
  Config,
  ConfigSchema,
  UserSchema,
  MultihostSchema,
  Multihost_PeerSchema,
  Multihost_Permission_Type,
} from "../../gen/ts/v1/config_pb";
import { PeerState } from "../../gen/ts/v1sync/syncservice_pb";
import { useSyncStates } from "../state/peerstates";
import { PeerStateConnectionStatusIcon } from "../components/SyncStateIcon";
import { isMultihostSyncEnabled } from "../state/buildcfg";
import * as m from "../paraglide/messages";

interface FormData {
  auth: {
    users: {
      name: string;
      passwordBcrypt: string;
      needsBcrypt?: boolean;
      isExisting?: boolean;
    }[];
  };
  instance: string;
  multihost: {
    identity: {
      keyId: string;
    };
    knownHosts: {
      instanceId: string;
      keyId: string;
      keyIdVerified?: boolean;
      instanceUrl: string;
      permissions?: {
        type: number;
        scopes: string[];
      }[];
    }[];
    authorizedClients: {
      instanceId: string;
      keyId: string;
      keyIdVerified?: boolean;
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
  const peerStates = useSyncStates();
  const [reloadOnCancel, setReloadOnCancel] = useState(false);
  const [formEdited, setFormEdited] = useState(false);

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
          delete user.isExisting;
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
      setReloadOnCancel(true);
      alertsApi.success(m.settings_success_updated(), 5);
      setFormEdited(false);
    } catch (e: any) {
      alertsApi.error(formatErrorAlert(e, m.settings_error_operation()), 15);
    }
  };

  const handleCancel = () => {
    showModal(null);
    if (reloadOnCancel) {
      window.location.reload();
    }
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
            {formEdited ? m.button_cancel() : m.button_close()}
          </Button>,
          <Button key="submit" type="primary" onClick={handleOk}>
            {m.button_save()}
          </Button>,
        ]}
      >
        <Form
          autoComplete="off"
          form={form}
          labelCol={{ span: 4 }}
          wrapperCol={{ span: 20 }}
          onValuesChange={() => setFormEdited(true)}
        >
          {users.length > 0 || config.auth?.disabled ? null : (
            <>
              <strong>{m.settings_initial_setup_title()}</strong>
              <p>
                {m.settings_initial_setup_message()}
              </p>
              <p>
                {m.settings_initial_setup_hint()}
              </p>
            </>
          )}
          <Form.Item
            hasFeedback
            name="instance"
            label={m.settings_field_instance_id()}
            required
            initialValue={config.instance || ""}
            tooltip={m.settings_field_instance_id_tooltip()}
            rules={[
              { required: true, message: m.settings_validation_instance_id_required() },
              {
                pattern: namePattern,
                message:
                  m.settings_validation_instance_id_pattern(),
              },
            ]}
          >
            <Input
              placeholder={
                m.settings_field_instance_id_placeholder()
              }
              disabled={!!config.instance}
            />
          </Form.Item>

          <Collapse
            items={[
              {
                key: "1",
                label: m.settings_section_authentication(),
                forceRender: true,
                children: <AuthenticationForm form={form} config={config} />,
              },
              {
                key: "2",
                label: m.settings_section_multihost(),
                forceRender: true,
                children: (
                  <MultihostIdentityForm
                    form={form}
                    config={config}
                    peerStates={peerStates}
                  />
                ),
                style: isMultihostSyncEnabled ? undefined : { display: "none" },
              },
              {
                key: "last",
                label: m.settings_section_preview(),
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
        label={m.settings_auth_disable()}
        name={["auth", "disabled"]}
        valuePropName="checked"
        initialValue={config.auth?.disabled || false}
      >
        <Checkbox />
      </Form.Item>

      <Form.Item label={m.settings_auth_users()} required={true}>
        <Form.List
          name={["auth", "users"]}
          initialValue={
            config.auth?.users?.map((u) => ({
              ...(toJson(UserSchema, u, { alwaysEmitImplicit: true }) as any),
              isExisting: true,
            })) || []
          }
        >
          {(fields, { add, remove }) => (
            <>
              {fields.map((field, index) => {
                return (
                  <Row key={field.key} gutter={16}>
                    <Col span={11}>
                      <Form.Item
                        name={[field.name, "isExisting"]}
                        hidden={true}
                        initialValue={false}
                      >
                        <Input />
                      </Form.Item>
                      <Form.Item shouldUpdate noStyle>
                        {(form) => {
                          const isExisting = form.getFieldValue([
                            "auth",
                            "users",
                            field.name,
                            "isExisting",
                          ]);
                          return (
                            <Form.Item
                              name={[field.name, "name"]}
                              rules={[
                                {
                                  required: true,
                                  message: m.settings_auth_name_required(),
                                },
                                {
                                  pattern: namePattern,
                                  message:
                                    m.settings_auth_name_pattern(),
                                },
                              ]}
                            >
                              <Input
                                placeholder={m.settings_auth_username_placeholder()}
                                disabled={isExisting}
                              />
                            </Form.Item>
                          );
                        }}
                      </Form.Item>
                    </Col>
                    <Col span={11}>
                      <Form.Item
                        name={[field.name, "passwordBcrypt"]}
                        rules={[
                          {
                            required: true,
                            message: m.settings_auth_password_required(),
                          },
                        ]}
                      >
                        <Input.Password
                          placeholder={m.settings_auth_password_placeholder()}
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
                  <PlusOutlined /> {m.settings_auth_add_user()}
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
  peerStates: PeerState[];
}> = ({ form, config, peerStates }) => {
  return (
    <>
      <Typography.Paragraph italic>
        {m.settings_multihost_intro()}
      </Typography.Paragraph>
      <Typography.Paragraph italic>
        {m.settings_multihost_warning()}
      </Typography.Paragraph>

      {/* Show the current instance's identity */}
      <Form.Item
        label={m.settings_multihost_identity()}
        name={["multihost", "identity", "keyId"]}
        initialValue={config.multihost?.identity?.keyid || ""}
        rules={[
          {
            required: true,
            message: m.settings_multihost_identity_required(),
          },
        ]}
        tooltip={m.settings_multihost_identity_tooltip()}
        wrapperCol={{ span: 16 }}
      >
        <Row>
          <Col flex="auto">
            <Input
              placeholder={m.settings_multihost_identity_placeholder()}
              disabled
              value={config.multihost?.identity?.keyid}
            />
          </Col>
          <Col>
            <Button
              type="link"
              onClick={() =>
                -navigator.clipboard.writeText(
                  config.multihost?.identity?.keyid || ""
                )
              }
            >
              {m.button_copy()}
            </Button>
          </Col>
        </Row>
      </Form.Item>

      {/* Authorized client peers. */}
      <Form.Item
        label={m.settings_multihost_authorized_clients()}
        tooltip={m.settings_multihost_authorized_clients_tooltip()}
      >
        <PeerFormList
          form={form}
          listName={["multihost", "authorizedClients"]}
          showInstanceUrl={false}
          itemTypeName={m.settings_multihost_authorized_client_item()}
          peerStates={peerStates}
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
        label={m.settings_multihost_known_hosts()}
        tooltip={m.settings_multihost_known_hosts_tooltip()}
      >
        <PeerFormList
          form={form}
          listName={["multihost", "knownHosts"]}
          showInstanceUrl={true}
          itemTypeName={m.settings_multihost_known_host_item()}
          peerStates={peerStates}
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
  itemTypeName: string;
  peerStates: PeerState[];
  initialValue: any[];
  config: Config;
  listType: "knownHosts" | "authorizedClients";
}> = ({
  form,
  listName,
  showInstanceUrl,
  itemTypeName,
  peerStates,
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
              peerStates={peerStates}
              isKnownHost={listType === "knownHosts"}
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
              {m.settings_peer_add_button({ itemTypeName: itemTypeName || m.peer_default_name() })}
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
  peerStates: PeerState[];
  isKnownHost?: boolean;
  index: number;
  config: Config;
  listType: "knownHosts" | "authorizedClients";
}> = ({
  form,
  fieldName,
  remove,
  showInstanceUrl,
  peerStates,
  isKnownHost = false,
  index,
  config,
  listType,
}) => {
  // Get the instance ID from the form to find the matching sync state, its a bit hacky but works reliably.
  const keyId = isKnownHost
    ? form.getFieldValue(["multihost", "knownHosts", index, "keyId"])
    : form.getFieldValue(["multihost", "authorizedClients", index, "keyId"]);

  const peerState = peerStates.find((state) => state.peerKeyid === keyId);

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
        {peerState && <PeerStateConnectionStatusIcon peerState={peerState} />}
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
            label={m.settings_peer_instance_id()}
            rules={[
              { required: true, message: m.settings_validation_instance_id_required() },
              {
                pattern: namePattern,
                message:
                  m.settings_validation_instance_id_pattern(),
              },
            ]}
          >
            <Input placeholder={m.settings_peer_instance_id_placeholder()} />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item
            name={[fieldName, "keyId"]}
            label={m.settings_peer_key_id()}
            rules={[{ required: true, message: m.settings_peer_key_id_required() }]}
          >
            <Input placeholder={m.settings_peer_key_id_placeholder()} />
          </Form.Item>
        </Col>
        <Col span={0}>
          <Form.Item
            name={[fieldName, "keyIdVerified"]}
            valuePropName="checked"
            // At the moment, we require clients to explicitly provide keys so there's nothing implicit. Manually checking the box doesn't add much value.
            // It will be more useful if we automate fetching keyids from known hosts in the future / provide a "connection token" like mechanism for easier setup.
            hidden={true}
          >
            <Checkbox defaultChecked={true}>{m.settings_peer_verified()}</Checkbox>
          </Form.Item>
        </Col>
      </Row>

      {showInstanceUrl && (
        <Row gutter={16}>
          <Col span={24}>
            <Form.Item
              name={[fieldName, "instanceUrl"]}
              label={m.settings_peer_instance_url()}
              rules={[
                {
                  required: showInstanceUrl,
                  message: m.settings_peer_instance_url_required(),
                },
                { type: "url", message: m.settings_peer_instance_url_pattern() },
              ]}
            >
              <Input placeholder={m.settings_peer_instance_url_placeholder()} />
            </Form.Item>
          </Col>
        </Row>
      )}

      {/* No meaningful permissions to grant to clients today, only show permissions UI for known hosts */}
      {isKnownHost ? (
        <PeerPermissionsTile
          form={form}
          fieldName={fieldName}
          listType={listType}
          config={config}
        />
      ) : null}
    </div>
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
    <div>
      <Typography.Text strong style={{ marginBottom: "8px", display: "block" }}>
        {m.settings_peer_permissions()}
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
                  <Col span={11}>
                    <Form.Item
                      name={[permissionField.name, "type"]}
                      label={m.settings_peer_permission_type()}
                      rules={[
                        {
                          required: true,
                          message: m.settings_peer_permission_type_required(),
                        },
                      ]}
                    >
                      <Select placeholder={m.settings_permission_type_placeholder()}>
                        <Select.Option
                          value={
                            Multihost_Permission_Type.PERMISSION_READ_WRITE_CONFIG
                          }
                        >
                          {m.settings_permission_edit_repo()}
                        </Select.Option>
                        <Select.Option
                          value={
                            Multihost_Permission_Type.PERMISSION_READ_OPERATIONS
                          }
                        >
                          {m.settings_permission_read_ops()}
                        </Select.Option>
                      </Select>
                    </Form.Item>
                  </Col>
                  <Col span={11}>
                    <Form.Item
                      name={[permissionField.name, "scopes"]}
                      label={m.settings_peer_permission_scopes()}
                      rules={[
                        {
                          required: true,
                          message: m.settings_peer_permission_scopes_required(),
                        },
                      ]}
                    >
                      <Select
                        mode="multiple"
                        placeholder={m.settings_permission_scope_placeholder()}
                        options={[
                          { label: m.settings_permission_scope_all(), value: "*" },
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
                  type: Multihost_Permission_Type.PERMISSION_READ_OPERATIONS,
                  scopes: ["*"],
                })
              }
              icon={<PlusOutlined />}
              size="small"
              style={{ width: "100%" }}
            >
              {m.settings_peer_add_permission()}
            </Button>
          </>
        )}
      </Form.List>
    </div>
  );
};
