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
} from "@chakra-ui/react";
import { Checkbox } from "../components/ui/checkbox";
import React, { useEffect, useState, useMemo } from "react";
import { useShowModal } from "../components/ModalManager";
import {
  ConfigSchema,
  PlanSchema,
  RetentionPolicySchema,
  Schedule_Clock,
  type Plan,
  type RetentionPolicy,
  type Schedule
} from "../../gen/ts/v1/config_pb";
import {
    FiPlus as Plus,
    FiMinus as Minus,
} from "react-icons/fi";
import { BsCalculator as Calculator } from "react-icons/bs";
import { formatErrorAlert, useAlertApi } from "../components/Alerts";
import { namePattern } from "../lib/formutil";
import { ConfirmButton } from "../components/SpinButton";
import { useConfig } from "../components/ConfigProvider";
import { backrestService } from "../api";
import { clone, create, equals, fromJson, toJson, JsonValue } from "@bufbuild/protobuf";
import { formatDuration } from "../lib/formatting";
import { getMinimumCronDuration } from "../lib/cronutil";
import * as m from "../paraglide/messages";
import { FormModal } from "../components/FormModal";
import { Button } from "../components/ui/button";
import { Field } from "../components/ui/field";
import { Tooltip } from "../components/ui/tooltip";
import { toaster } from "../components/ui/toaster";
import { NumberInputField } from "../components/NumberInput"; // Assuming I migrated this or will check
import { isWindows } from "../state/buildcfg";
import { ScheduleFormItem, ScheduleDefaultsDaily } from "../components/ScheduleFormItem";
// Use the real implementation
import { URIAutocomplete } from "../components/URIAutocomplete";

import { HooksFormList, hooksListTooltipText } from "../components/HooksFormList";
import { DynamicList } from "../components/DynamicList";

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
  const alertsApi = useAlertApi()!;
  const [config, setConfig] = useConfig();
  
  // Local State
  const [formData, setFormData] = useState<any>(
      template 
      ? toJson(PlanSchema, template, { alwaysEmitImplicit: true }) 
      : toJson(PlanSchema, planDefaults, { alwaysEmitImplicit: true })
  );

  // Helper to update fields
  const updateField = (path: string[], value: any) => {
      setFormData((prev: any) => {
          const next = { ...prev };
          let curr = next;
          for (let i = 0; i < path.length - 1; i++) {
              if (curr[path[i]] === undefined) curr[path[i]] = {};
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
      if (!template) throw new Error(m.add_plan_modal_error_template_not_found());

      const configCopy = clone(ConfigSchema, config);
      const idx = configCopy.plans.findIndex((r) => r.id === template.id);
      if (idx === -1) throw new Error(m.add_plan_modal_error_plan_delete_not_found());
      
      configCopy.plans.splice(idx, 1);
      setConfig(await backrestService.setConfig(configCopy));
      showModal(null);
      alertsApi.success(m.add_plan_modal_success_plan_deleted(), 30);
    } catch (e: any) {
      alertsApi.error(formatErrorAlert(e, m.add_plan_modal_error_destroy_prefix()), 15);
    } finally {
      setConfirmLoading(false);
    }
  };

  const handleOk = async () => {
    setConfirmLoading(true);
    try {
      // TODO: Validation (manual or schema based)
      const plan = fromJson(PlanSchema, formData, { ignoreUnknownFields: true });

      // Clean up retention if empty (logic from original)
      if (plan.retention && equals(RetentionPolicySchema, plan.retention, create(RetentionPolicySchema, {}))) {
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
      alertsApi.error(formatErrorAlert(e, m.add_plan_modal_error_operation_prefix()), 15);
    } finally {
      setConfirmLoading(false);
    }
  };

  const repos = config?.repos || [];
  const repoOptions = createListCollection({
      items: repos.map(r => ({ label: r.id, value: r.id }))
  });

  return (
    <FormModal
      isOpen={true}
      onClose={() => showModal(null)}
      title={template ? m.add_plan_modal_title_update() : m.add_plan_modal_title_add()}
      size="large"
      footer={
        <Flex gap={2} justify="flex-end" width="full">
            <Button variant="outline" disabled={confirmLoading} onClick={() => showModal(null)}>
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
          <a href="https://garethgeorge.github.io/backrest/introduction/getting-started" target="_blank" style={{textDecoration: 'underline'}}>
            {m.add_plan_modal_see_guide_link()}
          </a>{" "}
          {m.add_plan_modal_see_guide_suffix()}
        </p>

        {/* Plan ID */}
        <Field label={m.add_plan_modal_field_plan_name()} helperText={m.add_plan_modal_field_plan_name_tooltip()} required>
            <Input 
                value={getField(['id'])} 
                onChange={(e) => updateField(['id'], e.target.value)}
                disabled={!!template}
                placeholder={"plan" + ((config?.plans?.length || 0) + 1)}
            />
        </Field>

        {/* Repository */}
        <Field label={m.add_plan_modal_field_repository()} helperText={m.add_plan_modal_field_repository_tooltip()} required>
            <SelectRoot collection={repoOptions} size="sm"
                value={[getField(['repo'])]}
                onValueChange={(e: any) => updateField(['repo'], e.value[0])}
                disabled={!!template}
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

        {/* Paths */}
        <DynamicList 
            label={m.add_plan_modal_field_paths()}
            items={getField(['paths']) || []}
            onUpdate={(items: string[]) => updateField(['paths'], items)}
            tooltip={m.add_plan_modal_field_paths_tooltip()}
            placeholder="File Path (e.g. /home/user/data)"
            required
            autocompleteType="uri"
        />

        {/* Excludes */}
        <DynamicList 
            label={m.add_plan_modal_field_excludes()}
            items={getField(['excludes']) || []}
            onUpdate={(items: string[]) => updateField(['excludes'], items)}
            tooltip={m.add_plan_modal_field_excludes_tooltip_link()}
            placeholder="Exclude Pattern"
        />

        {/* IExcludes */}
        <DynamicList 
            label={m.add_plan_modal_field_iexcludes()}
            items={getField(['iexcludes']) || []}
            onUpdate={(items: string[]) => updateField(['iexcludes'], items)}
            tooltip="Case-insensitive excludes"
            placeholder="Case-insensitive Exclude Pattern"
        />

        {/* Schedule */}
        <Field label={m.add_plan_modal_field_schedule()}>
            <ScheduleFormItem 
                value={getField(['schedule'])} 
                onChange={(v: any) => updateField(['schedule'], v)}
                defaults={ScheduleDefaultsDaily}
            />
        </Field>

        {/* Backup Flags */}
        <DynamicList 
            label={m.add_plan_modal_field_backup_flags()}
            items={getField(['backup_flags']) || []}
            onUpdate={(items: string[]) => updateField(['backup_flags'], items)}
            tooltip={m.add_plan_modal_field_backup_flags_tooltip()}
            placeholder="--flag"
            autocompleteType="flag"
        />

        {/* Retention Policy */}
        <RetentionPolicyView 
            schedule={getField(['schedule'])}
            retention={getField(['retention'])}
            onChange={(v: any) => updateField(['retention'], v)}
        />

        {/* Hooks */}
        <Field label={m.add_plan_modal_field_hooks()} helperText={hooksListTooltipText} >
            <HooksFormList 
                value={getField(['hooks'])}
                onChange={(v: any) => updateField(['hooks'], v)}
            />
        </Field>
      </Stack>
    </FormModal>
  );
};

// --- Subcomponents ---


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
        () => schedule?.schedule?.value && !/^\d+ /.test(schedule.schedule.value) && schedule.schedule.case === "cron",
        [schedule]
    );

    // Helpers to update nested retention fields
    const updateRetentionField = (path: string[], val: any) => {
        const next = { ...retention };
        let curr = next;
        for (let i = 0; i < path.length - 1; i++) {
             if (!curr[path[i]]) curr[path[i]] = {};
             curr = curr[path[i]];
        }
        curr[path[path.length - 1]] = val;
        onChange(next);
    };

    return (
        <Field label="Retention Policy">
            <Stack gap={4}>
                {/* Mode Selector */}
                {/* Using a simple Button group or similar since RadioGroup might be complex to style perfectly immediately */}
                <Flex gap={2}>
                     {["policyKeepLastN", "policyTimeBucketed", "policyKeepAll"].map(m => (
                         <Button 
                            key={m} 
                            size="sm" 
                            variant={mode === m ? "solid" : "outline"} 
                            onClick={() => handleModeChange(m)}
                         >
                             {m === "policyKeepLastN" && "By Count"}
                             {m === "policyTimeBucketed" && "By Time Period"}
                             {m === "policyKeepAll" && "None"}
                         </Button>
                     ))}
                </Flex>

                {/* Mode Content */}
                <Card.Root variant="subtle">
                     <Card.Body>
                        {mode === "policyKeepAll" && (
                             <p>All backups are retained. Warning: this may result in slow backups if the repo grows very large.</p>
                        )}

                        {mode === "policyKeepLastN" && (
                             <NumberInputField 
                                label="Keep Last N Snapshots"
                                value={retention?.policyKeepLastN || 0}
                                onValueChange={(e: any) => onChange({ policyKeepLastN: e.valueAsNumber })}
                             />
                        )}

                        {mode === "policyTimeBucketed" && (
                            <Stack gap={2}>
                                <Flex gap={2}>
                                    <NumberInputField label="Hourly" value={retention?.policyTimeBucketed?.hourly || 0} onValueChange={(e: any) => updateRetentionField(['policyTimeBucketed', 'hourly'], e.valueAsNumber)} />
                                    <NumberInputField label="Daily" value={retention?.policyTimeBucketed?.daily || 0} onValueChange={(e: any) => updateRetentionField(['policyTimeBucketed', 'daily'], e.valueAsNumber)} />
                                </Flex>
                                <Flex gap={2}>
                                    <NumberInputField label="Weekly" value={retention?.policyTimeBucketed?.weekly || 0} onValueChange={(e: any) => updateRetentionField(['policyTimeBucketed', 'weekly'], e.valueAsNumber)} />
                                    <NumberInputField label="Monthly" value={retention?.policyTimeBucketed?.monthly || 0} onValueChange={(e: any) => updateRetentionField(['policyTimeBucketed', 'monthly'], e.valueAsNumber)} />
                                    <NumberInputField label="Yearly" value={retention?.policyTimeBucketed?.yearly || 0} onValueChange={(e: any) => updateRetentionField(['policyTimeBucketed', 'yearly'], e.valueAsNumber)} />
                                </Flex>
                                <Field 
                                    label="Latest snapshots to keep regardless of age"
                                    required={cronIsSubHourly}
                                    helperText={cronIsSubHourly ? "Schedule runs frequently; keep some recent snapshots." : undefined}
                                >
                                     <NumberInputField 
                                        value={retention?.policyTimeBucketed?.keepLastN || 0}
                                        onValueChange={(e: any) => updateRetentionField(['policyTimeBucketed', 'keepLastN'], e.valueAsNumber)}
                                     />
                                </Field>
                            </Stack>
                        )}
                     </Card.Body>
                </Card.Root>
            </Stack>
        </Field>
    );
};

