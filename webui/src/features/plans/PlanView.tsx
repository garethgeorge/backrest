import React, { useEffect, useState } from "react";
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
} from "../../../gen/ts/v1/service_pb";
import { StringValueSchema } from "../../../gen/ts/types/value_pb";
import { SpinButton } from "../../components/common/SpinButton";
import { useShowModal } from "../../components/common/ModalManager";
import { create } from "@bufbuild/protobuf";
import { useConfig } from "../../app/provider";
import { OperationListView } from "../operations/OperationListView";
import { OperationTreeView } from "../operations/OperationTreeView";
import * as m from "../../paraglide/messages";

export const PlanView = ({ plan }: React.PropsWithChildren<{ plan: Plan }>) => {
  const [config, _] = useConfig();
  const showModal = useShowModal();
  const repo = config?.repos.find((r) => r.id === plan.repo);

  const handleBackupNow = async () => {
    try {
      await backrestService.backup({ value: plan.id });
      alerts.success(m.plan_backup_scheduled());
    } catch (e: any) {
      alerts.error(m.plan_error_backup() + e.message);
    }
  };

  const handleDryRunBackup = async () => {
    try {
      await backrestService.dryRunBackup(
        create(StringValueSchema, { value: plan.id })
      );
      alerts.success(m.plan_dry_run_success());
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
          <SpinButton type="primary" onClickAsync={handleBackupNow}>
            {m.plan_button_backup()}
          </SpinButton>
          <MenuRoot>
            <MenuTrigger asChild>
              <IconButton
                variant="subtle"
                colorPalette="blue"
                aria-label="More actions"
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
                {m.repo_button_run_command()}
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
          <TabsTrigger value="tree">{m.repo_tab_tree()}</TabsTrigger>
          <TabsTrigger value="list">{m.repo_tab_list()}</TabsTrigger>
        </TabsList>

        <TabsContent value="tree">
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
          />
        </TabsContent>

        <TabsContent value="list">
          <Heading size="md" mb={4}>
            {m.repo_history_title()}
          </Heading>
          <OperationListView
            req={create(GetOperationsRequestSchema, {
              selector: {
                instanceId: config?.instance,
                repoGuid: repo.guid,
                planId: plan.id!,
              },
              lastN: BigInt(MAX_OPERATION_HISTORY),
            })}
            showDelete={true}
          />
        </TabsContent>
      </TabsRoot>
    </Box>
  );
};
