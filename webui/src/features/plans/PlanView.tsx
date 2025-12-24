import React, { useEffect, useState } from "react";
import { Plan } from "../../../gen/ts/v1/config_pb";
import { Button } from "../../components/ui/button";
import { Flex, Heading, Text, Box } from "@chakra-ui/react";
import {
  Tabs,
  TabsList,
  TabsTrigger,
  TabsContent,
  TabsRoot,
} from "../../components/ui/tabs";
import { Tooltip } from "../../components/ui/tooltip";
import { useAlertApi } from "../../components/common/Alerts";
import { MAX_OPERATION_HISTORY } from "../../constants";
import { backrestService } from "../../api/client";
import {
  ClearHistoryRequestSchema,
  DoRepoTaskRequest_Task,
  DoRepoTaskRequestSchema,
  GetOperationsRequestSchema,
} from "../../../gen/ts/v1/service_pb";
import { SpinButton } from "../../components/common/SpinButton";
import { useShowModal } from "../../components/common/ModalManager";
import { create } from "@bufbuild/protobuf";
import { useConfig } from "../../app/provider";
import { OperationListView } from "../operations/OperationListView";
import { OperationTreeView } from "../operations/OperationTreeView";
import * as m from "../../paraglide/messages";

export const PlanView = ({ plan }: React.PropsWithChildren<{ plan: Plan }>) => {
  const [config, _] = useConfig();
  const alertsApi = useAlertApi()!;
  const showModal = useShowModal();
  const repo = config?.repos.find((r) => r.id === plan.repo);

  const handleBackupNow = async () => {
    try {
      await backrestService.backup({ value: plan.id });
      alertsApi.success(m.plan_backup_scheduled());
    } catch (e: any) {
      alertsApi.error(m.plan_error_backup() + e.message);
    }
  };

  const handleUnlockNow = async () => {
    try {
      alertsApi.info(m.repo_info_unlocking());
      await backrestService.doRepoTask(
        create(DoRepoTaskRequestSchema, {
          repoId: plan.repo!,
          task: DoRepoTaskRequest_Task.UNLOCK,
        }),
      );
      alertsApi.success(m.repo_success_unlocked());
    } catch (e: any) {
      alertsApi.error(m.repo_error_unlock() + e.message);
    }
  };

  const handleClearErrorHistory = async () => {
    try {
      alertsApi.info(m.plan_clearing_history());
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
      alertsApi.success(m.plan_history_cleared());
    } catch (e: any) {
      alertsApi.error(m.plan_error_clear_history() + e.message);
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

        <SpinButton type="primary" onClickAsync={handleBackupNow}>
          {m.plan_button_backup()}
        </SpinButton>

        <Tooltip content={m.repo_tooltip_run_command()}>
          <Button
            variant="outline"
            onClick={async () => {
              const { RunCommandModal } =
                await import("../operations/RunCommandModal");
              showModal(<RunCommandModal repo={repo} />);
            }}
          >
            {m.repo_button_run_command()}
          </Button>
        </Tooltip>

        <Tooltip content={m.repo_tooltip_unlock()}>
          <SpinButton type="default" onClickAsync={handleUnlockNow}>
            {m.repo_button_unlock()}
          </SpinButton>
        </Tooltip>

        <Tooltip content={m.plan_tooltip_clear_history()}>
          <SpinButton type="default" onClickAsync={handleClearErrorHistory}>
            {m.plan_button_clear_history()}
          </SpinButton>
        </Tooltip>
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
