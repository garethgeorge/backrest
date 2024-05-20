import React, { Suspense, useContext, useEffect, useState } from "react";
import { Repo } from "../../gen/ts/v1/config_pb";
import { Flex, Tabs, Tooltip, Typography, Button } from "antd";
import { OperationList } from "../components/OperationList";
import { OperationTree } from "../components/OperationTree";
import { MAX_OPERATION_HISTORY, STATS_OPERATION_HISTORY } from "../constants";
import { GetOperationsRequest, OpSelector } from "../../gen/ts/v1/service_pb";
import { shouldHideStatus } from "../state/oplog";
import { backrestService } from "../api";
import { StringValue } from "@bufbuild/protobuf";
import { SpinButton } from "../components/SpinButton";
import { useConfig } from "../components/ConfigProvider";
import { useAlertApi } from "../components/Alerts";
import { useShowModal } from "../components/ModalManager";

const StatsPanel = React.lazy(() => import("../components/StatsPanel"));

export const RepoView = ({ repo }: React.PropsWithChildren<{ repo: Repo }>) => {
  const [config, setConfig] = useConfig();
  const showModal = useShowModal();
  const alertsApi = useAlertApi()!;

  // Task handlers
  const handleIndexNow = async () => {
    try {
      await backrestService.indexSnapshots(
        new StringValue({ value: repo.id! })
      );
    } catch (e: any) {
      alertsApi.error("Failed to index snapshots: " + e.message);
    }
  };

  const handleStatsNow = async () => {
    try {
      await backrestService.stats(new StringValue({ value: repo.id! }));
    } catch (e: any) {
      alertsApi.error("Failed to compute stats: " + e.message);
    }
  };

  const handlePruneNow = async () => {
    try {
      await backrestService.prune({ value: repo.id });
    } catch (e: any) {
      alertsApi.error("Failed to prune: " + e.message);
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
          <h3>Browse Backups</h3>
          <OperationTree
            req={
              new GetOperationsRequest({
                selector: new OpSelector({
                  repoId: repo.id!,
                }),
                lastN: BigInt(MAX_OPERATION_HISTORY),
              })
            }
          />
        </>
      ),
      destroyInactiveTabPane: true,
    },
    {
      key: "2",
      label: "Operation List",
      children: (
        <>
          <h3>Backup Action History</h3>
          <OperationList
            req={
              new GetOperationsRequest({
                selector: new OpSelector({
                  repoId: repo.id!,
                }),
                lastN: BigInt(MAX_OPERATION_HISTORY),
              })
            }
            showPlan={true}
            filter={(op) => !shouldHideStatus(op.status)}
          />
        </>
      ),
      destroyInactiveTabPane: true,
    },
    {
      key: "3",
      label: "Stats",
      children: (
        <Suspense fallback={<div>Loading...</div>}>
          <StatsPanel repoId={repo.id!} />
        </Suspense>
      ),
      destroyInactiveTabPane: true,
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
              showModal(<RunCommandModal repoId={repo.id!} />);
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

        <Tooltip title="Runs a prune operation on the repository that will remove old snapshots and free up space">
          <SpinButton type="default" onClickAsync={handlePruneNow}>
            Prune Now
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
