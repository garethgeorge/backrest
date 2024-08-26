import { Button, Input, Modal, Space } from "antd";
import React from "react";
import { useShowModal } from "../components/ModalManager";
import { backrestService } from "../api";
import { SpinButton } from "../components/SpinButton";
import { ConnectError } from "@connectrpc/connect";
import { useAlertApi } from "../components/Alerts";

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
  const [invocations, setInvocations] = React.useState<Invocation[]>([]);
  const [abortController, setAbortController] = React.useState<
    AbortController | undefined
  >();

  const handleCancel = () => {
    if (abortController) {
      alertApi.warning("In-progress restic command was aborted");
      abortController.abort();
    }
    showModal(null);
  };

  const doExecute = async () => {
    if (running) return;
    setRunning(true);

    const newInvocation = { command, output: "", error: "" };
    setInvocations((invocations) => [newInvocation, ...invocations]);

    let segments: string[] = [];

    try {
      const abortController = new AbortController();
      setAbortController(abortController);

      for await (const bytes of backrestService.runCommand(
        {
          repoId,
          command,
        },
        {
          signal: abortController.signal,
        }
      )) {
        const output = new TextDecoder("utf-8").decode(bytes.value);
        segments.push(output);
        setInvocations((invocations) => {
          const copy = [...invocations];
          copy[0] = {
            ...copy[0],
            output: segments.join(""),
          };
          return copy;
        });
      }
    } catch (e: any) {
      setInvocations((invocations) => {
        const copy = [...invocations];
        copy[0] = {
          ...copy[0],
          error: (e as Error).message,
        };
        return copy;
      });
    } finally {
      setRunning(false);
      setAbortController(undefined);
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
      {invocations.map((invocation, i) => (
        <div key={i}>
          {invocation.output ? <pre>{invocation.output}</pre> : null}
          {invocation.error ? (
            <pre style={{ color: "red" }}>{invocation.error}</pre>
          ) : null}
        </div>
      ))}
    </Modal>
  );
};
