import {
  Form,
  Modal,
  Input,
  Typography,
  Select,
  Button,
  Divider,
  Tooltip,
} from "antd";
import React, { useState } from "react";
import { useShowModal } from "../components/ModalManager";
import { Plan } from "../../gen/ts/v1/config.pb";
import { useRecoilState } from "recoil";
import { configState, fetchConfig, updateConfig } from "../state/config";
import { nameRegex } from "../lib/patterns";
import { MinusCircleOutlined, PlusOutlined } from "@ant-design/icons";
import { URIAutocomplete } from "../components/URIAutocomplete";
import { useAlertApi } from "../components/Alerts";
import { ResticUI } from "../../gen/ts/v1/service.pb";
import { Cron } from "react-js-cron";
import { validateForm } from "../lib/formutil";

export const AddPlanModal = ({
  template,
}: {
  template: Partial<Plan> | null;
}) => {
  const [config, setConfig] = useRecoilState(configState);
  const [deleteConfirmed, setDeleteConfirmed] = useState(false);
  const [confirmLoading, setConfirmLoading] = useState(false);
  const showModal = useShowModal();
  const alertsApi = useAlertApi()!;
  const [form] = Form.useForm<Plan>();

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
      config.plans = config.plans || [];

      if (!template) {
        throw new Error("template not found");
      }

      // Remove the plan from the config
      const idx = config.plans.findIndex((r) => r.id === template.id);
      if (idx === -1) {
        throw new Error("failed to update config, plan to delete not found");
      }

      config.plans.splice(idx, 1);

      // Update config and notify success.
      setConfig(await updateConfig(config));
      showModal(null);

      alertsApi.success(
        "Plan deleted from config, but not from restic repo. Snapshots will remain in storage and operations will be tracked until manually deleted. Reusing a deleted plan ID is not recommended if backups have already been performed."
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
      let plan = await validateForm<Plan>(form);

      let config = await fetchConfig();
      config.plans = config.plans || [];

      // Merge the new plan (or update) into the config
      if (template) {
        const idx = config.plans.findIndex((r) => r.id === template.id);
        if (idx === -1) {
          throw new Error("failed to update plan, not found");
        }
        config.plans[idx] = plan;
      } else {
        config.plans.push(plan);
      }

      // Update config and notify success.
      setConfig(await updateConfig(config));
      showModal(null);
    } catch (e: any) {
      alertsApi.error("Operation failed: " + e.message, 15);
    } finally {
      setConfirmLoading(false);
    }
  };

  const handleCancel = () => {
    showModal(null);
  };

  const repos = config?.repos || [];

  return (
    <>
      <Modal
        open={true}
        onCancel={handleCancel}
        title={template ? "Update Plan" : "Add Plan"}
        footer={[
          <Button loading={confirmLoading} key="back" onClick={handleCancel}>
            Cancel
          </Button>,
          template != null ? (
            <Button
              key="delete"
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
          {/* Plan.id */}
          <Form.Item<Plan>
            hasFeedback
            name="id"
            label="Plan Name"
            initialValue={template && template.id}
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
                  if (config?.plans?.find((r) => r.id === value)) {
                    throw new Error("Plan with name already exists");
                  }
                },
                message: "Plan with name already exists",
              },
            ]}
          >
            <Input
              placeholder={"plan" + ((config?.plans?.length || 0) + 1)}
              disabled={!!template}
            />
          </Form.Item>

          {/* Plan.repo */}
          <Form.Item<Plan>
            name="repo"
            label="Repository"
            validateTrigger={["onChange", "onBlur"]}
            initialValue={template && template.repo}
            rules={[
              {
                required: true,
                message: "Please select repository",
              },
            ]}
          >
            <Select
              // defaultValue={repos.length > 0 ? repos[0].id : undefined}
              options={repos.map((repo) => ({
                value: repo.id,
              }))}
              disabled={!!template}
            />
          </Form.Item>

          {/* Plan.paths */}
          <Form.List
            name="paths"
            rules={[]}
            initialValue={template ? template.paths : []}
          >
            {(fields, { add, remove }, { errors }) => (
              <>
                {fields.map((field, index) => (
                  <Form.Item
                    label={index === 0 ? "Paths" : ""}
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
                        },
                      ]}
                      noStyle
                    >
                      <URIAutocomplete
                        style={{ width: "60%" }}
                        onBlur={() => form.validateFields()}
                      />
                    </Form.Item>
                    <MinusCircleOutlined
                      className="dynamic-delete-button"
                      onClick={() => remove(field.name)}
                      style={{ paddingLeft: "5px" }}
                    />
                  </Form.Item>
                ))}
                <Form.Item label={fields.length === 0 ? "Paths" : ""}>
                  <Button
                    type="dashed"
                    onClick={() => add()}
                    style={{ width: "60%" }}
                    icon={<PlusOutlined />}
                  >
                    Add Path
                  </Button>
                  <Form.ErrorList errors={errors} />
                </Form.Item>
              </>
            )}
          </Form.List>
          {/* Plan.excludes */}
          <Form.List
            name="excludes"
            rules={[]}
            initialValue={template ? template.excludes : []}
          >
            {(fields, { add, remove }, { errors }) => (
              <>
                {fields.map((field, index) => (
                  <Form.Item
                    label={index === 0 ? "Excludes" : ""}
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
                        },
                      ]}
                      noStyle
                    >
                      <URIAutocomplete
                        style={{ width: "60%" }}
                        onBlur={() => form.validateFields()}
                      />
                    </Form.Item>
                    <MinusCircleOutlined
                      className="dynamic-delete-button"
                      onClick={() => remove(field.name)}
                      style={{ paddingLeft: "5px" }}
                    />
                  </Form.Item>
                ))}
                <Form.Item label={fields.length === 0 ? "Excludes" : ""}>
                  <Button
                    type="dashed"
                    onClick={() => add()}
                    style={{ width: "60%" }}
                    icon={<PlusOutlined />}
                  >
                    Add Exclusion Glob
                  </Button>
                  <Form.ErrorList errors={errors} />
                </Form.Item>
              </>
            )}
          </Form.List>

          {/* Plan.cron */}
          <Tooltip title="Cron expression to schedule the plan in 24 hour time">
            <Form.Item<Plan>
              name="cron"
              label="Schedule"
              initialValue={template ? template.cron : "0 0 * * *"}
              validateTrigger={["onChange", "onBlur"]}
              rules={[
                {
                  required: true,
                  message: "Please input schedule",
                },
              ]}
            >
              <Cron
                value={form.getFieldValue("cron")}
                setValue={(val: string) => {
                  form.setFieldValue("cron", val);
                }}
                clearButton={false}
              />
            </Form.Item>
          </Tooltip>

          {/* Plan.retention */}

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
