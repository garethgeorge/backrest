import {
  Flex,
  Stack,
  Input,
  Textarea,
  createListCollection,
  IconButton,
  Card,
  Heading,
  Text,
  Box,
} from "@chakra-ui/react";
import { Checkbox } from "../../components/ui/checkbox";
import React, { useEffect, useState, useMemo } from "react";
import { useShowModal } from "../../components/common/ModalManager";
import {
  FiPlus as Plus,
  FiMinus as Minus,
  FiCopy as Copy,
} from "react-icons/fi";
import { formatErrorAlert, alerts } from "../../components/common/Alerts";
import { namePattern } from "../../lib/util";
import { backrestService, authenticationService } from "../../api/client";
import { clone, fromJson, toJson } from "@bufbuild/protobuf";
import {
  AuthSchema,
  Config,
  ConfigSchema,
  UserSchema,
  MultihostSchema,
  Multihost_PeerSchema,
  Multihost_Permission_Type,
} from "../../../gen/ts/v1/config_pb";
import { PeerState } from "../../../gen/ts/v1sync/syncservice_pb";
import { useSyncStates } from "../../state/peerStates";
import { PeerStateConnectionStatusIcon } from "../../components/common/SyncStateIcon";
import { isMultihostSyncEnabled } from "../../state/buildcfg";
import * as m from "../../paraglide/messages";
import { FormModal } from "../../components/common/FormModal";
import { Button } from "../../components/ui/button";
import { Field } from "../../components/ui/field";
import { Tooltip } from "../../components/ui/tooltip";
import { PasswordInput } from "../../components/ui/password-input";
import {
  AccordionRoot,
  AccordionItem,
  AccordionItemTrigger,
  AccordionItemContent,
} from "../../components/ui/accordion";
import {
  SelectRoot,
  SelectTrigger,
  SelectContent,
  SelectItem,
  SelectValueText,
  SelectLabel,
} from "../../components/ui/select";
import { useConfig } from "../../app/provider";

interface FormData {
  auth: {
    disabled?: boolean;
    users: {
      name: string;
      passwordBcrypt: string;
      needsBcrypt?: boolean;
      isExisting?: boolean;
    }[];
  };
  instance: string;
  multihost: {
    identity: {
      keyid: string;
    };
    knownHosts: any[];
    authorizedClients: any[];
  };
}

export const SettingsModal = () => {
  const [config, setConfig] = useConfig();
  const showModal = useShowModal();
  const peerStates = useSyncStates();
  const [confirmLoading, setConfirmLoading] = useState(false);
  const [reloadOnCancel, setReloadOnCancel] = useState(false);

  // Local state initialized from config
  const [formData, setFormData] = useState<any>(() => {
    if (!config) return null;
    return {
      instance: config.instance || "",
      auth: {
        disabled: config.auth?.disabled || false,
        users:
          config.auth?.users?.map((u: any) => ({
            ...(toJson(UserSchema, u, { alwaysEmitImplicit: true }) as any),
            isExisting: true,
          })) || [],
      },
      multihost: {
        identity: { keyid: config.multihost?.identity?.keyid || "" },
        knownHosts:
          config.multihost?.knownHosts?.map((peer: any) =>
            toJson(Multihost_PeerSchema, peer, { alwaysEmitImplicit: true }),
          ) || [],
        authorizedClients:
          config.multihost?.authorizedClients?.map((peer: any) =>
            toJson(Multihost_PeerSchema, peer, { alwaysEmitImplicit: true }),
          ) || [],
      },
    };
  });

  if (!config || !formData) return null;

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

  const handleOk = async () => {
    setConfirmLoading(true);
    try {
      const workingData = JSON.parse(JSON.stringify(formData));

      // Hash passwords if needed
      if (workingData.auth?.users) {
        for (const user of workingData.auth.users) {
          if (user.needsBcrypt) {
            const hash = await authenticationService.hashPassword({
              value: user.passwordBcrypt,
            });
            user.passwordBcrypt = hash.value;
            delete user.needsBcrypt;
          }
          delete user.isExisting;
        }
      }

      // Update configuration
      let newConfig = clone(ConfigSchema, config);
      newConfig.auth = fromJson(AuthSchema, workingData.auth, {
        ignoreUnknownFields: false,
      });
      newConfig.multihost = fromJson(MultihostSchema, workingData.multihost, {
        ignoreUnknownFields: false,
      });
      newConfig.instance = workingData.instance;

      if (!newConfig.auth?.users && !newConfig.auth?.disabled) {
        throw new Error(
          "At least one user must be configured or authentication must be disabled",
        );
      }

      setConfig(await backrestService.setConfig(newConfig));
      setReloadOnCancel(true);
      alerts.success(m.settings_success_updated());
    } catch (e: any) {
      alerts.error(formatErrorAlert(e, m.settings_error_operation()));
    } finally {
      setConfirmLoading(false);
    }
  };

  const handleCancel = () => {
    showModal(null);
    if (reloadOnCancel) {
      window.location.reload();
    }
  };

  const users = getField(["auth", "users"]) || [];

  return (
    <FormModal
      isOpen={true}
      onClose={handleCancel}
      title="Settings"
      size="large"
      footer={
        <Flex gap={2} justify="flex-end" width="full">
          <Button variant="outline" onClick={handleCancel}>
            {m.button_close()}
          </Button>
          <Button loading={confirmLoading} onClick={handleOk}>
            {m.button_save()}
          </Button>
        </Flex>
      }
    >
      <Stack gap={6}>
        {users.length === 0 && !getField(["auth", "disabled"]) && (
          <Alert status="warning">
            <Stack gap={1}>
              <strong>{m.settings_initial_setup_title()}</strong>
              <Text fontSize="sm">{m.settings_initial_setup_message()}</Text>
              <Text fontSize="xs" fontStyle="italic">
                {m.settings_initial_setup_hint()}
              </Text>
            </Stack>
          </Alert>
        )}

        <Field
          label={m.settings_field_instance_id()}
          helperText={m.settings_field_instance_id_tooltip()}
          required
        >
          <Input
            value={getField(["instance"])}
            onChange={(e) => updateField(["instance"], e.target.value)}
            disabled={!!config.instance}
            placeholder={m.settings_field_instance_id_placeholder()}
          />
        </Field>

        {/* @ts-ignore */}
        <AccordionRoot collapsible defaultValue={["auth"]}>
          {/* Authentication Section */}
          {/* @ts-ignore */}
          <AccordionItem value="auth">
            <AccordionItemTrigger>
              {m.settings_section_authentication()}
            </AccordionItemTrigger>
            <AccordionItemContent>
              <Stack gap={4}>
                <Field label={m.settings_auth_disable()}>
                  <Checkbox
                    checked={getField(["auth", "disabled"])}
                    onCheckedChange={(e: any) =>
                      updateField(["auth", "disabled"], !!e.checked)
                    }
                  >
                    {m.settings_auth_disable()}
                  </Checkbox>
                </Field>

                <Field label={m.settings_auth_users()} required>
                  <Stack gap={3}>
                    {users.map((user: any, index: number) => (
                      <Flex key={index} gap={2} align="center">
                        <Input
                          placeholder={m.settings_auth_username_placeholder()}
                          value={user.name}
                          onChange={(e) => {
                            const newUsers = [...users];
                            newUsers[index].name = e.target.value;
                            updateField(["auth", "users"], newUsers);
                          }}
                          disabled={user.isExisting}
                          flex={1}
                        />
                        <PasswordInput
                          placeholder={m.settings_auth_password_placeholder()}
                          value={user.passwordBcrypt}
                          onChange={(e) => {
                            const newUsers = [...users];
                            newUsers[index].passwordBcrypt = e.target.value;
                            newUsers[index].needsBcrypt = true;
                            updateField(["auth", "users"], newUsers);
                          }}
                          flex={1}
                        />
                        <IconButton
                          size="sm"
                          variant="ghost"
                          aria-label="Remove"
                          onClick={() => {
                            const newUsers = [...users];
                            newUsers.splice(index, 1);
                            updateField(["auth", "users"], newUsers);
                          }}
                        >
                          <Minus />
                        </IconButton>
                      </Flex>
                    ))}
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => {
                        updateField(
                          ["auth", "users"],
                          [
                            ...users,
                            {
                              name: "",
                              passwordBcrypt: "",
                              needsBcrypt: true,
                              isExisting: false,
                            },
                          ],
                        );
                      }}
                    >
                      <Plus /> {m.settings_auth_add_user()}
                    </Button>
                  </Stack>
                </Field>
              </Stack>
            </AccordionItemContent>
          </AccordionItem>

          {/* Multihost Section */}
          {isMultihostSyncEnabled && (
            // @ts-ignore
            <AccordionItem value="multihost">
              <AccordionItemTrigger>
                {m.settings_section_multihost()}
              </AccordionItemTrigger>
              <AccordionItemContent>
                <Stack gap={4}>
                  <Text fontStyle="italic" fontSize="sm">
                    {m.settings_multihost_intro()}
                  </Text>
                  <Text fontStyle="italic" fontSize="sm" color="red.500">
                    {m.settings_multihost_warning()}
                  </Text>

                  <Field
                    label={m.settings_multihost_identity()}
                    helperText={m.settings_multihost_identity_tooltip()}
                  >
                    <Flex gap={2}>
                      <Input
                        value={getField(["multihost", "identity", "keyid"])}
                        disabled
                        flex={1}
                      />
                      <IconButton
                        size="sm"
                        variant="outline"
                        onClick={() =>
                          navigator.clipboard.writeText(
                            getField(["multihost", "identity", "keyid"]) || "",
                          )
                        }
                        aria-label="Copy"
                      >
                        <Copy />
                      </IconButton>
                    </Flex>
                  </Field>

                  <Field
                    label={m.settings_multihost_authorized_clients()}
                    helperText={m.settings_multihost_authorized_clients_tooltip()}
                  >
                    <PeerFormList
                      items={getField(["multihost", "authorizedClients"]) || []}
                      onUpdate={(items: any) =>
                        updateField(["multihost", "authorizedClients"], items)
                      }
                      itemTypeName={m.settings_multihost_authorized_client_item()}
                      peerStates={peerStates}
                      config={config}
                      showInstanceUrl={false}
                    />
                  </Field>

                  <Field
                    label={m.settings_multihost_known_hosts()}
                    helperText={m.settings_multihost_known_hosts_tooltip()}
                  >
                    <PeerFormList
                      items={getField(["multihost", "knownHosts"]) || []}
                      onUpdate={(items: any) =>
                        updateField(["multihost", "knownHosts"], items)
                      }
                      itemTypeName={m.settings_multihost_known_host_item()}
                      peerStates={peerStates}
                      config={config}
                      showInstanceUrl={true}
                    />
                  </Field>
                </Stack>
              </AccordionItemContent>
            </AccordionItem>
          )}

          {/* Preview Section */}
          {/* @ts-ignore */}
          <AccordionItem value="preview">
            <AccordionItemTrigger>
              {m.settings_section_preview()}
            </AccordionItemTrigger>
            <AccordionItemContent>
              <Box
                as="pre"
                p={2}
                bg="gray.900"
                color="white"
                borderRadius="md"
                fontSize="xs"
                overflowX="auto"
              >
                {JSON.stringify(formData, null, 2)}
              </Box>
            </AccordionItemContent>
          </AccordionItem>
        </AccordionRoot>
      </Stack>
    </FormModal>
  );
};

// --- Peer Sub-components ---

const PeerFormList = ({
  items,
  onUpdate,
  itemTypeName,
  peerStates,
  config,
  showInstanceUrl,
}: any) => {
  const handleAdd = () => {
    onUpdate([
      ...items,
      { instanceId: "", keyId: "", instanceUrl: "", permissions: [] },
    ]);
  };

  const handleRemove = (index: number) => {
    const next = [...items];
    next.splice(index, 1);
    onUpdate(next);
  };

  const handleItemUpdate = (index: number, val: any) => {
    const next = [...items];
    next[index] = val;
    onUpdate(next);
  };

  return (
    <Stack gap={4}>
      {items.map((item: any, index: number) => (
        <PeerFormListItem
          key={index}
          item={item}
          onChange={(val: any) => handleItemUpdate(index, val)}
          onRemove={() => handleRemove(index)}
          peerStates={peerStates}
          showInstanceUrl={showInstanceUrl}
          config={config}
        />
      ))}
      <Button size="sm" variant="outline" onClick={handleAdd} width="full">
        <Plus />{" "}
        {m.settings_peer_add_button({
          itemTypeName: itemTypeName || m.peer_default_name(),
        })}
      </Button>
    </Stack>
  );
};

const PeerFormListItem = ({
  item,
  onChange,
  onRemove,
  peerStates,
  showInstanceUrl,
  config,
}: any) => {
  const peerState = peerStates.find(
    (state: any) => state.peerKeyid === item.keyId,
  );

  const updateItem = (field: string, val: any) => {
    onChange({ ...item, [field]: val });
  };

  return (
    <Box p={4} borderWidth="1px" borderRadius="md" position="relative">
      <Flex position="absolute" top={2} right={2} gap={2} align="center">
        {peerState && <PeerStateConnectionStatusIcon peerState={peerState} />}
        <IconButton
          size="xs"
          variant="ghost"
          onClick={onRemove}
          aria-label="Remove"
        >
          <Minus />
        </IconButton>
      </Flex>

      <Stack gap={3}>
        <Flex gap={4}>
          <Field label={m.settings_peer_instance_id()} required flex={1}>
            <Input
              value={item.instanceId}
              onChange={(e) => updateItem("instanceId", e.target.value)}
              placeholder={m.settings_peer_instance_id_placeholder()}
            />
          </Field>
          <Field label={m.settings_peer_key_id()} required flex={1.2}>
            <Input
              value={item.keyId}
              onChange={(e) => updateItem("keyId", e.target.value)}
              placeholder={m.settings_peer_key_id_placeholder()}
            />
          </Field>
        </Flex>

        {showInstanceUrl && (
          <Field label={m.settings_peer_instance_url()} required>
            <Input
              value={item.instanceUrl}
              onChange={(e) => updateItem("instanceUrl", e.target.value)}
              placeholder={m.settings_peer_instance_url_placeholder()}
            />
          </Field>
        )}

        {/* Permissions (Only for known hosts logic in original? No, original had isKnownHost? logic) */}
        {/* PeerPermissionsTile logic */}
        <PeerPermissionsTile
          permissions={item.permissions || []}
          onUpdate={(perms: any) => updateItem("permissions", perms)}
          config={config}
        />
      </Stack>
    </Box>
  );
};

const PeerPermissionsTile = ({ permissions, onUpdate, config }: any) => {
  const repoOptions = createListCollection({
    items: [
      { label: m.settings_permission_scope_all(), value: "*" },
      ...(config.repos || []).map((repo: any) => ({
        label: repo.id,
        value: `repo:${repo.id}`,
      })),
    ],
  });

  const permissionTypeOptions = createListCollection({
    items: [
      {
        label: m.settings_permission_edit_repo(),
        value:
          Multihost_Permission_Type.PERMISSION_READ_WRITE_CONFIG.toString(),
      },
      {
        label: m.settings_permission_read_ops(),
        value: Multihost_Permission_Type.PERMISSION_READ_OPERATIONS.toString(),
      },
    ],
  });

  const handleAdd = () => {
    onUpdate([
      ...permissions,
      {
        type: Multihost_Permission_Type.PERMISSION_READ_OPERATIONS,
        scopes: ["*"],
      },
    ]);
  };

  const handleRemove = (index: number) => {
    const next = [...permissions];
    next.splice(index, 1);
    onUpdate(next);
  };

  const handleUpdate = (index: number, field: string, val: any) => {
    const next = [...permissions];
    next[index] = { ...next[index], [field]: val };
    onUpdate(next);
  };

  return (
    <Stack gap={2}>
      <Text fontWeight="bold" fontSize="sm">
        {m.settings_peer_permissions()}
      </Text>
      {permissions.map((perm: any, index: number) => (
        <Box
          key={index}
          p={3}
          borderWidth="1px"
          borderRadius="sm"
          bg="gray.50"
          _dark={{ bg: "gray.800" }}
        >
          <Flex gap={2} align="flex-end">
            <Field label={m.settings_peer_permission_type()} flex={1}>
              <SelectRoot
                collection={permissionTypeOptions}
                value={[perm.type.toString()]}
                onValueChange={(e: any) =>
                  handleUpdate(index, "type", parseInt(e.value[0]))
                }
              >
                {/* @ts-ignore */}
                <SelectTrigger>
                  {/* @ts-ignore */}
                  <SelectValueText
                    placeholder={m.settings_permission_type_placeholder()}
                  />
                </SelectTrigger>
                {/* @ts-ignore */}
                <SelectContent>
                  {permissionTypeOptions.items.map((o: any) => (
                    <SelectItem item={o} key={o.value}>
                      {o.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </SelectRoot>
            </Field>

            <Field label={m.settings_peer_permission_scopes()} flex={1}>
              <SelectRoot
                multiple
                collection={repoOptions}
                value={perm.scopes}
                onValueChange={(e: any) =>
                  handleUpdate(index, "scopes", e.value)
                }
              >
                {/* @ts-ignore */}
                <SelectTrigger>
                  {/* @ts-ignore */}
                  <SelectValueText
                    placeholder={m.settings_permission_scope_placeholder()}
                  />
                </SelectTrigger>
                {/* @ts-ignore */}
                <SelectContent>
                  {repoOptions.items.map((o: any) => (
                    <SelectItem item={o} key={o.value}>
                      {o.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </SelectRoot>
            </Field>

            <IconButton
              size="sm"
              variant="ghost"
              onClick={() => handleRemove(index)}
              aria-label="Remove Permission"
            >
              <Minus size={14} />
            </IconButton>
          </Flex>
        </Box>
      ))}
      <Button
        size="xs"
        variant="ghost"
        onClick={handleAdd}
        justifyContent="start"
      >
        <Plus size={14} /> {m.settings_peer_add_permission()}
      </Button>
    </Stack>
  );
};

// Mock Alert component if needed or use toast
const Alert = ({ status, children }: any) => (
  <Box
    p={4}
    borderRadius="md"
    bg={status === "warning" ? "orange.100" : "blue.100"}
    color={status === "warning" ? "orange.800" : "blue.800"}
    _dark={{ bg: "orange.900", color: "orange.200" }}
  >
    {children}
  </Box>
);
