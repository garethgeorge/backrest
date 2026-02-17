import {
  Flex,
  Stack,
  Input,
  Textarea,
  createListCollection,
  SelectContent,
  SelectItem,
  SelectLabel,
  SelectRoot,
  SelectTrigger,
  SelectValueText,
  IconButton,
  Card,
  Box,
  HStack,
  Text as CText,
  Grid,
  Code,
} from "@chakra-ui/react";
import { Checkbox } from "../../components/ui/checkbox";
import {
  AccordionItem,
  AccordionItemContent,
  AccordionItemTrigger,
  AccordionRoot,
} from "../../components/ui/accordion";
import React, { useEffect, useState, useMemo } from "react";
import { useShowModal } from "../../components/common/ModalManager";
import {
  ConfigSchema,
  PlanSchema,
  RetentionPolicySchema,
  Schedule_Clock,
  type Plan,
  type RetentionPolicy,
  type Schedule,
} from "../../../gen/ts/v1/config_pb";
import { FiPlus as Plus, FiMinus as Minus, FiMenu } from "react-icons/fi";
import { BsCalculator as Calculator } from "react-icons/bs";
import { alerts, formatErrorAlert } from "../../components/common/Alerts";
import { namePattern } from "../../lib/util";
import { ConfirmButton } from "../../components/common/SpinButton";
import { useConfig } from "../../app/provider";
import { backrestService } from "../../api/client";
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
} from "@dnd-kit/core";
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
  useSortable,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import {
  clone,
  create,
  equals,
  fromJson,
  toJson,
  JsonValue,
} from "@bufbuild/protobuf";
import * as m from "../../paraglide/messages";
import { FormModal } from "../../components/common/FormModal";
import { Button } from "../../components/ui/button";
import { Field } from "../../components/ui/field";
import { Tooltip } from "../../components/ui/tooltip";
import { NumberInputField } from "../../components/common/NumberInput"; // Assuming I migrated this or will check
import {
  ScheduleFormItem,
  ScheduleDefaultsDaily,
} from "../../components/common/ScheduleFormItem";
// Use the real implementation
import { URIAutocomplete } from "../../components/common/URIAutocomplete";
import {
  HooksFormList,
  hooksListTooltipText,
} from "../../components/common/HooksFormList";
import { DynamicList } from "../../components/common/DynamicList";

// Default Plan
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
  const [config, setConfig] = useConfig();

  // Local State
  const [formData, setFormData] = useState<any>(
    template
      ? toJson(PlanSchema, template, { alwaysEmitImplicit: true })
      : toJson(PlanSchema, planDefaults, { alwaysEmitImplicit: true }),
  );

  // Sync state with template prop
  useEffect(() => {
    setFormData(
      template
        ? toJson(PlanSchema, template, { alwaysEmitImplicit: true })
        : toJson(PlanSchema, planDefaults, { alwaysEmitImplicit: true }),
    );
  }, [template]);

  // Helper to update fields
  const updateField = (path: string[], value: any) => {
    setFormData((prev: any) => {
      const next = { ...prev };
      let curr = next;
      for (let i = 0; i < path.length - 1; i++) {
        // Create shallow copy of the next level if it exists, or new object if not
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
      // Validation
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
      if (!formData.paths || formData.paths.length === 0) {
        throw new Error(m.add_plan_modal_validation_paths_required());
      }
      if (
        formData.backup_flags &&
        formData.backup_flags.some((f: string) => !/^\-\-?.*$/.test(f))
      ) {
        throw new Error(m.add_plan_modal_validation_flag_pattern());
      }

      // Check retention for sub-hourly schedules
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

      // Clean up retention if empty (logic from original)
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

  return (
    <FormModal
      isOpen={true}
      onClose={() => showModal(null)}
      title={
        template
          ? m.add_plan_modal_title_update()
          : m.add_plan_modal_title_add()
      }
      size="large"
      footer={
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
      }
    >
      <Stack gap={6}>
        {/* Info Link */}
        <p>
          {m.add_plan_modal_see_guide_prefix()}{" "}
          <a
            href="https://garethgeorge.github.io/backrest/introduction/getting-started"
            target="_blank"
            style={{ textDecoration: "underline" }}
          >
            {m.add_plan_modal_see_guide_link()}
          </a>{" "}
          {m.add_plan_modal_see_guide_suffix()}
        </p>

        {/* Plan Details */}
        <Section title="Plan Details">
          <Card.Root variant="subtle">
            <Card.Body>
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
                      <SelectValueText placeholder="Select repository" />
                    </SelectTrigger>
                    {/* @ts-ignore */}
                    <SelectContent>
                      {repoOptions.items.map((item: any) => (
                        // @ts-ignore
                        <SelectItem item={item} key={item.value}>
                          {item.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </SelectRoot>
                </Field>
              </Stack>
            </Card.Body>
          </Card.Root>
        </Section>

        {/* Scope */}
        <Section title="Backup Scope">
          <Card.Root variant="subtle">
            <Card.Body>
              <Stack gap={4}>
                <DynamicList
                  label={m.add_plan_modal_field_paths()}
                  tooltip={"Paths to include in snapshots created by this plan"}
                  items={getField(["paths"]) || []}
                  onUpdate={(items: string[]) => updateField(["paths"], items)}
                  required
                  autocompleteType="uri"
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
                  placeholder="Exclude Pattern"
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
                  placeholder="Case-insensitive Exclude Pattern"
                />
              </Stack>
            </Card.Body>
          </Card.Root>
        </Section>

        {/* Schedule */}
        <Section title={m.add_plan_modal_field_schedule()}>
          <ScheduleFormItem
            value={getField(["schedule"])}
            onChange={(v: any) => updateField(["schedule"], v)}
            defaults={ScheduleDefaultsDaily}
          />
        </Section>

        {/* Retention Policy */}
        <Section title="Retention Policy">
          <RetentionPolicyView
            schedule={getField(["schedule"])}
            retention={getField(["retention"])}
            onChange={(v: any) => updateField(["retention"], v)}
          />
        </Section>

        {/* Advanced */}
        <Section title="Advanced">
          <Card.Root variant="subtle">
            <Card.Body>
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
            </Card.Body>
          </Card.Root>
        </Section>

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
      </Stack>
    </FormModal>
  );
};

const Section = ({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) => (
  <Stack gap={2}>
    <CText fontWeight="semibold" fontSize="sm">
      {title}
    </CText>
    {children}
  </Stack>
);

// Retention View
const RetentionPolicyView = ({ schedule, retention, onChange }: any) => {
  // Mode determination
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

  // Derived values
  const cronIsSubHourly = useMemo(
    () =>
      schedule?.schedule?.value &&
      !/^\d+ /.test(schedule.schedule.value) &&
      schedule.schedule.case === "cron",
    [schedule],
  );

  // Helpers to update nested retention fields
  const updateRetentionField = (path: string[], val: any) => {
    const next = { ...retention };
    let curr = next;
    for (let i = 0; i < path.length - 1; i++) {
      // Create shallow copy of the next level to ensure immutability
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
          {/* Mode Selector */}
          <Flex gap={2} wrap="wrap">
            {[
              {
                value: "policyKeepLastN",
                label: "By Count",
                tooltip:
                  "The last N snapshots will be kept by restic. Retention policy is applied to drop older snapshots after each backup run.",
              },
              {
                value: "policyTimeBucketed",
                label: "By Time Period",
                tooltip:
                  "The last N snapshots for each time period will be kept by restic. Retention policy is applied to drop older snapshots after each backup run.",
              },
              {
                value: "policyKeepAll",
                label: "None",
                tooltip:
                  "All backups will be retained. Note that this may result in slow backups if the set of snapshots grows very large.",
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
          {/* Mode Content */}
          {mode === "policyKeepAll" && (
            <p>
              All backups are retained. Warning: this may result in slow backups
              if the repo grows very large.
            </p>
          )}

          {mode === "policyKeepLastN" && (
            <NumberInputField
              label="Keep Last N Snapshots"
              value={retention?.policyKeepLastN || 0}
              onValueChange={(e: any) =>
                onChange({ ...schedule, policyKeepLastN: e.valueAsNumber })
              }
            />
          )}

          {mode === "policyTimeBucketed" && (
            <Grid templateColumns="repeat(3, 180px)" gap={4}>
              <NumberInputField
                label="Hourly"
                value={retention?.policyTimeBucketed?.hourly || 0}
                onValueChange={(e: any) =>
                  updateRetentionField(
                    ["policyTimeBucketed", "hourly"],
                    e.valueAsNumber,
                  )
                }
              />
              <NumberInputField
                label="Daily"
                value={retention?.policyTimeBucketed?.daily || 0}
                onValueChange={(e: any) =>
                  updateRetentionField(
                    ["policyTimeBucketed", "daily"],
                    e.valueAsNumber,
                  )
                }
              />
              <NumberInputField
                label="Weekly"
                value={retention?.policyTimeBucketed?.weekly || 0}
                onValueChange={(e: any) =>
                  updateRetentionField(
                    ["policyTimeBucketed", "weekly"],
                    e.valueAsNumber,
                  )
                }
              />
              <NumberInputField
                label="Monthly"
                value={retention?.policyTimeBucketed?.monthly || 0}
                onValueChange={(e: any) =>
                  updateRetentionField(
                    ["policyTimeBucketed", "monthly"],
                    e.valueAsNumber,
                  )
                }
              />
              <NumberInputField
                label="Yearly"
                value={retention?.policyTimeBucketed?.yearly || 0}
                onValueChange={(e: any) =>
                  updateRetentionField(
                    ["policyTimeBucketed", "yearly"],
                    e.valueAsNumber,
                  )
                }
              />
              <NumberInputField
                label="Latest (Count)"
                helperText={
                  cronIsSubHourly
                    ? "Keep recent snapshots (High-frequency schedule detected)"
                    : "Latest snapshots to keep regardless of age"
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
