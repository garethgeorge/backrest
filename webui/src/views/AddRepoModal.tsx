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
  const uri = Form.useWatch("uri", form);
  const [sftpIdentityFile, setSftpIdentityFile] = useState("");
  const [sftpPort, setSftpPort] = useState<number | null>(null);

  useEffect(() => {
    const initVal = template
      ? toJson(RepoSchema, template, {
          alwaysEmitImplicit: true,
        })
      : toJson(RepoSchema, repoDefaults, { alwaysEmitImplicit: true });
    form.setFieldsValue(initVal);

    // When creating a new repo, reset the SFTP fields.
    if (!template) {
      setSftpIdentityFile("");
      setSftpPort(null);
    }
  }, [template]);

  useEffect(() => {
    if (!uri?.startsWith("sftp:")) {
      setSftpIdentityFile("");
      setSftpPort(null);
    }
  }, [uri]);

  // This effect now only runs for NEW repos to build the flag.
  useEffect(() => {
    // If we are editing, we don't touch the flags. The user can edit them manually.
    if (template) {
      return;
    }

    const currentFlags = form.getFieldValue("flags") || [];
    const newFlags = currentFlags.filter(
      (f: string) => f && !f.startsWith("--option=sftp.args")
    );

    if (uri?.startsWith("sftp:")) {
      let sftpArgs = "-oBatchMode=yes";
      let argsChanged = false;

      if (sftpIdentityFile) {
        sftpArgs += ` -i ${sftpIdentityFile}`;
        argsChanged = true;
      }

      if (sftpPort && sftpPort !== 0 && sftpPort !== 22) {
        sftpArgs += ` -p ${sftpPort}`;
        argsChanged = true;
      }

      if (argsChanged) {
        newFlags.push(`--option=sftp.args='${sftpArgs}'`);
      }
    }

    const sortedCurrent = [...(currentFlags || [])].sort();
    const sortedNew = [...newFlags].sort();

    if (JSON.stringify(sortedCurrent) !== JSON.stringify(sortedNew)) {
      form.setFieldsValue({ flags: newFlags });
    }
  }, [uri, sftpIdentityFile, sftpPort, template]);

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
        "Deleted repo " +
          template!.id! +
          " from config but files remain. To release storage delete the files manually. URI: " +
          template!.uri
      );
    } catch (e: any) {
      alertsApi.error(formatErrorAlert(e, "Operation error: "), 15);
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
        alertsApi.success("Updated repo configuration " + repo.uri);
      } else {
        // We are in the create repo flow, create the new repo via the service
        setConfig(await backrestService.addRepo(repo));
        showModal(null);
        alertsApi.success("Added repo " + repo.uri);
      }

      try {
        // Update the snapshots for the repo to confirm the config works.
        // TODO: this operation is only used here, find a different RPC for this purpose.
        await backrestService.listSnapshots({ repoId: repo.id });
      } catch (e: any) {
        alertsApi.error(
          formatErrorAlert(
            e,
            "Failed to list snapshots for updated/added repo: "
          ),
          10
        );
      }
    } catch (e: any) {
      alertsApi.error(formatErrorAlert(e, "Operation error: "), 10);
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
        title={template ? "Edit Restic Repository" : "Add Restic Repository"}
        width="60vw"
        footer={[
          <Button loading={confirmLoading} key="back" onClick={handleCancel}>
            Cancel
          </Button>,
          template != null ? (
            <Tooltip
              key="delete-tooltip"
              title="Removes the repo from the config but will not delete the restic repository"
            >
              <ConfirmButton
                key="delete"
                type="primary"
                danger
                onClickAsync={handleDestroy}
                confirmTitle="Confirm Delete"
              >
                Delete
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
                    "Connected successfully to " +
                      repo.uri +
                      " and found an existing repo.",
                    10
                  );
                } else {
                  alertsApi.success(
                    "Connected successfully to " +
                      repo.uri +
                      ". No existing repo found at this location, a new one will be initialized",
                    10
                  );
                }
              } catch (e: any) {
                alertsApi.error(formatErrorAlert(e, "Check error: "), 10);
              }
            }}
          >
            Test Configuration
          </SpinButton>,
          <Button
            key="submit"
            type="primary"
            loading={confirmLoading}
            onClick={handleOk}
          >
            Submit
          </Button>,
        ]}
        maskClosable={false}
      >
        <p>
          See{" "}
          <a
            href="https://garethgeorge.github.io/backrest/introduction/getting-started"
            target="_blank"
          >
            backrest getting started guide
          </a>{" "}
          for repository configuration instructions or check the{" "}
          <a href="https://restic.readthedocs.io/" target="_blank">
            restic documentation
          </a>{" "}
          for more details about repositories.
        </p>
        <br />
        <Form
          autoComplete="off"
          form={form}
          labelCol={{ span: 4 }}
          wrapperCol={{ span: 18 }}
          disabled={confirmLoading}
        >
          {/* Repo.id */}
          <Tooltip
            title={
              "Unique ID that identifies this repo in the backrest UI (e.g. s3-mybucket). This cannot be changed after creation."
            }
          >
            <Form.Item<Repo>
              hasFeedback
              name="id"
              label="Repo Name"
              validateTrigger={["onChange", "onBlur"]}
              rules={[
                {
                  required: true,
                  message: "Please input repo name",
                },
                {
                  validator: async (_, value) => {
                    if (template) return;
                    if (config?.repos?.find((r) => r.id === value)) {
                      throw new Error();
                    }
                  },
                  message: "Repo with name already exists",
                },
                {
                  pattern: namePattern,
                  message:
                    "Name must be alphanumeric with dashes or underscores as separators",
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
                Valid Repo URIs are:
                <ul>
                  <li>Local filesystem path</li>
                  <li>S3 e.g. s3:// ...</li>
                  <li>SFTP e.g. sftp:user@host:/repo-path</li>
                  <li>
                    See{" "}
                    <a
                      href="https://restic.readthedocs.io/en/latest/030_preparing_a_new_repo.html#preparing-a-new-repository"
                      target="_blank"
                    >
                      restic docs
                    </a>{" "}
                    for more info.
                  </li>
                </ul>
              </>
            }
          >
            <Form.Item<Repo>
              hasFeedback
              name="uri"
              label="Repository URI"
              validateTrigger={["onChange", "onBlur"]}
              rules={[
                {
                  required: true,
                  message: "Please input repo URI",
                },
              ]}
            >
              <URIAutocomplete disabled={!!template} />
            </Form.Item>
          </Tooltip>

          {uri?.startsWith("sftp:") && !template && (
            <>
              <Tooltip title="Optional: Path to an SSH identity file for SFTP authentication. This path must be accessible on the machine running backrest.">
                <Form.Item label="SFTP Identity File">
                  <Input
                    placeholder="/home/user/.ssh/id_rsa"
                    value={sftpIdentityFile}
                    onChange={(e) => setSftpIdentityFile(e.target.value)}
                  />
                </Form.Item>
              </Tooltip>
              <Tooltip title="Optional: Specify a custom port for SFTP connection. Defaults to 22.">
                <Form.Item label="SFTP Port">
                  <InputNumber
                    min={1}
                    max={65535}
                    placeholder="22"
                    value={sftpPort}
                    onChange={(value) => setSftpPort(value)}
                  />
                </Form.Item>
              </Tooltip>
            </>
          )}

          {/* Repo.password */}
          <Tooltip
            title={
              <>
                This password that encrypts data in your repository.
                <ul>
                  <li>
                    Recommended to pick a value that is 128 bits of entropy (20
                    chars or longer)
                  </li>
                  <li>
                    You may alternatively provide env variable credentials e.g.
                    RESTIC_PASSWORD, RESTIC_PASSWORD_FILE, or
                    RESTIC_PASSWORD_COMMAND.
                  </li>
                  <li>
                    Click [Generate] to seed a random password from your
                    browser's crypto random API.
                  </li>
                </ul>
              </>
            }
          >
            <Form.Item label="Password">
              <Row>
                <Col span={16}>
                  <Form.Item<Repo>
                    hasFeedback
                    name="password"
                    validateTrigger={["onChange", "onBlur"]}
                  >
                    <Input disabled={!!template} />
                  </Form.Item>
                </Col>
                <Col
                  span={7}
                  offset={1}
                  style={{ display: "flex", justifyContent: "left" }}
                >
                  <Button
                    type="text"
                    onClick={() => {
                      if (template) return;
                      form.setFieldsValue({
                        password: cryptoRandomPassword(),
                      });
                    }}
                  >
                    [Generate]
                  </Button>
                </Col>
              </Row>
            </Form.Item>
          </Tooltip>

          {/* Repo.env */}
          <Tooltip
            title={
              "Environment variables that are passed to restic (e.g. to provide S3 or B2 credentials). References to parent-process env variables are supported as FOO=${MY_FOO_VAR}."
            }
          >
            <Form.Item label="Env Vars">
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
                          <Form.Item
                            {...restField}
                            validateTrigger={["onChange", "onBlur"]}
                            rules={[
                              {
                                required: true,
                                whitespace: true,
                                pattern: /^[\w-]+=.*$/,
                                message:
                                  "Environment variable must be in format KEY=VALUE",
                              },
                            ]}
                            noStyle
                          >
                            <Input
                              placeholder="KEY=VALUE"
                              onBlur={() => form.validateFields()}
                              style={{ width: "90%" }}
                            />
                          </Form.Item>
                          <MinusCircleOutlined
                            className="dynamic-delete-button"
                            onClick={() => remove(index)}
                            style={{ paddingLeft: "5px" }}
                          />
                        </Form.Item>
                      );
                    })}
                    <Form.Item>
                      <Button
                        type="dashed"
                        onClick={() => add("")}
                        style={{ width: "90%" }}
                        icon={<PlusOutlined />}
                      >
                        Set Environment Variable
                      </Button>
                      <Form.ErrorList errors={errors} />
                    </Form.Item>
                  </>
                )}
              </Form.List>
            </Form.Item>
          </Tooltip>

          {/* Repo.flags */}
          <Form.Item label="Flags">
            <Form.List name="flags">
              {(fields, { add, remove }, { errors }) => (
                <>
                  {fields.map((field, index) => {
                    const { key, ...restField } = field;
                    return (
                      <Form.Item required={false} key={field.key}>
                        <Form.Item
                          {...restField}
                          validateTrigger={["onChange", "onBlur"]}
                          rules={[
                            {
                              required: true,
                              whitespace: true,
                              pattern: /^\-\-?.*$/,
                              message:
                                "Value should be a CLI flag e.g. see restic --help",
                            },
                          ]}
                          noStyle
                        >
                          <Input
                            placeholder="--flag"
                            style={{ width: "90%" }}
                          />
                        </Form.Item>
                        <MinusCircleOutlined
                          className="dynamic-delete-button"
                          onClick={() => remove(index)}
                          style={{ paddingLeft: "5px" }}
                        />
                      </Form.Item>
                    );
                  })}
                  <Form.Item>
                    <Button
                      type="dashed"
                      onClick={() => add()}
                      style={{ width: "90%" }}
                      icon={<PlusOutlined />}
                    >
                      Set Flag
                    </Button>
                    <Form.ErrorList errors={errors} />
                  </Form.Item>
                </>
              )}
            </Form.List>
          </Form.Item>

          {/* Repo.prunePolicy */}
          <Form.Item
            label={
              <Tooltip
                title={
                  <span>
                    The schedule on which prune operations are run for this
                    repository. Read{" "}
                    <a
                      href="https://restic.readthedocs.io/en/stable/060_forget.html#customize-pruning"
                      target="_blank"
                    >
                      the restic docs on customizing prune operations
                    </a>{" "}
                    for more details.
                  </span>
                }
              >
                Prune Policy
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
                  <Tooltip title="The maximum percentage of the repo size that may be unused after a prune operation completes. High values reduce copying at the expense of storage.">
                    <div style={{ width: "12" }}>Max Unused After Prune</div>
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
                    The schedule on which check operations are run for this
                    repository. Restic check operations verify the integrity of
                    your repository by scanning the on-disk structures that make
                    up your backup data. Check can optionally be configured to
                    re-read and re-hash data, this is slow and can be bandwidth
                    expensive but will catch any bitrot or silent corruption in
                    the storage medium.
                  </span>
                }
              >
                Check Policy
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
                  <Tooltip title="The percentage of pack data in this repository that will be read and verified. Higher values will use more bandwidth (e.g. 100% will re-read the entire repository on each check).">
                    <div style={{ width: "12" }}>Read Data %</div>
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
                      Modifiers for the backup operation e.g. set the CPU or IO
                      priority.
                    </span>
                  }
                >
                  Command Modifiers
                </Tooltip>
              }
              colon={false}
            >
              <Row>
                <Col span={12} style={{ paddingLeft: "5px" }}>
                  <Tooltip
                    title={
                      <>
                        Available IO priority modes
                        <ul>
                          <li>
                            IO_BEST_EFFORT_LOW - runs at lower than default disk
                            priority (will prioritize other processes)
                          </li>
                          <li>
                            IO_BEST_EFFORT_HIGH - runs at higher than default
                            disk priority (top of disk IO queue)
                          </li>
                          <li>
                            IO_IDLE - only runs when disk bandwidth is idle
                            (e.g. no other operations are queued)
                          </li>
                        </ul>
                      </>
                    }
                  >
                    IO Priority:
                    <br />
                    <Form.Item
                      name={["commandPrefix", "ioNice"]}
                      required={false}
                    >
                      <Select
                        allowClear
                        style={{ width: "100%" }}
                        placeholder="Select an IO priority"
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
                        Available CPU priority modes:
                        <ul>
                          <li>CPU_DEFAULT - no change in priority</li>
                          <li>
                            CPU_HIGH - higher than default priority (backrest
                            must be running as root)
                          </li>
                          <li>CPU_LOW - lower than default priority</li>
                        </ul>
                      </>
                    }
                  >
                    CPU Priority:
                    <br />
                    <Form.Item
                      name={["commandPrefix", "cpuNice"]}
                      required={false}
                    >
                      <Select
                        allowClear
                        style={{ width: "100%" }}
                        placeholder="Select a CPU priority"
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
            label={
              <Tooltip
                title={
                  "Auto-unlock will remove lockfiles at the start of forget and prune operations. " +
                  "This is potentially unsafe if the repo is shared by multiple client devices. Disabled by default."
                }
              >
                Auto Unlock
              </Tooltip>
            }
            name="autoUnlock"
            valuePropName="checked"
          >
            <Checkbox />
          </Form.Item>

          <Form.Item
            label={<Tooltip title={hooksListTooltipText}>Hooks</Tooltip>}
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
                    label: "Repo Config as JSON",
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
        "Missing repo password. Either provide a password or set one of the env variables RESTIC_PASSWORD, RESTIC_PASSWORD_COMMAND, RESTIC_PASSWORD_FILE."
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
