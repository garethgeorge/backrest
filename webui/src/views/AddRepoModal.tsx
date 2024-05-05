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
} from "antd";
import React, { useEffect, useState } from "react";
import { useShowModal } from "../components/ModalManager";
import { Hook, Repo } from "../../gen/ts/v1/config_pb";
import { URIAutocomplete } from "../components/URIAutocomplete";
import { MinusCircleOutlined, PlusOutlined } from "@ant-design/icons";
import { useAlertApi } from "../components/Alerts";
import { namePattern, validateForm } from "../lib/formutil";
import { backrestService } from "../api";
import {
  HooksFormList,
  hooksListTooltipText,
  HookFormData,
} from "../components/HooksFormList";
import { ConfirmButton } from "../components/SpinButton";
import { useConfig } from "../components/ConfigProvider";

export const AddRepoModal = ({
  template,
}: {
  template: Repo | null;
}) => {
  const [confirmLoading, setConfirmLoading] = useState(false);
  const showModal = useShowModal();
  const alertsApi = useAlertApi()!;
  const [config, setConfig] = useConfig();
  const [form] = Form.useForm();
  useEffect(() => {
    form.setFieldsValue(template ? JSON.parse(template.toJsonString()) : {});
  }, [template])

  if (!config) {
    return null;
  }

  const handleDestroy = async () => {
    setConfirmLoading(true);

    try {
      config.repos = config.repos || [];

      if (!template) {
        throw new Error("template not found");
      }

      // Check if still in use by a plan
      for (const plan of config.plans || []) {
        if (plan.repo === template.id) {
          throw new Error("Can't delete repo, still in use by plan " + plan.id);
        }
      }

      // Remove the plan from the config
      const idx = config.repos.findIndex((r) => r.id === template.id);
      if (idx === -1) {
        throw new Error("failed to update config, plan to delete not found");
      }

      config.repos.splice(idx, 1);

      // Update config and notify success.
      setConfig(await backrestService.setConfig(config));
      showModal(null);
      alertsApi.success(
        "Deleted repo " +
        template.id +
        " from config but files remain. To release storage delete the files manually. URI: " +
        template.uri
      );
    } catch (e: any) {
      alertsApi.error("Operation failed: " + e.message, 15);
    } finally {
      setConfirmLoading(false);
    }
  };

  const handleOk = async () => {
    setConfirmLoading(true);

    try {
      let repoFormData = await validateForm(form);
      const repo = new Repo().fromJsonString(JSON.stringify(repoFormData), { ignoreUnknownFields: false });

      if (template !== null) {
        // We are in the edit repo flow, update the repo in the config
        const idx = config.repos!.findIndex((r) => r.id === template!.id);
        if (idx === -1) {
          alertsApi.error("Can't update repo, not found");
          return;
        }
        config.repos![idx] = repo;
        setConfig(await backrestService.setConfig(config));
        showModal(null);
        alertsApi.success("Updated repo " + repo.uri);

        // Update the snapshots for the repo to confirm the config works.
        // TODO: this operation is only used here, find a different RPC for this purpose.
        await backrestService.listSnapshots({ repoId: repo.id });
      } else {
        // We are in the create repo flow, create the new repo via the service
        setConfig(await backrestService.addRepo(repo));
        showModal(null);
        alertsApi.success("Added repo " + repo.uri);
      }
    } catch (e: any) {
      alertsApi.error("Operation failed: " + e.message, 15);
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
        width="40vw"
        footer={[
          <Button loading={confirmLoading} key="back" onClick={handleCancel}>
            Cancel
          </Button>,
          template != null ? (
            <ConfirmButton
              key="delete"
              type="primary"
              danger
              onClickAsync={handleDestroy}
              confirmTitle="Confirm Delete"
            >
              Delete
            </ConfirmButton>
          ) : null,
          <Button
            key="submit"
            type="primary"
            loading={confirmLoading}
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
          disabled={confirmLoading}
        >
          {/* Repo.id */}
          <Tooltip title={"Unique ID that identifies this repo in the backrest UI (e.g. s3-mybucket). This cannot be changed after creation."}>
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
                  message: "Name must be alphanumeric with dashes or underscores as separators",
                }
              ]}
            >
              <Input
                disabled={!!template}
                placeholder={"repo" + ((config?.repos?.length || 0) + 1)}
              />
            </Form.Item>
          </Tooltip>

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
                    <a href="https://restic.readthedocs.io/en/latest/030_preparing_a_new_repo.html#preparing-a-new-repository" target="_blank">
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

          {/* Repo.password */}
          <Tooltip
            title={<>
              This password that encrypts data in your repository.
              <ul>
                <li>Recommended to pick a value that is 128 bits of entropy (20 chars or longer)</li>
                <li>You may alternatively provide env variable credentials e.g. RESTIC_PASSWORD, RESTIC_PASSWORD_FILE, or RESTIC_PASSWORD_COMMAND.</li>
                <li>Click [Generate] to seed a random password from your browser's crypto random API.</li>
              </ul>
            </>}
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
          <Tooltip title={"Environment variables that are passed to restic (e.g. to provide S3 or B2 credentials). References to parent-process env variables are supported as FOO=${MY_FOO_VAR}."}>
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
                    {fields.map((field, index) => (
                      <Form.Item key={field.key}>
                        <Form.Item
                          {...field}
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
                    ))}
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
            <Form.List
              name="flags"
            >
              {(fields, { add, remove }, { errors }) => (
                <>
                  {fields.map((field, index) => (
                    <Form.Item required={false} key={field.key}>
                      <Form.Item
                        {...field}
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
                        <Input placeholder="--flag" style={{ width: "90%" }} />
                      </Form.Item>
                      <MinusCircleOutlined
                        className="dynamic-delete-button"
                        onClick={() => remove(index)}
                        style={{ paddingLeft: "5px" }}
                      />
                    </Form.Item>
                  ))}
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
            required={false}
            label={
              <Tooltip
                title={
                  <span>
                    The schedule on which prune operations are run for this
                    repository. Read{" "}
                    <a href="https://restic.readthedocs.io/en/stable/060_forget.html#customize-pruning" target="_blank">
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
              name={["prunePolicy", "maxFrequencyDays"]}
              initialValue={7}
              required={false}
            >
              <InputNumber
                addonBefore={
                  <div style={{ width: "12em" }}>Max Frequency Days</div>
                }
              />
            </Form.Item>
            <Form.Item
              name={["prunePolicy", "maxUnusedPercent"]}
              initialValue={25}
              required={false}
            >
              <InputNumber
                addonBefore={
                  <Tooltip title="The maximum percentage of the repo size that may be unused after a prune operation completes. High values reduce copying at the expense of storage.">
                    <div style={{ width: "12em" }}>Max Unused Percent</div>
                  </Tooltip>
                }
              />
            </Form.Item>
          </Form.Item>

          <Form.Item label={<Tooltip title={"Auto-unlock will remove lockfiles at the start of forget and prune operations. "
            + "This is potentially unsafe if the repo is shared by multiple client devices. Opt-in (and disabled) by default."}>
            Auto Unlock
          </Tooltip>} name="autoUnlock" valuePropName="checked">
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
                        <pre>{JSON.stringify(form.getFieldsValue(), undefined, 2)}</pre>
                      </Typography>
                    ),
                  },
                ]}
              />
            )}
          </Form.Item>
        </Form>
      </Modal >
    </>
  );
};

const expectedEnvVars: { [scheme: string]: string[][] } = {
  s3: [["AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"]],
  b2: [["B2_ACCOUNT_ID", "B2_ACCOUNT_KEY"]],
  azure: [["AZURE_ACCOUNT_NAME", "AZURE_ACCOUNT_KEY"], ["AZURE_ACCOUNT_NAME", "AZURE_ACCOUNT_SAS"]],
  gs: [["GOOGLE_APPLICATION_CREDENTIALS", "GOOGLE_PROJECT_ID"], ["GOOGLE_ACCESS_TOKEN"]],
};

const envVarSetValidator = (form: FormInstance<FormData>, envVars: string[]) => {
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

const checkSchemeEnvVars = (scheme: string, envVarNames: string[]): Promise<void> => {
  let expected = expectedEnvVars[scheme];
  if (!expected) {
    return Promise.resolve();
  }

  const missingVarsCollection: string[][] = [];

  for (let possibility of expected) {
    const missingVars = possibility.filter(envVar => !envVarNames.includes(envVar));

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
      "Missing env vars " + formatMissingEnvVars(missingVarsCollection) + " for scheme " + scheme
    )
  );
}

const formatMissingEnvVars = (partialMatches: string[][]): string => {
  return partialMatches.map(x => {
    if (x.length > 1) {
      return `[ ${x.join(", ")} ]`;
    }
    return x[0];
  }).join(" or ");
}
