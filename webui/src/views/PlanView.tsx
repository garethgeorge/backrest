import React, { useEffect, useState } from "react";
import { Plan } from "../../gen/ts/v1/config_pb";
import { Button, Flex, Tabs, Tooltip, Typography } from "antd";
import { useAlertApi } from "../components/Alerts";
import { MAX_OPERATION_HISTORY } from "../constants";
import { backrestService } from "../api";
import {
  ClearHistoryRequestSchema,
  DoRepoTaskRequest_Task,
  DoRepoTaskRequestSchema,
  GetOperationsRequestSchema,
} from "../../gen/ts/v1/service_pb";
import { SpinButton } from "../components/SpinButton";
import { useShowModal } from "../components/ModalManager";
import { create } from "@bufbuild/protobuf";
import { useConfig } from "../components/ConfigProvider";
import { OperationListView } from "../components/OperationListView";
import { OperationTreeView } from "../components/OperationTreeView";
import * as m from "../paraglide/messages";

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
        })
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
        })
      );
      alertsApi.success(m.plan_history_cleared());
    } catch (e: any) {
      alertsApi.error(m.plan_error_clear_history() + e.message);
    }
  };

  if (!repo) {
    return (
      <>
        <Typography.Title>
          {m.plan_repo_not_found({ repo: plan.repo!, planId: plan.id! })}
        </Typography.Title>
      </>
    );
  }

  return (
    <>
      <Flex gap="small" align="center" wrap="wrap">
        <Typography.Title>{plan.id}</Typography.Title>
      </Flex>
      <Flex gap="small" align="center" wrap="wrap">
        <SpinButton type="primary" onClickAsync={handleBackupNow}>
          {m.plan_button_backup()}
        </SpinButton>
        <Tooltip title={m.repo_tooltip_run_command()}>
          <Button
            type="default"
            onClick={async () => {
              const { RunCommandModal } = await import("./RunCommandModal");
              showModal(<RunCommandModal repo={repo} />);
            }}
          >
            {m.repo_button_run_command()}
          </Button>
        </Tooltip>
        <Tooltip title={m.repo_tooltip_unlock()}>
          <SpinButton type="default" onClickAsync={handleUnlockNow}>
            {m.repo_button_unlock()}
          </SpinButton>
        </Tooltip>
        <Tooltip title={m.plan_tooltip_clear_history()}>
          <SpinButton type="default" onClickAsync={handleClearErrorHistory}>
            {m.plan_button_clear_history()}
          </SpinButton>
        </Tooltip>
      </Flex>
      <Tabs
        defaultActiveKey="1"
        items={[
          {
            key: "1",
            label: m.repo_tab_tree(),
            children: (
              <>
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
              </>
            ),
            destroyOnHidden: true,
          },
          {
            key: "2",
            label: m.repo_tab_list(),
            children: (
              <>
                <h2>{m.repo_history_title()}</h2>
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
              </>
            ),
            destroyOnHidden: true,
          },
        ]}
      />
    </>
  );
};
