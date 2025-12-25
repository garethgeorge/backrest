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
} from "@chakra-ui/react";
import { FiPlus, FiTrash2, FiInfo } from "react-icons/fi";
import {
  SelectRoot,
  SelectTrigger,
  SelectValueText,
  SelectContent,
  SelectItem,
  SelectHiddenSelect,
} from "../ui/select";
import {
  MenuContent,
  MenuItem,
  MenuItemText,
  MenuRoot,
  MenuTrigger,
} from "../ui/menu";
import { Tooltip } from "../ui/tooltip";
import { createListCollection } from "@chakra-ui/react";
import { Link } from "../ui/link";

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
    Hooks let you configure actions e.g. notifications and scripts that run in
    response to the backup lifecycle. See{" "}
    <Link
      href="https://garethgeorge.github.io/backrest/docs/hooks"
      target="_blank"
      color="blue.500"
    >
      the hook documentation
    </Link>{" "}
    for available options, or{" "}
    <Link
      href="https://garethgeorge.github.io/backrest/cookbooks/command-hook-examples"
      target="_blank"
      color="blue.500"
    >
      the cookbook
    </Link>{" "}
    for scripting examples.
  </Text>
);

const hookConditionDescriptions: Record<string, string> = {
  CONDITION_SNAPSHOT_START:
    "Triggered when a backup operation begins and must complete successfully before the snapshot will begin.",
  CONDITION_SNAPSHOT_END:
    "Triggered when a backup operation completes (regardless of success/failure)",
  CONDITION_SNAPSHOT_SUCCESS:
    "Triggered when a backup operation completes successfully",
  CONDITION_SNAPSHOT_ERROR: "Triggered when a backup operation fails",
  CONDITION_SNAPSHOT_WARNING:
    "Triggered when a backup operation encounters non-fatal issues",
  CONDITION_PRUNE_START: "Triggered when a prune operation begins",
  CONDITION_PRUNE_SUCCESS:
    "Triggered when a prune operation completes successfully",
  CONDITION_PRUNE_ERROR: "Triggered when a prune operation fails",
  CONDITION_CHECK_START: "Triggered when a check operation begins",
  CONDITION_CHECK_SUCCESS:
    "Triggered when a check operation completes successfully",
  CONDITION_CHECK_ERROR: "Triggered when a check operation fails",
  CONDITION_FORGET_START: "Triggered when a forget operation begins",
  CONDITION_FORGET_SUCCESS:
    "Triggered when a forget operation completes successfully",
  CONDITION_FORGET_ERROR: "Triggered when a forget operation fails",
  CONDITION_ANY_ERROR: "Triggered when any operation fails",
};

const conditionCollection = createListCollection({
  items: Hook_ConditionSchema.values.map((v) => ({
    label: v.name,
    value: v.name,
    description: hookConditionDescriptions[v.name],
  })),
});

const onErrorCollection = createListCollection({
  items: Hook_OnErrorSchema.values.map((v) => ({
    label: v.name,
    value: v.name,
  })),
});

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
          <Button variant="outline" borderStyle="dashed" size="sm" width="full">
            <FiPlus /> Add Hook
          </Button>
        </MenuTrigger>
        {/* @ts-ignore */}
        <MenuContent zIndex={2000}>
          {hookTypes.map((type) => (
            // @ts-ignore
            <MenuItem
              key={type.name}
              onClick={(e) => {
                e.stopPropagation();
                addHook(JSON.parse(JSON.stringify(type.template))); // Deep clone
              }}
              cursor="pointer"
            >
              {/* @ts-ignore */}
              <MenuItemText>{type.name}</MenuItemText>
            </MenuItem>
          ))}
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
  const handleConditionChange = (details: { value: string[] }) => {
    onChange({ ...hook, conditions: details.value });
  };

  return (
    <Card.Root size="sm" variant="outline" width="full">
      <Card.Header pb={2}>
        <Flex align="center" justify="space-between">
          <Text fontWeight="bold">
            Hook {index + 1}: {typeName}
          </Text>
          <IconButton
            size="xs"
            variant="ghost"
            colorPalette="red"
            onClick={onRemove}
            aria-label="Remove hook"
          >
            <FiTrash2 />
          </IconButton>
        </Flex>
      </Card.Header>
      <Card.Body gap={3}>
        <HookConditionsTooltip>
          <SelectRoot
            multiple
            collection={conditionCollection}
            value={hook.conditions}
            // @ts-ignore
            onValueChange={handleConditionChange}
            size="sm"
          >
            {/* @ts-ignore */}
            <SelectHiddenSelect />
            <SelectTrigger>
              <Box flex="1" textAlign="left">
                {hook.conditions && hook.conditions.length > 0 ? (
                  hook.conditions.join(", ")
                ) : (
                  <Text color="fg.muted">Runs when...</Text>
                )}
              </Box>
            </SelectTrigger>
            <SelectContent zIndex={2000}>
              {conditionCollection.items.map((item: any) => (
                // @ts-ignore
                <SelectItem item={item} key={item.value}>
                  <span>
                    {item.label}
                    {item.description && (
                      <Text as="span" color="fg.muted" ml={2}>
                        - {item.description}
                      </Text>
                    )}
                  </span>
                </SelectItem>
              ))}
            </SelectContent>
          </SelectRoot>
        </HookConditionsTooltip>

        <HookBuilder hook={hook} onChange={onChange} />
      </Card.Body>
    </Card.Root>
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
    name: "Command",
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
            Script:
          </Text>
          <Textarea
            fontFamily="monospace"
            value={hook.actionCommand?.command || ""}
            onChange={(e) => updateCommand(e.target.value)}
            size="sm"
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
            placeholder="Shoutrrr URL"
            value={hook.actionShoutrrr?.shoutrrrUrl || ""}
            onChange={(e) => updateShoutrrr("shoutrrrUrl", e.target.value)}
            size="sm"
          />
          <Text fontSize="sm" mt={1}>
            Text Template:
          </Text>
          <Textarea
            fontFamily="monospace"
            value={hook.actionShoutrrr?.template || ""}
            onChange={(e) => updateShoutrrr("template", e.target.value)}
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
            placeholder="Discord Webhook URL"
            value={hook.actionDiscord?.webhookUrl || ""}
            onChange={(e) => updateDiscord("webhookUrl", e.target.value)}
            size="sm"
          />
          <Text fontSize="sm" mt={1}>
            Text Template:
          </Text>
          <Textarea
            fontFamily="monospace"
            value={hook.actionDiscord?.template || ""}
            onChange={(e) => updateDiscord("template", e.target.value)}
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
            placeholder="Gotify Base URL"
            value={hook.actionGotify?.baseUrl || ""}
            onChange={(e) => updateGotify("baseUrl", e.target.value)}
            size="sm"
          />
          <Input
            placeholder="Gotify Token"
            value={hook.actionGotify?.token || ""}
            onChange={(e) => updateGotify("token", e.target.value)}
            size="sm"
          />
          <Input
            placeholder="Title Template"
            value={hook.actionGotify?.titleTemplate || ""}
            onChange={(e) => updateGotify("titleTemplate", e.target.value)}
            size="sm"
          />
          <Text fontSize="sm" mt={1}>
            Text Template:
          </Text>
          <Textarea
            fontFamily="monospace"
            value={hook.actionGotify?.template || ""}
            onChange={(e) => updateGotify("template", e.target.value)}
            size="sm"
          />
          <SelectRoot
            collection={createListCollection({
              items: [
                { label: "0 - No notification", value: "0" },
                { label: "1 - Icon in notification bar", value: "1" },
                { label: "4 - Icon in notification bar + Sound", value: "4" },
                {
                  label: "8 - Icon in notification bar + Sound + Vibration",
                  value: "8",
                },
              ],
            })}
            value={[String(hook.actionGotify?.priority ?? 5)]}
            // @ts-ignore
            onValueChange={(e) =>
              updateGotify("priority", parseInt(e.value[0]))
            }
            size="sm"
          >
            {/* @ts-ignore */}
            <SelectHiddenSelect />
            <SelectTrigger>
              {/* @ts-ignore */}
              <SelectValueText placeholder="Priority" />
            </SelectTrigger>
            <SelectContent zIndex={2000}>
              {[
                { label: "0 - No notification", value: "0" },
                { label: "1 - Icon in notification bar", value: "1" },
                { label: "4 - Icon in notification bar + Sound", value: "4" },
                {
                  label: "8 - Icon in notification bar + Sound + Vibration",
                  value: "8",
                },
              ].map((item) => (
                // @ts-ignore
                <SelectItem item={item} key={item.value}>
                  {item.label}
                </SelectItem>
              ))}
            </SelectContent>
          </SelectRoot>
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
            placeholder="Slack Webhook URL"
            value={hook.actionSlack?.webhookUrl || ""}
            onChange={(e) => updateSlack("webhookUrl", e.target.value)}
            size="sm"
          />
          <Text fontSize="sm" mt={1}>
            Text Template:
          </Text>
          <Textarea
            fontFamily="monospace"
            value={hook.actionSlack?.template || ""}
            onChange={(e) => updateSlack("template", e.target.value)}
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
            placeholder="Ping URL"
            value={hook.actionHealthchecks?.webhookUrl || ""}
            onChange={(e) => updateHealthchecks("webhookUrl", e.target.value)}
            size="sm"
          />
          <Text fontSize="sm" mt={1}>
            Text Template:
          </Text>
          <Textarea
            fontFamily="monospace"
            value={hook.actionHealthchecks?.template || ""}
            onChange={(e) => updateHealthchecks("template", e.target.value)}
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
            placeholder="Bot Token"
            value={hook.actionTelegram?.botToken || ""}
            onChange={(e) => updateTelegram("botToken", e.target.value)}
            size="sm"
          />
          <Input
            placeholder="Chat ID"
            value={hook.actionTelegram?.chatId || ""}
            onChange={(e) => updateTelegram("chatId", e.target.value)}
            size="sm"
          />
          <Text fontSize="sm" mt={1}>
            Text Template:
          </Text>
          <Textarea
            fontFamily="monospace"
            value={hook.actionTelegram?.template || ""}
            onChange={(e) => updateTelegram("template", e.target.value)}
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
    return <Text>Unknown hook type</Text>;
  }

  for (const hookType of hookTypes) {
    if (hookType.oneofKey in hook) {
      return hookType.component({ hook, onChange });
    }
  }

  return <Text>Unknown hook type</Text>;
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
          Error Behavior:
        </Text>
        <Tooltip
          content={
            <Box>
              <Text fontWeight="bold">What happens when the hook fails</Text>
              <Text fontSize="xs">
                (only effective on start hooks e.g. backup start)
              </Text>
              <Stack gap={1} mt={1} fontSize="xs">
                <Text>• IGNORE - failure is ignored</Text>
                <Text>• FATAL - stops operation with error</Text>
                <Text>• CANCEL - stops operation (cancelled)</Text>
              </Stack>
            </Box>
          }
        >
          <IconButton aria-label="info" size="xs" variant="ghost">
            <FiInfo />
          </IconButton>
        </Tooltip>
      </Flex>
      <SelectRoot
        collection={onErrorCollection}
        value={[hook.onError || ""]}
        // @ts-ignore
        onValueChange={(e) => onChange({ ...hook, onError: e.value[0] })}
        size="sm"
      >
        {/* @ts-ignore */}
        <SelectHiddenSelect />
        <SelectTrigger>
          {/* @ts-ignore */}
          <SelectValueText placeholder="Error behavior..." />
        </SelectTrigger>
        <SelectContent zIndex={2000}>
          {onErrorCollection.items.map((item: any) => (
            // @ts-ignore
            <SelectItem item={item} key={item.value}>
              {item.label}
            </SelectItem>
          ))}
        </SelectContent>
      </SelectRoot>
    </Stack>
  );
};

const HookConditionsTooltip = ({ children }: { children: React.ReactNode }) => {
  return (
    <Tooltip
      content={
        <Box>
          <Text fontWeight="bold">Available conditions</Text>
          <Stack gap={0} fontSize="xs">
            <Text>• CONDITION_ANY_ERROR - error executing any task</Text>
            <Text>
              • CONDITION_SNAPSHOT_START - start of a backup operation
            </Text>
            <Text>• CONDITION_SNAPSHOT_END - end of backup operation</Text>
            <Text>• CONDITION_SNAPSHOT_SUCCESS - end of successful backup</Text>
            <Text>• CONDITION_SNAPSHOT_ERROR - end of failed backup</Text>
            <Text>• CONDITION_SNAPSHOT_WARNING - end of partial backup</Text>
            <Text>• CONDITION_PRUNE_START - start of prune operation</Text>
            <Text>• ... see docs for more</Text>
          </Stack>
        </Box>
      }
    >
      {children}
    </Tooltip>
  );
};
