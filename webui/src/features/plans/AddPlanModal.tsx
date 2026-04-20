import {
  Flex,
  Stack,
  Input,
  createListCollection,
  SelectContent,
  SelectItem,
  SelectRoot,
  SelectTrigger,
  SelectValueText,
  Card,
  Text as CText,
  Grid,
  Code,
} from "@chakra-ui/react";
import {
  AccordionItem,
  AccordionItemContent,
  AccordionItemTrigger,
  AccordionRoot,
} from "../../components/ui/accordion";
import { useEffect, useState, useMemo } from "react";
import { useShowModal } from "../../components/common/ModalManager";
import {
  ConfigSchema,
  PlanSchema,
  RetentionPolicySchema,
  Schedule_Clock,
  type Plan,
} from "../../../gen/ts/v1/config_pb";
import { FiFileText, FiFolder, FiClock, FiArchive, FiSliders } from "react-icons/fi";
import { alerts, formatErrorAlert } from "../../components/common/Alerts";
import { namePattern } from "../../lib/util";
import { ConfirmButton } from "../../components/common/SpinButton";
import { useConfig } from "../../app/provider";
import { backrestService } from "../../api/client";
import {
  clone,
  create,
  equals,
  fromJson,
  toJson,
} from "@bufbuild/protobuf";
import * as m from "../../paraglide/messages";
import { Button } from "../../components/ui/button";
import { Field } from "../../components/ui/field";
import { Tooltip } from "../../components/ui/tooltip";
import { NumberInputField } from "../../components/common/NumberInput";
import {
  ScheduleFormItem,
  ScheduleDefaultsDaily,
} from "../../components/common/ScheduleFormItem";
import {
  HooksFormList,
  hooksListTooltipText,
} from "../../components/common/HooksFormList";
import { DynamicList } from "../../components/common/DynamicList";
import {
  TwoPaneModal,
  TwoPaneSection,
  type SectionDef,
} from "../../components/common/TwoPaneModal";
import { SectionCard } from "../../components/common/SectionCard";

// Default Plan
const planDefaults = create(PlanSchema, {
  schedule: {
    schedule: {
      case: "cron",
      value: "0 * * * *",
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

export const AddPlanModal = ({ template, onSaveOverride }: { template: Plan | null, onSaveOverride?: (plan: Plan) => Promise<void> }) => {
  const [confirmLoading, setConfirmLoading] = useState(false);
  const showModal = useShowModal();
  const [config, setConfig] = useConfig();

  const [formData, setFormData] = useState<any>(
    template
      ? toJson(PlanSchema, template, { alwaysEmitImplicit: true })
      : toJson(PlanSchema, planDefaults, { alwaysEmitImplicit: true }),
  );

  useEffect(() => {
    setFormData(
      template
        ? toJson(PlanSchema, template, { alwaysEmitImplicit: true })
        : toJson(PlanSchema, planDefaults, { alwaysEmitImplicit: true }),
    );
  }, [template]);

  const updateField = (path: string[], value: any) => {
    setFormData((prev: any) => {
      const next = { ...prev };
      let curr = next;
      for (let i = 0; i < path.length - 1; i++) {
        curr[path[i]] = curr[path[i]] ? { ...curr[path[i]] } : {};
        curr = curr[path[i]];
      }
      curr[path[path.length - 1]] = value;
      return next;
    });
  };

  const getField = (path: string[]) => {
    let curr = formData;
    for (const p of path) {
      if (curr === undefined) return undefined;
      curr = curr[p];
    }
    return curr;
  };

  if (!config) return null;

  const handleDestroy = async () => {
    setConfirmLoading(true);
    try {
      if (!template)
        throw new Error(m.add_plan_modal_error_template_not_found());

      const configCopy = clone(ConfigSchema, config);
      const idx = configCopy.plans.findIndex((r) => r.id === template.id);
      if (idx === -1)
        throw new Error(m.add_plan_modal_error_plan_delete_not_found());

      configCopy.plans.splice(idx, 1);
      setConfig(await backrestService.setConfig(configCopy));
      showModal(null);
      alerts.success(m.add_plan_modal_success_plan_deleted());
    } catch (e: any) {
      alerts.error(
        formatErrorAlert(e, m.add_plan_modal_error_destroy_prefix()),
      );
    } finally {
      setConfirmLoading(false);
    }
  };

  const handleOk = async () => {
    setConfirmLoading(true);
    try {
      if (!formData.id?.trim()) {
        throw new Error(m.add_plan_modal_validation_plan_name_required());
      }
      if (!namePattern.test(formData.id)) {
        throw new Error(m.add_plan_modal_validation_plan_name_pattern());
      }
      if (!template && config.plans.find((p) => p.id === formData.id)) {
        throw new Error(m.add_plan_modal_validation_plan_exists());
      }
      if (!formData.repo) {
        throw new Error(m.add_plan_modal_validation_repository_required());
      }
      if (
        formData.backup_flags &&
        formData.backup_flags.some((f: string) => !/^\-\-?.*$/.test(f))
      ) {
        throw new Error(m.add_plan_modal_validation_flag_pattern());
      }

      const scheduleValue = formData.schedule?.schedule?.value;
      const isCron = formData.schedule?.schedule?.case === "cron";
      const isSubHourly =
        isCron && scheduleValue && !/^\d+ /.test(scheduleValue);

      if (
        isSubHourly &&
        formData.retention?.policyTimeBucketed &&
        !(formData.retention.policyTimeBucketed.keepLastN > 1)
      ) {
        throw new Error(
          "Your schedule runs more than once per hour; please specify a 'Latest (Count)' greater than 1 in Retention Policy.",
        );
      }

      const plan = fromJson(PlanSchema, formData, {
        ignoreUnknownFields: true,
      });

      if (
        plan.retention &&
        equals(
          RetentionPolicySchema,
          plan.retention,
          create(RetentionPolicySchema, {}),
        )
      ) {
        delete plan.retention;
      }

      if (onSaveOverride) {
        await onSaveOverride(plan);
        showModal(null);
        return;
      }

      const configCopy = clone(ConfigSchema, config);

      if (template) {
        const idx = configCopy.plans.findIndex((r) => r.id === template.id);
        if (idx === -1) throw new Error("failed to update plan, not found");
        configCopy.plans[idx] = plan;
      } else {
        configCopy.plans.push(plan);
      }

      setConfig(await backrestService.setConfig(configCopy));
      showModal(null);
    } catch (e: any) {
      alerts.error(
        formatErrorAlert(e, m.add_plan_modal_error_operation_prefix()),
      );
    } finally {
      setConfirmLoading(false);
    }
  };

  const repos = config?.repos || [];
  const repoOptions = createListCollection({
    items: repos.map((r) => ({ label: r.id, value: r.id })),
  });

  const sections: SectionDef[] = [
    { id: "details", label: "Details", icon: <FiFileText size={14} /> },
    { id: "scope", label: "Scope", icon: <FiFolder size={14} /> },
    { id: "schedule", label: "Schedule", icon: <FiClock size={14} /> },
    { id: "retention", label: "Retention", icon: <FiArchive size={14} /> },
    { id: "advanced", label: "Advanced", icon: <FiSliders size={14} /> },
  ];

  const footer = (
    <Flex gap={2} justify="flex-end" width="full">
      <Button
        variant="outline"
        disabled={confirmLoading}
        onClick={() => showModal(null)}
      >
        {m.add_plan_modal_button_cancel()}
      </Button>
      {template && (
        <ConfirmButton
          danger
          onClickAsync={handleDestroy}
          confirmTitle={m.add_plan_modal_button_confirm_delete()}
        >
          {m.add_plan_modal_button_delete()}
        </ConfirmButton>
      )}
      <Button loading={confirmLoading} onClick={handleOk}>
        {m.add_plan_modal_button_submit()}
      </Button>
    </Flex>
  );

  return (
    <TwoPaneModal
      isOpen={true}
      onClose={() => showModal(null)}
      title={
        template
          ? m.add_plan_modal_title_update()
          : m.add_plan_modal_title_add()
      }
      headerIcon={<FiFileText size={14} />}
      sections={sections}
      footer={footer}
    >
      {/* Details Section */}
      <TwoPaneSection id="details">
        <SectionCard
          icon={<FiFileText size={16} />}
          title={m.op_row_backup_details()}
          description="Plan name and target repository."
        >
          <Stack gap={4}>
            <Field
              label={m.add_plan_modal_field_plan_name()}
              helperText={m.add_plan_modal_field_plan_name_tooltip()}
              required
              invalid={
                !!formData.id &&
                (!namePattern.test(formData.id) ||
                  (!template &&
                    !!config.plans.find((p) => p.id === formData.id)))
              }
              errorText={
                !!formData.id && !namePattern.test(formData.id)
                  ? m.add_plan_modal_validation_plan_name_pattern()
                  : m.add_plan_modal_validation_plan_exists()
              }
            >
              <Input
                value={getField(["id"])}
                onChange={(e) => updateField(["id"], e.target.value)}
                disabled={!!template}
                placeholder={"plan" + ((config?.plans?.length || 0) + 1)}
              />
            </Field>

            <Field
              label={m.add_plan_modal_field_repository()}
              helperText={m.add_plan_modal_field_repository_tooltip()}
              required
              invalid={!getField(["repo"]) && confirmLoading}
            >
              <SelectRoot
                collection={repoOptions}
                size="sm"
                value={[getField(["repo"])]}
                onValueChange={(e: any) =>
                  updateField(["repo"], e.value[0])
                }
                disabled={!!template}
                width="full"
              >
                {/* @ts-ignore */}
                <SelectTrigger>
                  {/* @ts-ignore */}
                  <SelectValueText placeholder={m.add_plan_modal_field_repository_select()} />
                </SelectTrigger>
                {/* @ts-ignore */}
                <SelectContent>
                  {repoOptions.items.map((item: any) => (
                    <SelectItem item={item} key={item.value}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </SelectRoot>
            </Field>
          </Stack>
        </SectionCard>
      </TwoPaneSection>

      {/* Scope Section */}
      <TwoPaneSection id="scope">
        <SectionCard
          icon={<FiFolder size={16} />}
          title={m.settings_peer_permission_scopes()}
          description="Directories and exclusion patterns."
        >
          <Stack gap={4}>
            <DynamicList
              label={m.add_plan_modal_field_paths()}
              tooltip={m.add_plan_modal_field_paths_tooltip()}
              items={getField(["paths"]) || []}
              onUpdate={(items: string[]) => updateField(["paths"], items)}
              required
              autocompleteType="uri"
              placeholder={m.add_plan_modal_field_paths()}
            />

            <DynamicList
              label={m.add_plan_modal_field_excludes()}
              items={getField(["excludes"]) || []}
              onUpdate={(items: string[]) =>
                updateField(["excludes"], items)
              }
              tooltip={
                <>
                  {m.add_plan_modal_field_excludes_tooltip_prefix()}{" "}
                  <a
                    href="https://restic.readthedocs.io/en/latest/040_backup.html#excluding-files"
                    target="_blank"
                    style={{ textDecoration: "underline" }}
                  >
                    {m.add_plan_modal_field_excludes_tooltip_link()}
                  </a>{" "}
                  {m.add_plan_modal_field_excludes_tooltip_suffix()}
                </>
              }
              placeholder={m.add_plan_modal_field_excludes()}
            />

            <DynamicList
              label={m.add_plan_modal_field_iexcludes()}
              items={getField(["iexcludes"]) || []}
              onUpdate={(items: string[]) =>
                updateField(["iexcludes"], items)
              }
              tooltip={
                <>
                  {m.add_plan_modal_field_iexcludes_tooltip_prefix()}{" "}
                  <a
                    href="https://restic.readthedocs.io/en/latest/040_backup.html#excluding-files"
                    target="_blank"
                    style={{ textDecoration: "underline" }}
                  >
                    {m.add_plan_modal_field_excludes_tooltip_link()}
                  </a>{" "}
                  {m.add_plan_modal_field_excludes_tooltip_suffix()}
                </>
              }
              placeholder={m.add_plan_modal_field_iexcludes()}
            />
          </Stack>
        </SectionCard>
      </TwoPaneSection>

      {/* Schedule Section */}
      <TwoPaneSection id="schedule">
        <SectionCard
          icon={<FiClock size={16} />}
          title={m.add_plan_modal_field_schedule()}
          description="When backups run automatically."
        >
          <ScheduleFormItem
            value={getField(["schedule"])}
            onChange={(v: any) => updateField(["schedule"], v)}
            defaults={ScheduleDefaultsDaily}
          />
        </SectionCard>
      </TwoPaneSection>

      {/* Retention Section */}
      <TwoPaneSection id="retention">
        <SectionCard
          icon={<FiArchive size={16} />}
          title={m.add_plan_modal_retention_policy_label()}
          description="How long to keep snapshots before forgetting them."
        >
          <RetentionPolicyView
            schedule={getField(["schedule"])}
            retention={getField(["retention"])}
            onChange={(v: any) => updateField(["retention"], v)}
          />
        </SectionCard>
      </TwoPaneSection>

      {/* Advanced Section */}
      <TwoPaneSection id="advanced">
        <SectionCard
          icon={<FiSliders size={16} />}
          title={m.add_plan_modal_advanced_label()}
          description="Extra flags and notification hooks."
        >
          <Stack gap={4}>
            <DynamicList
              label={m.add_plan_modal_field_backup_flags()}
              items={getField(["backup_flags"]) || []}
              onUpdate={(items: string[]) =>
                updateField(["backup_flags"], items)
              }
              tooltip={m.add_plan_modal_field_backup_flags_tooltip()}
              placeholder="--flag"
              autocompleteType="flag"
            />

            <Field
              label={m.add_plan_modal_field_hooks()}
              helperText={hooksListTooltipText}
            >
              <HooksFormList
                value={getField(["hooks"])}
                onChange={(v: any) => updateField(["hooks"], v)}
              />
            </Field>
          </Stack>
        </SectionCard>
      </TwoPaneSection>

      {/* JSON Preview */}
      <AccordionRoot collapsible variant="plain">
        <AccordionItem value="json-preview">
          <AccordionItemTrigger>
            <CText fontSize="sm" color="fg.muted">
              {m.add_repo_modal_preview_json()}
            </CText>
          </AccordionItemTrigger>
          <AccordionItemContent>
            <Code
              display="block"
              whiteSpace="pre"
              overflowX="auto"
              p={2}
              borderRadius="md"
              fontSize="xs"
            >
              {JSON.stringify(formData, null, 2)}
            </Code>
          </AccordionItemContent>
        </AccordionItem>
      </AccordionRoot>
    </TwoPaneModal>
  );
};

// Retention View
const RetentionPolicyView = ({ schedule, retention, onChange }: any) => {
  const determineMode = () => {
    if (!retention) return "policyTimeBucketed";
    if (retention.policyKeepLastN) return "policyKeepLastN";
    if (retention.policyKeepAll) return "policyKeepAll";
    if (retention.policyTimeBucketed) return "policyTimeBucketed";
    return "policyTimeBucketed";
  };
  const mode = determineMode();

  const handleModeChange = (newMode: string) => {
    if (newMode === "policyKeepLastN") {
      onChange({ policyKeepLastN: 30 });
    } else if (newMode === "policyTimeBucketed") {
      onChange({
        policyTimeBucketed: {
          yearly: 0,
          monthly: 3,
          weekly: 4,
          daily: 7,
          hourly: 24,
          keepLastN: 0,
        },
      });
    } else {
      onChange({ policyKeepAll: true });
    }
  };

  const cronIsSubHourly = useMemo(
    () =>
      schedule?.schedule?.value &&
      !/^\d+ /.test(schedule.schedule.value) &&
      schedule.schedule.case === "cron",
    [schedule],
  );

  const updateRetentionField = (path: string[], val: any) => {
    const next = { ...retention };
    let curr = next;
    for (let i = 0; i < path.length - 1; i++) {
      curr[path[i]] = curr[path[i]] ? { ...curr[path[i]] } : {};
      curr = curr[path[i]];
    }
    curr[path[path.length - 1]] = val;
    onChange(next);
  };

  return (
    <Stack gap={4}>
      <Card.Root variant="subtle" width="fit-content">
        <Card.Header pb={0}>
          <Flex gap={2} wrap="wrap">
            {[
              {
                value: "policyKeepLastN",
                label: m.add_plan_modal_retention_policy_mode_count_label(),
                tooltip: m.add_plan_modal_retention_policy_keep_last_n_tooltip()
              },
              {
                value: "policyTimeBucketed",
                label: m.add_plan_modal_retention_policy_mode_time_label(),
                tooltip: m.add_plan_modal_retention_policy_time_bucketed_tooltip()
              },
              {
                value: "policyKeepAll",
                label: m.add_plan_modal_retention_policy_mode_none_label(),
                tooltip: m.add_plan_modal_retention_policy_keep_all_tooltip()
              },
            ].map((option) => (
              <Tooltip key={option.value} content={option.tooltip}>
                <Button
                  size="sm"
                  variant={mode === option.value ? "solid" : "outline"}
                  onClick={() => handleModeChange(option.value)}
                >
                  {option.label}
                </Button>
              </Tooltip>
            ))}
          </Flex>
        </Card.Header>

        <Card.Body>
          {mode === "policyKeepAll" && (
            <p>
              {m.add_plan_modal_retention_policy_keep_all_warning()}
            </p>
          )}

          {mode === "policyKeepLastN" && (
            <NumberInputField
              label={m.add_plan_modal_retention_policy_keep_last_n_snapshots_label()}
              value={retention?.policyKeepLastN || 0}
              onValueChange={(e: any) =>
                onChange({ ...schedule, policyKeepLastN: e.valueAsNumber })
              }
            />
          )}

          {mode === "policyTimeBucketed" && (
            <Grid templateColumns="repeat(3, 180px)" gap={4}>
              <NumberInputField
                label={m.add_plan_modal_retention_policy_hourly_label()}
                value={retention?.policyTimeBucketed?.hourly || 0}
                onValueChange={(e: any) =>
                  updateRetentionField(
                    ["policyTimeBucketed", "hourly"],
                    e.valueAsNumber,
                  )
                }
              />
              <NumberInputField
                label={m.add_plan_modal_retention_policy_daily_label()}
                value={retention?.policyTimeBucketed?.daily || 0}
                onValueChange={(e: any) =>
                  updateRetentionField(
                    ["policyTimeBucketed", "daily"],
                    e.valueAsNumber,
                  )
                }
              />
              <NumberInputField
                label={m.add_plan_modal_retention_policy_weekly_label()}
                value={retention?.policyTimeBucketed?.weekly || 0}
                onValueChange={(e: any) =>
                  updateRetentionField(
                    ["policyTimeBucketed", "weekly"],
                    e.valueAsNumber,
                  )
                }
              />
              <NumberInputField
                label={m.add_plan_modal_retention_policy_monthly_label()}
                value={retention?.policyTimeBucketed?.monthly || 0}
                onValueChange={(e: any) =>
                  updateRetentionField(
                    ["policyTimeBucketed", "monthly"],
                    e.valueAsNumber,
                  )
                }
              />
              <NumberInputField
                label={m.add_plan_modal_retention_policy_yearly_label()}
                value={retention?.policyTimeBucketed?.yearly || 0}
                onValueChange={(e: any) =>
                  updateRetentionField(
                    ["policyTimeBucketed", "yearly"],
                    e.valueAsNumber,
                  )
                }
              />
              <NumberInputField
                label={m.add_plan_modal_retention_policy_latest_label()}
                helperText={
                  cronIsSubHourly
                    ? "Keep recent snapshots (High-frequency schedule detected)"
                    : m.add_plan_modal_retention_policy_keep_regardless_label()
                }
                value={retention?.policyTimeBucketed?.keepLastN || 0}
                onValueChange={(e: any) =>
                  updateRetentionField(
                    ["policyTimeBucketed", "keepLastN"],
                    e.valueAsNumber,
                  )
                }
              />
            </Grid>
          )}
        </Card.Body>
      </Card.Root>
    </Stack>
  );
};
