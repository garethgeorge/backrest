import {
  Form,
  Modal,
  Input,
  Typography,
  AutoComplete,
  Tooltip,
  Button,
} from "antd";
import React, { useState } from "react";
import { useShowModal } from "../components/ModalManager";
import { Repo } from "../../gen/ts/v1/config.pb";
import { URIAutocomplete } from "../components/URIAutocomplete";
import { MinusCircleOutlined, PlusOutlined } from "@ant-design/icons";
import { useAlertApi } from "../components/Alerts";
import { ResticUI } from "../../gen/ts/v1/service.pb";
import {
  addRepo,
  configState,
  fetchConfig,
  updateConfig,
} from "../state/config";
import { useRecoilState } from "recoil";
import { nameRegex } from "../lib/patterns";
import { validateForm } from "../lib/formutil";

export const AddRepoModel = ({
  template,
}: {
  template: Partial<Repo> | null;
}) => {
  const [config, setConfig] = useRecoilState(configState);
  const [deleteConfirmed, setDeleteConfirmed] = useState(false);
  const [confirmLoading, setConfirmLoading] = useState(false);
  const showModal = useShowModal();
  const alertsApi = useAlertApi()!;
  const [form] = Form.useForm<Repo>();

  const handleDestroy = async () => {
    if (!deleteConfirmed) {
      setDeleteConfirmed(true);
      setTimeout(() => {
        setDeleteConfirmed(false);
      }, 2000);
      return;
    }

    setConfirmLoading(true);

    try {
      let config = await fetchConfig();
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
      setConfig(await updateConfig(config));
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
      setDeleteConfirmed(false);
      setConfirmLoading(false);
    }
  };

  const handleOk = async () => {
    setConfirmLoading(true);

    try {
      let repo = await validateForm<Repo>(form);

      if (template !== null) {
        // We are in the edit repo flow, update the repo in the config
        let config = await fetchConfig();
        const idx = config.repos!.findIndex((r) => r.id === template!.id);
        if (idx === -1) {
          alertsApi.error("Can't update repo, not found");
          return;
        }
        config.repos![idx] = { ...repo };
        setConfig(await updateConfig(config));
        showModal(null);
        alertsApi.success("Updated repo " + repo.uri);

        // Update the snapshots for the repo
        await ResticUI.ListSnapshots(
          {
            repoId: repo.id,
          },
          { pathPrefix: "/api" }
        );
      } else {
        // We are in the create repo flow, create the new repo via the service
        setConfig(await addRepo(repo));
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
        footer={[
          <Button loading={confirmLoading} key="back" onClick={handleCancel}>
            Cancel
          </Button>,
          template != null ? (
            <Button
              type="primary"
              danger
              loading={confirmLoading}
              onClick={handleDestroy}
            >
              {deleteConfirmed ? "Confirm delete?" : "Delete"}
            </Button>
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
        <Form layout={"vertical"} autoComplete="off" form={form}>
          {/* Repo.id */}
          <Form.Item<Repo>
            hasFeedback
            name="id"
            label="Repo Name"
            initialValue={template ? template.id : ""}
            validateTrigger={["onChange", "onBlur"]}
            rules={[
              {
                required: true,
                message: "Please input plan name",
              },
              {
                pattern: nameRegex,
                message: "Invalid symbol",
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
            ]}
          >
            <Input
              disabled={!!template}
              placeholder={"repo" + ((config?.repos?.length || 0) + 1)}
            />
          </Form.Item>

          {/* Repo.uri */}

          <Tooltip
            title={
              <>
                Valid Repo URIs are:
                <ul>
                  <li>Local filesystem path</li>
                  <li>S3 e.g. s3:// ...</li>
                  <li>SFTP e.g. sftp://user@host:/repo-path</li>
                  <li>
                    See{" "}
                    <a href="https://restic.readthedocs.io/en/latest/030_preparing_a_new_repo.html#preparing-a-new-repository">
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
              label="Repository URI (e.g. ./local-path or s3://bucket-name/path)"
              initialValue={template ? template.uri : ""}
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
          <Form.Item<Repo>
            hasFeedback
            name="password"
            label={
              <>
                Password{" "}
                <Button
                  type="text"
                  onClick={() => {
                    if (template) return;
                    form.setFieldsValue({
                      password: cryptoRandomPassword(),
                    });
                  }}
                >
                  [Create Crypto Random]
                </Button>
              </>
            }
            initialValue={template && template.password}
            validateTrigger={["onChange", "onBlur"]}
            rules={[
              {
                required: true,
                message: "Please input repo name",
              },
            ]}
          >
            <Input disabled={!!template} />
          </Form.Item>

          {/* Repo.env */}
          <Form.List
            name="env"
            rules={[
              {
                validator: async (_, envVars) => {
                  let uri = form.getFieldValue("uri");
                  return await envVarSetValidator(uri, envVars);
                },
              },
            ]}
            initialValue={template ? template.env : []}
          >
            {(fields, { add, remove }, { errors }) => (
              <>
                {fields.map((field, index) => (
                  <Form.Item
                    label={index === 0 ? "Environment Variables" : ""}
                    required={false}
                    key={field.key}
                  >
                    <Form.Item
                      {...field}
                      validateTrigger={["onChange", "onBlur"]}
                      initialValue={""}
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
                        style={{ width: "60%" }}
                      />
                    </Form.Item>
                    <MinusCircleOutlined
                      className="dynamic-delete-button"
                      onClick={() => remove(field.name)}
                      style={{ paddingLeft: "5px" }}
                    />
                  </Form.Item>
                ))}
                <Form.Item
                  label={fields.length === 0 ? "Environment Variables" : ""}
                >
                  <Button
                    type="dashed"
                    onClick={() => add()}
                    style={{ width: "60%" }}
                    icon={<PlusOutlined />}
                  >
                    Set Environment Variable
                  </Button>
                  <Form.ErrorList errors={errors} />
                </Form.Item>
              </>
            )}
          </Form.List>

          {/* Repo.flags */}
          <Form.List name="flags" initialValue={[]}>
            {(fields, { add, remove }, { errors }) => (
              <>
                {fields.map((field, index) => (
                  <Form.Item
                    label={index === 0 ? "(Advanced) Flag Overrides" : ""}
                    required={false}
                    key={field.key}
                  >
                    <Form.Item
                      {...field}
                      validateTrigger={["onChange", "onBlur"]}
                      initialValue={""}
                      rules={[
                        {
                          required: true,
                          whitespace: true,
                          pattern: /^\-\-[A-Za-z0-9_\-]*$/,
                          message:
                            "Value should be a CLI flag e.g. see restic --help",
                        },
                      ]}
                      noStyle
                    >
                      <Input placeholder="--flag" style={{ width: "60%" }} />
                    </Form.Item>
                    <MinusCircleOutlined
                      className="dynamic-delete-button"
                      onClick={() => remove(field.name)}
                      style={{ paddingLeft: "5px" }}
                    />
                  </Form.Item>
                ))}
                <Form.Item
                  label={fields.length === 0 ? "(Advanced) Flag Overrides" : ""}
                >
                  <Button
                    type="dashed"
                    onClick={() => add()}
                    style={{ width: "60%" }}
                    icon={<PlusOutlined />}
                  >
                    Set Flag
                  </Button>
                  <Form.ErrorList errors={errors} />
                </Form.Item>
              </>
            )}
          </Form.List>

          <Form.Item shouldUpdate label="Preview">
            {() => (
              <Typography>
                <pre>{JSON.stringify(form.getFieldsValue(), null, 2)}</pre>
              </Typography>
            )}
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

const expectedEnvVars: { [scheme: string]: string[] } = {
  s3: ["AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"],
  b2: ["B2_ACCOUNT_ID", "B2_ACCOUNT_KEY"],
};

const envVarSetValidator = (uri: string | undefined, envVars: string[]) => {
  if (!uri) {
    return Promise.resolve();
  }
  let schemeIdx = uri.indexOf(":");
  if (schemeIdx === -1) {
    return Promise.resolve();
  }

  let scheme = uri.substring(0, schemeIdx);
  let expected = expectedEnvVars[scheme];
  if (!expected) {
    return Promise.resolve();
  }

  const envVarNames = envVars.map((e) => e && e.split("=")[0]);

  let missing: string[] = [];
  for (let e of expected) {
    if (!envVarNames.includes(e)) {
      missing.push(e);
    }
  }

  if (missing.length === 0) {
    return Promise.resolve();
  }

  return Promise.reject(
    new Error(
      "Missing env vars " + missing.join(", ") + " for scheme " + scheme
    )
  );
};

const cryptoRandomPassword = (): string => {
  let vals = crypto.getRandomValues(new Uint8Array(64));
  // 48 chars is at least log2(64) * 48 = 288 bits of entropy.
  return btoa(String.fromCharCode(...vals)).slice(0, 48);
};
