import React, { useEffect, useState } from "react";
import { Plan } from "../../gen/ts/v1/config.pb";
import { Button, Flex } from "antd";
import { SettingOutlined } from "@ant-design/icons";
import { AddPlanModal } from "./AddPlanModel";
import { useShowModal } from "../components/ModalManager";
import { useRecoilValue } from "recoil";
import { configState } from "../state/config";
import { useAlertApi } from "../components/Alerts";
import { ResticUI } from "../../gen/ts/v1/service.pb";
import { Operation } from "../../gen/ts/v1/operations.pb";
import {
  buildOperationListListener,
  subscribeToOperations,
  unsubscribeFromOperations,
} from "../state/oplog";
import { OperationList } from "../components/OperationList";

export const PlanView = ({ plan }: React.PropsWithChildren<{ plan: Plan }>) => {
  const showModal = useShowModal();
  const alertsApi = useAlertApi()!;
  const [operations, setOperations] = useState<Operation[]>([]);

  useEffect(() => {
    const listener = buildOperationListListener(
      { planId: plan.id, lastN: "100" },
      (event, operations) => {
        setOperations([...operations]);
      }
    );
    subscribeToOperations(listener);

    return () => {
      unsubscribeFromOperations(listener);
    };
  }, [plan.id]);

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
    alertsApi.warning("Not implemented yet :(");
  };

  return (
    <>
      <Flex gap="small" align="center" wrap="wrap">
        <h1>{plan.id}</h1>
        <Button
          type="text"
          size="small"
          shape="circle"
          icon={<SettingOutlined />}
          onClick={() => {
            showModal(<AddPlanModal template={plan} />);
          }}
        />
      </Flex>
      <Flex gap="small" align="center" wrap="wrap">
        <Button type="primary" onClick={handleBackupNow}>
          Backup Now
        </Button>
        <Button type="primary" onClick={handlePruneNow}>
          Prune Now
        </Button>
      </Flex>
      <h2>Operations List</h2>
      <OperationList operations={operations} />
    </>
  );
};
