import React, { Suspense, useContext, useEffect, useState } from "react";
import { Repo } from "../../gen/ts/v1/config_pb";
import { Flex, Tabs, Tooltip, Typography, Button } from "antd";
import { OperationListView } from "../components/OperationListView";
import { OperationTreeView } from "../components/OperationTreeView";
import { MAX_OPERATION_HISTORY, STATS_OPERATION_HISTORY } from "../constants";
import {
  DoRepoTaskRequest_Task,
  DoRepoTaskRequestSchema,
  GetOperationsRequestSchema,
  OpSelectorSchema,
} from "../../gen/ts/v1/service_pb";
import { backrestService } from "../api";
import { SpinButton } from "../components/SpinButton";
import { useConfig } from "../components/ConfigProvider";
import { formatErrorAlert, useAlertApi } from "../components/Alerts";
import { useShowModal } from "../components/ModalManager";
import { create } from "@bufbuild/protobuf";
import { RepoProps } from "../state/peerstates";
import * as m from "../paraglide/messages";

const StatsPanel = React.lazy(() => import("../components/StatsPanel"));

export const RepoView = ({
  repo,
}: React.PropsWithChildren<{ repo: RepoProps }>) => {
  const [config, _] = useConfig();
  const showModal = useShowModal();
  const alertsApi = useAlertApi()!;

  // Task handlers
  const handleIndexNow = async () => {
    try {
      await backrestService.doRepoTask(
        create(DoRepoTaskRequestSchema, {
          repoId: repo.id!,
          task: DoRepoTaskRequest_Task.INDEX_SNAPSHOTS,
        })
      );
    } catch (e: any) {
      alertsApi.error(formatErrorAlert(e, m.repo_error_index()));
    }
  };

  const handleUnlockNow = async () => {
    try {
      alertsApi.info(m.repo_info_unlocking());
      await backrestService.doRepoTask(
        create(DoRepoTaskRequestSchema, {
          repoId: repo.id!,
          task: DoRepoTaskRequest_Task.UNLOCK,
        })
      );
      alertsApi.success(m.repo_success_unlocked());
    } catch (e: any) {
      alertsApi.error(m.repo_error_unlock() + e.message);
    }
  };

  const handleStatsNow = async () => {
    try {
      await backrestService.doRepoTask(
        create(DoRepoTaskRequestSchema, {
          repoId: repo.id!,
          task: DoRepoTaskRequest_Task.STATS,
        })
      );
    } catch (e: any) {
      alertsApi.error(formatErrorAlert(e, m.repo_error_stats()));
    }
  };

  const handlePruneNow = async () => {
    try {
      await backrestService.doRepoTask(
        create(DoRepoTaskRequestSchema, {
          repoId: repo.id!,
          task: DoRepoTaskRequest_Task.PRUNE,
        })
      );
    } catch (e: any) {
      alertsApi.error(formatErrorAlert(e, m.repo_error_prune()));
    }
  };

  const handleCheckNow = async () => {
    try {
      await backrestService.doRepoTask(
        create(DoRepoTaskRequestSchema, {
          repoId: repo.id!,
          task: DoRepoTaskRequest_Task.CHECK,
        })
      );
    } catch (e: any) {
      alertsApi.error(formatErrorAlert(e, m.repo_error_check()));
    }
  };

  // Gracefully handle deletions by checking if the plan is still in the config.
  let repoInConfig = config?.repos?.find((r) => r.id === repo.id);
  if (!repoInConfig) {
    return (
      <>
        {m.repo_deleted_message()}
        <pre>{JSON.stringify(config, null, 2)}</pre>
      </>
    );
  }
  repo = repoInConfig;

  const items = [
    {
      key: "1",
      label: m.repo_tab_tree(),
      children: (
        <>
          <OperationTreeView
            req={create(GetOperationsRequestSchema, {
              selector: {
                repoGuid: repo.guid,
              },
              lastN: BigInt(MAX_OPERATION_HISTORY),
            })}
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
           <h3>{m.repo_history_title()}</h3>
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
        </>
      ),
      destroyOnHidden: true,
    },
    {
      key: "3",
      label: m.repo_tab_stats(),
      children: (
        <Suspense fallback={<div>{m.loading()}</div>}>
          <StatsPanel
            selector={create(OpSelectorSchema, {
              repoGuid: repo.guid,
              instanceId: config?.instance,
            })}
          />
        </Suspense>
      ),
      destroyOnHidden: true,
    },
  ];
  return (
    <>
      <Flex gap="small" align="center" wrap="wrap">
        <Typography.Title>{repo.id}</Typography.Title>
      </Flex>
      <Flex gap="small" align="center" wrap="wrap">
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

        <Tooltip title={m.repo_tooltip_index()}>
          <SpinButton type="default" onClickAsync={handleIndexNow}>
            {m.repo_button_index()}
          </SpinButton>
        </Tooltip>

        <Tooltip title={m.repo_tooltip_unlock()}>
          <SpinButton type="default" onClickAsync={handleUnlockNow}>
            {m.repo_button_unlock()}
          </SpinButton>
        </Tooltip>

        <Tooltip title={m.repo_tooltip_prune()}>
          <SpinButton type="default" onClickAsync={handlePruneNow}>
            {m.repo_button_prune()}
          </SpinButton>
        </Tooltip>

        <Tooltip title={m.repo_tooltip_check()}>
          <SpinButton type="default" onClickAsync={handleCheckNow}>
            {m.repo_button_check()}
          </SpinButton>
        </Tooltip>

        <Tooltip title={m.repo_tooltip_stats()}>
          <SpinButton type="default" onClickAsync={handleStatsNow}>
            {m.repo_button_stats()}
          </SpinButton>
        </Tooltip>
      </Flex>
      <Tabs defaultActiveKey={items[0].key} items={items} />
    </>
  );
};
