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
  Collapse,
  FormInstance,
  Checkbox,
} from "antd";
import React, { useEffect, useState } from "react";
import { useShowModal } from "../components/ModalManager";
import { Plan, RetentionPolicy } from "../../gen/ts/v1/config_pb";
import { MinusCircleOutlined, PlusOutlined } from "@ant-design/icons";
import { URIAutocomplete } from "../components/URIAutocomplete";
import { useAlertApi } from "../components/Alerts";
import { Cron } from "react-js-cron";
import { namePattern, validateForm } from "../lib/formutil";
import {
  HooksFormList,
  hooksListTooltipText,
} from "../components/HooksFormList";
import { ConfirmButton, SpinButton } from "../components/SpinButton";
import { useConfig } from "../components/ConfigProvider";
import { backrestService } from "../api";

export const AddPlanModal = ({ template }: { template: Plan | null }) => {
  const [confirmLoading, setConfirmLoading] = useState(false);
  const showModal = useShowModal();
  const alertsApi = useAlertApi()!;
  const [config, setConfig] = useConfig();
  const [form] = Form.useForm();
  useEffect(() => {
    form.setFieldsValue(template ? JSON.parse(template.toJsonString()) : {});
  }, [template]);

  if (!config) {
    return null;
  }

  const handleDestroy = async () => {
    setConfirmLoading(true);

    try {
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
      setConfig(await backrestService.setConfig(config));
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
      let planFormData = await validateForm(form);
      const plan = new Plan().fromJsonString(JSON.stringify(planFormData), {
        ignoreUnknownFields: false,
      });

      if (plan.retention && plan.retention.equals(new RetentionPolicy())) {
        delete plan.retention;
      }

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
      setConfig(await backrestService.setConfig(config));
      showModal(null);
    } catch (e: any) {
      alertsApi.error("Operation failed: " + e.message, 15);
      console.error(e);
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
          <SpinButton key="submit" type="primary" onClickAsync={handleOk}>
            Submit
          </SpinButton>,
        ]}
      >
        <Form
          autoComplete="off"
          form={form}
          labelCol={{ span: 6 }}
          wrapperCol={{ span: 16 }}
          disabled={confirmLoading}
        >
          {/* Plan.id */}
          <Tooltip title="Unique ID that identifies this plan in the backrest UI (e.g. s3-myplan). This cannot be changed after creation.">
            <Form.Item<Plan>
              hasFeedback
              name="id"
              label="Plan Name"
              initialValue={template ? template.id : ""}
              validateTrigger={["onChange", "onBlur"]}
              rules={[
                {
                  required: true,
                  message: "Please input plan name",
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
                {
                  pattern: namePattern,
                  message:
                    "Name must be alphanumeric with dashes or underscores as separators",
                },
              ]}
            >
              <Input
                placeholder={"plan" + ((config?.plans?.length || 0) + 1)}
                disabled={!!template}
              />
            </Form.Item>
          </Tooltip>

          {/* Plan.repo */}
          <Tooltip title="The repo that backrest will store your snapshots in.">
            <Form.Item<Plan>
              name="repo"
              label="Repository"
              validateTrigger={["onChange", "onBlur"]}
              initialValue={template ? template.repo : ""}
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
          </Tooltip>

          {/* Plan.paths */}
          <Form.Item label="Paths" required={true}>
            <Form.List
              name="paths"
              rules={[]}
              initialValue={template ? template.paths : []}
            >
              {(fields, { add, remove }, { errors }) => (
                <>
                  {fields.map((field, index) => (
                    <Form.Item key={field.key}>
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
                          style={{ width: "90%" }}
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
                  <Form.Item>
                    <Button
                      type="dashed"
                      onClick={() => add()}
                      style={{ width: "90%" }}
                      icon={<PlusOutlined />}
                    >
                      Add Path
                    </Button>
                    <Form.ErrorList errors={errors} />
                  </Form.Item>
                </>
              )}
            </Form.List>
          </Form.Item>

          {/* Plan.excludes */}
          <Tooltip
            title={
              <>
                Paths to exclude from your backups. See the{" "}
                <a
                  href="https://restic.readthedocs.io/en/latest/040_backup.html#excluding-files"
                  target="_blank"
                >
                  restic docs
                </a>{" "}
                for more info.
              </>
            }
          >
            <Form.Item label="Excludes" required={false}>
              <Form.List
                name="excludes"
                rules={[]}
                initialValue={template ? template.excludes : []}
              >
                {(fields, { add, remove }, { errors }) => (
                  <>
                    {fields.map((field, index) => (
                      <Form.Item required={false} key={field.key}>
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
                            style={{ width: "90%" }}
                            onBlur={() => form.validateFields()}
                            globAllowed={true}
                          />
                        </Form.Item>
                        <MinusCircleOutlined
                          className="dynamic-delete-button"
                          onClick={() => remove(field.name)}
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
                        Add Exclusion Glob
                      </Button>
                      <Form.ErrorList errors={errors} />
                    </Form.Item>
                  </>
                )}
              </Form.List>
            </Form.Item>
          </Tooltip>

          {/* Plan.iexcludes */}
          <Tooltip
            title={
              <>
                Case insensitive paths to exclude from your backups. See the{" "}
                <a
                  href="https://restic.readthedocs.io/en/latest/040_backup.html#excluding-files"
                  target="_blank"
                >
                  restic docs
                </a>{" "}
                for more info.
              </>
            }
          >
            <Form.Item label="Excludes (Case Insensitive)" required={false}>
              <Form.List
                name="iexcludes"
                rules={[]}
                initialValue={template ? template.iexcludes : []}
              >
                {(fields, { add, remove }, { errors }) => (
                  <>
                    {fields.map((field, index) => (
                      <Form.Item required={false} key={field.key}>
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
                            style={{ width: "90%" }}
                            onBlur={() => form.validateFields()}
                            globAllowed={true}
                          />
                        </Form.Item>
                        <MinusCircleOutlined
                          className="dynamic-delete-button"
                          onClick={() => remove(field.name)}
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
                        Add Case Insensitive Exclusion Glob
                      </Button>
                      <Form.ErrorList errors={errors} />
                    </Form.Item>
                  </>
                )}
              </Form.List>
            </Form.Item>
          </Tooltip>

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

          {/* Plan.backup_flags */}
          <Form.Item
            label={
              <Tooltip title="Extra flags to add to the 'restic backup' command">
                Backup Flags
              </Tooltip>
            }
          >
            <Form.List name="backup_flags">
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
                              "Value should be a CLI flag e.g. see restic backup --help",
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

          {/* Plan.retention */}
          <RetentionPolicyView />

          {/* Plan.hooks */}
          <Form.Item
            label={<Tooltip title={hooksListTooltipText}>Hooks</Tooltip>}
          >
            <HooksFormList />
          </Form.Item>

          {/* Disabled? toggles whether the plan will be scheduled. */}
          <Form.Item
            label={
              <Tooltip
                title={
                  "Toggles whether the plan's scheduling is enabled. If disabled no scheduled operations will be run."
                }
              >
                Disable Scheduling
              </Tooltip>
            }
            name="disabled"
            valuePropName="checked"
          >
            <Checkbox />
          </Form.Item>

          <Form.Item shouldUpdate label="Preview">
            {() => (
              <Collapse
                size="small"
                items={[
                  {
                    key: "1",
                    label: "Plan Config as JSON",
                    children: (
                      <Typography>
                        <pre>
                          {JSON.stringify(form.getFieldsValue(), null, 2)}
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

const RetentionPolicyView = () => {
  const form = Form.useFormInstance();
  const retention = Form.useWatch("retention", { form, preserve: true }) as any;

  const determineMode = () => {
    if (!retention) {
      return "policyTimeBucketed";
    } else if (retention.policyKeepLastN) {
      return "policyKeepLastN";
    } else if (retention.policyKeepAll) {
      return "policyKeepAll";
    } else if (retention.policyTimeBucketed) {
      return "policyTimeBucketed";
    }
  };

  const mode = determineMode();

  let elem: React.ReactNode = null;
  if (mode === "policyKeepAll") {
    elem = (
      <>
        <p>
          All backups are retained e.g. for append-only repos. Ensure that you
          manually forget / prune backups elsewhere. Backrest will register
          forgets performed externally on the next backup.
        </p>
        <Form.Item
          name={["retention", "policyKeepAll"]}
          valuePropName="checked"
          initialValue={true}
          hidden={true}
        >
          <Checkbox />
        </Form.Item>
      </>
    );
  } else if (mode === "policyKeepLastN") {
    elem = (
      <Form.Item
        name={["retention", "policyKeepLastN"]}
        initialValue={0}
        validateTrigger={["onChange", "onBlur"]}
        rules={[
          {
            required: true,
            message: "Please input keep last N",
          },
        ]}
      >
        <InputNumber
          addonBefore={<div style={{ width: "5em" }}>Count</div>}
          type="number"
        />
      </Form.Item>
    );
  } else if (mode === "policyTimeBucketed") {
    elem = (
      <Row>
        <Col span={11}>
          <Form.Item
            name={["retention", "policyTimeBucketed", "yearly"]}
            validateTrigger={["onChange", "onBlur"]}
            initialValue={0}
            required={false}
          >
            <InputNumber
              addonBefore={<div style={{ width: "5em" }}>Yearly</div>}
              type="number"
            />
          </Form.Item>
          <Form.Item
            name={["retention", "policyTimeBucketed", "monthly"]}
            initialValue={0}
            validateTrigger={["onChange", "onBlur"]}
            required={false}
          >
            <InputNumber
              addonBefore={<div style={{ width: "5em" }}>Monthly</div>}
              type="number"
            />
          </Form.Item>
          <Form.Item
            name={["retention", "policyTimeBucketed", "weekly"]}
            initialValue={0}
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
            name={["retention", "policyTimeBucketed", "daily"]}
            validateTrigger={["onChange", "onBlur"]}
            initialValue={0}
            required={false}
          >
            <InputNumber
              addonBefore={<div style={{ width: "5em" }}>Daily</div>}
              type="number"
            />
          </Form.Item>
          <Form.Item
            name={["retention", "policyTimeBucketed", "hourly"]}
            validateTrigger={["onChange", "onBlur"]}
            initialValue={0}
            required={false}
          >
            <InputNumber
              addonBefore={<div style={{ width: "5em" }}>Hourly</div>}
              type="number"
            />
          </Form.Item>
        </Col>
      </Row>
    );
  }

  return (
    <>
      <Form.Item label="Retention Policy">
        <Row>
          <Radio.Group
            value={mode}
            onChange={(e) => {
              const selected = e.target.value;
              if (selected === "policyKeepLastN") {
                form.setFieldValue("retention", { policyKeepLastN: 30 });
              } else if (selected === "policyTimeBucketed") {
                form.setFieldValue("retention", {
                  policyTimeBucketed: {
                    yearly: 0,
                    monthly: 3,
                    weekly: 4,
                    daily: 7,
                    hourly: 24,
                  },
                });
              } else {
                form.setFieldValue("retention", { policyKeepAll: true });
              }
            }}
          >
            <Radio.Button value={"policyKeepLastN"}>
              <Tooltip title="The last N snapshots will be kept by restic. Retention policy is applied to drop older snapshots after each backup run.">
                By Count
              </Tooltip>
            </Radio.Button>
            <Radio.Button value={"policyTimeBucketed"}>
              <Tooltip title="Snapshots older than the specified time period will be dropped by restic. Retention policy is applied to drop older snapshots after each backup run.">
                By Time Period
              </Tooltip>
            </Radio.Button>
            <Radio.Button value={"policyKeepAll"}>
              <Tooltip title="All backups will be retained. Note that this may result in slow backups if the set of snapshots grows very large.">
                None
              </Tooltip>
            </Radio.Button>
          </Radio.Group>
        </Row>
        <br />
        <Row>
          <Form.Item>{elem}</Form.Item>
        </Row>
      </Form.Item>
    </>
  );
};
