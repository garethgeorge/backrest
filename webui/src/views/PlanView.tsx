import React, { useEffect, useState } from "react";
import { Plan } from "../../gen/ts/v1/config_pb";
import { Flex, Tabs, Tooltip, Typography } from "antd";
import { useAlertApi } from "../components/Alerts";
import { OperationList } from "../components/OperationList";
import { OperationTree } from "../components/OperationTree";
import { MAX_OPERATION_HISTORY } from "../constants";
import { backrestService } from "../api";
import { GetOperationsRequest } from "../../gen/ts/v1/service_pb";
import { SpinButton } from "../components/SpinButton";

export const PlanView = ({ plan }: React.PropsWithChildren<{ plan: Plan }>) => {
  const alertsApi = useAlertApi()!;

  const handleBackupNow = async () => {
    try {
      await backrestService.backup({ value: plan.id });
      alertsApi.success("Backup scheduled.");
    } catch (e: any) {
      alertsApi.error("Failed to schedule backup: " + e.message);
    }
  };

  const handlePruneNow = async () => {
    try {
      await backrestService.prune({ value: plan.id });
      alertsApi.success("Prune scheduled.");
    } catch (e: any) {
      alertsApi.error("Failed to schedule prune: " + e.message);
    }
  };

  const handleUnlockNow = async () => {
    try {
      alertsApi.info("Unlocking repo...");
      await backrestService.unlock({ value: plan.repo! });
      alertsApi.success("Repo unlocked.");
    } catch (e: any) {
      alertsApi.error("Failed to unlock repo: " + e.message);
    }
  };

  const handleClearErrorHistory = async () => {
    try {
      alertsApi.info("Clearing error history...");
      await backrestService.clearHistory({ planId: plan.id, onlyFailed: true });
      alertsApi.success("Error history cleared.");
    } catch (e: any) {
      alertsApi.error("Failed to clear error history: " + e.message);
    }
  }

  return (
    <>
      <Flex gap="small" align="center" wrap="wrap">
        <Typography.Title>
          {plan.id}
        </Typography.Title>
      </Flex>
      <Flex gap="small" align="center" wrap="wrap">
        <SpinButton type="primary" onClickAsync={handleBackupNow}>
          Backup Now
        </SpinButton>
        <Tooltip title="Runs a prune operation on the repository that will remove old snapshots and free up space">
          <SpinButton type="default" onClickAsync={handlePruneNow}>
            Prune Now
          </SpinButton>
        </Tooltip>
        <Tooltip title="Removes lockfiles and checks the repository for errors. Only run if you are sure the repo is not being accessed by another system">
          <SpinButton type="default" onClickAsync={handleUnlockNow}>
            Unlock Repo
          </SpinButton>
        </Tooltip>
        <Tooltip title="Removes failed operations from the list">
          <SpinButton type="default" onClickAsync={handleClearErrorHistory}>
            Clear Error History
          </SpinButton>
        </Tooltip>
      </Flex>
      <Tabs
        defaultActiveKey="1"
        items={[
          {
            key: "1",
            label: "Tree View",
            children: (
              <>
                <OperationTree
                  req={new GetOperationsRequest({ planId: plan.id!, lastN: BigInt(MAX_OPERATION_HISTORY) })}
                />
              </>
            ),
          },
          {
            key: "2",
            label: "Operation List",
            children: (
              <>
                <h2>Backup Action History</h2>
                <OperationList
                  req={new GetOperationsRequest({ planId: plan.id!, lastN: BigInt(MAX_OPERATION_HISTORY) })}
                />
              </>
            ),
          },
        ]}
      />
    </>
  );
};
