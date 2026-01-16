import {
  Stack,
  Flex,
  Input,
  Card,
  Text as CText,
  Grid,
  Code,
  Box,
} from "@chakra-ui/react";
import { EnumSelector, EnumOption } from "../../components/common/EnumSelector";
import { Checkbox } from "../../components/ui/checkbox";
import {
  AccordionItem,
  AccordionItemContent,
  AccordionItemTrigger,
  AccordionRoot,
} from "../../components/ui/accordion";
import React, { useEffect, useState } from "react";
import { useShowModal } from "../../components/common/ModalManager";
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
import { ConfirmButton, SpinButton } from "../../components/common/SpinButton";
import { useConfig } from "../../app/provider";
import {
  ScheduleFormItem,
  ScheduleDefaultsInfrequent,
} from "../../components/common/ScheduleFormItem";
import { isWindows } from "../../state/buildcfg";
import { create, fromJson, toJson, JsonValue } from "@bufbuild/protobuf";
import * as m from "../../paraglide/messages";
import { FormModal } from "../../components/common/FormModal";
import { Button } from "../../components/ui/button";
import { Field } from "../../components/ui/field";
import { PasswordInput } from "../../components/ui/password-input";
import { Tooltip } from "../../components/ui/tooltip";
import { NumberInputField } from "../../components/common/NumberInput";
import {
  HooksFormList,
  hooksListTooltipText,
} from "../../components/common/HooksFormList";
import { DynamicList } from "../../components/common/DynamicList";

const repoDefaults = create(RepoSchema, {
  prunePolicy: {
    maxUnusedPercent: 10,
    schedule: {
      schedule: {
        case: "cron",
        value: "0 0 1 * *", // 1st of the month
      },
      clock: Schedule_Clock.LAST_RUN_TIME,
    },
  },
  checkPolicy: {
    schedule: {
      schedule: {
        case: "cron",
        value: "0 0 1 * *", // 1st of the month
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
  const [formData, setFormData] = useState<any>(
    template
      ? toJson(RepoSchema, template, { alwaysEmitImplicit: true })
      : toJson(RepoSchema, repoDefaults, { alwaysEmitImplicit: true }),
  );

  useEffect(() => {
    setFormData(
      template
        ? toJson(RepoSchema, template, { alwaysEmitImplicit: true })
        : toJson(RepoSchema, repoDefaults, { alwaysEmitImplicit: true }),
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

  const validateLocal = async () => {
    const id = getField(["id"]);
    if (!id?.trim()) {
      throw new Error(m.add_repo_modal_error_repo_name_required());
    }
    if (!namePattern.test(id)) {
      throw new Error(m.add_plan_modal_validation_plan_name_pattern());
    }
    if (!template && config.repos.find((r) => r.id === id)) {
      throw new Error(m.add_repo_modal_error_repo_exists());
    }

    const uri = getField(["uri"]);
    if (!uri?.trim()) {
      throw new Error(m.add_repo_modal_error_uri_required());
    }

    // Env and Password validation
    await envVarSetValidator(formData);

    // Flags validation
    const flags = getField(["flags"]);
    if (flags && flags.some((f: string) => !/^\-\-?.*$/.test(f))) {
      throw new Error(m.add_repo_modal_error_flag_format());
    }
  };

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
      );
    } finally {
      setConfirmLoading(false);
    }
  };

  const handleOk = async () => {
    setConfirmLoading(true);
    try {
      await validateLocal();

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

      try {
        await backrestService.listSnapshots({ repoId: repo.id });
      } catch (e: any) {
        alerts.error(
          formatErrorAlert(e, m.add_repo_modal_error_list_snapshots()),
        );
        return;
      }
    } catch (e: any) {
      alerts.error(
        formatErrorAlert(e, m.add_plan_modal_error_operation_prefix()),
      );
      return;
    } finally {
      setConfirmLoading(false);
    }
  };

  const handleTest = async () => {
    try {
      await validateLocal();
      const repo = fromJson(RepoSchema, formData, {
        ignoreUnknownFields: true,
      });
      const exists = await backrestService.checkRepoExists(repo);
      if (exists.value) {
        alerts.success(
          m.add_repo_modal_test_success_existing({ uri: repo.uri }),
        );
      } else {
        alerts.success(m.add_repo_modal_test_success_new({ uri: repo.uri }));
      }
    } catch (e: any) {
      alerts.error(formatErrorAlert(e, m.add_repo_modal_test_error()));
    }
  };

  const ioNiceOptions: EnumOption<string>[] = [
    {
      label: "IO_BEST_EFFORT_LOW",
      value: "IO_BEST_EFFORT_LOW",
      description: m.add_repo_modal_field_io_priority_low(),
    },
    {
      label: "IO_BEST_EFFORT_HIGH",
      value: "IO_BEST_EFFORT_HIGH",
      description: m.add_repo_modal_field_io_priority_high(),
    },
    {
      label: "IO_IDLE",
      value: "IO_IDLE",
      description: m.add_repo_modal_field_io_priority_idle(),
    },
    {
      label: "IO_DEFAULT",
      value: "IO_DEFAULT",
      description: "Default system priority",
    },
  ];

  const cpuNiceOptions: EnumOption<string>[] = [
    {
      label: "CPU_DEFAULT",
      value: "CPU_DEFAULT",
      description: m.add_repo_modal_field_cpu_priority_default(),
    },
    {
      label: "CPU_HIGH",
      value: "CPU_HIGH",
      description: m.add_repo_modal_field_cpu_priority_high(),
    },
    {
      label: "CPU_LOW",
      value: "CPU_LOW",
      description: m.add_repo_modal_field_cpu_priority_low(),
    },
  ];

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
              danger
              onClickAsync={handleDestroy}
              confirmTitle={m.add_plan_modal_button_confirm_delete()}
            >
              {m.add_plan_modal_button_delete()}
            </ConfirmButton>
          )}
          <SpinButton onClickAsync={handleTest}>
            {m.add_repo_modal_test_config()}
          </SpinButton>
          <Button loading={confirmLoading} onClick={handleOk}>
            {m.add_plan_modal_button_submit()}
          </Button>
        </Flex>
      }
    >
      <Stack gap={6}>
        <p>
          {m.add_repo_modal_guide_text_p1()}{" "}
          <a
            href="https://garethgeorge.github.io/backrest/introduction/getting-started"
            target="_blank"
            style={{ textDecoration: "underline" }}
          >
            {m.add_repo_modal_guide_link_text()}
          </a>{" "}
          {m.add_repo_modal_guide_text_p2()}{" "}
          <a
            href="https://restic.readthedocs.io/"
            target="_blank"
            style={{ textDecoration: "underline" }}
          >
            {m.add_repo_modal_guide_restic_link_text()}
          </a>{" "}
          {m.add_repo_modal_guide_text_p3()}
        </p>

        <Section title="Repo Details">
          <Card.Root variant="subtle">
            <Card.Body>
              <Stack gap={4}>
                <Field
                  label={m.add_repo_modal_field_repo_name()}
                  helperText={
                    !template
                      ? m.add_repo_modal_field_repo_name_tooltip()
                      : undefined
                  }
                  required
                  invalid={
                    !!getField(["id"]) &&
                    (!namePattern.test(getField(["id"])) ||
                      (!template &&
                        !!config.repos.find((r) => r.id === getField(["id"]))))
                  }
                  errorText={
                    !!getField(["id"]) && !namePattern.test(getField(["id"]))
                      ? m.add_plan_modal_validation_plan_name_pattern()
                      : m.add_repo_modal_error_repo_exists()
                  }
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
                  helperText={
                    <>
                      {m.add_repo_modal_field_uri_tooltip_title()}
                      <Box as="ul" ml={4} mt={1}>
                        <li>{m.add_repo_modal_field_uri_tooltip_local()}</li>
                        <li>{m.add_repo_modal_field_uri_tooltip_s3()}</li>
                        <li>{m.add_repo_modal_field_uri_tooltip_sftp()}</li>
                        <li>
                          {m.add_repo_modal_field_uri_tooltip_see()}{" "}
                          <a
                            href="https://restic.readthedocs.io/en/latest/030_preparing_a_new_repo.html#preparing-a-new-repository"
                            target="_blank"
                            style={{ textDecoration: "underline" }}
                          >
                            {m.add_repo_modal_field_uri_tooltip_restic_docs()}
                          </a>{" "}
                          {m.add_repo_modal_field_uri_tooltip_info()}
                        </li>
                      </Box>
                    </>
                  }
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
                  helperText={
                    !template ? (
                      <>
                        {m.add_repo_modal_field_password_tooltip_intro()}
                        <Box as="ul" ml={4} mt={1}>
                          <li>
                            {m.add_repo_modal_field_password_tooltip_entropy()}
                          </li>
                          <li>
                            {m.add_repo_modal_field_password_tooltip_env()}
                          </li>
                          <li>
                            {m.add_repo_modal_field_password_tooltip_generate()}
                          </li>
                        </Box>
                      </>
                    ) : undefined
                  }
                >
                  <Flex gap={2} width="full">
                    <Box flex={1}>
                      <PasswordInput
                        value={getField(["password"])}
                        onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                          updateField(["password"], e.target.value)
                        }
                        disabled={!!template}
                      />
                    </Box>
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

                <Field label={m.add_repo_modal_field_auto_unlock()}>
                  <Checkbox
                    checked={getField(["autoUnlock"])}
                    onCheckedChange={(e: {
                      checked: boolean | "indeterminate";
                    }) => updateField(["autoUnlock"], !!e.checked)}
                  >
                    {m.add_repo_modal_field_auto_unlock()}
                  </Checkbox>
                  <CText fontSize="xs" color="fg.muted" ml={6}>
                    {m.add_repo_modal_field_auto_unlock_tooltip()}
                  </CText>
                </Field>
              </Stack>
            </Card.Body>
          </Card.Root>
        </Section>

        <Section title="Environment & Flags">
          <Card.Root variant="subtle">
            <Card.Body>
              <Stack gap={4}>
                <DynamicList
                  label={m.add_repo_modal_field_env_vars()}
                  items={getField(["env"]) || []}
                  onUpdate={(items: string[]) => updateField(["env"], items)}
                  tooltip={
                    <Stack gap={2}>
                      <CText>
                        {m.add_repo_modal_field_env_vars_tooltip()}
                      </CText>
                      <EnvVarTooltip uri={getField(["uri"])} />
                    </Stack>
                  }
                  placeholder="KEY=VALUE"
                />

                <DynamicList
                  label={m.add_repo_modal_field_flags()}
                  items={getField(["flags"]) || []}
                  onUpdate={(items: string[]) => updateField(["flags"], items)}
                  placeholder="--flag"
                />
              </Stack>
            </Card.Body>
          </Card.Root>
        </Section>

        <Section
          title={
            <Tooltip
              content={
                <span>
                  {m.add_repo_modal_field_prune_policy_tooltip_p1()}{" "}
                  <a
                    href="https://restic.readthedocs.io/en/stable/060_forget.html#customize-pruning"
                    target="_blank"
                    style={{ textDecoration: "underline" }}
                  >
                    {m.add_repo_modal_field_prune_policy_tooltip_link()}
                  </a>{" "}
                  {m.add_repo_modal_field_prune_policy_tooltip_p2()}
                </span>
              }
            >
              {m.add_repo_modal_field_prune_policy()}
            </Tooltip>
          }
        >
          <Card.Root variant="subtle" size="sm">
            <Card.Body>
              <Stack gap={4}>
                <NumberInputField
                  label={m.add_repo_modal_field_max_unused()}
                  helperText={m.add_repo_modal_field_max_unused_tooltip()}
                  value={getField(["prunePolicy", "maxUnusedPercent"])}
                  onValueChange={(e: {
                    value: string;
                    valueAsNumber: number;
                  }) =>
                    updateField(
                      ["prunePolicy", "maxUnusedPercent"],
                      e.valueAsNumber,
                    )
                  }
                />
                <ScheduleFormItem
                  value={getField(["prunePolicy", "schedule"])}
                  onChange={(val: any) =>
                    updateField(["prunePolicy", "schedule"], val)
                  }
                  defaults={ScheduleDefaultsInfrequent}
                />
              </Stack>
            </Card.Body>
          </Card.Root>
        </Section>

        <Section
          title={
            <Tooltip content={m.add_repo_modal_field_check_policy_tooltip()}>
              {m.add_repo_modal_field_check_policy()}
            </Tooltip>
          }
        >
          <Card.Root variant="subtle" size="sm">
            <Card.Body>
              <Stack gap={4}>
                <NumberInputField
                  label={m.add_repo_modal_field_read_data()}
                  helperText={m.add_repo_modal_field_read_data_tooltip()}
                  value={getField(["checkPolicy", "readDataSubsetPercent"])}
                  onValueChange={(e: {
                    value: string;
                    valueAsNumber: number;
                  }) =>
                    updateField(
                      ["checkPolicy", "readDataSubsetPercent"],
                      e.valueAsNumber,
                    )
                  }
                />
                <ScheduleFormItem
                  value={getField(["checkPolicy", "schedule"])}
                  onChange={(val: any) =>
                    updateField(["checkPolicy", "schedule"], val)
                  }
                  defaults={ScheduleDefaultsInfrequent}
                />
              </Stack>
            </Card.Body>
          </Card.Root>
        </Section>

        <Section title="Advanced">
          <Card.Root variant="subtle">
            <Card.Body>
              <Stack gap={4} width="full">
                {!isWindows && (
                  <Field label={m.add_repo_modal_field_command_modifiers()}>
                    <Grid templateColumns="1fr 1fr" gap={4} width="full">
                      <Field
                        label={m.add_repo_modal_field_io_priority()}
                        helperText={m.add_repo_modal_field_io_priority_tooltip_intro()}
                      >
                        <EnumSelector
                          options={ioNiceOptions}
                          size="sm"
                          value={getField(["commandPrefix", "ioNice"])}
                          onChange={(val) =>
                            updateField(
                              ["commandPrefix", "ioNice"],
                              val as string,
                            )
                          }
                          placeholder={m.add_repo_modal_field_io_priority_placeholder()}
                        />
                      </Field>
                      <Field
                        label={m.add_repo_modal_field_cpu_priority()}
                        helperText={m.add_repo_modal_field_cpu_priority_tooltip_intro()}
                      >
                        <EnumSelector
                          options={cpuNiceOptions}
                          size="sm"
                          value={getField(["commandPrefix", "cpuNice"])}
                          onChange={(val) =>
                            updateField(
                              ["commandPrefix", "cpuNice"],
                              val as string,
                            )
                          }
                          placeholder={m.add_repo_modal_field_cpu_priority_placeholder()}
                        />
                      </Field>
                    </Grid>
                  </Field>
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
  title: React.ReactNode;
  children: React.ReactNode;
}) => (
  <Stack gap={2}>
    <CText fontWeight="semibold" fontSize="sm">
      {title}
    </CText>
    {children}
  </Stack>
);

// Utils
const cryptoRandomPassword = (): string => {
  let vals = crypto.getRandomValues(new Uint8Array(64));
  return btoa(String.fromCharCode(...vals)).slice(0, 48);
};

// Validation Logic
const expectedEnvVars: { [scheme: string]: string[][] } = {
  s3: [
    ["AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"],
    ["AWS_SHARED_CREDENTIALS_FILE"],
  ],
  b2: [["B2_ACCOUNT_ID", "B2_ACCOUNT_KEY"]],
  azure: [
    ["AZURE_ACCOUNT_NAME", "AZURE_ACCOUNT_KEY"],
    ["AZURE_ACCOUNT_NAME", "AZURE_ACCOUNT_SAS"],
  ],
  gs: [
    ["GOOGLE_APPLICATION_CREDENTIALS", "GOOGLE_PROJECT_ID"],
    ["GOOGLE_ACCESS_TOKEN"],
  ],
};

const envVarSetValidator = async (formData: any) => {
  const envVars = formData.env || [];
  const uri = formData.uri;

  if (!uri) {
    return;
  }

  const envVarNames = envVars.map((e: string) => {
    if (!e) return "";
    let idx = e.indexOf("=");
    if (idx === -1) return "";
    return e.substring(0, idx);
  });

  const password = formData.password;
  if (
    (!password || password.length === 0) &&
    !envVarNames.includes("RESTIC_PASSWORD") &&
    !envVarNames.includes("RESTIC_PASSWORD_COMMAND") &&
    !envVarNames.includes("RESTIC_PASSWORD_FILE")
  ) {
    throw new Error(m.add_repo_modal_error_missing_password());
  }

  let schemeIdx = uri.indexOf(":");
  if (schemeIdx === -1) {
    return;
  }

  let scheme = uri.substring(0, schemeIdx);
  await checkSchemeEnvVars(scheme, envVarNames);
};

const checkSchemeEnvVars = async (
  scheme: string,
  envVarNames: string[],
): Promise<void> => {
  let expected = expectedEnvVars[scheme];
  if (!expected) {
    return;
  }

  const missingVarsCollection: string[][] = [];

  for (let possibility of expected) {
    const missingVars = possibility.filter(
      (envVar) => !envVarNames.includes(envVar),
    );

    if (missingVars.length === 0) {
      return;
    }

    if (missingVars.length < possibility.length) {
      missingVarsCollection.push(missingVars);
    }
  }

  if (!missingVarsCollection.length) {
    missingVarsCollection.push(...expected);
  }

  throw new Error(
    "Missing env vars " +
      formatMissingEnvVars(missingVarsCollection) +
      " for scheme " +
      scheme,
  );
};

const formatMissingEnvVars = (partialMatches: string[][]): string => {
  return partialMatches
    .map((x) => {
      if (x.length > 1) {
        return `[ ${x.join(", ")} ]`;
      }
      return x[0];
    })
    .join(" or ");
};
const EnvVarTooltip = ({ uri }: { uri: string }) => {
  if (!uri) return null;
  const scheme = uri.split(":")[0];
  const expected = expectedEnvVars[scheme];
  if (!expected) return null;
  return (
    <Box mt={2} p={2} bg="bg.muted" borderRadius="md" borderWidth="1px">
      <CText fontWeight="bold" mb={1}>
        Recommended for {scheme}:
      </CText>
      <ul style={{ paddingLeft: "1.2em" }}>
        {expected.map((set, i) => (
          <li key={i}>
            {i > 0 && "or "}
            {set.join(" + ")}
          </li>
        ))}
      </ul>
    </Box>
  );
};
