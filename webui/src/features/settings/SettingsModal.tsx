import {
  Flex,
  Stack,
  Input,
  createListCollection,
  IconButton,
  Text,
  Box,
} from "@chakra-ui/react";
import { useState } from "react";
import { useShowModal } from "../../components/common/ModalManager";
import {
  FiPlus as Plus,
  FiMinus as Minus,
  FiCopy as Copy,
  FiEye,
  FiEyeOff,
  FiSettings,
  FiLock,
  FiGlobe,
} from "react-icons/fi";
import { formatErrorAlert, alerts } from "../../components/common/Alerts";
import { backrestService, authenticationService } from "../../api/client";
import { clone, create, fromJson, toJson } from "@bufbuild/protobuf";
import {
  AuthSchema,
  ConfigSchema,
  UserSchema,
  MultihostSchema,
  Multihost_PeerSchema,
  Multihost_Permission_Type,
} from "../../../gen/ts/v1/config_pb";
import { GeneratePairingTokenRequestSchema } from "../../../gen/ts/v1/service_pb";
import { useSyncStates } from "../../state/peerStates";
import { PeerStateConnectionStatusIcon } from "../../components/common/SyncStateIcon";
import { isMultihostSyncEnabled } from "../../state/buildcfg";
import * as m from "../../paraglide/messages";
import { Button } from "../../components/ui/button";
import { Field } from "../../components/ui/field";
import { PasswordInput } from "../../components/ui/password-input";
import {
  SelectRoot,
  SelectTrigger,
  SelectContent,
  SelectItem,
  SelectValueText,
} from "../../components/ui/select";
import { useConfig } from "../../app/provider";
import { useUserPreferences } from "../../lib/userPreferences";
import {
  TwoPaneModal,
  TwoPaneSection,
  type SectionDef,
} from "../../components/common/TwoPaneModal";
import { SectionCard } from "../../components/common/SectionCard";
import { ToggleField } from "../../components/common/ToggleField";

export const SettingsModal = () => {
  const [config, setConfig] = useConfig();
  const showModal = useShowModal();
  const peerStates = useSyncStates();
  const [confirmLoading, setConfirmLoading] = useState(false);
  const [reloadOnCancel, setReloadOnCancel] = useState(false);

  // Pairing token generation state
  const [showGenerateForm, setShowGenerateForm] = useState(false);
  const [tokenLabel, setTokenLabel] = useState("");
  const [tokenTtl, setTokenTtl] = useState("3600");
  const [tokenMaxUses, setTokenMaxUses] = useState(1);
  const [generatedToken, setGeneratedToken] = useState("");
  const [generateLoading, setGenerateLoading] = useState(false);
  const [initialTokenCount] = useState(
    () => config?.multihost?.pairingTokens?.length || 0,
  );


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

  const [initialFormData, setInitialFormData] = useState(() =>
    JSON.stringify(formData),
  );
  const dirty = JSON.stringify(formData) !== initialFormData;

  const ttlOptions = createListCollection({
    items: [
      { label: "15 minutes", value: "900" },
      { label: "1 hour", value: "3600" },
      { label: "24 hours", value: "86400" },
      { label: "7 days", value: "604800" },
      { label: "Forever", value: "0" },
    ],
  });

  const refreshConfig = async () => {
    const freshConfig = await backrestService.getConfig({});
    setConfig(freshConfig);
  };

  const handleGenerateToken = async () => {
    setGenerateLoading(true);
    try {
      const resp = await backrestService.generatePairingToken(
        create(GeneratePairingTokenRequestSchema, {
          label: tokenLabel,
          ttlSeconds: BigInt(parseInt(tokenTtl)),
          maxUses: tokenMaxUses,
        }),
      );
      setGeneratedToken(resp.token);
      await refreshConfig();
    } catch (e: any) {
      alerts.error(formatErrorAlert(e, "Failed to generate pairing token"));
    } finally {
      setGenerateLoading(false);
    }
  };

  const handleRemovePairingToken = async (index: number) => {
    if (!config) return;
    try {
      const newConfig = clone(ConfigSchema, config);
      if (newConfig.multihost) {
        newConfig.multihost.pairingTokens.splice(index, 1);
      }
      setConfig(await backrestService.setConfig(newConfig));
      alerts.success("Pairing token removed.");
    } catch (e: any) {
      alerts.error(formatErrorAlert(e, "Failed to remove pairing token"));
    }
  };


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
      setInitialFormData(JSON.stringify(formData));
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

  const sections: SectionDef[] = [
    { id: "general", label: "General", icon: <FiSettings size={14} /> },
    { id: "auth", label: "Authentication", icon: <FiLock size={14} /> },
    ...(isMultihostSyncEnabled
      ? [
          {
            id: "multihost",
            label: "Multihost",
            icon: <FiGlobe size={14} />,
          } as SectionDef,
        ]
      : []),
  ];

  return (
    <TwoPaneModal
      isOpen={true}
      onClose={handleCancel}
      title={m.settings_modal_title()}
      headerIcon={<FiSettings size={14} />}
      sections={sections}
      dirty={dirty}
      dirtyCount={1}
      onSave={handleOk}
      onDiscard={() => {
        setFormData(JSON.parse(initialFormData));
      }}
      saving={confirmLoading}
    >
      {/* General Section */}
      <TwoPaneSection id="general">
        <SectionCard
          icon={<FiSettings size={16} />}
          title="General"
          description="Instance identity and display preferences."
        >
          <Stack gap={4}>
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

            <UserSettingsForm />
          </Stack>
        </SectionCard>
      </TwoPaneSection>

      {/* Authentication Section */}
      <TwoPaneSection id="auth">
        <SectionCard
          icon={<FiLock size={16} />}
          title={m.settings_section_authentication()}
          description="User accounts and access control."
        >
          <Stack gap={4}>
            <ToggleField
              checked={getField(["auth", "disabled"]) || false}
              onChange={(v) => updateField(["auth", "disabled"], v)}
              label={m.settings_auth_disable()}
              hint="When disabled, no login is required to access Backrest."
            />

            <Field label={m.settings_auth_users()} required>
              <Stack gap={3} width="full">
                {users.map((user: any, index: number) => (
                  <Flex key={index} gap={2} align="center" width="full">
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
                      rootProps={{ flex: 1 }}
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
                  width="full"
                >
                  <Plus /> {m.settings_auth_add_user()}
                </Button>
              </Stack>
            </Field>
          </Stack>
        </SectionCard>
      </TwoPaneSection>

      {/* Multihost Section */}
      {isMultihostSyncEnabled && (
        <TwoPaneSection id="multihost">
          <SectionCard
            icon={<FiGlobe size={16} />}
            title={m.settings_section_multihost()}
            description="Peer-to-peer synchronisation between Backrest instances."
          >
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
                <Flex gap={2} width="full">
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

              {/* Pairing Tokens */}
              <Field
                label="Pairing Tokens"
                helperText="Tokens that can be shared with other Backrest instances to simplify peering."
                width="full"
              >
                <Stack gap={3} width="full">
                  {(config.multihost?.pairingTokens || []).map(
                    (token, index) => (
                      <PairingTokenItem
                        key={index}
                        token={token}
                        isNew={index >= initialTokenCount}
                        generatedTokenString={
                          index >= initialTokenCount ? generatedToken : undefined
                        }
                        config={config}
                        onRemove={() => handleRemovePairingToken(index)}
                      />
                    ),
                  )}

                  {showGenerateForm && (
                    <Box p={4} borderWidth="1px" borderRadius="md">
                      <Stack gap={3}>
                        <Field label="Label (optional)">
                          <Input
                            value={tokenLabel}
                            onChange={(e) => setTokenLabel(e.target.value)}
                            placeholder="e.g. laptop-2"
                            width="full"
                          />
                        </Field>
                        <Field label="TTL">
                          <SelectRoot
                            collection={ttlOptions}
                            value={[tokenTtl]}
                            onValueChange={(e: any) =>
                              setTokenTtl(e.value[0])
                            }
                          >
                            {/* @ts-ignore */}
                            <SelectTrigger>
                              {/* @ts-ignore */}
                              <SelectValueText placeholder="Select TTL" />
                            </SelectTrigger>
                            {/* @ts-ignore */}
                            <SelectContent zIndex={2000}>
                              {ttlOptions.items.map((o: any) => (
                                <SelectItem item={o} key={o.value}>
                                  {o.label}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </SelectRoot>
                        </Field>
                        <Field label="Max Uses" helperText="0 = unlimited">
                          <Input
                            type="number"
                            value={tokenMaxUses}
                            onChange={(e) =>
                              setTokenMaxUses(parseInt(e.target.value) || 0)
                            }
                            min={0}
                            width="full"
                          />
                        </Field>
                        <Flex gap={2}>
                          <Button
                            size="sm"
                            onClick={handleGenerateToken}
                            loading={generateLoading}
                          >
                            Generate
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => {
                              setShowGenerateForm(false);
                              setGeneratedToken("");
                            }}
                          >
                            Cancel
                          </Button>
                        </Flex>
                      </Stack>
                    </Box>
                  )}

                  {!showGenerateForm && (
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => {
                        setShowGenerateForm(true);
                        setGeneratedToken("");
                        setTokenLabel("");
                        setTokenTtl("3600");
                        setTokenMaxUses(1);
                      }}
                      width="full"
                    >
                      <Plus /> Generate Pairing Token
                    </Button>
                  )}
                </Stack>
              </Field>

              <Field
                label={m.settings_multihost_authorized_clients()}
                helperText={m.settings_multihost_authorized_clients_tooltip()}
                width="full"
              >
                <PeerFormList
                  items={getField(["multihost", "authorizedClients"]) || []}
                  onUpdate={(items: any) =>
                    updateField(["multihost", "authorizedClients"], items)
                  }
                  peerStates={peerStates}
                  config={config}
                  showInstanceUrl={false}
                />
              </Field>

              <Field
                label={m.settings_multihost_known_hosts()}
                helperText={m.settings_multihost_known_hosts_tooltip()}
                width="full"
              >
                <KnownHostsList
                  items={getField(["multihost", "knownHosts"]) || []}
                  onUpdate={(items: any) =>
                    updateField(["multihost", "knownHosts"], items)
                  }
                  peerStates={peerStates}
                  config={config}
                />
              </Field>
            </Stack>
          </SectionCard>
        </TwoPaneSection>
      )}
    </TwoPaneModal>
  );
};

// --- Pairing Token Item ---

const PairingTokenItem = ({
  token,
  isNew,
  generatedTokenString,
  config,
  onRemove,
}: {
  token: any;
  isNew: boolean;
  generatedTokenString?: string;
  config: any;
  onRemove: () => void;
}) => {
  const [showToken, setShowToken] = useState(isNew);

  // Build the full token string: <keyid>:<secret>#<instanceid>
  const fullTokenString =
    generatedTokenString ||
    `${config.multihost?.identity?.keyid || ""}:${token.secret || ""}#${config.instance || ""}`;

  const isExpired =
    token.expiresAtUnix > 0n &&
    token.expiresAtUnix < BigInt(Math.floor(Date.now() / 1000));
  const usesText =
    token.maxUses === 0
      ? `${token.uses} uses (unlimited)`
      : `${token.uses}/${token.maxUses} uses`;
  const expiryText =
    token.expiresAtUnix === 0n
      ? "Never expires"
      : isExpired
        ? `Expired ${new Date(Number(token.expiresAtUnix) * 1000).toLocaleString()}`
        : `Expires ${new Date(Number(token.expiresAtUnix) * 1000).toLocaleString()}`;

  return (
    <Box p={3} borderWidth="1px" borderRadius="md">
      <Flex justify="space-between" align="center" width="full">
        <Stack gap={0}>
          <Text fontSize="sm" fontWeight="medium">
            {token.label || "(no label)"}
          </Text>
          <Text fontSize="xs" color={isExpired ? "red.500" : "gray.500"}>
            {expiryText} -- {usesText}
          </Text>
        </Stack>
        <Flex gap={1} align="center">
          <IconButton
            size="xs"
            variant="ghost"
            onClick={() => setShowToken(!showToken)}
            aria-label={showToken ? "Hide token" : "Show token"}
          >
            {showToken ? <FiEyeOff size={14} /> : <FiEye size={14} />}
          </IconButton>
          <IconButton
            size="xs"
            variant="ghost"
            onClick={onRemove}
            aria-label="Remove token"
          >
            <Minus />
          </IconButton>
        </Flex>
      </Flex>
      {showToken && (
        <Flex gap={2} mt={2} width="full">
          <Input value={fullTokenString} readOnly flex={1} size="sm" />
          <IconButton
            size="sm"
            variant="outline"
            onClick={() => navigator.clipboard.writeText(fullTokenString)}
            aria-label="Copy token"
          >
            <Copy />
          </IconButton>
        </Flex>
      )}
    </Box>
  );
};

// --- Known Hosts List (with integrated pairing) ---

const KnownHostsList = ({
  items,
  onUpdate,
  peerStates,
  config,
}: any) => {
  const [showAddForm, setShowAddForm] = useState(false);
  const [pairToken, setPairToken] = useState("");
  const [pairInstanceUrl, setPairInstanceUrl] = useState("");

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

  const handleAdd = () => {
    try {
      if (!pairToken.trim()) {
        onUpdate([
          ...items,
          {
            instanceId: "",
            keyId: "",
            instanceUrl: pairInstanceUrl,
            permissions: [
              {
                type: Multihost_Permission_Type.PERMISSION_READ_OPERATIONS,
                scopes: ["*"],
              },
            ],
          },
        ]);
        setShowAddForm(false);
        setPairToken("");
        setPairInstanceUrl("");
        return;
      }

      const hashIdx = pairToken.indexOf("#");
      const colonIdx = pairToken.indexOf(":");
      if (hashIdx === -1 || colonIdx === -1 || colonIdx > hashIdx) {
        throw new Error(
          'Invalid token format. Expected "<keyid>:<secret>#<instanceid>"',
        );
      }
      const keyId = pairToken.substring(0, colonIdx);
      const secret = pairToken.substring(colonIdx + 1, hashIdx);
      const instanceId = pairToken.substring(hashIdx + 1);

      if (!keyId || !secret || !instanceId) {
        throw new Error("Token is missing required fields");
      }
      if (!pairInstanceUrl) {
        throw new Error("Instance URL is required");
      }

      onUpdate([
        ...items,
        {
          instanceId,
          keyId,
          instanceUrl: pairInstanceUrl,
          initialPairingSecret: secret,
          permissions: [
            {
              type: Multihost_Permission_Type.PERMISSION_READ_OPERATIONS,
              scopes: ["*"],
            },
          ],
        },
      ]);

      setPairToken("");
      setPairInstanceUrl("");
      setShowAddForm(false);
      alerts.success("Server added to known hosts. Save settings to apply.");
    } catch (e: any) {
      alerts.error(formatErrorAlert(e, "Failed to add known host"));
    }
  };

  return (
    <Stack gap={4} width="full">
      {items.map((item: any, index: number) => (
        <PeerFormListItem
          key={index}
          item={item}
          onChange={(val: any) => handleItemUpdate(index, val)}
          onRemove={() => handleRemove(index)}
          peerStates={peerStates}
          showInstanceUrl={true}
          config={config}
        />
      ))}

      {showAddForm ? (
        <Box p={4} borderWidth="1px" borderRadius="md">
          <Stack gap={3}>
            <Text fontSize="sm" color="gray.500">
              Paste a pairing token from another Backrest server, or leave blank
              to configure manually.
            </Text>
            <Field label="Pairing Token (optional)">
              <Input
                value={pairToken}
                onChange={(e) => setPairToken(e.target.value)}
                placeholder='<keyid>:<secret>#<instanceid>'
                width="full"
              />
            </Field>
            <Field label="Instance URL" required>
              <Input
                value={pairInstanceUrl}
                onChange={(e) => setPairInstanceUrl(e.target.value)}
                placeholder="e.g. http://server:9898"
                width="full"
              />
            </Field>
            <Flex gap={2}>
              <Button size="sm" onClick={handleAdd}>
                {pairToken.trim() ? "Pair" : "Add"}
              </Button>
              <Button
                size="sm"
                variant="outline"
                onClick={() => {
                  setShowAddForm(false);
                  setPairToken("");
                  setPairInstanceUrl("");
                }}
              >
                Cancel
              </Button>
            </Flex>
          </Stack>
        </Box>
      ) : (
        <Button
          size="sm"
          variant="outline"
          onClick={() => setShowAddForm(true)}
          width="full"
        >
          <Plus />{" "}
          {m.settings_peer_add_button({
            itemTypeName: m.settings_multihost_known_host_item(),
          })}
        </Button>
      )}
    </Stack>
  );
};

// --- Peer Sub-components ---

const PeerFormList = ({
  items,
  onUpdate,
  peerStates,
  config,
  showInstanceUrl,
}: any) => {
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
    <Stack gap={4} width="full">
      {items.length === 0 && (
        <Text fontSize="sm" color="fg.muted" fontStyle="italic">
          No trusted peers yet. Generate a pairing token above and share it with
          another instance to get started.
        </Text>
      )}
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
      {
        label: "Receive shared repos",
        value: Multihost_Permission_Type.PERMISSION_RECEIVE_SHARED_REPOS.toString(),
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
                <SelectContent zIndex={2000}>
                  {permissionTypeOptions.items.map((o: any) => (
                    <SelectItem item={o} key={o.value}>
                      {o.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </SelectRoot>
            </Field>

            {perm.type !== Multihost_Permission_Type.PERMISSION_RECEIVE_SHARED_REPOS && (
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
                  <SelectContent zIndex={2000}>
                    {repoOptions.items.map((o: any) => (
                      <SelectItem item={o} key={o.value}>
                        {o.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </SelectRoot>
              </Field>
            )}

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

const languageNames: Record<string, string> = {
  en: "English",
  de: "Deutsch",
  zh: "中文",
  hi: "हिन्दी",
  es: "Español",
  ar: "العربية",
  fr: "Français",
  bn: "বাংলা",
  pt: "Português",
  ru: "Русский",
  id: "Bahasa Indonesia",
  it: "Italiano",
  ja: "日本語",
};

const UserSettingsForm = () => {
  const { preferences, updatePreference, availableLanguages } =
    useUserPreferences();

  const languageOptions = createListCollection({
    items: availableLanguages.map((tag: string) => ({
      label: languageNames[tag] || tag,
      value: tag,
    })),
  });

  return (
    <Field
      label={
        // @ts-ignore
        m.settings_field_language
          ? m.settings_field_language()
          : "Display Language"
      }
    >
      <SelectRoot
        collection={languageOptions}
        value={[preferences.language]}
        onValueChange={(e: any) => updatePreference("language", e.value[0])}
      >
        {/* @ts-ignore */}
        <SelectTrigger>
          {/* @ts-ignore */}
          <SelectValueText placeholder="Select language" />
        </SelectTrigger>
        {/* @ts-ignore */}
        <SelectContent zIndex={2000}>
          {languageOptions.items.map((option: any) => (
            <SelectItem item={option} key={option.value}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </SelectRoot>
    </Field>
  );
};
