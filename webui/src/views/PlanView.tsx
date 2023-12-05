import React, { useEffect, useState } from "react";
import { Plan } from "../../gen/ts/v1/config.pb";
import { Button, Flex, Tabs, Tooltip } from "antd";
import { SettingOutlined } from "@ant-design/icons";
import { AddPlanModal } from "./AddPlanModal";
import { useShowModal } from "../components/ModalManager";
import { useRecoilValue } from "recoil";
import { configState } from "../state/config";
import { useAlertApi } from "../components/Alerts";
import { ResticUI } from "../../gen/ts/v1/service.pb";
import {
  EOperation,
  buildOperationListListener,
  subscribeToOperations,
  unsubscribeFromOperations,
} from "../state/oplog";
import { OperationList } from "../components/OperationList";
import { OperationTree } from "../components/OperationTree";
import { MAX_OPERATION_HISTORY } from "../constants";

export const PlanView = ({ plan }: React.PropsWithChildren<{ plan: Plan }>) => {
  const showModal = useShowModal();
  const alertsApi = useAlertApi()!;

  // Gracefully handle deletions by checking if the plan is still in the config.
  const config = useRecoilValue(configState);
  let planInConfig = config.plans?.find((p) => p.id === plan.id);
  if (!planInConfig) {
    return <p>Plan was deleted.</p>;
  }
  plan = planInConfig;

  const handleBackupNow = async () => {
    try {
      ResticUI.Backup({ value: plan.id }, { pathPrefix: "/api" });
      alertsApi.success("Backup scheduled.");
    } catch (e: any) {
      alertsApi.error("Failed to schedule backup: " + e.message);
    }
  };

  const handlePruneNow = () => {
    try {
      ResticUI.Prune({ value: plan.id }, { pathPrefix: "/api" });
      alertsApi.success("Prune scheduled.");
    } catch (e: any) {
      alertsApi.error("Failed to schedule prune: " + e.message);
    }
  };

  const handleUnlockNow = () => {
    try {
      alertsApi.info("Unlocking repo...");
      ResticUI.Unlock({ value: plan.repo! }, { pathPrefix: "/api" });
      alertsApi.success("Repo unlocked.");
    } catch (e: any) {
      alertsApi.error("Failed to unlock repo: " + e.message);
    }
  };

  return (
    <>
      <Flex gap="small" align="center" wrap="wrap">
        <h1>{plan.id}</h1>
      </Flex>
      <Flex gap="small" align="center" wrap="wrap">
        <Button type="primary" onClick={handleBackupNow}>
          Backup Now
        </Button>
        <Tooltip title="Runs a prune operation on the repository that will remove old snapshots and free up space">
          <Button type="default" onClick={handlePruneNow}>
            Prune Now
          </Button>
        </Tooltip>
        <Tooltip title="Removes lockfiles and checks the repository for errors. Only run if you are sure the repo is not being accessed by another system">
          <Button type="default" onClick={handleUnlockNow}>
            Unlock Repo
          </Button>
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
                  req={{ planId: plan.id!, lastN: "" + MAX_OPERATION_HISTORY }}
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
                  req={{ planId: plan.id!, lastN: "" + MAX_OPERATION_HISTORY }}
                />
              </>
            ),
          },
        ]}
      />
    </>
  );
};
