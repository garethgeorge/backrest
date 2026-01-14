import React, { Suspense, useContext, useEffect, useState } from "react";
import { Repo } from "../../../gen/ts/v1/config_pb";
import { Button } from "../../components/ui/button";
import { Flex, Heading, Box, Group, IconButton } from "@chakra-ui/react";
import { FiChevronDown } from "react-icons/fi";
import {
  Tabs,
  TabsRoot,
  TabsList,
  TabsTrigger,
  TabsContent,
} from "../../components/ui/tabs";
import {
  MenuContent,
  MenuItem,
  MenuRoot,
  MenuTrigger,
} from "../../components/ui/menu";
import { Tooltip } from "../../components/ui/tooltip";
import { OperationListView } from "../operations/OperationListView";
import { OperationTreeView } from "../operations/OperationTreeView";
import {
  MAX_OPERATION_HISTORY,
  STATS_OPERATION_HISTORY,
} from "../../constants";
import {
  DoRepoTaskRequest_Task,
  DoRepoTaskRequestSchema,
  GetOperationsRequestSchema,
  OpSelectorSchema,
} from "../../../gen/ts/v1/service_pb";
import { backrestService } from "../../api/client";
import { SpinButton } from "../../components/common/SpinButton";
import { useConfig } from "../../app/provider";
import { formatErrorAlert, alerts } from "../../components/common/Alerts";
import { useShowModal } from "../../components/common/ModalManager";
import { create } from "@bufbuild/protobuf";
import { RepoProps } from "../../state/peerStates";
import * as m from "../../paraglide/messages";

const StatsPanel = React.lazy(() => import("../dashboard/StatsPanel"));

export const RepoView = ({
  repo,
}: React.PropsWithChildren<{ repo: RepoProps }>) => {
  const [config, _] = useConfig();
  const showModal = useShowModal();

  // Task handlers
  const handleIndexNow = async () => {
    try {
      await backrestService.doRepoTask(
        create(DoRepoTaskRequestSchema, {
          repoId: repo.id!,
          task: DoRepoTaskRequest_Task.INDEX_SNAPSHOTS,
        }),
      );
    } catch (e: any) {
      alerts.error(formatErrorAlert(e, m.repo_error_index()));
    }
  };

  const handleUnlockNow = async () => {
    try {
      alerts.info(m.repo_info_unlocking());
      await backrestService.doRepoTask(
        create(DoRepoTaskRequestSchema, {
          repoId: repo.id!,
          task: DoRepoTaskRequest_Task.UNLOCK,
        }),
      );
      alerts.success(m.repo_success_unlocked());
    } catch (e: any) {
      alerts.error(m.repo_error_unlock() + e.message);
    }
  };

  const handleStatsNow = async () => {
    try {
      await backrestService.doRepoTask(
        create(DoRepoTaskRequestSchema, {
          repoId: repo.id!,
          task: DoRepoTaskRequest_Task.STATS,
        }),
      );
    } catch (e: any) {
      alerts.error(formatErrorAlert(e, m.repo_error_stats()));
    }
  };

  const handlePruneNow = async () => {
    try {
      await backrestService.doRepoTask(
        create(DoRepoTaskRequestSchema, {
          repoId: repo.id!,
          task: DoRepoTaskRequest_Task.PRUNE,
        }),
      );
    } catch (e: any) {
      alerts.error(formatErrorAlert(e, m.repo_error_prune()));
    }
  };

  const handleCheckNow = async () => {
    try {
      await backrestService.doRepoTask(
        create(DoRepoTaskRequestSchema, {
          repoId: repo.id!,
          task: DoRepoTaskRequest_Task.CHECK,
        }),
      );
    } catch (e: any) {
      alerts.error(formatErrorAlert(e, m.repo_error_check()));
    }
  };

  // Gracefully handle deletions by checking if the plan is still in the config.
  let repoInConfig = config?.repos?.find((r) => r.id === repo.id);
  if (!repoInConfig) {
    return (
      <Box>
        {m.repo_deleted_message()}
        <Box as="pre" p={2} bg="gray.100" borderRadius="md" overflowX="auto">
          {JSON.stringify(config, null, 2)}
        </Box>
      </Box>
    );
  }
  repo = repoInConfig;

  return (
    <Box>
      <Flex gap={4} align="center" wrap="wrap" mb={4}>
        <Heading size="xl">{repo.id}</Heading>
        <Box flex="1" />

        <Group attached>
          <SpinButton type="primary" onClickAsync={handleIndexNow}>
            {m.repo_button_index()}
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
              <MenuItem value="prune" onClick={handlePruneNow}>
                {m.repo_button_prune()}
              </MenuItem>
              <MenuItem value="check" onClick={handleCheckNow}>
                {m.repo_button_check()}
              </MenuItem>
              <MenuItem value="stats" onClick={handleStatsNow}>
                {m.repo_button_stats()}
              </MenuItem>
            </MenuContent>
          </MenuRoot>
        </Group>
      </Flex>

      <TabsRoot defaultValue="tree" lazyMount>
        <TabsList>
          <TabsTrigger value="tree">{m.repo_tab_tree()}</TabsTrigger>
          <TabsTrigger value="list">{m.repo_tab_list()}</TabsTrigger>
          <TabsTrigger value="stats">{m.repo_tab_stats()}</TabsTrigger>
        </TabsList>

        <TabsContent value="tree">
          <OperationTreeView
            req={create(GetOperationsRequestSchema, {
              selector: {
                repoGuid: repo.guid,
              },
              lastN: BigInt(MAX_OPERATION_HISTORY),
            })}
          />
        </TabsContent>

        <TabsContent value="list">
          <Heading size="md" mb={4}>
            {m.repo_history_title()}
          </Heading>
          <OperationListView
            req={create(GetOperationsRequestSchema, {
              selector: {
                repoGuid: repo.guid,
              },
              lastN: BigInt(MAX_OPERATION_HISTORY),
            })}
            showPlan={true}
            showDelete={true}
          />
        </TabsContent>

        <TabsContent value="stats">
          <Suspense fallback={<div>{m.loading()}</div>}>
            <StatsPanel
              selector={create(OpSelectorSchema, {
                repoGuid: repo.guid,
                instanceId: config?.instance,
              })}
            />
          </Suspense>
        </TabsContent>
      </TabsRoot>
    </Box>
  );
};
