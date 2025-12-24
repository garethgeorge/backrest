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
} from "@chakra-ui/react";
import { Checkbox } from "../../components/ui/checkbox";
import React, { useEffect, useState } from "react";
import { useShowModal } from "../../components/common/ModalManager";
// removed ../state/repo_state import
// valid imports
import {
  CommandPrefix_CPUNiceLevel,
  CommandPrefix_CPUNiceLevelSchema,
  CommandPrefix_IONiceLevel,
  CommandPrefix_IONiceLevelSchema,
  Repo,
  RepoSchema,
  Schedule_Clock,
} from "../../../gen/ts/v1/config_pb";
import { StringValueSchema } from "../../../gen/ts/types/value_pb";
import { URIAutocomplete } from "../../components/common/URIAutocomplete";
import { alerts, formatErrorAlert } from "../../components/common/Alerts";
import { namePattern } from "../../lib/util";
import { backrestService } from "../../api/client";
// import { HooksFormList } from "../components/HooksFormList"; // TODO: Migrate
import { ConfirmButton } from "../../components/common/SpinButton";
import { useConfig } from "../../app/provider";
import {
  ScheduleFormItem,
  ScheduleDefaultsDaily,
} from "../../components/common/ScheduleFormItem";
import { isWindows } from "../../state/buildcfg";
import { create, fromJson, toJson, JsonValue } from "@bufbuild/protobuf";
import * as m from "../../paraglide/messages";
import { FormModal } from "../../components/common/FormModal";
import { Button } from "../../components/ui/button";
import { Field } from "../../components/ui/field";
import { PasswordInput } from "../../components/ui/password-input";
import { Tooltip } from "../../components/ui/tooltip";
import { toaster } from "../../components/ui/toaster";
import { NumberInputField } from "../../components/common/NumberInput";

import {
  HooksFormList,
  hooksListTooltipText,
} from "../../components/common/HooksFormList";

const repoDefaults = create(RepoSchema, {
  prunePolicy: {
    maxUnusedPercent: 10,
    schedule: {
      schedule: {
        case: "cron",
        value: "0 0 1 * *",
      },
      clock: Schedule_Clock.LAST_RUN_TIME,
    },
  },
  checkPolicy: {
    schedule: {
      schedule: {
        case: "cron",
        value: "0 0 1 * *",
      },
      clock: Schedule_Clock.LAST_RUN_TIME,
    },
  },
  commandPrefix: {
    ioNice: CommandPrefix_IONiceLevel.IO_DEFAULT,
    cpuNice: CommandPrefix_CPUNiceLevel.CPU_DEFAULT,
  },
});

export const AddRepoModal = ({ template }: { template: Repo | null }) => {
  const [confirmLoading, setConfirmLoading] = useState(false);
  const showModal = useShowModal();
  const [config, setConfig] = useConfig();

  // Local state for form fields
  // Using a simple object state to mimic the form values
  const [formData, setFormData] = useState<any>(
    template
      ? toJson(RepoSchema, template, { alwaysEmitImplicit: true })
      : toJson(RepoSchema, repoDefaults, { alwaysEmitImplicit: true }),
  );

  const updateField = (path: string[], value: any) => {
    setFormData((prev: any) => {
      const next = { ...prev };
      let curr = next;
      for (let i = 0; i < path.length - 1; i++) {
        if (!curr[path[i]]) curr[path[i]] = {};
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
      setConfig(
        await backrestService.removeRepo(
          create(StringValueSchema, { value: template!.id }),
        ),
      );
      showModal(null);
      alerts.success(
        m.add_repo_modal_success_deleted({
          id: template!.id!,
          uri: template!.uri,
        }),
      );
    } catch (e: any) {
      alerts.error(
        formatErrorAlert(e, m.add_plan_modal_error_destroy_prefix()),
        15,
      );
    } finally {
      setConfirmLoading(false);
    }
  };

  const handleOk = async () => {
    setConfirmLoading(true);
    try {
      // Validation logic would go here
      const repo = fromJson(RepoSchema, formData, {
        ignoreUnknownFields: true,
      });

      if (template !== null) {
        setConfig(await backrestService.addRepo(repo));
        showModal(null);
        alerts.success(m.add_repo_modal_success_updated({ uri: repo.uri }));
      } else {
        setConfig(await backrestService.addRepo(repo));
        showModal(null);
        alerts.success(m.add_repo_modal_success_added({ uri: repo.uri }));
      }

      // Trigger generic "check" or "list snapshots" to verify
      try {
        await backrestService.listSnapshots({ repoId: repo.id });
      } catch (e: any) {
        alerts.error(
          formatErrorAlert(e, m.add_repo_modal_error_list_snapshots()),
          10,
        );
      }
    } catch (e: any) {
      alerts.error(
        formatErrorAlert(e, m.add_plan_modal_error_operation_prefix()),
        10,
      );
    } finally {
      setConfirmLoading(false);
    }
  };

  const ioNiceOptions = createListCollection({
    items: CommandPrefix_IONiceLevelSchema.values.map((v) => ({
      label: v.name,
      value: v.name,
    })),
  });

  const cpuNiceOptions = createListCollection({
    items: CommandPrefix_CPUNiceLevelSchema.values.map((v) => ({
      label: v.name,
      value: v.name,
    })),
  });

  return (
    <FormModal
      isOpen={true}
      onClose={() => showModal(null)}
      title={
        template ? m.add_repo_modal_title_edit() : m.add_repo_modal_title_add()
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
              colorPalette="red"
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
        <Field
          label={m.add_repo_modal_field_repo_name()}
          helperText={m.add_repo_modal_field_repo_name_tooltip()}
          required
        >
          <Input
            value={getField(["id"])}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              updateField(["id"], e.target.value)
            }
            disabled={!!template}
            placeholder={"repo" + ((config?.repos?.length || 0) + 1)}
          />
        </Field>

        <Field
          label={m.add_repo_modal_field_uri()}
          helperText="Supports local paths, sftp, s3, etc."
          required
        >
          <URIAutocomplete
            disabled={!!template}
            value={getField(["uri"])}
            onChange={(val: string) => updateField(["uri"], val)}
          />
        </Field>

        <Field
          label={m.add_repo_modal_field_password()}
          helperText={m.add_repo_modal_field_password_tooltip_intro()}
        >
          <Flex gap={2}>
            <PasswordInput
              value={getField(["password"])}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                updateField(["password"], e.target.value)
              }
              disabled={!!template}
              flex={1}
            />
            {!template && (
              <Button
                variant="ghost"
                onClick={() =>
                  updateField(["password"], cryptoRandomPassword())
                }
              >
                {m.add_repo_modal_button_generate()}
              </Button>
            )}
          </Flex>
        </Field>

        {/* Env Vars Placeholder */}
        <Field label={m.add_repo_modal_field_env_vars()}>
          <Textarea
            placeholder="KEY=VALUE (One per line)"
            value={getField(["env"]) || ""}
            onChange={(e) => updateField(["env"], e.target.value)}
          />
        </Field>

        <Field label={m.add_repo_modal_field_auto_unlock()}>
          <Checkbox
            checked={getField(["autoUnlock"])}
            onCheckedChange={(e: { checked: boolean | "indeterminate" }) =>
              updateField(["autoUnlock"], !!e.checked)
            }
          >
            {m.add_repo_modal_field_auto_unlock()}
          </Checkbox>
        </Field>

        <Field label={m.add_repo_modal_field_prune_policy()}>
          <Stack gap={4} p={4} borderWidth="1px" borderRadius="md">
            <NumberInputField
              label={m.add_repo_modal_field_max_unused()}
              value={getField(["prunePolicy", "maxUnusedPercent"])}
              onValueChange={(e: { value: string; valueAsNumber: number }) =>
                updateField(
                  ["prunePolicy", "maxUnusedPercent"],
                  e.valueAsNumber,
                )
              }
            />
            <Field label="Prune Schedule">
              <ScheduleFormItem
                value={getField(["prunePolicy", "schedule"])}
                onChange={(val: any) =>
                  updateField(["prunePolicy", "schedule"], val)
                }
                defaults={ScheduleDefaultsDaily}
              />
            </Field>
          </Stack>
        </Field>

        <Field label="Backend Check Policy">
          <Stack gap={4} p={4} borderWidth="1px" borderRadius="md">
            <Field label="Check Schedule">
              <ScheduleFormItem
                value={getField(["checkPolicy", "schedule"])}
                onChange={(val: any) =>
                  updateField(["checkPolicy", "schedule"], val)
                }
                defaults={ScheduleDefaultsDaily}
              />
            </Field>
          </Stack>
        </Field>

        {!isWindows && (
          <Flex gap={4}>
            <Field label={m.add_repo_modal_field_io_priority()} flex={1}>
              <SelectRoot
                collection={ioNiceOptions}
                size="sm"
                value={[getField(["commandPrefix", "ioNice"])]}
                onValueChange={(e: any) =>
                  updateField(["commandPrefix", "ioNice"], e.value[0])
                }
              >
                {/* @ts-ignore */}
                <SelectTrigger>
                  {/* @ts-ignore */}
                  <SelectValueText placeholder="Select priority" />
                </SelectTrigger>
                {/* @ts-ignore */}
                <SelectContent>
                  {ioNiceOptions.items.map((item: any) => (
                    // @ts-ignore
                    <SelectItem item={item} key={item.value}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </SelectRoot>
            </Field>
            <Field label={m.add_repo_modal_field_cpu_priority()} flex={1}>
              <SelectRoot
                collection={cpuNiceOptions}
                size="sm"
                value={[getField(["commandPrefix", "cpuNice"])]}
                onValueChange={(e: any) =>
                  updateField(["commandPrefix", "cpuNice"], e.value[0])
                }
              >
                {/* @ts-ignore */}
                <SelectTrigger>
                  {/* @ts-ignore */}
                  <SelectValueText placeholder="Select priority" />
                </SelectTrigger>
                {/* @ts-ignore */}
                <SelectContent>
                  {cpuNiceOptions.items.map((item: any) => (
                    // @ts-ignore
                    <SelectItem item={item} key={item.value}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </SelectRoot>
            </Field>
          </Flex>
        )}
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
    </FormModal>
  );
};

// Utils
const cryptoRandomPassword = (): string => {
  let vals = crypto.getRandomValues(new Uint8Array(64));
  return btoa(String.fromCharCode(...vals)).slice(0, 48);
};
