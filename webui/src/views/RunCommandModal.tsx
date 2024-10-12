import { Button, Input, Modal, Space } from "antd";
import React from "react";
import { useShowModal } from "../components/ModalManager";
import { backrestService } from "../api";
import { SpinButton } from "../components/SpinButton";
import { ConnectError } from "@connectrpc/connect";
import { useAlertApi } from "../components/Alerts";
import {
  GetOperationsRequest,
  RunCommandRequest,
} from "../../gen/ts/v1/service_pb";
import { OperationList } from "../components/OperationList";

interface Invocation {
  command: string;
  output: string;
  error: string;
}

export const RunCommandModal = ({ repoId }: { repoId: string }) => {
  const showModal = useShowModal();
  const alertApi = useAlertApi()!;
  const [command, setCommand] = React.useState("");
  const [running, setRunning] = React.useState(false);

  const handleCancel = () => {
    showModal(null);
  };

  const doExecute = async () => {
    if (!command) return;
    setRunning(true);

    const toRun = command.trim();
    setCommand("");

    try {
      const opID = await backrestService.runCommand(
        new RunCommandRequest({
          repoId,
          command: toRun,
        })
      );
    } catch (e: any) {
      alertApi.error("Command failed: " + e.message);
    } finally {
      setRunning(false);
    }
  };

  return (
    <Modal
      open={true}
      onCancel={handleCancel}
      title={"Run Command in repo " + repoId}
      width="80vw"
      footer={[]}
    >
      <Space.Compact style={{ width: "100%" }}>
        <Input
          placeholder="Run a restic comamnd e.g. 'help' to print help text"
          value={command}
          onChange={(e) => setCommand(e.target.value)}
          onKeyUp={(e) => {
            if (e.key === "Enter") {
              doExecute();
            }
          }}
        />
        <SpinButton type="primary" onClickAsync={doExecute}>
          Execute
        </SpinButton>
      </Space.Compact>
      {running && command ? (
        <em style={{ color: "gray" }}>
          Warning: another command is already running. Wait for it to finish
          before running another operation that requires the repo lock.
        </em>
      ) : null}
      <OperationList
        req={
          new GetOperationsRequest({
            selector: {
              repoId: repoId,
              planId: "_system_", // run commands are not associated with a plan
            },
          })
        }
        filter={(op) => op.op.case === "operationRunCommand"}
      />
    </Modal>
  );
};
