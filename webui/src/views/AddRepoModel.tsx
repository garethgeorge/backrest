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

export const AddRepoModel = ({
  template,
}: {
  template: Partial<Repo> | null;
}) => {
  const [confirmLoading, setConfirmLoading] = useState(false);
  const showModal = useShowModal();
  const alertsApi = useAlertApi()!;
  const [form] = Form.useForm<Repo>();

  template = template || {};

  const handleOk = () => {
    const errors = form
      .getFieldsError()
      .map((e) => e.errors)
      .flat();
    if (errors.length > 0) {
      alertsApi.warning("Please fix form errors " + errors.join(", "));
      return;
    }
    setConfirmLoading(true);

    const repo = form.getFieldsValue() as Repo;

    if (template === null) {
      // We are in the create repo flow, create the new repo via the service
      ResticUI.AddRepo(repo, {
        pathPrefix: "/api",
      })
        .then((res) => {
          showModal(null);
          alertsApi.success("Added repo " + repo.uri);
        })
        .catch((e) => {
          alertsApi.error("Error adding repo: " + e.message, 15);
        })
        .finally(() => {
          setConfirmLoading(false);
        });
    } else {
    }
  };

  const handleCancel = () => {
    showModal(null);
  };

  return (
    <>
      <Modal
        open={true}
        title={template ? "Add Restic Repository" : "Edit Restic Repository"}
        onOk={handleOk}
        confirmLoading={confirmLoading}
        onCancel={handleCancel}
      >
        <Form layout={"vertical"} autoComplete="off" form={form}>
          {/* Repo.id */}
          <Form.Item<Repo>
            hasFeedback
            name="id"
            label="Repo Name"
            initialValue={template.id}
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
            ]}
          >
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
              label="Repo URI"
              initialValue={template.id}
              validateTrigger={["onChange", "onBlur"]}
              rules={[
                {
                  required: true,
                  message: "Please input repo URI",
                },
              ]}
            >
              <Input />
            </Form.Item>
          </Tooltip>

          {/* Repo.password */}
          <Form.Item<Repo>
            hasFeedback
            name="password"
            label="Password"
            initialValue={template.password}
            validateTrigger={["onChange", "onBlur"]}
            rules={[
              {
                required: true,
                message: "Please input repo name",
              },
              {
                pattern: nameRegex,
                message: "Invalid symbol",
              },
            ]}
          >
            <Input />
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
            initialValue={[]}
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
                      <Input placeholder="KEY=VALUE" style={{ width: "60%" }} />
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
                    Set Environment Variable
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

const nameRegex = /^[a-zA-Z0-9\-_ ]+$/;

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

  let missing: string[] = [];
  for (let e of expected) {
    if (!envVars.includes(e)) {
      missing.push(e);
    }
  }

  return Promise.reject(
    new Error(
      "Missing env vars " + missing.join(", ") + " for scheme " + scheme
    )
  );
};
