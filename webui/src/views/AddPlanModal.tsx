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
} from "antd";
import React, { useState } from "react";
import { useShowModal } from "../components/ModalManager";
import { Plan, RetentionPolicy } from "../../gen/ts/v1/config.pb";
import { useRecoilState } from "recoil";
import { configState, fetchConfig, updateConfig } from "../state/config";
import { nameRegex } from "../lib/patterns";
import { MinusCircleOutlined, PlusOutlined } from "@ant-design/icons";
import { URIAutocomplete } from "../components/URIAutocomplete";
import { useAlertApi } from "../components/Alerts";
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
        "Plan deleted from config, but not from restic repo. Snapshots will remain in storage and operations will be tracked until manually deleted. Reusing a deleted plan ID is not recommended if backups have already been performed.",
        30
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
        width="40vw"
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
        <Form
          autoComplete="off"
          form={form}
          labelCol={{ span: 6 }}
          wrapperCol={{ span: 16 }}
        >
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
                    label={index === 0 ? "Paths" : " "}
                    required={false}
                    key={field.key}
                    colon={index === 0}
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
                        style={{ width: "100%" }}
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
                <Form.Item
                  label={fields.length === 0 ? "Paths" : " "}
                  colon={fields.length === 0}
                >
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
                    label={index === 0 ? "Excludes" : " "}
                    colon={index === 0}
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
                <Form.Item
                  label={fields.length === 0 ? "Excludes" : " "}
                  colon={fields.length === 0}
                >
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
          <RetentionPolicyView policy={template?.retention} />

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

const RetentionPolicyView = ({ policy }: { policy?: RetentionPolicy }) => {
  enum PolicyType {
    TimeBased,
    CountBased,
  }

  policy = policy || {};

  const [policyType, setPolicyType] = useState<PolicyType>(
    policy.keepLastN ? PolicyType.CountBased : PolicyType.TimeBased
  );

  let elem = null;
  switch (policyType) {
    case PolicyType.TimeBased:
      elem = (
        <Form.Item
          required={true}
          label={
            <Tooltip title="The specified numbers of daily, weekly, monthly, etc backups will be retained. Retention policy is applied to drop older snapshots after each backup run.">
              Retain by Time
            </Tooltip>
          }
        >
          <Card>
            <Row>
              <Col span={11}>
                <Form.Item
                  name={["retention", "keepYearly"]}
                  initialValue={policy.keepYearly || 0}
                  validateTrigger={["onChange", "onBlur"]}
                  required={false}
                >
                  <InputNumber
                    addonBefore={<div style={{ width: "5em" }}>Yearly</div>}
                    type="number"
                  />
                </Form.Item>
                <Form.Item
                  name={["retention", "keepMonthly"]}
                  initialValue={policy.keepMonthly || 3}
                  validateTrigger={["onChange", "onBlur"]}
                  required={false}
                >
                  <InputNumber
                    addonBefore={<div style={{ width: "5em" }}>Monthly</div>}
                    type="number"
                  />
                </Form.Item>
                <Form.Item
                  name={["retention", "keepWeekly"]}
                  initialValue={policy.keepWeekly || 4}
                  validateTrigger={["onChange", "onBlur"]}
                  required={false}
                >
                  <InputNumber
                    addonBefore={<div style={{ width: "5em" }}>Weekly</div>}
                    type="number"
                  />
                </Form.Item>
              </Col>
              <Col span={11} offset={1}>
                <Form.Item
                  name={["retention", "keepDaily"]}
                  initialValue={policy.keepDaily || 7}
                  validateTrigger={["onChange", "onBlur"]}
                  required={false}
                >
                  <InputNumber
                    addonBefore={<div style={{ width: "5em" }}>Daily</div>}
                    type="number"
                  />
                </Form.Item>
                <Form.Item
                  name={["retention", "keepHourly"]}
                  initialValue={policy.keepHourly || 24}
                  validateTrigger={["onChange", "onBlur"]}
                  required={false}
                >
                  <InputNumber
                    addonBefore={<div style={{ width: "5em" }}>Hourly</div>}
                    type="number"
                  />
                </Form.Item>
              </Col>
            </Row>
          </Card>
        </Form.Item>
      );
      break;
    case PolicyType.CountBased:
      elem = (
        <Form.Item
          name={["retention", "keepLastN"]}
          label={
            <Tooltip title="The last N snapshots will be kept by restic. Retention policy is applied to drop older snapshots after each backup run.">
              Retain Last N
            </Tooltip>
          }
          initialValue={policy.keepLastN || 30}
          validateTrigger={["onChange", "onBlur"]}
          rules={[
            {
              required: true,
              message: "Please input keep last N",
            },
          ]}
        >
          <InputNumber type="number" />
        </Form.Item>
      );
      break;
  }

  return (
    <>
      <Form.Item label="Retention Policy">
        <Radio.Group
          value={policyType}
          onChange={(e) => {
            setPolicyType(e.target.value);
          }}
        >
          <Radio.Button value={PolicyType.CountBased}>By Count</Radio.Button>
          <Radio.Button value={PolicyType.TimeBased}>
            By Time Period
          </Radio.Button>
        </Radio.Group>
      </Form.Item>
      {elem}
    </>
  );
};
