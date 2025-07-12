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
import { SyncRepoMetadata } from "../../gen/ts/v1/syncservice_pb";

const StatsPanel = React.lazy(() => import("../components/StatsPanel"));

// Type intersection to combine properties from Repo and SyncRepoMetadata
interface RepoProps {
  id: string;
  guid: string;
}

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
      alertsApi.error(formatErrorAlert(e, "Failed to index snapshots: "));
    }
  };

  const handleUnlockNow = async () => {
    try {
      alertsApi.info("Unlocking repo...");
      await backrestService.doRepoTask(
        create(DoRepoTaskRequestSchema, {
          repoId: repo.id!,
          task: DoRepoTaskRequest_Task.UNLOCK,
        })
      );
      alertsApi.success("Repo unlocked.");
    } catch (e: any) {
      alertsApi.error("Failed to unlock repo: " + e.message);
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
      alertsApi.error(formatErrorAlert(e, "Failed to compute stats: "));
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
      alertsApi.error(formatErrorAlert(e, "Failed to prune: "));
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
      alertsApi.error(formatErrorAlert(e, "Failed to check: "));
    }
  };

  // Gracefully handle deletions by checking if the plan is still in the config.
  let repoInConfig = config?.repos?.find((r) => r.id === repo.id);
  if (!repoInConfig) {
    return (
      <>
        Repo was deleted
        <pre>{JSON.stringify(config, null, 2)}</pre>
      </>
    );
  }
  repo = repoInConfig;

  const items = [
    {
      key: "1",
      label: "Tree View",
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
      label: "List View",
      children: (
        <>
          <h3>Backup Action History</h3>
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
      label: "Stats",
      children: (
        <Suspense fallback={<div>Loading...</div>}>
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
        <Tooltip title="Advanced users: open a restic shell to run commands on the repository. Re-index snapshots to reflect any changes in Backrest.">
          <Button
            type="default"
            onClick={async () => {
              const { RunCommandModal } = await import("./RunCommandModal");
              showModal(<RunCommandModal repo={repo} />);
            }}
          >
            Run Command
          </Button>
        </Tooltip>

        <Tooltip title="Indexes the snapshots in the repository. Snapshots are also indexed automatically after each backup.">
          <SpinButton type="default" onClickAsync={handleIndexNow}>
            Index Snapshots
          </SpinButton>
        </Tooltip>

        <Tooltip title="Removes lockfiles and checks the repository for errors. Only run if you are sure the repo is not being accessed by another system">
          <SpinButton type="default" onClickAsync={handleUnlockNow}>
            Unlock Repo
          </SpinButton>
        </Tooltip>

        <Tooltip title="Runs a prune operation on the repository that will remove old snapshots and free up space">
          <SpinButton type="default" onClickAsync={handlePruneNow}>
            Prune Now
          </SpinButton>
        </Tooltip>

        <Tooltip title="Runs a check operation on the repository that will verify the integrity of the repository">
          <SpinButton type="default" onClickAsync={handleCheckNow}>
            Check Now
          </SpinButton>
        </Tooltip>

        <Tooltip title="Runs restic stats on the repository, this may be a slow operation">
          <SpinButton type="default" onClickAsync={handleStatsNow}>
            Compute Stats
          </SpinButton>
        </Tooltip>
      </Flex>
      <Tabs defaultActiveKey={items[0].key} items={items} />
    </>
  );
};
