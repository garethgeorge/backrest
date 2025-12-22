import {
  Form,
  Modal,
  Input,
  Typography,
  AutoComplete,
  Tooltip,
  Button,
  Row,
  Col,
  Card,
  InputNumber,
  FormInstance,
  Collapse,
  Checkbox,
  Select,
  Space,
  Flex,
} from "antd";
import React, { useEffect, useState } from "react";
import { useShowModal } from "../components/ModalManager";
import {
  CommandPrefix_CPUNiceLevel,
  CommandPrefix_CPUNiceLevelSchema,
  CommandPrefix_IONiceLevel,
  CommandPrefix_IONiceLevelSchema,
  type Repo,
  RepoSchema,
  Schedule_Clock,
} from "../../gen/ts/v1/config_pb";
import { StringValueSchema } from "../../gen/ts/types/value_pb";
import { URIAutocomplete } from "../components/URIAutocomplete";
import { MinusCircleOutlined, PlusOutlined } from "@ant-design/icons";
import { formatErrorAlert, useAlertApi } from "../components/Alerts";
import { namePattern, validateForm } from "../lib/formutil";
import { backrestService } from "../api";
import {
  HooksFormList,
  hooksListTooltipText,
} from "../components/HooksFormList";
import { ConfirmButton, SpinButton } from "../components/SpinButton";
import { useConfig } from "../components/ConfigProvider";
import Cron from "react-js-cron";
import {
  ScheduleDefaultsInfrequent,
  ScheduleFormItem,
} from "../components/ScheduleFormItem";
import { isWindows } from "../state/buildcfg";
import { create, fromJson, JsonValue, toJson } from "@bufbuild/protobuf";
import * as m from "../paraglide/messages";

const repoDefaults = create(RepoSchema, {
  prunePolicy: {
    maxUnusedPercent: 10,
    schedule: {
      schedule: {
        case: "cron",
        value: "0 0 1 * *", // 1st of the month,
      },
      clock: Schedule_Clock.LAST_RUN_TIME,
    },
  },
  checkPolicy: {
    schedule: {
      schedule: {
        case: "cron",
        value: "0 0 1 * *", // 1st of the month,
      },
      clock: Schedule_Clock.LAST_RUN_TIME,
    },
  },
  commandPrefix: {
    ioNice: CommandPrefix_IONiceLevel.IO_DEFAULT,
    cpuNice: CommandPrefix_CPUNiceLevel.CPU_DEFAULT,
  },
});

export const AddRepoModal = ({ template }: { template: Repo | null }) => {
  const [confirmLoading, setConfirmLoading] = useState(false);
  const showModal = useShowModal();
  const alertsApi = useAlertApi()!;
  const [config, setConfig] = useConfig();
  const [form] = Form.useForm<JsonValue>();
  useEffect(() => {
    const initVal = template
      ? toJson(RepoSchema, template, {
          alwaysEmitImplicit: true,
        })
      : toJson(RepoSchema, repoDefaults, { alwaysEmitImplicit: true });
    form.setFieldsValue(initVal);
  }, [template]);

  if (!config) {
    return null;
  }

  const handleDestroy = async () => {
    setConfirmLoading(true);

    try {
      // Update config and notify success.
      setConfig(
        await backrestService.removeRepo(
          create(StringValueSchema, { value: template!.id })
        )
      );
      showModal(null);
      alertsApi.success(
        m.add_repo_modal_success_deleted({ id: template!.id!, uri: template!.uri })
      );
    } catch (e: any) {
      alertsApi.error(formatErrorAlert(e, m.add_plan_modal_error_destroy_prefix()), 15);
    } finally {
      setConfirmLoading(false);
    }
  };

  const handleOk = async () => {
    setConfirmLoading(true);

    try {
      let repoFormData = await validateForm(form);
      const repo = fromJson(RepoSchema, repoFormData, {
        ignoreUnknownFields: false,
      });

      if (template !== null) {
        // We are in the update repo flow, update the repo via the service
        setConfig(await backrestService.addRepo(repo));
        showModal(null);
        alertsApi.success(m.add_repo_modal_success_updated({ uri: repo.uri }));
      } else {
        // We are in the create repo flow, create the new repo via the service
        setConfig(await backrestService.addRepo(repo));
        showModal(null);
        alertsApi.success(m.add_repo_modal_success_added({ uri: repo.uri }));
      }

      try {
        // Update the snapshots for the repo to confirm the config works.
        // TODO: this operation is only used here, find a different RPC for this purpose.
        await backrestService.listSnapshots({ repoId: repo.id });
      } catch (e: any) {
        alertsApi.error(
          formatErrorAlert(
            e,
            m.add_repo_modal_error_list_snapshots()
          ),
          10
        );
      }
    } catch (e: any) {
      alertsApi.error(formatErrorAlert(e, m.add_plan_modal_error_operation_prefix()), 10);
    } finally {
      setConfirmLoading(false);
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
        title={template ? m.add_repo_modal_title_edit() : m.add_repo_modal_title_add()}
        width="60vw"
        footer={[
          <Button loading={confirmLoading} key="back" onClick={handleCancel}>
            {m.add_plan_modal_button_cancel()}
          </Button>,
          template != null ? (
            <Tooltip
              key="delete-tooltip"
              title={m.add_repo_modal_delete_tooltip()}
            >
              <ConfirmButton
                key="delete"
                type="primary"
                danger
                onClickAsync={handleDestroy}
                confirmTitle={m.add_plan_modal_button_confirm_delete()}
              >
                {m.add_plan_modal_button_delete()}
              </ConfirmButton>
            </Tooltip>
          ) : null,
          <SpinButton
            key="check"
            onClickAsync={async () => {
              let repoFormData = await validateForm(form);
              console.log("checking repo", repoFormData);
              const repo = fromJson(RepoSchema, repoFormData, {
                ignoreUnknownFields: false,
              });
              try {
                const exists = await backrestService.checkRepoExists(repo);
                if (exists.value) {
                  alertsApi.success(
                    m.add_repo_modal_test_success_existing({ uri: repo.uri }),
                    10
                  );
                } else {
                  alertsApi.success(
                    m.add_repo_modal_test_success_new({ uri: repo.uri }),
                    10
                  );
                }
              } catch (e: any) {
                alertsApi.error(formatErrorAlert(e, m.add_repo_modal_test_error()), 10);
              }
            }}
          >
            {m.add_repo_modal_test_config()}
          </SpinButton>,
          <Button
            key="submit"
            type="primary"
            loading={confirmLoading}
            onClick={handleOk}
          >
            {m.add_plan_modal_button_submit()}
          </Button>,
        ]}
        maskClosable={false}
      >
        <p>
          {m.add_repo_modal_guide_text_p1()}
          <a
            href="https://garethgeorge.github.io/backrest/introduction/getting-started"
            target="_blank"
          >
            {m.add_repo_modal_guide_link_text()}
          </a>{" "}
          {m.add_repo_modal_guide_text_p2()}
          <a href="https://restic.readthedocs.io/" target="_blank">
            {m.add_repo_modal_guide_restic_link_text()}
          </a>{" "}
          {m.add_repo_modal_guide_text_p3()}
        </p>
        <br />
        <Form
          autoComplete="off"
          form={form}
          labelCol={{ flex: "160px" }}
          wrapperCol={{ flex: "auto" }}
          disabled={confirmLoading}
        >
          {/* Repo.id */}
          <Tooltip
            title={
              m.add_repo_modal_field_repo_name_tooltip()
            }
          >
            <Form.Item<Repo>
              hasFeedback
              name="id"
              label={m.add_repo_modal_field_repo_name()}
              validateTrigger={["onChange", "onBlur"]}
              rules={[
                {
                  required: true,
                  message: m.add_repo_modal_error_repo_name_required(),
                },
                {
                  validator: async (_, value) => {
                    if (template) return;
                    if (config?.repos?.find((r) => r.id === value)) {
                      throw new Error();
                    }
                  },
                  message: m.add_repo_modal_error_repo_exists(),
                },
                {
                  pattern: namePattern,
                  message:
                    m.add_plan_modal_validation_plan_name_pattern(),
                },
              ]}
            >
              <Input
                disabled={!!template}
                placeholder={"repo" + ((config?.repos?.length || 0) + 1)}
              />
            </Form.Item>
          </Tooltip>

          <Form.Item<Repo> name="guid" hidden>
            <Input />
          </Form.Item>

          {/* Repo.uri */}

          <Tooltip
            title={
              <>
                {m.add_repo_modal_field_uri_tooltip_title()}
                <ul>
                  <li>{m.add_repo_modal_field_uri_tooltip_local()}</li>
                  <li>{m.add_repo_modal_field_uri_tooltip_s3()}</li>
                  <li>{m.add_repo_modal_field_uri_tooltip_sftp()}</li>
                  <li>
                    {m.add_repo_modal_field_uri_tooltip_see()}
                    <a
                      href="https://restic.readthedocs.io/en/latest/030_preparing_a_new_repo.html#preparing-a-new-repository"
                      target="_blank"
                    >
                      {m.add_repo_modal_field_uri_tooltip_restic_docs()}
                    </a>{" "}
                    {m.add_repo_modal_field_uri_tooltip_info()}
                  </li>
                </ul>
              </>
            }
          >
            <Form.Item<Repo>
              hasFeedback
              name="uri"
              label={m.add_repo_modal_field_uri()}
              validateTrigger={["onChange", "onBlur"]}
              rules={[
                {
                  required: true,
                  message: m.add_repo_modal_error_uri_required(),
                },
              ]}
            >
              <URIAutocomplete disabled={!!template} />
            </Form.Item>
          </Tooltip>

          {/* Repo.password */}
          <Tooltip
            title={
              <>
                {m.add_repo_modal_field_password_tooltip_intro()}
                <ul>
                  <li>
                    {m.add_repo_modal_field_password_tooltip_entropy()}
                  </li>
                  <li>
                    {m.add_repo_modal_field_password_tooltip_env()}
                  </li>
                  <li>
                    {m.add_repo_modal_field_password_tooltip_generate()}
                  </li>
                </ul>
              </>
            }
          >
            <Form.Item label={m.add_repo_modal_field_password()}>
              <Flex gap="small">
                <Form.Item<Repo>
                  hasFeedback
                  name="password"
                  validateTrigger={["onChange", "onBlur"]}
                  noStyle
                >
                  <Input disabled={!!template} style={{ flex: 1 }} />
                </Form.Item>
                <Button
                  type="text"
                  onClick={() => {
                    if (template) return;
                    form.setFieldsValue({
                      password: cryptoRandomPassword(),
                    });
                  }}
                >
                  {m.add_repo_modal_button_generate()}
                </Button>
              </Flex>
            </Form.Item>
          </Tooltip>

          {/* Repo.env */}
          <Tooltip
            title={
              m.add_repo_modal_field_env_vars_tooltip({ MY_FOO_VAR: "$MY_FOO_VAR" })
            }
          >
            <Form.Item label={m.add_repo_modal_field_env_vars()}>
              <Form.List
                name="env"
                rules={[
                  {
                    validator: async (_, envVars) => {
                      return await envVarSetValidator(form, envVars);
                    },
                  },
                ]}
              >
                {(fields, { add, remove }, { errors }) => (
                  <>
                    {fields.map((field, index) => {
                      const { key, ...restField } = field;
                      return (
                        <Form.Item key={field.key}>
                          <Flex gap="small" align="center">
                            <Form.Item
                              {...restField}
                              validateTrigger={["onChange", "onBlur"]}
                              rules={[
                                {
                                  required: true,
                                  whitespace: true,
                                  pattern: /^[\w-]+=.*$/,
                                  message: m.add_repo_modal_error_env_format(),
                                },
                              ]}
                              noStyle
                            >
                              <Input
                                placeholder="KEY=VALUE"
                                onBlur={() => form.validateFields()}
                                style={{ flex: 1 }}
                              />
                            </Form.Item>
                            <MinusCircleOutlined
                              className="dynamic-delete-button"
                              onClick={() => remove(index)}
                            />
                          </Flex>
                        </Form.Item>
                      );
                    })}
                    <Form.Item>
                      <Button
                        type="dashed"
                        onClick={() => add("")}
                        block
                        icon={<PlusOutlined />}
                      >
                        {m.add_repo_modal_button_set_env()}
                      </Button>
                      <Form.ErrorList errors={errors} />
                    </Form.Item>
                  </>
                )}
              </Form.List>
            </Form.Item>
          </Tooltip>

          {/* Repo.flags */}
          <Form.Item label={m.add_repo_modal_field_flags()}>
            <Form.List name="flags">
              {(fields, { add, remove }, { errors }) => (
                <>
                  {fields.map((field, index) => {
                    const { key, ...restField } = field;
                    return (
                      <Form.Item required={false} key={field.key}>
                        <Flex gap="small" align="center">
                          <Form.Item
                            {...restField}
                            validateTrigger={["onChange", "onBlur"]}
                            rules={[
                              {
                                required: true,
                                whitespace: true,
                                pattern: /^\-\-?.*$/,
                                message: m.add_repo_modal_error_flag_format(),
                              },
                            ]}
                            noStyle
                          >
                            <Input
                              placeholder="--flag"
                              style={{ flex: 1 }}
                            />
                          </Form.Item>
                          <MinusCircleOutlined
                            className="dynamic-delete-button"
                            onClick={() => remove(index)}
                          />
                        </Flex>
                      </Form.Item>
                    );
                  })}
                  <Form.Item>
                    <Button
                      type="dashed"
                      onClick={() => add()}
                      block
                      icon={<PlusOutlined />}
                    >
                      {m.add_repo_modal_button_set_flag()}
                    </Button>
                    <Form.ErrorList errors={errors} />
                  </Form.Item>
                </>
              )}
            </Form.List>
          </Form.Item>

          {/* Repo.autoUnlock */}
          <Form.Item
            label={
              <Tooltip
                title={
                  m.add_repo_modal_field_auto_unlock_tooltip()
                }
              >
                {m.add_repo_modal_field_auto_unlock()}
              </Tooltip>
            }
            name="autoUnlock"
            valuePropName="checked"
          >
            <Checkbox />
          </Form.Item>

          {/* Repo.prunePolicy */}
          <Form.Item
            label={
              <Tooltip
                title={
                  <span>
                    {m.add_repo_modal_field_prune_policy_tooltip_p1()}
                    <a
                      href="https://restic.readthedocs.io/en/stable/060_forget.html#customize-pruning"
                      target="_blank"
                    >
                      {m.add_repo_modal_field_prune_policy_tooltip_link()}
                    </a>{" "}
                    {m.add_repo_modal_field_prune_policy_tooltip_p2()}
                  </span>
                }
              >
                {m.add_repo_modal_field_prune_policy()}
              </Tooltip>
            }
          >
            <Form.Item
              name={["prunePolicy", "maxUnusedPercent"]}
              initialValue={10}
              required={false}
            >
              <InputPercent
                addonBefore={
                  <Tooltip title={m.add_repo_modal_field_max_unused_tooltip()}>
                    <div style={{ width: "12" }}>{m.add_repo_modal_field_max_unused()}</div>
                  </Tooltip>
                }
              />
            </Form.Item>
            <ScheduleFormItem
              name={["prunePolicy", "schedule"]}
              defaults={ScheduleDefaultsInfrequent}
            />
          </Form.Item>

          {/* Repo.checkPolicy */}
          <Form.Item
            label={
              <Tooltip
                title={
                  <span>
                    {m.add_repo_modal_field_check_policy_tooltip()}
                  </span>
                }
              >
                {m.add_repo_modal_field_check_policy()}
              </Tooltip>
            }
          >
            <Form.Item
              name={["checkPolicy", "readDataSubsetPercent"]}
              initialValue={0}
              required={false}
            >
              <InputPercent
                addonBefore={
                  <Tooltip title={m.add_repo_modal_field_read_data_tooltip()}>
                    <div style={{ width: "12" }}>{m.add_repo_modal_field_read_data()}</div>
                  </Tooltip>
                }
              />
            </Form.Item>
            <ScheduleFormItem
              name={["checkPolicy", "schedule"]}
              defaults={ScheduleDefaultsInfrequent}
            />
          </Form.Item>

          {/* Repo.commandPrefix */}
          {!isWindows && (
            <Form.Item
              label={
                <Tooltip
                  title={
                    <span>
                      {m.add_repo_modal_field_command_modifiers_tooltip()}
                    </span>
                  }
                >
                  {m.add_repo_modal_field_command_modifiers()}
                </Tooltip>
              }
              colon={false}
            >
              <Row>
                <Col span={12} style={{ paddingLeft: "5px" }}>
                  <Tooltip
                    title={
                      <>
                         {m.add_repo_modal_field_io_priority_tooltip_intro()}
                        <ul>
                          <li>
                            {m.add_repo_modal_field_io_priority_low()}
                          </li>
                          <li>
                            {m.add_repo_modal_field_io_priority_high()}
                          </li>
                          <li>
                            {m.add_repo_modal_field_io_priority_idle()}
                          </li>
                        </ul>
                      </>
                    }
                  >
                    {m.add_repo_modal_field_io_priority()}
                    <br />
                    <Form.Item
                      name={["commandPrefix", "ioNice"]}
                      required={false}
                    >
                      <Select
                        allowClear
                        style={{ width: "100%" }}
                        placeholder={m.add_repo_modal_field_io_priority_placeholder()}
                        options={CommandPrefix_IONiceLevelSchema.values.map(
                          (v) => ({
                            label: v.name,
                            value: v.name,
                          })
                        )}
                      />
                    </Form.Item>
                  </Tooltip>
                </Col>
                <Col span={12} style={{ paddingLeft: "5px" }}>
                  <Tooltip
                    title={
                      <>
                         {m.add_repo_modal_field_cpu_priority_tooltip_intro()}
                        <ul>
                          <li>{m.add_repo_modal_field_cpu_priority_default()}</li>
                          <li>
                             {m.add_repo_modal_field_cpu_priority_high()}
                          </li>
                          <li>{m.add_repo_modal_field_cpu_priority_low()}</li>
                        </ul>
                      </>
                    }
                  >
                    {m.add_repo_modal_field_cpu_priority()}
                    <br />
                    <Form.Item
                      name={["commandPrefix", "cpuNice"]}
                      required={false}
                    >
                      <Select
                        allowClear
                        style={{ width: "100%" }}
                        placeholder={m.add_repo_modal_field_cpu_priority_placeholder()}
                        options={CommandPrefix_CPUNiceLevelSchema.values.map(
                          (v) => ({
                            label: v.name,
                            value: v.name,
                          })
                        )}
                      />
                    </Form.Item>
                  </Tooltip>
                </Col>
              </Row>
            </Form.Item>
          )}

          <Form.Item
            label={<Tooltip title={hooksListTooltipText}>{m.add_plan_modal_field_hooks()}</Tooltip>}
          >
            <HooksFormList />
          </Form.Item>

          <Form.Item shouldUpdate label="Preview">
            {() => (
              <Collapse
                size="small"
                items={[
                  {
                    key: "1",
                    label: m.add_repo_modal_preview_json(),
                    children: (
                      <Typography>
                        <pre>
                          {JSON.stringify(form.getFieldsValue(), undefined, 2)}
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

const expectedEnvVars: { [scheme: string]: string[][] } = {
  s3: [
    ["AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"],
    ["AWS_SHARED_CREDENTIALS_FILE"],
  ],
  b2: [["B2_ACCOUNT_ID", "B2_ACCOUNT_KEY"]],
  azure: [
    ["AZURE_ACCOUNT_NAME", "AZURE_ACCOUNT_KEY"],
    ["AZURE_ACCOUNT_NAME", "AZURE_ACCOUNT_SAS"],
  ],
  gs: [
    ["GOOGLE_APPLICATION_CREDENTIALS", "GOOGLE_PROJECT_ID"],
    ["GOOGLE_ACCESS_TOKEN"],
  ],
};

const envVarSetValidator = (form: FormInstance<any>, envVars: string[]) => {
  if (!envVars) {
    return Promise.resolve();
  }

  let uri = form.getFieldValue("uri");
  if (!uri) {
    return Promise.resolve();
  }

  const envVarNames = envVars.map((e) => {
    if (!e) {
      return "";
    }
    let idx = e.indexOf("=");
    if (idx === -1) {
      return "";
    }
    return e.substring(0, idx);
  });

  // check that password is provided in some form
  const password = form.getFieldValue("password");
  if (
    (!password || password.length === 0) &&
    !envVarNames.includes("RESTIC_PASSWORD") &&
    !envVarNames.includes("RESTIC_PASSWORD_COMMAND") &&
    !envVarNames.includes("RESTIC_PASSWORD_FILE")
  ) {
    return Promise.reject(
      new Error(
        m.add_repo_modal_error_missing_password()
      )
    );
  }

  // find expected env for scheme
  let schemeIdx = uri.indexOf(":");
  if (schemeIdx === -1) {
    return Promise.resolve();
  }

  let scheme = uri.substring(0, schemeIdx);

  return checkSchemeEnvVars(scheme, envVarNames);
};

const cryptoRandomPassword = (): string => {
  let vals = crypto.getRandomValues(new Uint8Array(64));
  // 48 chars is at least log2(64) * 48 = ~288 bits of entropy.
  return btoa(String.fromCharCode(...vals)).slice(0, 48);
};

const checkSchemeEnvVars = (
  scheme: string,
  envVarNames: string[]
): Promise<void> => {
  let expected = expectedEnvVars[scheme];
  if (!expected) {
    return Promise.resolve();
  }

  const missingVarsCollection: string[][] = [];

  for (let possibility of expected) {
    const missingVars = possibility.filter(
      (envVar) => !envVarNames.includes(envVar)
    );

    // If no env vars are missing, we have a full match and are good
    if (missingVars.length === 0) {
      return Promise.resolve();
    }

    // First pass: Only add those missing vars from sets where at least one existing env var already exists
    if (missingVars.length < possibility.length) {
      missingVarsCollection.push(missingVars);
    }
  }

  // If we didn't find any env var set with a partial match, then add all expected sets
  if (!missingVarsCollection.length) {
    missingVarsCollection.push(...expected);
  }

  return Promise.reject(
    new Error(
      "Missing env vars " +
        formatMissingEnvVars(missingVarsCollection) +
        " for scheme " +
        scheme
    )
  );
};

const formatMissingEnvVars = (partialMatches: string[][]): string => {
  return partialMatches
    .map((x) => {
      if (x.length > 1) {
        return `[ ${x.join(", ")} ]`;
      }
      return x[0];
    })
    .join(" or ");
};

const InputPercent = ({ ...props }) => {
  return (
    <InputNumber
      step={1}
      min={0}
      max={100}
      precision={2}
      controls={false}
      suffix="%"
      {...props}
    />
  );
};
