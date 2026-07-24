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
  Text as CText,
  Code,
} from "@chakra-ui/react";
import {
  AccordionItem,
  AccordionItemContent,
  AccordionItemTrigger,
  AccordionRoot,
} from "../../components/ui/accordion";
import { useEffect, useState } from "react";
import { useShowModal } from "../../components/common/ModalManager";
import {
  ConfigSchema,
  PlanSchema,
  RetentionPolicySchema,
  Schedule_Clock,
  type Plan,
} from "../../../gen/ts/v1/config_pb";
import {
  FiFileText,
  FiFolder,
  FiClock,
  FiArchive,
  FiSliders,
} from "react-icons/fi";
import { alerts, formatErrorAlert } from "../../components/common/Alerts";
import { namePattern } from "../../lib/util";
import { ConfirmButton } from "../../components/common/SpinButton";
import { useConfig } from "../../app/provider";
import { backrestService } from "../../api/client";
import { clone, create, equals, fromJson, toJson } from "@bufbuild/protobuf";
import * as m from "../../paraglide/messages";
import { Button } from "../../components/ui/button";
import { Field } from "../../components/ui/field";
import { RetentionPolicyView } from "../../components/common/RetentionPolicyView";
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

export const AddPlanModal = ({
  template,
  onSaveOverride,
}: {
  template: Plan | null;
  onSaveOverride?: (plan: Plan) => Promise<void>;
}) => {
  const [confirmLoading, setConfirmLoading] = useState(false);
  const showModal = useShowModal();
  const [config, setConfig] = useConfig();

  const [formData, setFormData] = useState<any>(
    template
      ? toJson(PlanSchema, template, { alwaysEmitImplicit: true })
      : toJson(PlanSchema, planDefaults, { alwaysEmitImplicit: true }),
  );

  const selectedRepo = config?.repos.find(
    (r) => r.id === (template?.repo || formData?.repo),
  );
  const repoHasScheduledForget =
    !!selectedRepo?.forgetPolicy?.schedule &&
    selectedRepo.forgetPolicy.schedule.schedule.case !== undefined &&
    selectedRepo.forgetPolicy.schedule.schedule.case !== "disabled";

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
        throw new Error(m.settings_auth_name_pattern());
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
          m.add_plan_modal_your_schedule_runs_more_than_once_per_hour_please_specify_a({ count: 1 }),
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
        if (idx === -1) throw new Error(m.add_plan_modal_failed_to_update_plan_not_found());
        configCopy.plans[idx] = plan;
      } else {
        configCopy.plans.push(plan);
      }

      setConfig(await backrestService.setConfig(configCopy));
      showModal(null);
    } catch (e: any) {
      alerts.error(
        formatErrorAlert(e, m.settings_error_operation()),
      );
    } finally {
      setConfirmLoading(false);
    }
  };

  const allRepos = config?.repos || [];
  const localRepos = allRepos.filter((r) => !r.originInstanceId);
  const remoteRepos = allRepos.filter((r) => !!r.originInstanceId);
  const repoOptions = createListCollection({
    items: [
      ...localRepos.map((r) => ({ label: r.id, value: r.id })),
      ...remoteRepos.map((r) => ({
        label: `${r.id} (from ${r.originInstanceId})`,
        value: r.id,
      })),
    ],
  });

  const sections: SectionDef[] = [
    { id: "details", label: m.op_row_details(), icon: <FiFileText size={14} /> },
    { id: "scope", label: m.add_plan_modal_scope(), icon: <FiFolder size={14} /> },
    { id: "schedule", label: m.add_plan_modal_schedule(), icon: <FiClock size={14} /> },
    { id: "retention", label: m.add_plan_modal_retention(), icon: <FiArchive size={14} /> },
    { id: "advanced", label: m.add_plan_modal_advanced(), icon: <FiSliders size={14} /> },
  ];

  const footer = (
    <Flex gap={2} justify="flex-end" width="full">
      <Button
        variant="outline"
        disabled={confirmLoading}
        onClick={() => showModal(null)}
      >
        {m.button_cancel()}
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
      <Button
        loading={confirmLoading}
        onClick={handleOk}
        data-testid="add-plan-submit"
      >
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
          : m.app_menu_add_plan()
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
          description={m.add_plan_modal_plan_name_and_target_repository()}
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
                  ? m.settings_auth_name_pattern()
                  : m.add_plan_modal_validation_plan_exists()
              }
            >
              <Input
                data-testid="add-plan-name"
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
                onValueChange={(e: any) => updateField(["repo"], e.value[0])}
                disabled={!!template}
                width="full"
              >
                {/* @ts-ignore */}
                <SelectTrigger data-testid="add-plan-repo-select">
                  {/* @ts-ignore */}
                  <SelectValueText
                    placeholder={m.add_plan_modal_field_repository_select()}
                  />
                </SelectTrigger>
                {/* @ts-ignore */}
                <SelectContent>
                  {localRepos.length > 0 && remoteRepos.length > 0 && (
                    <CText
                      fontSize="xs"
                      fontWeight="bold"
                      color="fg.muted"
                      px={2}
                      py={1}
                    >
                      {m.add_plan_modal_local()}
                    </CText>
                  )}
                  {repoOptions.items
                    .slice(0, localRepos.length)
                    .map((item: any) => (
                      <SelectItem item={item} key={item.value}>
                        {item.label}
                      </SelectItem>
                    ))}
                  {remoteRepos.length > 0 && (
                    <CText
                      fontSize="xs"
                      fontWeight="bold"
                      color="fg.muted"
                      px={2}
                      py={1}
                      mt={1}
                      borderTopWidth="1px"
                      borderColor="border"
                    >
                      {m.app_remote()}
                    </CText>
                  )}
                  {repoOptions.items
                    .slice(localRepos.length)
                    .map((item: any) => (
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
          description={m.add_plan_modal_directories_and_exclusion_patterns()}
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
              testId="add-plan-path"
            />

            <DynamicList
              label={m.add_plan_modal_field_excludes()}
              items={getField(["excludes"]) || []}
              onUpdate={(items: string[]) => updateField(["excludes"], items)}
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
                  {m.add_repo_modal_field_uri_tooltip_info()}
                </>
              }
              placeholder={m.add_plan_modal_field_excludes()}
            />

            <DynamicList
              label={m.add_plan_modal_field_iexcludes()}
              items={getField(["iexcludes"]) || []}
              onUpdate={(items: string[]) => updateField(["iexcludes"], items)}
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
                  {m.add_repo_modal_field_uri_tooltip_info()}
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
          description={m.add_plan_modal_when_backups_run_automatically()}
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
          description={m.add_plan_modal_how_long_to_keep_snapshots_before_forgetting_them()}
        >
          {repoHasScheduledForget ? (
            <CText color="fg.muted" fontStyle="italic">
              {m.add_plan_modal_retention_managed_by_repo()}
            </CText>
          ) : (
            <RetentionPolicyView
              schedule={getField(["schedule"])}
              retention={getField(["retention"])}
              onChange={(v: any) => updateField(["retention"], v)}
            />
          )}
        </SectionCard>
      </TwoPaneSection>

      {/* Advanced Section */}
      <TwoPaneSection id="advanced">
        <SectionCard
          icon={<FiSliders size={16} />}
          title={m.add_plan_modal_advanced()}
          description={m.add_plan_modal_extra_flags_and_notification_hooks()}
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
              label={m.add_repo_modal_hooks()}
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
