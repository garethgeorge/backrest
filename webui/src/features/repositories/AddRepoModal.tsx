import {
  Stack,
  Flex,
  Input,
  Text as CText,
  Grid,
  Code,
  Box,
} from "@chakra-ui/react";
import { EnumSelector, EnumOption } from "../../components/common/EnumSelector";

import React, { useEffect, useRef, useState } from "react";
import { useShowModal } from "../../components/common/ModalManager";
import {
  CommandPrefix_CPUNiceLevel,
  CommandPrefix_IONiceLevel,
  Repo,
  RepoSchema,
  Schedule_Clock,
} from "../../../gen/ts/v1/config_pb";
import {
  AddRepoRequestSchema,
  CheckRepoExistsRequestSchema,
  RemoveRepoRequestSchema,
} from "../../../gen/ts/v1/service_pb";
import { URIAutocomplete } from "../../components/common/URIAutocomplete";
import { alerts, formatErrorAlert } from "../../components/common/Alerts";
import { namePattern } from "../../lib/util";
import { backrestService } from "../../api/client";
import { ConfirmButton } from "../../components/common/SpinButton";
import { useConfig } from "../../app/provider";
import {
  ScheduleFormItem,
  ScheduleDefaultsInfrequent,
  ScheduleDefaultsDaily,
} from "../../components/common/ScheduleFormItem";
import { isWindows } from "../../state/buildcfg";
import { create, fromJson, toJson } from "@bufbuild/protobuf";
import * as m from "../../paraglide/messages";
import { Button } from "../../components/ui/button";
import { Field } from "../../components/ui/field";
import { PasswordInput } from "../../components/ui/password-input";
import { NumberInputField } from "../../components/common/NumberInput";
import {
  HooksFormList,
  hooksListTooltipText,
} from "../../components/common/HooksFormList";
import { DynamicList } from "../../components/common/DynamicList";
import {
  DialogActionTrigger,
  DialogBody,
  DialogCloseTrigger,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogRoot,
  DialogTitle,
} from "../../components/ui/dialog";
import { FiTag, FiLink, FiClock, FiZap, FiSliders } from "react-icons/fi";
import {
  TwoPaneModal,
  TwoPaneSection,
  type SectionDef,
} from "../../components/common/TwoPaneModal";
import { SectionCard } from "../../components/common/SectionCard";
import { ToggleField } from "../../components/common/ToggleField";
import { RetentionPolicyView } from "../../components/common/RetentionPolicyView";
import {
  AccordionRoot,
  AccordionItem,
  AccordionItemTrigger,
  AccordionItemContent,
} from "../../components/ui/accordion";

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
  // ForgetPolicy defaults to a disabled schedule. Per-plan retention
  // policies handle the common case; the repo-level forget is opt-in.
  // We still seed a clock and a reasonable retention so the UI doesn't
  // start with empty/zero fields when the user enables it.
  forgetPolicy: {
    schedule: {
      schedule: {
        case: "disabled",
        value: true,
      },
      clock: Schedule_Clock.LAST_RUN_TIME,
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
  },
  commandPrefix: {
    ioNice: CommandPrefix_IONiceLevel.IO_DEFAULT,
    cpuNice: CommandPrefix_CPUNiceLevel.CPU_DEFAULT,
  },
});

interface ConfirmationState {
  open: boolean;
  title: string;
  content: React.ReactNode;
  onOk: () => void;
}

// parseSftpUri understands both restic sftp URI shapes:
//   - scp-style:  sftp:[user@]host:/path            (cannot carry a port)
//   - URL-style:  sftp://[user@]host[:port]/path
// It returns whichever of user/host/port are present. host is "" if the URI
// cannot be parsed.
const parseSftpUri = (
  uri: string,
): { user?: string; host: string; port?: number } => {
  let rest = uri.replace(/^sftp:/, "");

  const splitUser = (
    authority: string,
  ): { user?: string; hostPart: string } => {
    const at = authority.lastIndexOf("@");
    if (at === -1) return { hostPart: authority };
    return { user: authority.slice(0, at), hostPart: authority.slice(at + 1) };
  };

  if (rest.startsWith("//")) {
    // URL-style: //[user@]host[:port]/path
    rest = rest.slice(2);
    const slash = rest.indexOf("/");
    const authority = slash === -1 ? rest : rest.slice(0, slash);
    const { user, hostPart } = splitUser(authority);
    const colon = hostPart.lastIndexOf(":");
    if (colon !== -1) {
      const port = parseInt(hostPart.slice(colon + 1), 10);
      return {
        user,
        host: hostPart.slice(0, colon),
        port: Number.isNaN(port) ? undefined : port,
      };
    }
    return { user, host: hostPart };
  }

  // scp-style: [user@]host:/path — the first ':' delimits the path, no port.
  const colon = rest.indexOf(":");
  const authority = colon === -1 ? rest : rest.slice(0, colon);
  const { user, hostPart } = splitUser(authority);
  return { user, host: hostPart };
};

// isHostKeyError detects ssh host-key verification failures in an error
// message. restic's sftp backend forwards ssh's stderr into the error it
// returns; ssh prints these markers when the server's host key is unknown or
// does not match the pinned known_hosts entry. Kept in sync with the Go-side
// markers in internal/api/backresthandler.go.
const isHostKeyError = (message: string | undefined): boolean => {
  if (!message) return false;
  const lower = message.toLowerCase();
  return (
    lower.includes("host key verification failed") ||
    lower.includes("remote host identification has changed")
  );
};

const hostKeyUntrustedContent = (
  <>
    {m.add_repo_modal_the_host_key_for_this_sftp_server_is_not_known_or_does_not_m()}
    <br />
    <br />
    {m.add_repo_modal_to_proceed_manually_add_the_correct_host_key_to_your_known_h()}
  </>
);

interface SftpConfigSectionProps {
  uri: string | undefined;
  identityFile: string;
  onChangeIdentityFile: (path: string) => void;
  port: number | null;
  onChangePort: (port: number | null) => void;
  knownHostsPath: string;
  onChangeKnownHostsPath: (path: string) => void;
  isWindows: boolean;
}

const SftpConfigSection = ({
  uri,
  identityFile,
  onChangeIdentityFile,
  port,
  onChangePort,
  knownHostsPath,
  onChangeKnownHostsPath,
  isWindows,
}: SftpConfigSectionProps) => {
  const [setupLoading, setSetupLoading] = useState(false);
  const [generatedPublicKey, setGeneratedPublicKey] = useState<string | null>(
    null,
  );
  const [hostKeyWarning, setHostKeyWarning] = useState<string | null>(null);
  const [keyCopied, setKeyCopied] = useState(false);

  if (isWindows) return null;

  const handleGenerateKey = async () => {
    setSetupLoading(true);
    setGeneratedPublicKey(null);
    setHostKeyWarning(null);
    try {
      if (!uri) return;

      const { user, host, port: uriPort } = parseSftpUri(uri);
      if (!host) return;

      // Precedence for the port used to keyscan the host: the explicit SFTP Port
      // field wins, then a port carried in a URL-style URI, then ssh's default.
      // When the URI supplies a port and the field is empty, prefill the field so
      // the two stay in agreement.
      if (uriPort != null && port == null) {
        onChangePort(uriPort);
      }
      const effectivePort = port ?? uriPort ?? 22;

      const res = await backrestService.setupSftp({
        host,
        port: effectivePort.toString(),
        username: user ?? "",
      });

      onChangeIdentityFile(res.keyPath);
      onChangeKnownHostsPath(res.knownHostsPath);
      if (res.publicKey) {
        setGeneratedPublicKey(res.publicKey);
      }
      if (res.error) {
        setHostKeyWarning(res.error);
      }
      alerts.success(m.add_repo_modal_generated_ssh_keypair_at_keyPath({ keyPath: res.keyPath }));
    } catch (e: any) {
      alerts.error(formatErrorAlert(e, m.add_repo_modal_sftp_setup_failed()));
    } finally {
      setSetupLoading(false);
    }
  };

  return (
    <Stack gap={4} ml={2} borderLeftWidth={2} pl={4}>
      {!generatedPublicKey && !identityFile && (
        <AccordionRoot collapsible variant="enclosed">
          <AccordionItem value="bootstrap">
            <AccordionItemTrigger>
              {m.add_repo_modal_setup_ssh_key_optional()}
            </AccordionItemTrigger>
            <AccordionItemContent>
              <Stack gap={3} p={2}>
                <CText fontSize="sm">
                  {m.add_repo_modal_click_generate_key_to_create_an_ssh_key_pair_for_this_host_b()} <Code>{m.add_repo_modal_sshauthorized_keys()}</Code> {m.add_repo_modal_on_the_remote_server()}
                </CText>
                <Button
                  size="sm"
                  onClick={handleGenerateKey}
                  loading={setupLoading}
                  data-testid="add-repo-sftp-generate-key"
                >
                  {m.add_repo_modal_generate_key()}
                </Button>
              </Stack>
            </AccordionItemContent>
          </AccordionItem>
        </AccordionRoot>
      )}

      {generatedPublicKey && (
        <Box p={4} borderWidth={1} borderRadius="md" bg="bg.subtle">
          <Stack gap={2}>
            <CText fontWeight="bold" color="green.500">
              {m.add_repo_modal_key_generated_successfully()}
            </CText>
            <CText fontSize="sm">
              {m.add_repo_modal_add_the_following_public_key_to()}{" "}
              <Code>{m.add_repo_modal_sshauthorized_keys()}</Code> {m.add_repo_modal_on_the_remote_server_2()}
            </CText>
            <Box position="relative">
              <Code
                p={3}
                display="block"
                whiteSpace="pre-wrap"
                wordBreak="break-all"
              >
                {generatedPublicKey}
              </Code>
              <Box position="absolute" top={1} right={1}>
                <Button
                  size="xs"
                  onClick={() => {
                    navigator.clipboard.writeText(generatedPublicKey || "");
                    setKeyCopied(true);
                    setTimeout(() => setKeyCopied(false), 2000);
                  }}
                  colorPalette={keyCopied ? "green" : undefined}
                >
                  {keyCopied ? "Copied!" : "Copy"}
                </Button>
              </Box>
            </Box>
            {hostKeyWarning && (
              <Box
                p={3}
                borderWidth={1}
                borderRadius="md"
                borderColor="yellow.400"
                bg="yellow.subtle"
              >
                <CText fontSize="sm" color="yellow.700">
                  <strong>{m.add_repo_modal_host_key_scan_failed()}</strong> {hostKeyWarning}
                </CText>
              </Box>
            )}
          </Stack>
        </Box>
      )}

      <Field
        label={m.add_repo_modal_sftp_identity_file()}
        helperText={m.add_repo_modal_optional_path_to_an_ssh_identity_file_for_sftp_authenticatio()}
      >
        <Input
          data-testid="add-repo-sftp-identity"
          placeholder="/home/user/.ssh/id_rsa"
          value={identityFile}
          onChange={(e) => onChangeIdentityFile(e.target.value)}
        />
      </Field>

      <Field
        label={m.add_repo_modal_sftp_port()}
        helperText={m.add_repo_modal_optional_specify_a_custom_port_for_sftp_connection_defaults({ port: 22 })}
      >
        <NumberInputField
          data-testid="add-repo-sftp-port"
          value={port ? port.toString() : undefined}
          onValueChange={(e) => onChangePort(e.valueAsNumber)}
          min={1}
          max={65535}
          defaultValue={"22"}
        />
      </Field>

      <Field
        label={m.add_repo_modal_known_hosts_file()}
        helperText={m.add_repo_modal_optional_path_to_a_known_hosts_file_for_host_key_verificatio()}
      >
        <Input
          data-testid="add-repo-sftp-known-hosts"
          placeholder={m.add_repo_modal_homeusersshknown_hosts()}
          value={knownHostsPath}
          onChange={(e) => onChangeKnownHostsPath(e.target.value)}
        />
      </Field>
    </Stack>
  );
};

export const AddRepoModal = ({
  template,
  onSaveOverride,
}: {
  template: Repo | null;
  onSaveOverride?: (repo: Repo) => Promise<void>;
}) => {
  const [confirmLoading, setConfirmLoading] = useState(false);
  const showModal = useShowModal();
  const [config, setConfig] = useConfig();
  const isRemoteOrigin = !!template?.originInstanceId;

  const [formData, setFormData] = useState<any>(
    template
      ? toJson(RepoSchema, template, { alwaysEmitImplicit: true })
      : toJson(RepoSchema, repoDefaults, { alwaysEmitImplicit: true }),
  );

  const [sftpIdentityFile, setSftpIdentityFile] = useState("");
  const [sftpPort, setSftpPort] = useState<number | null>(null);
  const [sftpKnownHostsPath, setSftpKnownHostsPath] = useState("");

  const flagsRef = useRef<string[]>([]);

  const [confirmation, setConfirmation] = useState<ConfirmationState>({
    open: false,
    title: "",
    content: null,
    onOk: () => {},
  });

  useEffect(() => {
    setFormData(
      template
        ? toJson(RepoSchema, template, { alwaysEmitImplicit: true })
        : toJson(RepoSchema, repoDefaults, { alwaysEmitImplicit: true }),
    );

    setSftpIdentityFile("");
    setSftpPort(null);
    setSftpKnownHostsPath("");

    if (template?.uri?.startsWith("sftp:")) {
      const sftpArgsFlag = (template.flags || []).find(
        (f) => f.includes("sftp.args") || f.includes("sftp.command"),
      );
      if (sftpArgsFlag) {
        const argsMatch = sftpArgsFlag.match(/sftp\.args=['"]?(.+?)['"]?\s*$/);
        if (argsMatch) {
          const argsStr = argsMatch[1].replace(/^'|'$/g, "");
          const identityMatch = argsStr.match(/-i\s+["']?([^\s"']+)["']?/);
          if (identityMatch) setSftpIdentityFile(identityMatch[1]);
          const portMatch = argsStr.match(/-p\s+(\d+)/);
          if (portMatch) setSftpPort(parseInt(portMatch[1], 10));
          const knownHostsMatch = argsStr.match(
            /-oUserKnownHostsFile=["']?([^\s"']+)["']?/,
          );
          if (knownHostsMatch) setSftpKnownHostsPath(knownHostsMatch[1]);
        }
      }
    }
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

  flagsRef.current = (formData.flags as string[]) || [];

  useEffect(() => {
    const uri = getField(["uri"]);
    if (!uri?.startsWith("sftp:")) {
      return;
    }

    const currentFlags = flagsRef.current;
    const newFlags = currentFlags.filter(
      (f: string) =>
        f && !f.includes("sftp.args") && !f.includes("sftp.command"),
    );

    let sftpArgs = "-oBatchMode=yes";

    if (sftpIdentityFile) {
      let cleanPath = sftpIdentityFile;
      if (cleanPath.startsWith("@")) {
        cleanPath = cleanPath.substring(1);
      }
      sftpArgs += ` -i "${cleanPath}"`;
    }

    if (sftpPort && sftpPort !== 0 && sftpPort !== 22) {
      sftpArgs += ` -p ${sftpPort}`;
    }

    if (sftpKnownHostsPath) {
      sftpArgs += ` -oUserKnownHostsFile="${sftpKnownHostsPath}"`;
    }

    newFlags.push(`--option=sftp.args='${sftpArgs}'`);

    const sortedCurrent = [...currentFlags].sort();
    const sortedNew = [...newFlags].sort();

    if (JSON.stringify(sortedCurrent) !== JSON.stringify(sortedNew)) {
      updateField(["flags"], newFlags);
    }
  }, [getField(["uri"]), sftpIdentityFile, sftpPort, sftpKnownHostsPath]);

  if (!config) return null;

  const validateLocal = async () => {
    const id = getField(["id"]);
    if (!id?.trim()) {
      throw new Error(m.add_repo_modal_error_repo_name_required());
    }
    if (!namePattern.test(id)) {
      throw new Error(m.settings_auth_name_pattern());
    }
    if (!template && config.repos.find((r) => r.id === id)) {
      throw new Error(m.add_repo_modal_error_repo_exists());
    }

    const uri = getField(["uri"]);
    if (!uri?.trim()) {
      throw new Error(m.add_repo_modal_error_uri_required());
    }

    await envVarSetValidator(formData);

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
          create(RemoveRepoRequestSchema, { repoId: template!.id }),
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

      const doSubmit = async () => {
        const repo = fromJson(RepoSchema, formData, {
          ignoreUnknownFields: true,
        });

        if (onSaveOverride) {
          await onSaveOverride(repo);
          showModal(null);
          alerts.success(m.add_repo_modal_success_updated({ uri: repo.uri }));
          return;
        }

        const req = create(AddRepoRequestSchema, {
          repo: repo,
        });

        if (template !== null) {
          setConfig(await backrestService.addRepo(req));
          showModal(null);
          alerts.success(m.add_repo_modal_success_updated({ uri: repo.uri }));
        } else {
          setConfig(await backrestService.addRepo(req));
          showModal(null);
          alerts.success(m.add_repo_modal_success_added({ uri: repo.uri }));
        }

        try {
          await backrestService.listSnapshots({ repoId: repo.id });
        } catch (e: any) {
          alerts.error(
            formatErrorAlert(e, m.add_repo_modal_error_list_snapshots()),
          );
        }
      };

      try {
        await doSubmit();
      } catch (e: any) {
        if (isHostKeyError(e.message)) {
          setConfirmation({
            open: true,
            title: m.add_repo_modal_unknown_sftp_host_key(),
            content: hostKeyUntrustedContent,
            onOk: () => {
              setConfirmation((prev) => ({ ...prev, open: false }));
            },
          });
        } else {
          throw e;
        }
      }
    } catch (e: any) {
      alerts.error(
        formatErrorAlert(e, m.settings_error_operation()),
      );
    } finally {
      setConfirmLoading(false);
    }
  };

  const handleTest = async () => {
    setConfirmLoading(true);
    try {
      await validateLocal();
      const doCheck = async () => {
        const repo = fromJson(RepoSchema, formData, {
          ignoreUnknownFields: true,
        });
        const req = create(CheckRepoExistsRequestSchema, {
          repo: repo,
        });

        const response = await backrestService.checkRepoExists(req);

        if (response.hostKeyUntrusted || isHostKeyError(response.error)) {
          setConfirmation({
            open: true,
            title: m.add_repo_modal_unknown_sftp_host_key(),
            content: hostKeyUntrustedContent,
            onOk: () => {
              setConfirmation((prev) => ({ ...prev, open: false }));
            },
          });
          return;
        }

        if (response.error) {
          throw new Error(response.error);
        }

        if (response.exists) {
          alerts.success(
            m.add_repo_modal_test_success_existing({ uri: repo.uri }),
          );
        } else {
          alerts.success(m.add_repo_modal_test_success_new({ uri: repo.uri }));
        }
      };

      try {
        await doCheck();
      } catch (e: any) {
        alerts.error(formatErrorAlert(e, m.add_repo_modal_test_error()));
      }
    } catch (e: any) {
      alerts.error(formatErrorAlert(e, m.add_repo_modal_test_error()));
    } finally {
      setConfirmLoading(false);
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
      description: m.add_repo_modal_default_system_priority(),
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

  const sections: SectionDef[] = [
    { id: "identity", label: m.add_repo_modal_identity(), icon: <FiTag size={14} /> },
    { id: "connection", label: m.add_repo_modal_connection(), icon: <FiLink size={14} /> },
    { id: "scheduling", label: m.add_repo_modal_scheduling(), icon: <FiClock size={14} /> },
    { id: "hooks", label: m.add_repo_modal_hooks(), icon: <FiZap size={14} /> },
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
      {!isRemoteOrigin && (
        <>
          <Button
            variant="subtle"
            loading={confirmLoading}
            onClick={handleTest}
            data-testid="add-repo-test-config"
          >
            {m.add_repo_modal_test_config()}
          </Button>
          <Button
            loading={confirmLoading}
            onClick={handleOk}
            data-testid="add-repo-submit"
          >
            {m.add_plan_modal_button_submit()}
          </Button>
        </>
      )}
    </Flex>
  );

  return (
    <>
      <DialogRoot
        open={confirmation.open}
        onOpenChange={(e) =>
          setConfirmation((prev) => ({ ...prev, open: e.open }))
        }
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{confirmation.title}</DialogTitle>
          </DialogHeader>
          <DialogBody>{confirmation.content}</DialogBody>
          <DialogFooter>
            <DialogActionTrigger asChild>
              <Button variant="outline">{m.button_cancel()}</Button>
            </DialogActionTrigger>
            <Button onClick={confirmation.onOk}>{m.add_repo_modal_confirm()}</Button>
          </DialogFooter>
          <DialogCloseTrigger />
        </DialogContent>
      </DialogRoot>

      <TwoPaneModal
        isOpen={true}
        onClose={() => showModal(null)}
        title={
          template
            ? m.add_repo_modal_title_edit()
            : m.add_repo_modal_title_add()
        }
        headerIcon={<FiTag size={14} />}
        sections={sections}
        footer={footer}
      >
        <Box
          opacity={isRemoteOrigin ? 0.7 : 1}
          pointerEvents={isRemoteOrigin ? "none" : undefined}
        >
          {isRemoteOrigin && (
            <Box
              p={3}
              mb={4}
              borderWidth={1}
              borderRadius="md"
              bg="blue.subtle"
              borderColor="blue.400"
              pointerEvents="auto"
            >
              <CText fontSize="sm">
                {m.add_repo_modal_this_repository_is_managed_by_remote_instance()}{" "}
                <strong>{template?.originInstanceId}</strong> {m.add_repo_modal_and_cannot_be_edited_you_may_delete_it_to_remove_the_local_c()}
              </CText>
            </Box>
          )}

          {/* Identity Section */}
          <TwoPaneSection id="identity">
            <SectionCard
              icon={<FiTag size={16} />}
              title={m.add_repo_modal_identity()}
              description={m.add_repo_modal_display_name_identifiers_and_unlock_behaviour()}
            >
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
                      ? m.settings_auth_name_pattern()
                      : m.add_repo_modal_error_repo_exists()
                  }
                >
                  <Input
                    data-testid="add-repo-name"
                    value={getField(["id"])}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                      updateField(["id"], e.target.value)
                    }
                    disabled={!!template}
                    placeholder={"repo" + ((config?.repos?.length || 0) + 1)}
                  />
                </Field>

                <ToggleField
                  checked={getField(["autoUnlock"]) || false}
                  onChange={(v) => updateField(["autoUnlock"], v)}
                  label={m.add_repo_modal_field_auto_unlock()}
                  hint={m.add_repo_modal_field_auto_unlock_tooltip()}
                />

                <ToggleField
                  checked={getField(["shared"]) || false}
                  onChange={(v) => updateField(["shared"], v)}
                  label={m.add_repo_modal_shared()}
                  hint="If using multihost management, enables sharing this repo's configuration to all authorized clients with read permission."
                />
                {getField(["shared"]) && (
                  <CText fontSize="sm" color="orange.500">
                    {m.add_repo_modal_shared_forget_hint()}
                  </CText>
                )}
              </Stack>
            </SectionCard>
          </TwoPaneSection>

          {/* Connection Section */}
          <TwoPaneSection id="connection">
            <SectionCard
              icon={<FiLink size={16} />}
              title={m.add_repo_modal_connection()}
              description={m.add_repo_modal_where_the_repo_lives_and_how_backrest_authenticates()}
            >
              <Stack gap={4}>
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
                          {m.add_repo_modal_guide_text_p1()}{" "}
                          <a
                            href="https://restic.readthedocs.io/en/latest/030_preparing_a_new_repo.html#preparing-a-new-repository"
                            target="_blank"
                            style={{ textDecoration: "underline" }}
                          >
                            {m.add_plan_modal_field_excludes_tooltip_link()}
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
                    inputProps={{ "data-testid": "add-repo-uri" }}
                  />
                </Field>

                {getField(["uri"])?.startsWith("sftp:") && (
                  <SftpConfigSection
                    uri={getField(["uri"])}
                    identityFile={sftpIdentityFile}
                    onChangeIdentityFile={setSftpIdentityFile}
                    port={sftpPort}
                    onChangePort={setSftpPort}
                    knownHostsPath={sftpKnownHostsPath}
                    onChangeKnownHostsPath={setSftpKnownHostsPath}
                    isWindows={isWindows}
                  />
                )}

                <Field
                  label={m.login_password_placeholder()}
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
                        data-testid="add-repo-password"
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

                <DynamicList
                  label={m.add_repo_modal_field_env_vars()}
                  items={getField(["env"]) || []}
                  onUpdate={(items: string[]) => updateField(["env"], items)}
                  tooltip={
                    <Stack gap={2}>
                      <CText>{m.add_repo_modal_field_env_vars_tooltip()}</CText>
                      <EnvVarTooltip uri={getField(["uri"])} />
                    </Stack>
                  }
                  placeholder={"KEY=VALUE"}
                />
              </Stack>
            </SectionCard>
          </TwoPaneSection>

          {/* Scheduling Section */}
          <TwoPaneSection id="scheduling">
            <SectionCard
              icon={<FiClock size={16} />}
              title={m.add_repo_modal_prune_policy()}
              description={m.add_repo_modal_field_prune_policy_help()}
            >
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
            </SectionCard>

            <SectionCard
              icon={<FiClock size={16} />}
              title={m.add_repo_modal_check_policy()}
              description={m.add_repo_modal_field_check_policy_help()}
            >
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
            </SectionCard>

            <SectionCard
              icon={<FiClock size={16} />}
              title={m.add_repo_modal_field_forget_policy()}
              description={m.add_repo_modal_field_forget_policy_help()}
            >
              <Stack gap={4}>
                <ScheduleFormItem
                  value={getField(["forgetPolicy", "schedule"])}
                  onChange={(val: any) =>
                    updateField(["forgetPolicy", "schedule"], val)
                  }
                  defaults={ScheduleDefaultsDaily}
                />
                {(() => {
                  const sched = getField(["forgetPolicy", "schedule"]);
                  return sched && !sched.disabled ? (
                    <Field
                      label={m.add_repo_modal_retention_policy_label()}
                      helperText={m.add_repo_modal_retention_policy_help()}
                    >
                      <RetentionPolicyView
                        retention={getField(["forgetPolicy", "retention"])}
                        onChange={(v: any) =>
                          updateField(["forgetPolicy", "retention"], v)
                        }
                      />
                    </Field>
                  ) : null;
                })()}
              </Stack>
            </SectionCard>
          </TwoPaneSection>

          {/* Hooks Section */}
          <TwoPaneSection id="hooks">
            <SectionCard
              icon={<FiZap size={16} />}
              title={m.add_repo_modal_hooks()}
              description={m.add_repo_modal_run_commands_or_send_notifications_on_operation_events()}
            >
              <Field
                label={m.add_repo_modal_hooks()}
                helperText={hooksListTooltipText}
              >
                <HooksFormList
                  value={getField(["hooks"])}
                  onChange={(v: any) => updateField(["hooks"], v)}
                />
              </Field>
            </SectionCard>
          </TwoPaneSection>

          {/* Advanced Section */}
          <TwoPaneSection id="advanced">
            <SectionCard
              icon={<FiSliders size={16} />}
              title={m.add_plan_modal_advanced()}
              description={m.add_repo_modal_command_priority_extra_flags_and_raw_restic_options()}
            >
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

                <DynamicList
                  label={m.add_repo_modal_field_flags()}
                  items={getField(["flags"]) || []}
                  onUpdate={(items: string[]) => updateField(["flags"], items)}
                  placeholder="--flag"
                />
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
        </Box>
      </TwoPaneModal>
    </>
  );
};

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
  const flags = formData.flags || [];
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
    !envVarNames.includes("RESTIC_PASSWORD_FILE") &&
    !flags.includes("--insecure-no-password")
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
    m.add_repo_modal_missing_env_vars_for_scheme({
      missing: formatMissingEnvVars(missingVarsCollection),
      scheme,
    }),
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
        {m.add_repo_modal_recommended_for()} {scheme}:
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
