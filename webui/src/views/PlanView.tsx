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
import {
  Operation,
  OperationEvent,
  OperationEventType,
} from "../../gen/ts/v1/operations.pb";
import { operationEmitter } from "../state/oplog";

export const PlanView = ({ plan }: React.PropsWithChildren<{ plan: Plan }>) => {
  const showModal = useShowModal();
  const alertsApi = useAlertApi()!;
  const [operations, setOperations] = useState<Operation[]>([]);

  useEffect(() => {
    (async () => {
      try {
        const ops = await ResticUI.GetOperations(
          { planId: plan.id },
          { pathPrefix: "/api" }
        );
        if (!ops.operations) throw new Error("No operations returned");
        setOperations(ops.operations);
      } catch (e: any) {
        alertsApi.error("Failed to fetch operations: " + e.message);
      }
    })();

    const listener = (opEvent: OperationEvent) => {
      setOperations((operations) => {
        if (opEvent.type === OperationEventType.EVENT_CREATED) {
          operations.push(opEvent.operation!);
        } else if (opEvent.type === OperationEventType.EVENT_UPDATED) {
          // We iterate from the back since the most recent operations are at the end and
          // only recent ops receive updates.
          for (let i = operations.length - 1; i >= 0; i--) {
            if (operations[i].id === opEvent.operation?.id) {
              operations[i] = opEvent.operation!;
              break;
            }
          }
        }
        return operations;
      });
    };

    operationEmitter.on("operation", listener);

    return () => {
      operationEmitter.removeListener("operation", listener);
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
      <OperationsPanel operations={operations} />
    </>
  );
};

const OperationsPanel = ({ operations }: { operations: Operation[] }) => {
  return (
    <>
      <h2>Operations List</h2>
      {operations.map((op) => {
        return (
          <div key={op.id}>
            <h3>{op.id}</h3>
          </div>
        );
      })}
    </>
  );
};
