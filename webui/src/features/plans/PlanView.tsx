import React, { useCallback, useEffect, useState } from "react";
import { Plan } from "../../../gen/ts/v1/config_pb";
import { Button } from "../../components/ui/button";
import { Flex, Heading, Text, Box, Group, IconButton } from "@chakra-ui/react";
import { FiChevronDown } from "react-icons/fi";
import {
  Tabs,
  TabsList,
  TabsTrigger,
  TabsContent,
  TabsRoot,
} from "../../components/ui/tabs";
import {
  MenuContent,
  MenuItem,
  MenuRoot,
  MenuTrigger,
} from "../../components/ui/menu";
import { Tooltip } from "../../components/ui/tooltip";
import { alerts } from "../../components/common/Alerts";
import { MAX_OPERATION_HISTORY } from "../../constants";
import { backrestService } from "../../api/client";
import {
  ClearHistoryRequestSchema,
  DoRepoTaskRequest_Task,
  DoRepoTaskRequestSchema,
  GetOperationsRequestSchema,
  BackupRequestSchema,
} from "../../../gen/ts/v1/service_pb";
import { SpinButton } from "../../components/common/SpinButton";
import { useShowModal } from "../../components/common/ModalManager";
import { create } from "@bufbuild/protobuf";
import { useConfig } from "../../app/provider";
import { OperationListView } from "../operations/OperationListView";
import { OperationTreeView } from "../operations/OperationTreeView";
import {
  DisplayType,
  FlowDisplayInfo,
  getTypeForDisplay,
} from "../../api/flowDisplayAggregator";
import { Operation } from "../../../gen/ts/v1/operations_pb";
import { OperationFilter } from "../operations/OperationFilter";
import * as m from "../../paraglide/messages";

const FILTER_TYPES = [
  DisplayType.BACKUP,
  DisplayType.BACKUP_DRYRUN,
  DisplayType.SNAPSHOT,
  DisplayType.FORGET,
  DisplayType.PRUNE,
  DisplayType.CHECK,
  DisplayType.RESTORE,
  DisplayType.STATS,
  DisplayType.RUNHOOK,
  DisplayType.RUNCOMMAND,
];

export const PlanView = ({ plan }: React.PropsWithChildren<{ plan: Plan }>) => {
  const [config, _] = useConfig();
  const showModal = useShowModal();
  const repo = config?.repos.find((r) => r.id === plan.repo);

  const [activeTypes, setActiveTypes] = useState<Set<DisplayType>>(
    new Set(FILTER_TYPES),
  );

  const toggleType = (type: DisplayType) => {
    setActiveTypes((prev) => {
      const next = new Set(prev);
      if (next.has(type)) {
        next.delete(type);
      } else {
        next.add(type);
      }
      return next;
    });
  };

  const selectAll = () => {
    setActiveTypes(new Set(FILTER_TYPES));
  };

  const selectNone = () => {
    setActiveTypes(new Set());
  };

  const treeFilter = useCallback(
    (flow: FlowDisplayInfo) => activeTypes.has(flow.type),
    [activeTypes],
  );

  const listFilter = useCallback(
    (op: Operation) => activeTypes.has(getTypeForDisplay(op)),
    [activeTypes],
  );

  const handleBackupNow = async () => {
    try {
      await backrestService.backup(
        create(BackupRequestSchema, { value: plan.id }),
      );
      alerts.success(m.plan_backup_scheduled());
    } catch (e: any) {
      alerts.error(m.plan_error_backup() + e.message);
    }
  };

  const handleDryRunBackup = async () => {
    try {
      await backrestService.backup(
        create(BackupRequestSchema, { value: plan.id, dryRun: true }),
      );
      alerts.success(m.plan_dry_run_scheduled());
    } catch (e: any) {
      alerts.error(m.plan_dry_run_error() + e.message);
    }
  };

  const handleUnlockNow = async () => {
    try {
      alerts.info(m.repo_info_unlocking());
      await backrestService.doRepoTask(
        create(DoRepoTaskRequestSchema, {
          repoId: plan.repo!,
          task: DoRepoTaskRequest_Task.UNLOCK,
        }),
      );
      alerts.success(m.repo_success_unlocked());
    } catch (e: any) {
      alerts.error(m.repo_error_unlock() + e.message);
    }
  };

  const handleClearErrorHistory = async () => {
    try {
      alerts.info(m.plan_clearing_history());
      await backrestService.clearHistory(
        create(ClearHistoryRequestSchema, {
          selector: {
            planId: plan.id,
            repoGuid: repo!.guid,
            originalInstanceKeyid: "",
          },
          onlyFailed: true,
        }),
      );
      alerts.success(m.plan_history_cleared());
    } catch (e: any) {
      alerts.error(m.plan_error_clear_history() + e.message);
    }
  };

  if (!repo) {
    return (
      <Heading size="lg" color="red.500">
        {m.plan_repo_not_found({ repo: plan.repo!, planId: plan.id! })}
      </Heading>
    );
  }

  return (
    <Box>
      <Flex gap={4} align="center" wrap="wrap" mb={4}>
        <Heading size="xl">{plan.id}</Heading>
        <Box flex="1" />

        <Group attached>
          <SpinButton
            type="primary"
            onClickAsync={handleBackupNow}
            data-testid="plan-backup-now"
          >
            {m.plan_button_backup()}
          </SpinButton>
          <MenuRoot>
            <MenuTrigger asChild>
              <IconButton
                variant="subtle"
                colorPalette="blue"
                aria-label={m.plan_view_more_actions()}
              >
                <FiChevronDown />
              </IconButton>
            </MenuTrigger>
            <MenuContent>
              <MenuItem value="dry-run-backup" onClick={handleDryRunBackup}>
                {m.op_type_dry_run_backup()}
              </MenuItem>
              <MenuItem
                value="run-command"
                onClick={async () => {
                  const { RunCommandModal } =
                    await import("../operations/RunCommandModal");
                  showModal(<RunCommandModal repo={repo} />);
                }}
              >
                {m.op_type_run_command()}
              </MenuItem>
              <MenuItem value="unlock" onClick={handleUnlockNow}>
                {m.repo_button_unlock()}
              </MenuItem>
              <MenuItem value="clear-history" onClick={handleClearErrorHistory}>
                {m.plan_button_clear_history()}
              </MenuItem>
            </MenuContent>
          </MenuRoot>
        </Group>
      </Flex>

      <TabsRoot defaultValue="tree" lazyMount>
        <TabsList>
          <TabsTrigger value="tree" data-testid="view-tab-tree">
            {m.repo_tab_tree()}
          </TabsTrigger>
          <TabsTrigger value="list" data-testid="view-tab-list">
            {m.repo_tab_list()}
          </TabsTrigger>
        </TabsList>

        <TabsContent value="tree">
          <Box mb={2}>
            <OperationFilter
              types={FILTER_TYPES}
              activeTypes={activeTypes}
              onToggle={toggleType}
              onSelectAll={selectAll}
              onSelectNone={selectNone}
            />
          </Box>
          <OperationTreeView
            req={create(GetOperationsRequestSchema, {
              selector: {
                instanceId: config?.instance,
                repoGuid: repo.guid,
                planId: plan.id!,
              },
              lastN: BigInt(MAX_OPERATION_HISTORY),
            })}
            isPlanView={true}
            filter={treeFilter}
          />
        </TabsContent>

        <TabsContent value="list">
          <Flex mb={4} align="center" justify="space-between" wrap="wrap">
            <Heading size="md">{m.repo_history_title()}</Heading>
            <OperationFilter
              types={FILTER_TYPES}
              activeTypes={activeTypes}
              onToggle={toggleType}
              onSelectAll={selectAll}
              onSelectNone={selectNone}
            />
          </Flex>
          <OperationListView
            req={create(GetOperationsRequestSchema, {
              selector: {
                instanceId: config?.instance,
                repoGuid: repo.guid,
                planId: plan.id!,
              },
              lastN: BigInt(MAX_OPERATION_HISTORY),
            })}
            filter={listFilter}
            showDelete={true}
          />
        </TabsContent>
      </TabsRoot>
    </Box>
  );
};
