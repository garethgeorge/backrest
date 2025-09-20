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
  Col,
  Collapse,
  Checkbox,
} from "antd";
import React, { useEffect, useMemo, useState } from "react";
import { useShowModal } from "../components/ModalManager";
import {
  ConfigSchema,
  PlanSchema,
  RetentionPolicySchema,
  Schedule_Clock,
  type Plan,
} from "../../gen/ts/v1/config_pb";
import {
  CalculatorOutlined,
  MinusCircleOutlined,
  PlusOutlined,
} from "@ant-design/icons";
import { URIAutocomplete } from "../components/URIAutocomplete";
import { formatErrorAlert, useAlertApi } from "../components/Alerts";
import { namePattern, validateForm } from "../lib/formutil";
import {
  HooksFormList,
  hooksListTooltipText,
} from "../components/HooksFormList";
import { ConfirmButton, SpinButton } from "../components/SpinButton";
import { useConfig } from "../components/ConfigProvider";
import { backrestService } from "../api";
import {
  ScheduleDefaultsDaily,
  ScheduleFormItem,
} from "../components/ScheduleFormItem";
import { clone, create, equals, fromJson, toJson } from "@bufbuild/protobuf";
import { formatDuration } from "../lib/formatting";
import { getMinimumCronDuration } from "../lib/cronutil";

const planDefaults = create(PlanSchema, {
  schedule: {
    schedule: {
      case: "cron",
      value: "0 * * * *", // every hour
    },
    clock: Schedule_Clock.LOCAL,
  },
  retention: {
    policy: {
      case: "policyTimeBucketed",
      value: {
        hourly: 24,
        daily: 30,
        monthly: 12,
      },
    },
  },
});

export const AddPlanModal = ({ template, templateClone }: { template: Plan | null, templateClone?: Plan | null }) => {
  const [confirmLoading, setConfirmLoading] = useState(false);
  const showModal = useShowModal();
  const alertsApi = useAlertApi()!;
  const [config, setConfig] = useConfig();
  const [form] = Form.useForm();

  // This sets the default values for the form
  useEffect(() => {
    if (templateClone) { // If the user is cloning an existing plan
      form.setFieldsValue(
        templateClone
          ? toJson(PlanSchema, templateClone, { alwaysEmitImplicit: true })
          : toJson(PlanSchema, planDefaults, { alwaysEmitImplicit: true })
      );
    } else { // If the user is opening an existing plan or creating a new one
      form.setFieldsValue(
        template
          ? toJson(PlanSchema, template, { alwaysEmitImplicit: true })
          : toJson(PlanSchema, planDefaults, { alwaysEmitImplicit: true })
      );
    }
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

      const configCopy = clone(ConfigSchema, config);

      // Remove the plan from the config
      const idx = configCopy.plans.findIndex((r) => r.id === template.id);
      if (idx === -1) {
        throw new Error("failed to update config, plan to delete not found");
      }
      configCopy.plans.splice(idx, 1);

      // Update config and notify success.
      setConfig(await backrestService.setConfig(configCopy));
      showModal(null);

      alertsApi.success(
        "Plan deleted from config, but not from restic repo. Snapshots will remain in storage and operations will be tracked until manually deleted. Reusing a deleted plan ID is not recommended if backups have already been performed.",
        30
      );
    } catch (e: any) {
      alertsApi.error(formatErrorAlert(e, "Destroy error:"), 15);
    } finally {
      setConfirmLoading(false);
    }
  };

  const handleClone = async () => {
    setConfirmLoading(true);

    try {
      if (!template) {
        throw new Error("template not found");
      }

      // Reopen the modal with the cloned template as the default values
      showModal(<AddPlanModal template={null} templateClone={template} />);

    } catch (e: any) {
      alertsApi.error(formatErrorAlert(e, "Clone error:"), 15);
    } finally {
      setConfirmLoading(false);
    }
  }

  const handleOk = async () => {
    setConfirmLoading(true);

    try {
      let planFormData = await validateForm(form);
      const plan = fromJson(PlanSchema, planFormData, {
        ignoreUnknownFields: false,
      });

      if (
        plan.retention &&
        equals(
          RetentionPolicySchema,
          plan.retention,
          create(RetentionPolicySchema, {})
        )
      ) {
        delete plan.retention;
      }

      const configCopy = clone(ConfigSchema, config);

      // Merge the new plan (or update) into the config
      if (template) {
        const idx = configCopy.plans.findIndex((r) => r.id === template.id);
        if (idx === -1) {
          throw new Error("failed to update plan, not found");
        }
        configCopy.plans[idx] = plan;
      } else {
        configCopy.plans.push(plan);
      }

      // Update config and notify success.
      setConfig(await backrestService.setConfig(configCopy));
      showModal(null);
    } catch (e: any) {
      alertsApi.error(formatErrorAlert(e, "Operation error: "), 15);
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
        width="60vw"
        footer={[
          <Button loading={confirmLoading} key="back" onClick={handleCancel}>
            Cancel
          </Button>,
          template != null ? (
            <><ConfirmButton
              key="delete"
              type="primary"
              danger
              onClickAsync={handleDestroy}
              confirmTitle="Confirm Delete"
            >
              Delete
            </ConfirmButton>
            <Button loading={confirmLoading} key="clone" onClick={handleClone}>
              Clone Plan
            </Button></>
          ) : null,
          <SpinButton key="submit" type="primary" onClickAsync={handleOk}>
            Submit
          </SpinButton>,
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
          for plan configuration instructions.
        </p>
        <br />
        <Form
          autoComplete="off"
          form={form}
          labelCol={{ span: 6 }}
          wrapperCol={{ span: 16 }}
          disabled={confirmLoading}
        >
          {/* Plan.id */}
          <Form.Item<Plan>
            hasFeedback
            name="id"
            label="Plan Name"
            initialValue={template ? template.id : ""}
            validateTrigger={["onChange", "onBlur"]}
            tooltip="Unique ID that identifies this plan in the backrest UI (e.g. s3-myplan). This cannot be changed after creation."
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

          {/* Plan.repo */}
          <Form.Item<Plan>
            name="repo"
            label="Repository"
            validateTrigger={["onChange", "onBlur"]}
            initialValue={template ? template.repo : ""}
            tooltip="The repo that backrest will store your snapshots in."
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
          <Form.Item label="Paths" required={true}>
            <Form.List
              name="paths"
              rules={[]}
              initialValue={template ? template.paths : []}
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
                    );
                  })}
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
          <Form.Item
            label="Excludes"
            required={false}
            tooltip={
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
            <Form.List
              name="excludes"
              rules={[]}
              initialValue={template ? template.excludes : []}
            >
              {(fields, { add, remove }, { errors }) => (
                <>
                  {fields.map((field, index) => {
                    const { key, ...restField } = field;
                    return (
                      <Form.Item required={false} key={field.key}>
                        <Form.Item
                          {...restField}
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
                    );
                  })}
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

          {/* Plan.iexcludes */}
          <Form.Item
            label="Excludes (Case Insensitive)"
            required={false}
            tooltip={
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
            <Form.List
              name="iexcludes"
              rules={[]}
              initialValue={template ? template.iexcludes : []}
            >
              {(fields, { add, remove }, { errors }) => (
                <>
                  {fields.map((field, index) => {
                    const { key, ...restField } = field;
                    return (
                      <Form.Item required={false} key={field.key}>
                        <Form.Item
                          {...restField}
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
                    );
                  })}
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

          {/* Plan.cron */}
          <Form.Item label="Backup Schedule">
            <ScheduleFormItem
              name={["schedule"]}
              defaults={ScheduleDefaultsDaily}
            />
          </Form.Item>

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
                                "Value should be a CLI flag e.g. see restic backup --help",
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

          {/* Plan.retention */}
          <RetentionPolicyView />

          {/* Plan.hooks */}
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
  const schedule = Form.useWatch("schedule", { form }) as any;
  const retention = Form.useWatch("retention", { form, preserve: true }) as any;
  // If the first value in the cron expression (minutes) is not just a plain number (e.g. 30), the
  // cron will hit more than once per hour (e.g. "*/15" "1,30" and "*").
  const cronIsSubHourly = useMemo(
    () => schedule?.cron && !/^\d+ /.test(schedule.cron),
    [schedule?.cron]
  );
  // Translates the number of snapshots retained to a retention duration for cron schedules.
  const minRetention = useMemo(() => {
    const keepLastN = retention?.policyTimeBucketed?.keepLastN;
    if (!keepLastN) {
      return null;
    }
    const msPerHour = 60 * 60 * 1000;
    const msPerDay = 24 * msPerHour;
    let duration = 0;
    // Simple calculations for non-cron schedules
    if (schedule?.maxFrequencyHours) {
      duration = schedule.maxFrequencyHours * (keepLastN - 1) * msPerHour;
    } else if (schedule?.maxFrequencyDays) {
      duration = schedule.maxFrequencyDays * (keepLastN - 1) * msPerDay;
    } else if (schedule?.cron && retention.policyTimeBucketed?.keepLastN) {
      duration = getMinimumCronDuration(
        schedule.cron,
        retention.policyTimeBucketed?.keepLastN
      );
    }
    return duration ? formatDuration(duration, { minUnit: "h" }) : null;
  }, [schedule, retention?.policyTimeBucketed?.keepLastN]);

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
      <>
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
        <Form.Item
          name={["retention", "policyTimeBucketed", "keepLastN"]}
          label="Latest snapshots to keep regardless of age"
          validateTrigger={["onChange", "onBlur"]}
          initialValue={0}
          required={cronIsSubHourly}
          rules={[
            {
              validator: async (_, value) => {
                if (cronIsSubHourly && !(value > 1)) {
                  throw new Error("Specify a number greater than 1");
                }
              },
              message:
                "Your schedule runs more than once per hour; choose how many snapshots to keep before handing off to the retention policy.",
            },
          ]}
        >
          <InputNumber
            type="number"
            min={0}
            addonAfter={
              <Tooltip
                title={
                  minRetention
                    ? `${retention?.policyTimeBucketed?.keepLastN} snapshots represents an expected retention duration of at least 
                ${minRetention}, but this may vary with manual backups or if intermittently online.`
                    : "Choose how many snapshots to retain, then use the calculator to see the expected duration they would cover."
                }
              >
                <CalculatorOutlined
                  style={{
                    padding: ".5em",
                    margin: "0 -.5em",
                  }}
                />
              </Tooltip>
            }
          />
        </Form.Item>
      </>
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
              <Tooltip title="The last N snapshots for each time period will be kept by restic. Retention policy is applied to drop older snapshots after each backup run.">
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
