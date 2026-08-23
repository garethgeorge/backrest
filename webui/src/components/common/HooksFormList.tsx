import React from "react";
import {
  Hook_Condition,
  Hook_ConditionSchema,
  Hook_OnErrorSchema,
} from "../../../gen/ts/v1/config_pb";
import {
  Box,
  Button,
  Stack,
  Input,
  Text,
  Badge,
  IconButton,
  Card,
  Textarea,
  Flex,
  useControllableState,
  SimpleGrid,
} from "@chakra-ui/react";
import { FiPlus, FiTrash2, FiInfo, FiChevronDown, FiChevronUp } from "react-icons/fi";
import {
  MenuTrigger,
  MenuContent,
  MenuItem,
  MenuItemText,
  MenuRoot,
} from "../ui/menu";
import { Tooltip } from "../ui/tooltip";
import { Link } from "../ui/link";
import { EnumSelector, EnumOption } from "./EnumSelector";
import * as m from "../../paraglide/messages";

export interface HookFields {
  conditions: string[];
  onError?: string;
  actionCommand?: any;
  actionGotify?: any;
  actionDiscord?: any;
  actionWebhook?: any;
  actionSlack?: any;
  actionShoutrrr?: any;
  actionHealthchecks?: any;
  actionTelegram?: any;
}

export const hooksListTooltipText = (
  <Text as="span">
    {m.hooks_from_list_a()}
    <Link
      href="https://garethgeorge.github.io/backrest/docs/hooks"
      target="_blank"
      color="blue.500"
    >
      {m.hooks_from_list_b()}
    </Link>
    {m.hooks_from_list_c()}
    <Link
      href="https://garethgeorge.github.io/backrest/cookbooks/command-hook-examples"
      target="_blank"
      color="blue.500"
    >
      {m.hooks_from_list_d()}
    </Link>
    {m.hooks_from_list_e()}
  </Text>
);

const hookConditionDescriptions: Record<string, string> = {
  CONDITION_SNAPSHOT_START:
    m.repo_hooks_command_runs_condition_snapshot_start(),
  CONDITION_SNAPSHOT_END: m.repo_hooks_command_runs_condition_snapshot_end(),
  CONDITION_SNAPSHOT_SUCCESS:
    m.repo_hooks_command_runs_condition_snapshot_success(),
  CONDITION_SNAPSHOT_ERROR:
    m.repo_hooks_command_runs_condition_snapshot_error(),
  CONDITION_SNAPSHOT_WARNING:
    m.repo_hooks_command_runs_condition_snapshot_warning(),
  CONDITION_SNAPSHOT_SKIPPED:
    m.repo_hooks_command_runs_condition_snapshot_skipped(),
  CONDITION_PRUNE_START: m.repo_hooks_command_runs_condition_prune_start(),
  CONDITION_PRUNE_SUCCESS: m.repo_hooks_command_runs_condition_prune_success(),
  CONDITION_PRUNE_ERROR: m.repo_hooks_command_runs_condition_prune_error(),
  CONDITION_CHECK_START: m.repo_hooks_command_runs_condition_check_start(),
  CONDITION_CHECK_SUCCESS: m.repo_hooks_command_runs_condition_check_success(),
  CONDITION_CHECK_ERROR: m.repo_hooks_command_runs_condition_check_error(),
  CONDITION_FORGET_START: m.repo_hooks_command_runs_condition_forget_start(),
  CONDITION_FORGET_SUCCESS:
    m.repo_hooks_command_runs_condition_forget_success(),
  CONDITION_FORGET_ERROR: m.repo_hooks_command_runs_condition_forget_error(),
  CONDITION_ANY_ERROR: m.repo_hooks_command_runs_condition_any_error(),
  CONDITION_UNKNOWN: m.repo_hooks_command_runs_condition_unknown(),
};

const conditionOptions: EnumOption<string>[] = Hook_ConditionSchema.values.map(
  (v) => ({
    label: v.name,
    value: v.name,
    description: hookConditionDescriptions[v.name],
  }),
);

const onErrorOptions: EnumOption<string>[] = Hook_OnErrorSchema.values.map(
  (v) => ({
    label: v.name,
    value: v.name,
  }),
);

interface HooksFormListProps {
  value?: HookFields[];
  defaultValue?: HookFields[];
  onChange?: (value: HookFields[]) => void;
}

/**
 * HooksFormList is a UI component for editing a list of hooks that can apply either at the repo level or at the plan level.
 */
export const HooksFormList = ({
  value,
  defaultValue = [],
  onChange,
}: HooksFormListProps) => {
  const [hooks, setHooks] = useControllableState({
    value,
    defaultValue,
    onChange,
  });

  const addHook = (template: HookFields) => {
    setHooks([...(hooks || []), template]);
  };

  const removeHook = (index: number) => {
    const newHooks = [...(hooks || [])];
    newHooks.splice(index, 1);
    setHooks(newHooks);
  };

  const updateHook = (index: number, newHook: HookFields) => {
    const newHooks = [...(hooks || [])];
    newHooks[index] = newHook;
    setHooks(newHooks);
  };

  return (
    <Stack gap={4} width="full">
      {(hooks || []).map((hook, index) => (
        <HookItem
          key={index}
          index={index}
          hook={hook}
          onRemove={() => removeHook(index)}
          onChange={(updated) => updateHook(index, updated)}
        />
      ))}

      {/* @ts-ignore */}
      <MenuRoot>
        {/* @ts-ignore */}
        <MenuTrigger asChild>
          <Button
            variant="outline"
            borderStyle="dashed"
            size="sm"
            width="full"
            data-testid="hooks-add"
          >
            <FiPlus /> {m.add_plan_modal_field_add_hook()}
          </Button>
        </MenuTrigger>
        {/* @ts-ignore */}
        <MenuContent zIndex={2000}>
          <SimpleGrid columns={3} gap={2} p={2}>
            {hookTypes.map((type) => (
              // @ts-ignore
              <MenuItem
                key={type.name}
                onClick={(e) => {
                  e.stopPropagation();
                  addHook(JSON.parse(JSON.stringify(type.template))); // Deep clone
                }}
                cursor="pointer"
                justifyContent="center"
                borderRadius="md"
                _hover={{ bg: "bg.muted" }}
              >
                {/* @ts-ignore */}
                <MenuItemText textAlign="center">{type.name}</MenuItemText>
              </MenuItem>
            ))}
          </SimpleGrid>
        </MenuContent>
      </MenuRoot>
    </Stack>
  );
};

const HookItem = ({
  index,
  hook,
  onRemove,
  onChange,
}: {
  index: number;
  hook: HookFields;
  onRemove: () => void;
  onChange: (h: HookFields) => void;
}) => {
  const typeName = findHookTypeName(hook);

  // @ts-ignore
  const handleConditionChange = (value: string | string[]) => {
    onChange({
      ...hook,
      conditions: Array.isArray(value) ? value : [value],
    });
  };

  return (
    <Card.Root size="sm" variant="outline" width="full">
      <Card.Header pb={2}>
        <Flex align="center" justify="space-between">
          <Text fontWeight="bold">
            {m.hooks_form_list_hook()} {index + 1}: {typeName}
          </Text>
          <IconButton
            size="xs"
            variant="ghost"
            colorPalette="red"
            onClick={onRemove}
            aria-label={m.hooks_form_list_remove_hook()}
            data-testid="hook-remove"
          >
            <FiTrash2 />
          </IconButton>
        </Flex>
      </Card.Header>
      <Card.Body gap={3}>
        <HookConditionsTooltip>
          <Box width="full" data-testid="hook-conditions">
            <EnumSelector
              multiSelect
              options={conditionOptions}
              value={hook.conditions}
              onChange={handleConditionChange}
              placeholder={m.repo_hooks_command_runs_when()}
              size="sm"
            />
          </Box>
        </HookConditionsTooltip>

        <HookBuilder hook={hook} onChange={onChange} />
      </Card.Body>
    </Card.Root>
  );
};

const templateSnippets: { name: string; description: string; snippet: string }[] = [
  {
    name: "Task",
    description: "Name of the task that triggered the hook",
    snippet: "{{ .Task }}",
  },
  {
    name: "Event",
    description: "Triggering event (numeric value)",
    snippet: "{{ .Event }}",
  },
  {
    name: "EventName",
    description: "Human-readable name of the event",
    snippet: "{{ .EventName .Event }}",
  },
  {
    name: "Repo",
    description: "Repository information",
    snippet: "{{ .Repo.Id }}",
  },
  {
    name: "Repo.Id",
    description: "Repository ID",
    snippet: "{{ .Repo.Id }}",
  },
  {
    name: "Repo.Uri",
    description: "Repository URI",
    snippet: "{{ .Repo.Uri }}",
  },
  {
    name: "Repo.Guid",
    description: "Repository GUID",
    snippet: "{{ .Repo.Guid }}",
  },
  {
    name: "Plan",
    description: "Plan information",
    snippet: "{{ .Plan.Id }}",
  },
  {
    name: "Plan.Id",
    description: "Plan ID",
    snippet: "{{ .Plan.Id }}",
  },
  {
    name: "Plan.Paths",
    description: "Included paths",
    snippet: "{{ .Plan.Paths }}",
  },
  {
    name: "Plan.Repo",
    description: "ID of the plan's repo",
    snippet: "{{ .Plan.Repo }}",
  },
  {
    name: "SnapshotId",
    description: "ID of the associated snapshot",
    snippet: "{{ .SnapshotId }}",
  },
  {
    name: "SnapshotStats.DataAdded",
    description: "Amount of new data",
    snippet: "{{ .SnapshotStats.DataAdded }}",
  },
  {
    name: "SnapshotStats.TotalFilesProcessed",
    description: "Total files processed",
    snippet: "{{ .SnapshotStats.TotalFilesProcessed }}",
  },
  {
    name: "SnapshotStats.TotalBytesProcessed",
    description: "Total bytes processed",
    snippet: "{{ .SnapshotStats.TotalBytesProcessed }}",
  },
  {
    name: "SnapshotStats.FilesNew",
    description: "New files",
    snippet: "{{ .SnapshotStats.FilesNew }}",
  },
  {
    name: "SnapshotStats.FilesChanged",
    description: "Changed files",
    snippet: "{{ .SnapshotStats.FilesChanged }}",
  },
  {
    name: "SnapshotStats.FilesUnmodified",
    description: "Unmodified files",
    snippet: "{{ .SnapshotStats.FilesUnmodified }}",
  },
  {
    name: "SnapshotStats.DirsNew",
    description: "New directories",
    snippet: "{{ .SnapshotStats.DirsNew }}",
  },
  {
    name: "SnapshotStats.DirsChanged",
    description: "Changed directories",
    snippet: "{{ .SnapshotStats.DirsChanged }}",
  },
  {
    name: "SnapshotStats.DirsUnmodified",
    description: "Unmodified directories",
    snippet: "{{ .SnapshotStats.DirsUnmodified }}",
  },
  {
    name: "SnapshotStats.DataBlobs",
    description: "Data blobs",
    snippet: "{{ .SnapshotStats.DataBlobs }}",
  },
  {
    name: "SnapshotStats.TreeBlobs",
    description: "Tree blobs",
    snippet: "{{ .SnapshotStats.TreeBlobs }}",
  },
  {
    name: "SnapshotStats.TotalDuration",
    description: "Total duration in seconds",
    snippet: "{{ .SnapshotStats.TotalDuration }}",
  },
  {
    name: "CurTime",
    description: "Current timestamp",
    snippet: "{{ .CurTime }}",
  },
  {
    name: "Duration",
    description: "Operation duration",
    snippet: "{{ .Duration }}",
  },
  {
    name: "Error",
    description: "Error message if any",
    snippet: "{{ .Error }}",
  },
  {
    name: "Summary",
    description: "Default event summary",
    snippet: "{{ .Summary }}",
  },
  {
    name: "FormatTime",
    description: "Format a timestamp",
    snippet: "{{ .FormatTime .CurTime }}",
  },
  {
    name: "FormatDuration",
    description: "Format a duration",
    snippet: "{{ .FormatDuration .Duration }}",
  },
  {
    name: "FormatSizeBytes",
    description: "Format a byte size",
    snippet: "{{ .FormatSizeBytes .SnapshotStats.DataAdded }}",
  },
  {
    name: "ShellEscape",
    description: "Escape a string for shell usage",
    snippet: "{{ .ShellEscape \"my string\" }}",
  },
  {
    name: "JsonMarshal",
    description: "Convert a value to JSON",
    snippet: "{{ .JsonMarshal .SnapshotStats }}",
  },
  {
    name: "IsError",
    description: "True for error events",
    snippet: "{{ .IsError .Event }}",
  },
];

const TemplateTextarea = ({
  value,
  onChange,
  size = "sm",
}: {
  value: string;
  onChange: (val: string) => void;
  size?: "sm" | "md" | "lg";
}) => {
  const ref = React.useRef<HTMLTextAreaElement>(null);
  const [open, setOpen] = React.useState(false);
  const [cursor, setCursor] = React.useState({ start: 0, end: 0 });

  const handleSelect = () => {
    if (!ref.current) return;
    setCursor({
      start: ref.current.selectionStart,
      end: ref.current.selectionEnd,
    });
  };

  const insert = (snippet: string) => {
    if (!ref.current) return;
    const { start, end } = cursor;
    const next = value.slice(0, start) + snippet + value.slice(end);
    onChange(next);
    const pos = start + snippet.length;
    setCursor({ start: pos, end: pos });
    setTimeout(() => {
      ref.current?.focus();
      ref.current?.setSelectionRange(pos, pos);
    }, 0);
  };

  return (
    <Stack gap={2} width="full">
      <Textarea
        ref={ref}
        fontFamily="monospace"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onSelect={handleSelect}
        onClick={handleSelect}
        onKeyUp={handleSelect}
        onBlur={handleSelect}
        size={size}
      />
      <Button
        variant="outline"
        borderStyle="dashed"
        size="sm"
        width="full"
        onClick={() => setOpen(!open)}
      >
        {open ? <FiChevronUp /> : <FiChevronDown />}{" "}
        {open
          ? "Hide available variables and helper functions"
          : "Show available variables and helper functions"}
      </Button>
      {open && (
        <Box p={2} borderWidth="1px" borderRadius="md" bg="bg.muted">
          <Stack gap={1}>
            {templateSnippets.map((item) => (
              <Flex key={item.name} align="center" gap={2}>
                <Button
                  size="xs"
                  variant="outline"
                  onClick={() => insert(item.snippet)}
                  flexShrink={0}
                >
                  {item.name}
                </Button>
                <Text fontSize="xs" color="fg.muted">
                  {item.description}
                </Text>
              </Flex>
            ))}
          </Stack>
        </Box>
      )}
    </Stack>
  );
};

const hookTypes: {
  name: string;
  template: HookFields;
  oneofKey: string;
  component: ({
    hook,
    onChange,
  }: {
    hook: HookFields;
    onChange: (h: HookFields) => void;
  }) => React.ReactNode;
}[] = [
  {
    name: m.repo_hooks_command_label(),
    template: {
      actionCommand: {
        command: "echo {{ .ShellEscape .Summary }}",
      },
      conditions: [],
    },
    oneofKey: "actionCommand",
    component: ({ hook, onChange }) => {
      const updateCommand = (val: string) => {
        onChange({
          ...hook,
          actionCommand: { ...hook.actionCommand, command: val },
        });
      };
      return (
        <Stack gap={2}>
          <Text fontSize="sm" fontWeight="medium">
            {m.repo_hooks_command_script_label()}
          </Text>
          <Textarea
            fontFamily="monospace"
            value={hook.actionCommand?.command || ""}
            onChange={(e) => updateCommand(e.target.value)}
            size="sm"
            data-testid="hook-command"
          />
          <ItemOnErrorSelector hook={hook} onChange={onChange} />
        </Stack>
      );
    },
  },
  {
    name: "Shoutrrr",
    template: {
      actionShoutrrr: {
        template: "{{ .Summary }}",
      },
      conditions: [],
    },
    oneofKey: "actionShoutrrr",
    component: ({ hook, onChange }) => {
      const updateShoutrrr = (field: string, val: string) => {
        onChange({
          ...hook,
          actionShoutrrr: { ...hook.actionShoutrrr, [field]: val },
        });
      };
      return (
        <Stack gap={2}>
          <Input
            placeholder={m.hooks_form_list_service_url({ service: "Shoutrrr" })}
            value={hook.actionShoutrrr?.shoutrrrUrl || ""}
            onChange={(e) => updateShoutrrr("shoutrrrUrl", e.target.value)}
            size="sm"
          />
          <Text fontSize="sm" mt={1}>
            {m.repo_hooks_command_template_label()}
          </Text>
          <TemplateTextarea
            value={hook.actionShoutrrr?.template || ""}
            onChange={(val) => updateShoutrrr("template", val)}
            size="sm"
          />
        </Stack>
      );
    },
  },
  {
    name: "Discord",
    template: {
      actionDiscord: {
        webhookUrl: "",
        template: "{{ .Summary }}",
      },
      conditions: [],
    },
    oneofKey: "actionDiscord",
    component: ({ hook, onChange }) => {
      const updateDiscord = (field: string, val: string) => {
        onChange({
          ...hook,
          actionDiscord: { ...hook.actionDiscord, [field]: val },
        });
      };
      return (
        <Stack gap={2}>
          <Input
            placeholder={m.hooks_form_list_webhook_url({ service: "Discord" })}
            value={hook.actionDiscord?.webhookUrl || ""}
            onChange={(e) => updateDiscord("webhookUrl", e.target.value)}
            size="sm"
          />
          <Text fontSize="sm" mt={1}>
            {m.repo_hooks_command_template_label()}
          </Text>
          <TemplateTextarea
            value={hook.actionDiscord?.template || ""}
            onChange={(val) => updateDiscord("template", val)}
            size="sm"
          />
        </Stack>
      );
    },
  },
  {
    name: "Gotify",
    template: {
      actionGotify: {
        baseUrl: "",
        token: "",
        template: "{{ .Summary }}",
        titleTemplate:
          "Backrest {{ .EventName .Event }} in plan {{ .Plan.Id }}",
        priority: 5,
      },
      conditions: [],
    },
    oneofKey: "actionGotify",
    component: ({ hook, onChange }) => {
      const updateGotify = (field: string, val: any) => {
        onChange({
          ...hook,
          actionGotify: { ...hook.actionGotify, [field]: val },
        });
      };
      return (
        <Stack gap={2}>
          <Input
            placeholder={m.hooks_form_list_service_base_url({ service: "Gotify" })}
            value={hook.actionGotify?.baseUrl || ""}
            onChange={(e) => updateGotify("baseUrl", e.target.value)}
            size="sm"
          />
          <Input
            placeholder={m.hooks_form_list_service_token({ service: "Gotify" })}
            value={hook.actionGotify?.token || ""}
            onChange={(e) => updateGotify("token", e.target.value)}
            size="sm"
          />
          <Input
            placeholder={m.hooks_form_list_title_template()}
            value={hook.actionGotify?.titleTemplate || ""}
            onChange={(e) => updateGotify("titleTemplate", e.target.value)}
            size="sm"
          />
          <Text fontSize="sm" mt={1}>
            {m.repo_hooks_command_template_label()}
          </Text>
          <TemplateTextarea
            value={hook.actionGotify?.template || ""}
            onChange={(val) => updateGotify("template", val)}
            size="sm"
          />
          <EnumSelector
            options={[
              { label: "0 - " + m.hooks_form_list_no_notification(), value: "0" },
              { label: "1 - " + m.hooks_form_list_icon_in_notification_bar(), value: "1" },
              { label: "4 - " + m.hooks_form_list_icon_in_notification_bar_sound(), value: "4" },
              {
                label: "8 - " + m.hooks_form_list_icon_in_notification_bar_sound_vibration(),
                value: "8",
              },
            ]}
            value={String(hook.actionGotify?.priority ?? 5)}
            onChange={(val) =>
              updateGotify("priority", parseInt(val as string))
            }
            placeholder={m.hooks_form_list_priority()}
            size="sm"
          />
        </Stack>
      );
    },
  },
  {
    name: "Slack",
    template: {
      actionSlack: {
        webhookUrl: "",
        template: "{{ .Summary }}",
      },
      conditions: [],
    },
    oneofKey: "actionSlack",
    component: ({ hook, onChange }) => {
      const updateSlack = (field: string, val: string) => {
        onChange({
          ...hook,
          actionSlack: { ...hook.actionSlack, [field]: val },
        });
      };
      return (
        <Stack gap={2}>
          <Input
            placeholder={m.hooks_form_list_service_url({ service: "Slack" })}
            value={hook.actionSlack?.webhookUrl || ""}
            onChange={(e) => updateSlack("webhookUrl", e.target.value)}
            size="sm"
          />
          <Text fontSize="sm" mt={1}>
            {m.repo_hooks_command_template_label()}
          </Text>
          <TemplateTextarea
            value={hook.actionSlack?.template || ""}
            onChange={(val) => updateSlack("template", val)}
            size="sm"
          />
        </Stack>
      );
    },
  },
  {
    name: "Healthchecks",
    template: {
      actionHealthchecks: {
        webhookUrl: "",
        template: "{{ .Summary }}",
      },
      conditions: [],
    },
    oneofKey: "actionHealthchecks",
    component: ({ hook, onChange }) => {
      const updateHealthchecks = (field: string, val: string) => {
        onChange({
          ...hook,
          actionHealthchecks: { ...hook.actionHealthchecks, [field]: val },
        });
      };
      return (
        <Stack gap={2}>
          <Input
            placeholder={m.hooks_form_list_ping_url({ service: "Ping" })}
            value={hook.actionHealthchecks?.webhookUrl || ""}
            onChange={(e) => updateHealthchecks("webhookUrl", e.target.value)}
            size="sm"
          />
          <Text fontSize="sm" mt={1}>
            {m.repo_hooks_command_template_label()}
          </Text>
          <TemplateTextarea
            value={hook.actionHealthchecks?.template || ""}
            onChange={(val) => updateHealthchecks("template", val)}
            size="sm"
          />
        </Stack>
      );
    },
  },
  {
    name: "Telegram",
    template: {
      actionTelegram: {
        botToken: "",
        chatId: "",
        template: "{{ .Summary }}",
      },
      conditions: [],
    },
    oneofKey: "actionTelegram",
    component: ({ hook, onChange }) => {
      const updateTelegram = (field: string, val: string) => {
        onChange({
          ...hook,
          actionTelegram: { ...hook.actionTelegram, [field]: val },
        });
      };
      return (
        <Stack gap={2}>
          <Input
            placeholder={m.hooks_form_list_service_token({ service: "Bot" })}
            value={hook.actionTelegram?.botToken || ""}
            onChange={(e) => updateTelegram("botToken", e.target.value)}
            size="sm"
          />
          <Input
            placeholder={m.hooks_form_list_chat_id()}
            value={hook.actionTelegram?.chatId || ""}
            onChange={(e) => updateTelegram("chatId", e.target.value)}
            size="sm"
          />
          <Text fontSize="sm" mt={1}>
            {m.repo_hooks_command_template_label()}
          </Text>
          <TemplateTextarea
            value={hook.actionTelegram?.template || ""}
            onChange={(val) => updateTelegram("template", val)}
            size="sm"
          />
        </Stack>
      );
    },
  },
];

const findHookTypeName = (field: HookFields): string => {
  if (!field) {
    return "Unknown";
  }
  for (const hookType of hookTypes) {
    if (hookType.oneofKey in field) {
      return hookType.name;
    }
  }
  return "Unknown";
};

const HookBuilder = ({
  hook,
  onChange,
}: {
  hook: HookFields;
  onChange: (h: HookFields) => void;
}) => {
  if (!hook) {
    return <Text>{m.hooks_form_list_unknown_hook_type()}</Text>;
  }

  for (const hookType of hookTypes) {
    if (hookType.oneofKey in hook) {
      return hookType.component({ hook, onChange });
    }
  }

  return <Text>{m.hooks_form_list_unknown_hook_type()}</Text>;
};

const ItemOnErrorSelector = ({
  hook,
  onChange,
}: {
  hook: HookFields;
  onChange: (h: HookFields) => void;
}) => {
  return (
    <Stack gap={2}>
      <Flex align="center" gap={1}>
        <Text fontSize="sm" fontWeight="medium">
          {m.repo_hooks_command_error_label()}
        </Text>
        <Tooltip
          content={
            <Box>
              <Text fontWeight="bold">
                {m.repo_hooks_command_error_info_what()}
              </Text>
              <Text fontSize="xs">
                {m.repo_hooks_command_error_info_only()}
              </Text>
              <Stack gap={1} mt={1} fontSize="xs">
                <Text>• {m.repo_hooks_command_error_info_ignore()}</Text>
                <Text>• {m.repo_hooks_command_error_info_fatal()}</Text>
                <Text>• {m.repo_hooks_command_error_info_cancel()}</Text>
              </Stack>
            </Box>
          }
        >
          <IconButton aria-label="info" size="xs" variant="ghost">
            <FiInfo />
          </IconButton>
        </Tooltip>
      </Flex>
      <EnumSelector
        options={onErrorOptions}
        value={hook.onError || ""}
        onChange={(val) => onChange({ ...hook, onError: val as string })}
        placeholder={m.repo_hooks_command_error_tooltip()}
        size="sm"
      />
    </Stack>
  );
};

const HookConditionsTooltip = ({ children }: { children: React.ReactNode }) => {
  return (
    <Tooltip
      content={
        <Box>
          <Text fontWeight="bold">
            {m.repo_hooks_command_runs_info_available()}
          </Text>
          <Stack gap={0} fontSize="xs">
            <Text>• {m.repo_hooks_command_runs_info_any_error()}</Text>
            <Text>• {m.repo_hooks_command_runs_info_start()}</Text>
            <Text>• {m.repo_hooks_command_runs_info_end()}</Text>
            <Text>• {m.repo_hooks_command_runs_info_success()}</Text>
            <Text>• {m.repo_hooks_command_runs_info_error()}</Text>
            <Text>• {m.repo_hooks_command_runs_info_warning()}</Text>
            <Text>• {m.repo_hooks_command_runs_info_prune_start()}</Text>
            <Text>• {m.repo_hooks_command_runs_info_docs()}</Text>
          </Stack>
        </Box>
      }
    >
      {children}
    </Tooltip>
  );
};
