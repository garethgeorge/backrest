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
  AutoComplete,
  Flex,
} from "antd";
import React, { useEffect, useMemo, useRef, useState } from "react";
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
import { debounce } from "../lib/util";
import { StringList } from "../../gen/ts/types/value_pb";
import { isWindows } from "../state/buildcfg";
import * as m from "../paraglide/messages";

const { TextArea } = Input;
const sep = isWindows ? "\\" : "/";



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

export const AddPlanModal = ({ template }: { template: Plan | null }) => {
  const [confirmLoading, setConfirmLoading] = useState(false);
  const showModal = useShowModal();
  const alertsApi = useAlertApi()!;
  const [config, setConfig] = useConfig();
  const [form] = Form.useForm();
  useEffect(() => {
    const formData = template
      ? toJson(PlanSchema, template, { alwaysEmitImplicit: true })
      : toJson(PlanSchema, planDefaults, { alwaysEmitImplicit: true });

    form.setFieldsValue(formData);
  }, [template]);

  if (!config) {
    return null;
  }

  const handleDestroy = async () => {
    setConfirmLoading(true);

    try {
      if (!template) {
        throw new Error(m.add_plan_modal_error_template_not_found());
      }

      const configCopy = clone(ConfigSchema, config);

      // Remove the plan from the config
      const idx = configCopy.plans.findIndex((r) => r.id === template.id);
      if (idx === -1) {
        throw new Error(m.add_plan_modal_error_plan_delete_not_found());
      }
      configCopy.plans.splice(idx, 1);

      // Update config and notify success.
      setConfig(await backrestService.setConfig(configCopy));
      showModal(null);

      alertsApi.success(
        m.add_plan_modal_success_plan_deleted(),
        30
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
      alertsApi.error(formatErrorAlert(e, m.add_plan_modal_error_operation_prefix()), 15);
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
        title={template ? m.add_plan_modal_title_update() : m.add_plan_modal_title_add()}
        width="60vw"
        footer={[
          <Button loading={confirmLoading} key="back" onClick={handleCancel}>
            {m.add_plan_modal_button_cancel()}
          </Button>,
          template != null ? (
            <ConfirmButton
              key="delete"
              type="primary"
              danger
              onClickAsync={handleDestroy}
              confirmTitle={m.add_plan_modal_button_confirm_delete()}
            >
              {m.add_plan_modal_button_delete()}
            </ConfirmButton>
          ) : null,
          <SpinButton key="submit" type="primary" onClickAsync={handleOk}>
            {m.add_plan_modal_button_submit()}
          </SpinButton>,
        ]}
        maskClosable={false}
      >
        <p>
          {m.add_plan_modal_see_guide_prefix()}{" "}
          <a
            href="https://garethgeorge.github.io/backrest/introduction/getting-started"
            target="_blank"
          >
            {m.add_plan_modal_see_guide_link()}
          </a>{" "}
          {m.add_plan_modal_see_guide_suffix()}
        </p>
        <br />
        <Form
          autoComplete="off"
          form={form}
          labelCol={{ flex: "160px" }}
          wrapperCol={{ flex: "auto" }}
          disabled={confirmLoading}
        >
          {/* Plan.id */}
          <Form.Item<Plan>
            hasFeedback
            name="id"
            label={m.add_plan_modal_field_plan_name()}
            initialValue={template ? template.id : ""}
            validateTrigger={["onChange", "onBlur"]}
            tooltip={m.add_plan_modal_field_plan_name_tooltip()}
            rules={[
              {
                required: true,
                message: m.add_plan_modal_validation_plan_name_required(),
              },
              {
                validator: async (_, value) => {
                  if (template) return;
                  if (config?.plans?.find((r) => r.id === value)) {
                    throw new Error(m.add_plan_modal_validation_plan_exists());
                  }
                },
                message: m.add_plan_modal_validation_plan_exists(),
              },
              {
                pattern: namePattern,
                message:
                  m.add_plan_modal_validation_plan_name_pattern(),
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
            label={m.add_plan_modal_field_repository()}
            validateTrigger={["onChange", "onBlur"]}
            initialValue={template ? template.repo : ""}
            tooltip={m.add_plan_modal_field_repository_tooltip()}
            rules={[
              {
                required: true,
                message: m.add_plan_modal_validation_repository_required(),
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
          <Form.Item
            label={m.add_plan_modal_field_paths()}
            required={true}
            tooltip={m.add_plan_modal_field_paths_tooltip()}
          >
            <Form.List
              name="paths"
              rules={[
                {
                  validator: async (_, paths) => {
                    if (!paths || paths.length === 0) {
                      throw new Error(
                        m.add_plan_modal_validation_paths_required()
                      );
                    }
                  },
                },
              ]}
              initialValue={template ? template.paths : []}
            >
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
                            initialValue={""}
                            rules={[
                              {
                                required: true,
                                message:
                                  m.add_plan_modal_validation_paths_valid_required(),
                              },
                            ]}
                            noStyle
                          >
                            <URIAutocomplete
                              style={{ flex: 1 }}
                              onBlur={() => form.validateFields()}
                              globAllowed={true}
                            />
                          </Form.Item>
                          <MinusCircleOutlined
                            className="dynamic-delete-button"
                            onClick={() => remove(field.name)}
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
            label={m.add_plan_modal_field_excludes()}
            required={false}
            tooltip={
              <>
                {m.add_plan_modal_field_excludes_tooltip_prefix()}{" "}
                <a
                  href="https://restic.readthedocs.io/en/latest/040_backup.html#excluding-files"
                  target="_blank"
                >
                  {m.add_plan_modal_field_excludes_tooltip_link()}
                </a>{" "}
                {m.add_plan_modal_field_excludes_tooltip_suffix()}
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
                        <Flex gap="small" align="center">
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
                              style={{ flex: 1 }}
                              onBlur={() => form.validateFields()}
                              globAllowed={true}
                            />
                          </Form.Item>
                          <MinusCircleOutlined
                            className="dynamic-delete-button"
                            onClick={() => remove(field.name)}
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
                      {m.add_plan_modal_field_excludes_add()}
                    </Button>
                    <Form.ErrorList errors={errors} />
                  </Form.Item>
                </>
              )}
            </Form.List>
          </Form.Item>

          {/* Plan.iexcludes */}
          <Form.Item
            label={m.add_plan_modal_field_iexcludes()}
            required={false}
            tooltip={
              <>
                {m.add_plan_modal_field_iexcludes_tooltip_prefix()}{" "}
                <a
                  href="https://restic.readthedocs.io/en/latest/040_backup.html#excluding-files"
                  target="_blank"
                >
                  {m.add_plan_modal_field_excludes_tooltip_link()}
                </a>{" "}
                {m.add_plan_modal_field_excludes_tooltip_suffix()}
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
                        <Flex gap="small" align="center">
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
                              style={{ flex: 1 }}
                              onBlur={() => form.validateFields()}
                              globAllowed={true}
                            />
                          </Form.Item>
                          <MinusCircleOutlined
                            className="dynamic-delete-button"
                            onClick={() => remove(field.name)}
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
                      {m.add_plan_modal_field_iexcludes_add()}
                    </Button>
                    <Form.ErrorList errors={errors} />
                  </Form.Item>
                </>
              )}
            </Form.List>
          </Form.Item>

          {/* Plan.cron */}
          <Form.Item label={m.add_plan_modal_field_schedule()}>
            <ScheduleFormItem
              name={["schedule"]}
              defaults={ScheduleDefaultsDaily}
            />
          </Form.Item>

          {/* Plan.backup_flags */}
          <Form.Item
            label={
              <Tooltip title={m.add_plan_modal_field_backup_flags_tooltip()}>
                {m.add_plan_modal_field_backup_flags()}
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
                        <Flex gap="small" align="center">
                          <Form.Item
                            {...restField}
                            validateTrigger={["onChange", "onBlur"]}
                            rules={[
                              {
                                required: true,
                                whitespace: true,
                                pattern: /^\-\-?.*$/,
                                message:
                                  m.add_plan_modal_validation_flag_pattern(),
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
                      {m.add_plan_modal_field_backup_flags_add()}
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
                    label: m.add_plan_modal_preview_json(),
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
